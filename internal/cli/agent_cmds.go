package cli

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/agents"
	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/project"
	"github.com/rapibase/cli/internal/ui"
)

// agentOpts are the flags shared by init, skill install and mcp add.
type agentOpts struct {
	agents []string
	scope  string
}

func (o *agentOpts) bind(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&o.agents, "agent", nil, i18n.T("flag.agent"))
	cmd.Flags().StringVar(&o.scope, "scope", "project", i18n.T("flag.scope"))
}

// root is the repo root, or the current folder outside a repo.
func (a *app) root() string {
	if r := project.RepoRoot(a.cwd); r != "" {
		return r
	}
	return a.cwd
}

// pickAgents resolves --agent, or detects the agent (spec §7, step 7): the
// one running the CLI, then the repo's files; with a TTY it asks, without
// one it falls back to AGENTS.md, which most agents read.
func (a *app) pickAgents(o agentOpts) ([]agents.Agent, agents.Scope, error) {
	sc, ok := agents.ParseScope(o.scope)
	if !ok {
		return nil, "", cerr.New(cerr.Usage, i18n.T("agent.bad_scope", o.scope))
	}
	var out []agents.Agent
	seen := map[agents.Agent]bool{}
	add := func(x agents.Agent) {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	for _, s := range o.agents {
		for _, part := range strings.Split(s, ",") {
			x, ok := agents.Parse(part)
			if !ok {
				return nil, "", cerr.New(cerr.Usage, i18n.T("agent.bad", part, agentNames()))
			}
			if x == agents.None {
				return nil, sc, nil
			}
			add(x)
		}
	}
	if len(out) > 0 {
		return out, sc, nil
	}
	if r := agents.Running(); r != "" {
		add(r)
		a.ui.Progress(i18n.T("agent.detected_running", r.Label()))
	}
	for _, x := range agents.FromFiles(a.root()) {
		add(x)
	}
	if len(out) > 0 {
		return out, sc, nil
	}
	if !a.ui.Interactive {
		a.ui.Progress(i18n.T("agent.fallback_other"))
		return []agents.Agent{agents.Other}, sc, nil
	}
	installed := map[agents.Agent]bool{}
	for _, x := range agents.FromPath() {
		installed[x] = true
	}
	opts := []ui.Option{}
	def := -1
	for i, x := range agents.All {
		hint := ""
		if installed[x] {
			hint = i18n.T("agent.installed")
			if def < 0 {
				def = i
			}
		}
		if x == agents.Other {
			hint = i18n.T("agent.other_hint")
		}
		opts = append(opts, ui.Option{Label: x.Label(), Hint: hint})
	}
	opts = append(opts, ui.Option{Label: i18n.T("agent.none")})
	if def < 0 {
		def = 0
	}
	i, err := a.ui.Select(i18n.T("agent.ask"), opts, def)
	if err != nil {
		return nil, "", err
	}
	if i >= len(agents.All) {
		return nil, sc, nil
	}
	return []agents.Agent{agents.All[i]}, sc, nil
}

func agentNames() string {
	names := []string{}
	for _, x := range agents.All {
		names = append(names, string(x))
	}
	return strings.Join(append(names, string(agents.None)), ", ")
}

// --- skill ---

func (a *app) skillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "skill",
		Short:   i18n.T("cmd.skill.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase skill install", "rapibase skill install --agent codex --scope user", "rapibase skill update"),
	}
	var o agentOpts
	var force bool
	install := &cobra.Command{
		Use: "install", Short: i18n.T("cmd.skill.install.short"), Long: i18n.T("cmd.skill.install.long"), Args: cobra.NoArgs,
		Example: examples("rapibase skill install", "rapibase skill install --agent claude --agent codex", "rapibase skill install --scope local"),
	}
	o.bind(install)
	install.Flags().BoolVar(&force, "force", false, i18n.T("flag.force_skill"))
	install.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		list, sc, err := a.pickAgents(o)
		if err != nil {
			return nil, err
		}
		res, err := a.installSkills(list, sc, force)
		if err != nil {
			return nil, err
		}
		return res, nil
	})
	update := &cobra.Command{
		Use: "update", Short: i18n.T("cmd.skill.update.short"), Args: cobra.NoArgs,
		Example: examples("rapibase skill update"),
	}
	update.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		res, err := agents.UpdateSkills(a.root())
		if err != nil {
			return nil, err
		}
		for _, u := range res.Updated {
			a.ui.Success(i18n.T("skill.updated", u))
		}
		switch {
		case len(res.Updated) == 0 && len(res.Current) == 0:
			a.ui.Println(i18n.T("skill.none_installed"))
		case len(res.Updated) == 0:
			a.ui.Success(i18n.T("skill.up_to_date", len(res.Current)))
		}
		return res, nil
	})
	cmd.AddCommand(install, update)
	return cmd
}

