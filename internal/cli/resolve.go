package cli

import (
	"os"
	"strconv"
	"strings"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/project"
	"github.com/rapibase/cli/internal/ui"
)

// Where the project of a command came from (spec §4.5.5).
const (
	srcFlag = "flag"
	srcEnv  = "env"
	srcLink = "link"
	srcOnly = "only"
	srcPick = "pick"
)

// projectChoice is a project in an error list (not_linked, missing_flag).
type projectChoice struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Org  string `json:"org"`
	Mode string `json:"mode"`
}

// target is the project a command acts on.
type target struct {
	*api.Project
	Source string
	Link   *project.Found
}

// view is the JSON shape of the target in every command's output.
func (t *target) view() map[string]any {
	v := map[string]any{"id": t.ID, "slug": t.Slug}
	if t.Name != "" {
		v["name"] = t.Name
	}
	if t.OrgName != "" {
		v["org"] = t.OrgName
	}
	if t.Plan != "" {
		v["plan"] = t.Plan
	}
	if t.Status != "" {
		v["status"] = t.Status
	}
	if t.AppURL != "" {
		v["url"] = t.AppURL
	}
	if t.Mode != "" {
		v["mode"] = t.Mode
	}
	return v
}

func (a *app) listProjects() ([]api.Project, error) {
	if a.projects != nil {
		return a.projects, nil
	}
	var ps []api.Project
	if err := a.api.Do(a.ctx, "GET", "/api/projects", nil, &ps); err != nil {
		return nil, err
	}
	if ps == nil {
		ps = []api.Project{}
	}
	a.projects = ps
	return ps, nil
}

func choices(ps []api.Project) []projectChoice {
	out := make([]projectChoice, 0, len(ps))
	for _, p := range ps {
		out = append(out, projectChoice{p.ID, p.Slug, p.Name, p.OrgName, p.Mode})
	}
	return out
}

// findProject looks a project up by id or slug among the ones the session
// reaches. If it isn't there, it asks the server to tell 404 (gone or not
// yours) from 403 (yours, but this session can't use it, with action_url).
func (a *app) findProject(ref string) (*api.Project, error) {
	ref = strings.TrimSpace(ref)
	ps, err := a.listProjects()
	if err != nil {
		return nil, err
	}
	id, numErr := strconv.ParseInt(ref, 10, 64)
	for i := range ps {
		p := &ps[i]
		if (numErr == nil && p.ID == id) || strings.EqualFold(p.Slug, ref) || strings.EqualFold(p.Subdomain, ref) {
			return p, nil
		}
	}
	if numErr == nil {
		var p api.Project
		if err := a.api.Do(a.ctx, "GET", "/api/projects/"+ref, nil, &p); err != nil {
			e := cerr.As(err)
			if e.Code == cerr.ProjectNotFound {
				e.Message = i18n.T("project.not_found", ref)
			}
			return nil, e
		}
		return &p, nil
	}
	return nil, cerr.New(cerr.ProjectNotFound, i18n.T("project.not_found", ref)).
		With("hint", i18n.T("project.see_list")).With("projects", choices(ps))
}

