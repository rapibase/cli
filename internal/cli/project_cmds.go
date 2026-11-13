package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/browser"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/project"
	"github.com/rapibase/cli/internal/ui"
)

func (a *app) projectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Short:   i18n.T("cmd.projects.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase projects", "rapibase projects --json", "rapibase projects create --name my-app"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		ps, err := a.listProjects()
		if err != nil {
			return nil, err
		}
		link, _ := project.Find(a.cwd)
		if len(ps) == 0 {
			a.ui.Println(i18n.T("projects.empty"))
			a.ui.Note(i18n.T("project.none_hint"))
		}
		lastOrg := "\x00"
		for _, p := range ps {
			if p.OrgName != lastOrg {
				a.ui.Println(a.ui.Bold(p.OrgName))
				lastOrg = p.OrgName
			}
			mark := " "
			if link != nil && link.ProjectID == p.ID {
				mark = a.ui.Green("*")
			}
			a.ui.Printf(" %s %-28s %s\n", mark, p.Slug, a.ui.Dim(planLabel(p.Plan)+" · "+statusLabel(p.Status)+" · "+modeLabel(p.Mode)))
		}
		out := []map[string]any{}
		for _, p := range ps {
			out = append(out, map[string]any{
				"id": p.ID, "slug": p.Slug, "name": p.Name, "org": p.OrgName, "org_id": p.OrgID,
				"plan": p.Plan, "status": p.Status, "mode": p.Mode, "url": p.AppURL, "console_url": p.ConsoleURL,
				"linked": link != nil && link.ProjectID == p.ID,
			})
		}
		return map[string]any{"projects": out}, nil
	})
	cmd.AddCommand(a.projectsCreateCmd())
	return cmd
}

func (a *app) projectsCreateCmd() *cobra.Command {
	var name, org, region string
	var wait bool
	cmd := &cobra.Command{
		Use:     "create",
		Short:   i18n.T("cmd.projects.create.short"),
		Long:    i18n.T("cmd.projects.create.long"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase projects create --name my-app", "rapibase projects create --name shop --org acme --wait"),
	}
	cmd.Flags().StringVar(&name, "name", "", i18n.T("flag.name"))
	cmd.Flags().StringVar(&org, "org", "", i18n.T("flag.org"))
	cmd.Flags().StringVar(&region, "region", "", i18n.T("flag.region"))
	cmd.Flags().BoolVar(&wait, "wait", false, i18n.T("flag.wait_running"))
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		me, err := a.me()
		if err != nil {
			return nil, err
		}
		p, err := a.createProject(me, name, org, region, false)
		if err != nil {
			return nil, err
		}
		if wait {
			if p, err = a.waitRunning(p); err != nil {
				return nil, err
			}
		}
		return map[string]any{"project": (&target{Project: p}).view(), "console_url": p.ConsoleURL}, nil
	})
	return cmd
}