// installSkills writes the skills for each agent, replaces the old single
// `rapibase` skill and prints what happened.
func (a *app) installSkills(list []agents.Agent, sc agents.Scope, force bool) (map[string]any, error) {
	root := a.root()
	results := []*agents.SkillResult{}
	done := map[string]bool{}
	for _, x := range list {
		r, err := agents.InstallSkills(x, sc, root, force)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
		if !done[r.Dir] && len(r.Installed) > 0 {
			a.ui.Success(i18n.T("skill.installed", x.Label(), strings.Join(r.Installed, ", "), r.Dir))
		}
		done[r.Dir] = true
		for _, s := range r.Skipped {
			a.ui.Note(i18n.T("skill.kept_newer", s, r.Dir))
		}
		if r.AgentsMD {
			a.ui.Note(i18n.T("skill.agents_md"))
		}
		if r.Excluded {
			a.ui.Note(i18n.T("skill.git_excluded"))
		}
	}
	// the old skill: in the repo always; in the home folder only when the new
	// ones also go there, so other repos keep a guide
	var old []agents.OldSkill
	for _, o := range agents.FindOldSkill(root) {
		inRepo := strings.HasPrefix(filepath.Clean(o.Path), filepath.Clean(root)+string(filepath.Separator))
		if inRepo || sc == agents.ScopeUser {
			old = append(old, o)
		}
	}
	if len(old) > 0 {
		if err := agents.RemoveOldSkill(old); err != nil {
			return nil, err
		}
		a.ui.Note(i18n.T("skill.old_replaced", len(old)))
	}
	out := map[string]any{"skills": results}
	if len(old) > 0 {
		out["old_skill_removed"] = old
	}
	return out, nil
}

// --- mcp ---

func (a *app) mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mcp",
		Short:   i18n.T("cmd.mcp.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase mcp add", "rapibase mcp add --agent cursor"),
	}
	var o agentOpts
	add := &cobra.Command{
		Use: "add", Short: i18n.T("cmd.mcp.add.short"), Long: i18n.T("cmd.mcp.add.long"), Args: cobra.NoArgs,
		Example: examples("rapibase mcp add", "rapibase mcp add --agent claude --scope user", "rapibase mcp add --agent codex"),
	}
	o.bind(add)
	add.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		list, sc, err := a.pickAgents(o)
		if err != nil {
			return nil, err
		}
		slug := ""
		if a.flags.project != "" {
			p, err := a.findProject(a.flags.project)
			if err != nil {
				return nil, err
			}
			slug = p.Slug
		} else if link, _ := project.Find(a.cwd); link != nil {
			slug = link.Slug
		}
		res := a.addMCP(list, sc, slug)
		return map[string]any{"mcp": res}, nil
	})
	cmd.AddCommand(add)
	return cmd
}