// resolve finds the project of the command, in the order of spec §4.5.2:
// --project, RAPIBASE_PROJECT, the nearest .rapibase/project.json, the only
// project the session reaches, a prompt (TTY), or not_linked.
func (a *app) resolve(changing bool) (*target, error) {
	link, err := project.Find(a.cwd)
	if err != nil {
		return nil, cerr.New(cerr.Usage, err.Error())
	}
	ref, src := a.flags.project, srcFlag
	if ref == "" {
		ref, src = strings.TrimSpace(os.Getenv("RAPIBASE_PROJECT")), srcEnv
	}

	// a deploy token can't list projects: it needs a numeric id
	if a.sess.IsDeployToken() {
		if ref == "" && link != nil {
			ref, src = strconv.FormatInt(link.ProjectID, 10), srcLink
		}
		id, err := strconv.ParseInt(ref, 10, 64)
		if err != nil || id <= 0 {
			return nil, a.missing(i18n.T("project.deploy_token_id"), "--project <id>")
		}
		t := &target{Project: &api.Project{ID: id}, Source: src, Link: link}
		if link != nil && link.ProjectID == id {
			t.Slug = link.Slug
		}
		if t.Slug == "" {
			t.Slug = ref
		}
		return t, nil
	}

	var t *target
	switch {
	case ref != "":
		p, err := a.findProject(ref)
		if err != nil {
			return nil, err
		}
		t = &target{Project: p, Source: src, Link: link}
		if link != nil && link.ProjectID != p.ID {
			a.ui.Warn(i18n.T("project.other_than_linked", p.Slug, link.Slug))
		}
	case link != nil:
		p, err := a.findProject(strconv.FormatInt(link.ProjectID, 10))
		if err != nil {
			if e := cerr.As(err); e.Code == cerr.ProjectNotFound {
				return nil, cerr.New(cerr.ProjectNotFound, i18n.T("project.linked_gone", link.Slug)).
					With("link", link.Path).With("hint", i18n.T("project.relink_hint"))
			}
			return nil, err
		}
		t = &target{Project: p, Source: srcLink, Link: link}
	default:
		ps, err := a.listProjects()
		if err != nil {
			return nil, err
		}
		switch {
		case len(ps) == 1:
			t = &target{Project: &ps[0], Source: srcOnly}
		case len(ps) == 0:
			return nil, cerr.New(cerr.NotLinked, i18n.T("project.none")).With("hint", i18n.T("project.none_hint"))
		case a.ui.Interactive:
			p, err := a.pickProject(ps, i18n.T("project.pick"))
			if err != nil {
				return nil, err
			}
			t = &target{Project: p, Source: srcPick}
			if ok, _ := a.ui.Confirm(i18n.T("project.offer_link", p.Slug), true); ok {
				if _, err := a.writeLink(p, ""); err != nil {
					return nil, err
				}
				t.Source = srcLink
			}
		default:
			return nil, cerr.New(cerr.NotLinked, i18n.T("project.not_linked")).
				With("hint", i18n.T("project.not_linked_hint")).With("projects", choices(ps))
		}
	}
	if changing {
		a.ui.Target(t.Slug, t.OrgName)
	}
	return t, nil
}

// pickProject shows the projects grouped by organization.
func (a *app) pickProject(ps []api.Project, prompt string) (*api.Project, error) {
	opts, idx := projectOptions(ps)
	i, err := a.ui.Select(prompt, opts, firstSelectable(opts))
	if err != nil {
		return nil, err
	}
	return &ps[idx[i]], nil
}

// projectOptions groups projects by organization for a Select; idx maps an
// option to its project.
func projectOptions(ps []api.Project) ([]ui.Option, map[int]int) {
	var opts []ui.Option
	idx := map[int]int{}
	var orgs []string
	byOrg := map[string][]int{}
	for i, p := range ps {
		if _, ok := byOrg[p.OrgName]; !ok {
			orgs = append(orgs, p.OrgName)
		}
		byOrg[p.OrgName] = append(byOrg[p.OrgName], i)
	}
	for _, o := range orgs {
		if len(orgs) > 1 || o != "" {
			opts = append(opts, ui.Option{Label: o, Header: true})
		}
		for _, i := range byOrg[o] {
			p := ps[i]
			idx[len(opts)] = i
			opts = append(opts, ui.Option{Label: p.Slug, Hint: planLabel(p.Plan) + " · " + statusLabel(p.Status)})
		}
	}
	return opts, idx
}

// firstSelectable is the first option that can be chosen, or -1.
func firstSelectable(opts []ui.Option) int {
	for i, o := range opts {
		if !o.Header && !o.Disabled {
			return i
		}
	}
	return -1
}

// writeLink writes .rapibase/project.json in the repo root (or the current
// folder outside a repo), or where the existing link is.
func (a *app) writeLink(p *api.Project, dir string) (string, error) {
	if dir == "" {
		dir = a.cwd
	}
	path, err := project.Write(dir, project.Link{ProjectID: p.ID, Slug: p.Slug, OrgID: p.OrgID})
	if err != nil {
		return "", err
	}
	return path, nil
}

func planLabel(plan string) string {
	switch plan {
	case "free":
		return i18n.T("plan.free")
	case "pro":
		return "Pro"
	case "team":
		return "Team"
	}
	return plan
}

func statusLabel(s string) string {
	if k := "status." + s; i18n.Has(k) {
		return i18n.T(k)
	}
	return s
}