var nameRe = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} _.-]{0,39}$`)

// createProject creates a free project (or one covered by a Team). Paid
// projects are created in the console (spec §6.3). With back, Esc in its
// questions returns ui.ErrBack (init goes back to its project menu).
func (a *app) createProject(me *api.Me, name, org, region string, back bool) (*api.Project, error) {
	if !me.CanCreate() {
		return nil, cerr.New(cerr.Forbidden, i18n.T("create.no_scope")).With("action_url", me.Connection.URL)
	}
	validName := func(s string) string {
		if !nameRe.MatchString(strings.TrimSpace(s)) {
			return i18n.T("create.bad_name")
		}
		return ""
	}
	if name == "" {
		if !a.ui.Interactive {
			return nil, a.missing(i18n.T("create.need_name"), "--name")
		}
		def := a.draftName
		if def == "" {
			def = filepath.Base(a.cwd)
		}
		if validName(def) != "" {
			def = "my-app"
		}
		ask := a.ui.Input
		if back {
			ask = a.ui.InputBack
		}
		n, err := ask(i18n.T("create.ask_name"), def, validName)
		if err != nil {
			return nil, err
		}
		name = n
		a.draftName = n
	} else if msg := validName(name); msg != "" {
		return nil, cerr.New(cerr.Usage, msg)
	}

	orgID, orgName, err := a.pickOrg(me, org, back)
	if err != nil {
		return nil, err
	}
	if region == "" {
		var rs []api.Region
		if err := a.api.Do(a.ctx, "GET", "/api/regions", nil, &rs); err == nil && len(rs) > 1 {
			if !a.ui.Interactive {
				ids := []string{}
				for _, r := range rs {
					ids = append(ids, r.ID)
				}
				return nil, a.missing(i18n.T("create.need_region", strings.Join(ids, ", ")), "--region")
			}
			opts := []ui.Option{}
			for _, r := range rs {
				opts = append(opts, ui.Option{Label: r.ID, Hint: r.Name})
			}
			sel := a.ui.Select
			if back {
				sel = a.ui.SelectBack
			}
			i, err := sel(i18n.T("create.ask_region"), opts, 0)
			if err != nil {
				return nil, err
			}
			region = rs[i].ID
		}
	}
	a.ui.Target(strings.TrimSpace(name), orgName)
	stop := a.ui.Spinner(i18n.T("create.creating"))
	var p api.Project
	body := map[string]any{"name": strings.TrimSpace(name), "plan": "free", "org_id": orgID, "region": region}
	err = a.api.Do(a.ctx, "POST", "/api/projects", api.JSONBody(body), &p)
	stop()
	if err != nil {
		e := cerr.As(err)
		if e.Code == cerr.PlanLimit && e.Str("project") != "" {
			e.With("hint", i18n.T("create.free_used_hint", e.Str("project")))
		}
		return nil, e
	}
	a.projects = nil
	a.draftName = ""
	a.ui.Success(i18n.T("create.done", p.Slug))
	return &p, nil
}

// orgChoice says whether a new project can go to an organization and what
// its option shows: the free project lives in the personal account and is
// one per person; in a Team the plan covers it (spec §6.3). It's a guess
// from what this session sees: the server has the last word.
type orgChoice struct {
	org  api.Org
	ok   bool
	hint string
}

func orgChoices(me *api.Me, ps []api.Project) []orgChoice {
	out := make([]orgChoice, 0, len(me.Orgs))
	for _, o := range me.Orgs {
		c := orgChoice{org: o, ok: true, hint: "Team"}
		switch {
		case o.Kind == "personal":
			c.hint = planLabel("free")
			for _, p := range ps {
				if p.Plan == "free" && p.OrgID == o.ID && p.Status != "deleting" {
					c.ok, c.hint = false, i18n.T("create.free_taken", p.Slug)
					break
				}
			}
		case o.Role == "limited":
			c.ok, c.hint = false, i18n.T("org.limited")
		}
		out = append(out, c)
	}
	return out
}

// newProjectOption is "Create a new one" in init's menu: free if the
// personal account still can, else where it can go, else shown disabled
// with the reason (never pushing to a paid plan).
func newProjectOption(cs []orgChoice) ui.Option {
	opt := ui.Option{Label: i18n.T("init.create_new"), Hint: planLabel("free")}
	var ok []orgChoice
	for _, c := range cs {
		if c.ok {
			ok = append(ok, c)
		}
	}
	switch {
	case len(cs) == 0:
		// no organizations listed: the server decides
	case len(ok) == 0:
		opt.Disabled = true
		opt.Hint = cs[0].hint
		for _, c := range cs {
			if c.org.Kind == "personal" {
				opt.Hint = c.hint
			}
		}
	default:
		for _, c := range ok {
			if c.org.Kind == "personal" {
				return opt
			}
		}
		opt.Hint = i18n.T("create.in_team")
		if len(ok) == 1 {
			opt.Hint = i18n.T("create.in_org", ok[0].org.Name)
		}
	}
	return opt
}

// pickOrg: --org by id or name; by default the personal one; with a TTY and
// Teams, it asks (spec §4.5.8), with the organizations where it can't go
// shown disabled.
func (a *app) pickOrg(me *api.Me, org string, back bool) (int64, string, error) {
	var personal *api.Org
	for i := range me.Orgs {
		if me.Orgs[i].Kind == "personal" {
			personal = &me.Orgs[i]
		}
	}
	if org != "" {
		id, _ := strconv.ParseInt(org, 10, 64)
		for _, o := range me.Orgs {
			if o.ID == id || strings.EqualFold(o.Name, org) {
				return o.ID, o.Name, nil
			}
		}
		names := []string{}
		for _, o := range me.Orgs {
			names = append(names, o.Name)
		}
		return 0, "", cerr.New(cerr.Usage, i18n.T("create.bad_org", org, strings.Join(names, ", ")))
	}
	if len(me.Orgs) > 1 && a.ui.Interactive {
		ps, _ := a.listProjects() // only to know if the free one is taken
		cs := orgChoices(me, ps)
		opts := []ui.Option{}
		def := -1
		for i, c := range cs {
			opts = append(opts, ui.Option{Label: c.org.Name, Hint: c.hint, Disabled: !c.ok})
			if c.ok && (def < 0 || c.org.Kind == "personal") {
				def = i
			}
		}
		// with none available, the server says why
		if def >= 0 {
			sel := a.ui.Select
			if back {
				sel = a.ui.SelectBack
			}
			i, err := sel(i18n.T("create.ask_org"), opts, def)
			if err != nil {
				return 0, "", err
			}
			return me.Orgs[i].ID, me.Orgs[i].Name, nil
		}
	}
	if personal != nil {
		return personal.ID, personal.Name, nil
	}
	return 0, "", nil
}

// waitRunning polls until the instance runs (TTY: up to 5 min; otherwise
// 90 s, then `provisioning`, exit 10).
func (a *app) waitRunning(p *api.Project) (*api.Project, error) {
	if p.Status == "running" {
		return p, nil
	}
	start := time.Now()
	deadline := start.Add(a.waitBudget(5 * time.Minute))
	stop := a.ui.Spinner(i18n.T("status.waiting", p.Slug))
	defer stop()
	for {
		var cur api.Project
		if err := a.api.Do(a.ctx, "GET", "/api/projects/"+p.IDString(), nil, &cur); err != nil {
			return nil, err
		}
		switch cur.Status {
		case "running":
			stop()
			a.ui.Success(i18n.T("status.running_after", cur.AppURL, int(time.Since(start).Seconds())))
			return &cur, nil
		case "error":
			return nil, cerr.New(cerr.ProjectNotRunning, i18n.T("status.failed_err", cur.Error)).
				With("status", "error").With("action_url", cur.ConsoleURL)
		case "paused":
			return nil, cerr.New(cerr.ProjectNotRunning, i18n.T("status.paused_err")).
				With("status", "paused").With("action_url", cur.ConsoleURL)
		}
		if time.Now().After(deadline) {
			return nil, cerr.New(cerr.Provisioning, i18n.T("status.still_provisioning")).
				With("status", cur.Status).With("hint", "rapibase status --wait")
		}
		select {
		case <-a.ctx.Done():
			return nil, a.ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func (a *app) linkCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "link [slug|id]",
		Short:   i18n.T("cmd.link.short"),
		Long:    i18n.T("cmd.link.long"),
		Args:    cobra.MaximumNArgs(1),
		Example: examples("rapibase link", "rapibase link my-app-x7k2", "rapibase link 31 --force"),
	}
	cmd.Flags().BoolVar(&force, "force", false, i18n.T("flag.force_relink"))
	cmd.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		ref := a.flags.project
		if len(args) == 1 {
			ref = args[0]
		}
		var p *api.Project
		var err error
		switch {
		case ref != "":
			p, err = a.findProject(ref)
		case a.ui.Interactive:
			var ps []api.Project
			if ps, err = a.listProjects(); err == nil {
				if len(ps) == 0 {
					return nil, cerr.New(cerr.NotLinked, i18n.T("project.none")).With("hint", i18n.T("project.none_hint"))
				}
				p, err = a.pickProject(ps, i18n.T("link.pick"))
			}
		default:
			ps, lerr := a.listProjects()
			if lerr != nil {
				return nil, lerr
			}
			return nil, a.missing(i18n.T("link.need_project"), "<slug|id>").With("projects", choices(ps))
		}
		if err != nil {
			return nil, err
		}
		path, err := a.linkTo(p, force)
		if err != nil {
			return nil, err
		}
		return map[string]any{"project": (&target{Project: p}).view(), "linked": path}, nil
	})
	return cmd
}

// linkTo writes the link, asking before replacing a different one (spec
// §4.5.7): without a TTY, --force.
func (a *app) linkTo(p *api.Project, force bool) (string, error) {
	cur, err := project.Find(a.cwd)
	if err != nil {
		return "", cerr.New(cerr.Usage, err.Error())
	}
	dir := a.cwd
	if cur != nil {
		dir = cur.Root
		if cur.ProjectID == p.ID {
			if cur.Slug != p.Slug { // the subdomain changed: refresh the slug
				_, _ = a.writeLink(p, dir)
			}
			a.ui.Success(i18n.T("link.already", p.Slug))
			return rel(a.cwd, cur.Path), nil
		}
		if !force {
			if !a.ui.Interactive {
				return "", cerr.New(cerr.Usage, i18n.T("link.relink_needs_force", rel(a.cwd, cur.Path), cur.Slug, p.Slug)).
					With("linked", cur.Slug).With("hint", i18n.T("link.relink_hint"))
			}
			ok, err := a.ui.Confirm(i18n.T("link.confirm_relink", rel(a.cwd, cur.Path), cur.Slug, p.Slug), false)
			if err != nil {
				return "", err
			}
			if !ok {
				return "", cerr.New(cerr.Usage, i18n.T("link.kept", cur.Slug))
			}
		}
	}
	path, err := a.writeLink(p, dir)
	if err != nil {
		return "", err
	}
	a.ui.Success(i18n.T("link.done", p.Slug, rel(a.cwd, path)))
	return rel(a.cwd, path), nil
}

func (a *app) unlinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unlink",
		Short:   i18n.T("cmd.unlink.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase unlink"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		cur, err := project.Find(a.cwd)
		if err != nil {
			return nil, cerr.New(cerr.Usage, err.Error())
		}
		if cur == nil {
			a.ui.Success(i18n.T("unlink.none"))
			return map[string]any{"unlinked": false}, nil
		}
		if err := project.Remove(cur); err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("unlink.done", cur.Slug, rel(a.cwd, cur.Path)))
		return map[string]any{"unlinked": true, "slug": cur.Slug, "path": rel(a.cwd, cur.Path)}, nil
	})
	return cmd
}

func (a *app) statusCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:     "status",
		Short:   i18n.T("cmd.status.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase status", "rapibase status --wait", "rapibase status --json"),
	}
	cmd.Flags().BoolVar(&wait, "wait", false, i18n.T("flag.wait_running"))
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		var p api.Project
		if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString(), nil, &p); err != nil {
			return nil, err
		}
		p.OrgName = t.OrgName
		if wait && p.Status != "running" {
			np, err := a.waitRunning(&p)
			if err != nil {
				return nil, err
			}
			np.OrgName = t.OrgName
			p = *np
		}
		var services any
		var svcList []map[string]any
		if p.Status == "running" {
			var raw json.RawMessage
			if err := a.api.Do(a.ctx, "GET", "/api/projects/"+p.IDString()+"/services", nil, &raw); err == nil {
				services = raw
				svcList = serviceRows(raw)
			}
		}
		linkPath := ""
		if t.Link != nil {
			linkPath = rel(a.cwd, t.Link.Path)
		}

		a.ui.Printf("%s %s  %s\n", a.ui.Bold(p.Slug), a.ui.Dim("("+t.OrgName+")"), statusColored(a.ui, p.Status))
		a.ui.Printf("  %-12s %s\n", i18n.T("status.plan"), planLabel(p.Plan))
		if p.AppURL != "" {
			a.ui.Printf("  %-12s %s\n", "App", a.ui.Cyan(p.AppURL))
			a.ui.Printf("  %-12s %s\n", "Dashboard", a.ui.Cyan(p.DashboardURL))
		}
		a.ui.Printf("  %-12s %s\n", i18n.T("status.console"), a.ui.Cyan(p.ConsoleURL))
		if p.CustomDomain != "" {
			a.ui.Printf("  %-12s %s\n", i18n.T("status.domain"), p.CustomDomain)
		}
		if p.ActiveDeployID != "" {
			a.ui.Printf("  %-12s %s\n", i18n.T("status.release"), p.ActiveDeployID)
		}
		for _, s := range svcList {
			where, _ := s["url"].(string)
			if where == "" {
				where, _ = s["internal_url"].(string)
			}
			state := ""
			if asleep, _ := s["asleep"].(bool); asleep {
				state = i18n.T("services.asleep")
			}
			a.ui.Printf("  %-12s %v %s %s\n", i18n.T("status.service"), s["name"], a.ui.Cyan(where), a.ui.Dim(state))
		}
		if p.Free != nil {
			printFree(a.ui, p.Free)
		}
		a.ui.Printf("  %-12s %s %s\n", i18n.T("status.mode"), modeLabel(p.Mode), a.ui.Dim("· "+sourceText(t.Source, linkPath)))

		out := map[string]any{
			"project":        projectFull(&p),
			"project_source": t.Source,
		}
		if linkPath != "" {
			out["link"] = linkPath
		}
		if services != nil {
			out["services"] = services
		}
		if p.Free != nil {
			out["free"] = p.Free
		}
		return out, nil
	})
	return cmd
}

func projectFull(p *api.Project) map[string]any {
	v := (&target{Project: p}).view()
	v["subdomain"] = p.Subdomain
	v["region"] = p.Region
	v["app_url"] = p.AppURL
	v["dashboard_url"] = p.DashboardURL
	v["console_url"] = p.ConsoleURL
	if p.ActiveDeployID != "" {
		v["active_release"] = p.ActiveDeployID
	}
	if p.CustomDomain != "" {
		v["custom_domain"] = p.CustomDomain
	}
	if p.Error != "" {
		v["error"] = p.Error
	}
	return v
}

func sourceText(src, link string) string {
	switch src {
	case srcLink:
		return i18n.T("status.src_link", link)
	case srcFlag:
		return i18n.T("status.src_flag")
	case srcEnv:
		return i18n.T("status.src_env")
	case srcOnly:
		return i18n.T("status.src_only")
	}
	return src
}

func statusColored(u *ui.UI, s string) string {
	switch s {
	case "running":
		return u.Green("● " + statusLabel(s))
	case "error":
		return u.Red("● " + statusLabel(s))
	}
	return u.Yellow("● " + statusLabel(s))
}

// serviceRows reads {"services":[{name,status…}]} loosely: the services
// view may grow; status only needs name and state.
func serviceRows(raw json.RawMessage) []map[string]any {
	var v struct {
		Services []map[string]any `json:"services"`
	}
	if json.Unmarshal(raw, &v) == nil {
		return v.Services
	}
	return nil
}

// printFree shows the free-plan usage against its limits and dates.
func printFree(u *ui.UI, raw json.RawMessage) {
	var f map[string]any
	if json.Unmarshal(raw, &f) != nil {
		return
	}
	limits, _ := f["limits"].(map[string]any)
	for _, k := range []string{"db_bytes", "storage_bytes", "egress_bytes"} {
		lim, ok := limits[k].(float64)
		if !ok || lim == 0 {
			continue
		}
		used, _ := f[k].(float64)
		u.Printf("  %-12s %s / %s\n", i18n.T("free."+k), humanSize(used), humanSize(lim))
	}
	if d, ok := f["pauses_at"].(string); ok && d != "" {
		u.Printf("  %-12s %s\n", i18n.T("free.pauses_at"), shortDate(d))
	}
}

func humanSize(n float64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", n/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", n/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", n/(1<<10))
	}
	return fmt.Sprintf("%.0f B", n)
}

func shortDate(s string) string {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local().Format("2006-01-02")
	}
	return s
}

func (a *app) openCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "open [console|dashboard|app|connections]",
		Short:     i18n.T("cmd.open.short"),
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"console", "dashboard", "app", "connections"},
		Example:   examples("rapibase open", "rapibase open app", "rapibase open dashboard"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		what := "console"
		if len(args) == 1 {
			what = args[0]
		}
		var url string
		switch what {
		case "connections":
			url = a.host + "/app/connections"
		case "console", "dashboard", "app":
			t, err := a.resolve(false)
			if err != nil {
				return nil, err
			}
			var p api.Project
			if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString(), nil, &p); err != nil {
				return nil, err
			}
			url = map[string]string{"console": p.ConsoleURL, "dashboard": p.DashboardURL, "app": p.AppURL}[what]
			if url == "" {
				return nil, cerr.New(cerr.ProjectNotRunning, i18n.T("open.not_running", statusLabel(p.Status))).
					With("status", p.Status).With("action_url", p.ConsoleURL)
			}
		default:
			return nil, cerr.New(cerr.Usage, i18n.T("open.bad_target", what))
		}
		opened := false
		if !a.ui.JSON && browser.Allowed(a.flags.noBrowser, ui.IsCI()) {
			opened = browser.Open(url)
		}
		if opened {
			a.ui.Success(i18n.T("open.opened", url))
		} else {
			a.ui.Println(url)
		}
		return map[string]any{"url": url, "opened": opened}, nil
	})
	return cmd
}

// rel shows a path relative to the current folder when it is inside it.
func rel(base, p string) string {
	if r, err := filepath.Rel(base, p); err == nil && !strings.HasPrefix(r, "..") {
		return filepath.ToSlash(r)
	}
	return filepath.ToSlash(p)
}