// addMCP registers the MCP for each agent. A failure with one agent does not
// stop the others: what is left to do is printed (and in the JSON).
func (a *app) addMCP(list []agents.Agent, sc agents.Scope, slug string) []*agents.MCPResult {
	out := []*agents.MCPResult{}
	for _, x := range list {
		r, err := agents.AddMCP(agents.MCPOptions{
			Agent: x, Scope: sc, Root: a.root(), AccountURL: strings.TrimRight(a.host, "/") + "/mcp",
			Slug: slug, Interactive: a.ui.Interactive,
		})
		if r == nil {
			a.ui.Warn(i18n.T("mcp.failed", x.Label(), err))
			continue
		}
		out = append(out, r)
		switch {
		case err != nil:
			a.ui.Warn(i18n.T("mcp.failed", x.Label(), err))
		case r.Unchanged:
			a.ui.Success(i18n.T("mcp.already", x.Label()))
		case r.Registered:
			where := r.File
			if where == "" {
				where = r.Command
			}
			a.ui.Success(i18n.T("mcp.added", x.Label(), where))
		}
		if r.Manual != "" {
			a.ui.Note(i18n.T("mcp.manual", x.Label()))
			for _, l := range strings.Split(r.Manual, "\n") {
				a.ui.Println("    " + a.ui.Cyan(l))
			}
		}
		if r.Login != "" {
			a.ui.Note(i18n.T("mcp.login", a.ui.Cyan(r.Login)))
		}
		if r.Registered && !r.Unchanged && r.RestartNeeded {
			a.ui.Note(i18n.T("mcp.restart_" + string(x)))
		}
	}
	return out
}

// --- init (spec §7) ---

func (a *app) initCmd() *cobra.Command {
	var name, org, region, ref, framework string
	var noEnv, noSkill, noMCP, force bool
	var o agentOpts
	cmd := &cobra.Command{
		Use:   "init",
		Short: i18n.T("cmd.init.short"),
		Long:  i18n.T("cmd.init.long"),
		Args:  cobra.NoArgs,
		Example: examples(
			"npx rapibase@latest init",
			"rapibase init --name my-app --agent claude --yes",
			"rapibase init --project my-app-staging --no-mcp",
		),
	}
	cmd.Flags().StringVar(&name, "name", "", i18n.T("flag.init_name"))
	cmd.Flags().StringVar(&org, "org", "", i18n.T("flag.org"))
	cmd.Flags().StringVar(&region, "region", "", i18n.T("flag.region"))
	cmd.Flags().StringVar(&ref, "ref", "", i18n.T("flag.ref"))
	cmd.Flags().StringVar(&framework, "framework", "", i18n.T("flag.framework", "vite|next|astro|sveltekit|expo"))
	cmd.Flags().BoolVar(&noEnv, "no-env", false, i18n.T("flag.no_env"))
	cmd.Flags().BoolVar(&noSkill, "no-skill", false, i18n.T("flag.no_skill"))
	cmd.Flags().BoolVar(&noMCP, "no-mcp", false, i18n.T("flag.no_mcp"))
	cmd.Flags().BoolVar(&force, "force", false, i18n.T("flag.force_link"))
	o.bind(cmd)
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		out := map[string]any{}
		// 2. session
		me, err := a.ensureLogin(ref)
		if err != nil {
			return nil, err
		}
		out["account"] = map[string]any{"email": maskEmail(me.Email)}

		// 3. project
		p, created, err := a.initProject(me, name, org, region)
		if err != nil {
			return nil, err
		}
		out["created"] = created

		// 4. provisioning
		if p, err = a.waitRunning(p); err != nil {
			if e := cerr.As(err); e.Code == cerr.Provisioning {
				// link anyway, so `status --wait` and a second init pick it up
				_, _ = a.linkTo(p, force)
				e.With("project", (&target{Project: p}).view()).With("hint", "rapibase status --wait && rapibase init")
				return nil, e
			}
			return nil, err
		}
		t := &target{Project: p, Source: srcLink}
		out["project"] = t.view()

		// 5. link
		linkPath, err := a.linkTo(p, force)
		if err != nil {
			return nil, err
		}
		out["link"] = linkPath

		// 6. env
		if !noEnv {
			res, err := a.envPull(t, ".env", framework)
			if err != nil {
				// a tracked .env must not stop the rest: say it and go on
				e := cerr.As(err)
				a.ui.Warn(e.Message)
				if h := e.Str("hint"); h != "" {
					a.ui.Note(h)
				}
				out["env_error"] = map[string]any{"code": e.Code, "message": e.Message}
			} else {
				out["env_file"] = rel(a.cwd, res.File)
				out["env_vars"] = res.Vars
				out["framework"] = res.Framework
			}
		}

		// 7–9. agent, skills, MCP
		if !noSkill || !noMCP {
			list, sc, err := a.pickAgents(o)
			if err != nil {
				return nil, err
			}
			names := []string{}
			for _, x := range list {
				names = append(names, string(x))
			}
			out["agents"] = names
			if len(list) > 0 && !noSkill {
				res, err := a.installSkills(list, sc, false)
				if err != nil {
					return nil, err
				}
				out["skills"] = res["skills"]
			}
			if len(list) > 0 && !noMCP {
				out["mcp"] = a.addMCP(list, sc, p.Slug)
			}
		}

		// 10. closing
		a.ui.Println()
		a.ui.Printf("  %-9s %s\n", i18n.T("init.app"), a.ui.Cyan(p.AppURL))
		if p.ConsoleURL != "" {
			a.ui.Printf("  %-9s %s\n", i18n.T("init.console"), a.ui.Cyan(p.ConsoleURL))
		}
		a.ui.Println()
		a.ui.Println(i18n.T("init.next"))
		a.ui.Println("  " + i18n.T("init.next_prompt"))
		a.ui.Println("  " + i18n.T("init.next_deploy"))
		out["console_url"] = p.ConsoleURL
		out["next_steps"] = []string{"rapibase deploy", "rapibase status"}
		return out, nil
	})
	return cmd
}

// initProject picks or creates the project of init (spec §7, step 3).
func (a *app) initProject(me *api.Me, name, org, region string) (*api.Project, bool, error) {
	link, err := project.Find(a.cwd)
	if err != nil {
		return nil, false, cerr.New(cerr.Usage, err.Error())
	}
	switch {
	case a.flags.project != "":
		p, err := a.findProject(a.flags.project)
		return p, false, err
	case link != nil:
		p, err := a.findProject(strconv.FormatInt(link.ProjectID, 10))
		if err == nil {
			a.ui.Progress(i18n.T("init.using_linked", p.Slug, rel(a.cwd, link.Path)))
			return p, false, nil
		}
		if e := cerr.As(err); e.Code != cerr.ProjectNotFound {
			return nil, false, err
		}
		a.ui.Warn(i18n.T("project.linked_gone", link.Slug))
	case name != "":
		p, err := a.createProject(me, name, org, region)
		return p, err == nil, err
	}
	ps, err := a.listProjects()
	if err != nil {
		return nil, false, err
	}
	if !a.ui.Interactive {
		if len(ps) == 0 {
			return nil, false, a.missing(i18n.T("init.need_name_new"), "--name")
		}
		return nil, false, a.missing(i18n.T("init.need_project"), "--project", "--name").With("projects", choices(ps))
	}
	opts, idx := projectOptions(ps)
	canCreate := me.CanCreate()
	if canCreate {
		opts = append([]ui.Option{{Label: i18n.T("init.create_new"), Hint: planLabel("free")}}, opts...)
	}
	if len(opts) == 0 {
		return nil, false, cerr.New(cerr.Forbidden, i18n.T("create.no_scope")).With("action_url", me.Connection.URL)
	}
	def := 0
	if !canCreate {
		def = firstSelectable(opts)
	}
	i, err := a.ui.Select(i18n.T("init.ask_project"), opts, def)
	if err != nil {
		return nil, false, err
	}
	if canCreate {
		if i == 0 {
			p, err := a.createProject(me, "", org, region)
			// the free project is taken: say so and offer the existing ones,
			// without pushing to a paid plan
			if e := cerr.As(err); err != nil && e.Code == cerr.PlanLimit && len(ps) > 0 {
				a.ui.Warn(e.Message)
				p, err := a.pickProject(ps, i18n.T("init.ask_existing"))
				return p, false, err
			}
			return p, err == nil, err
		}
		i--
	}
	return &ps[idx[i]], false, nil
}
