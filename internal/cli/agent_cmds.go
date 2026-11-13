package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/agents"
	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/project"
	"github.com/rapibase/cli/internal/skills"
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
		Example: examples("rapibase skill install", "rapibase skill install --agent codex --scope user", "rapibase skill update", "rapibase skill uninstall"),
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
		root := a.root()
		res, err := agents.UpdateSkills(root)
		if err != nil {
			return nil, err
		}
		for _, u := range res.Updated {
			a.ui.Success(i18n.T("skill.updated", u, skills.SetVersion()))
		}
		switch {
		case len(res.Updated) == 0 && len(res.Current) == 0:
			// only the old single skill: updating it means moving to the four
			// new ones, with the same confirmation as `skill install`
			if old := agents.FindOldSkill(root); len(old) > 0 {
				list, sc := oldSkillTargets(root, old)
				return a.installSkills(list, sc, false)
			}
			a.ui.Println(i18n.T("skill.none_installed"))
		case len(res.Updated) == 0:
			a.ui.Success(i18n.T("skill.up_to_date", len(res.Current), skills.SetVersion()))
		}
		return res, nil
	})
	var usc string
	var uagents []string
	uninstall := &cobra.Command{
		Use: "uninstall", Aliases: []string{"remove", "rm"},
		Short: i18n.T("cmd.skill.uninstall.short"), Long: i18n.T("cmd.skill.uninstall.long"), Args: cobra.NoArgs,
		Example: examples("rapibase skill uninstall", "rapibase skill uninstall --scope user --yes", "rapibase skill uninstall --agent codex"),
	}
	uninstall.Flags().StringVar(&usc, "scope", "project", i18n.T("flag.scope_remove"))
	uninstall.Flags().StringArrayVar(&uagents, "agent", nil, i18n.T("flag.agent_remove"))
	uninstall.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		sc, ok := agents.ParseScope(usc)
		if !ok {
			return nil, cerr.New(cerr.Usage, i18n.T("agent.bad_scope", usc))
		}
		var only []agents.Agent // nil: every agent, and the old single skill
		if len(uagents) > 0 {
			list, err := agentList(uagents)
			if err != nil {
				return nil, err
			}
			only = list
		}
		found := agents.FindSkills(a.root(), sc == agents.ScopeUser, only)
		if len(found) == 0 {
			a.ui.Println(i18n.T("skill.none_found"))
			return map[string]any{"removed": []agents.OldSkill{}}, nil
		}
		a.ui.Println(i18n.T("skill.found", len(found)))
		for _, f := range found {
			a.ui.Println("    " + a.shortPath(f.Path))
		}
		ok, err := a.sure(i18n.T("skill.uninstall_confirm", len(found)), i18n.T("skill.uninstall_needs_yes", len(found)))
		if err != nil {
			return nil, cerr.As(err).With("found", found)
		}
		if !ok {
			return nil, cerr.New(cerr.Usage, i18n.T("deploy.cancelled"))
		}
		if err := agents.RemoveOldSkill(found); err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("skill.uninstalled", len(found)))
		return map[string]any{"removed": found}, nil
	})
	cmd.AddCommand(install, update, uninstall)
	return cmd
}

// shortPath shows a path relative to the current folder, or with ~ when it
// is in the home folder outside the repo.
func (a *app) shortPath(p string) string {
	if h, _ := os.UserHomeDir(); h != "" && strings.HasPrefix(p, h) && rel(a.root(), p) == filepath.ToSlash(p) {
		return "~" + filepath.ToSlash(strings.TrimPrefix(p, h))
	}
	return rel(a.cwd, p)
}

// installSkills writes the skills for each agent, replaces the old single
// `rapibase` skill and prints what happened.
func (a *app) installSkills(list []agents.Agent, sc agents.Scope, force bool) (map[string]any, error) {
	root := a.root()
	// the old skill: in the repo always; in the home folder only when the new
	// ones also go there, so other repos keep a guide. Replacing it deletes
	// files the person may have edited, so it is confirmed first (spec §9.1):
	// a question with a TTY, --yes without one. Without --yes nothing is
	// installed either, so two guides that disagree never end up side by side.
	var old []agents.OldSkill
	for _, o := range agents.FindOldSkill(root) {
		inRepo := strings.HasPrefix(filepath.Clean(o.Path), filepath.Clean(root)+string(filepath.Separator))
		if inRepo || sc == agents.ScopeUser {
			old = append(old, o)
		}
	}
	replaceOld := len(old) > 0
	if replaceOld && !a.flags.yes {
		if !a.ui.Interactive {
			return nil, a.missing(i18n.T("skill.old_needs_yes", len(old)), "--yes").With("old_skill", old)
		}
		ok, err := a.ui.Confirm(i18n.T("skill.old_confirm", len(old)), true)
		if err != nil {
			return nil, err
		}
		replaceOld = ok
	}

	results := []*agents.SkillResult{}
	done := map[string]bool{}
	for _, x := range list {
		r, err := agents.InstallSkills(x, sc, root, force)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
		if !done[r.Dir] && len(r.Installed) > 0 {
			a.ui.Success(i18n.T("skill.installed", skills.SetVersion(), x.Label(), strings.Join(r.Installed, ", "), r.Dir))
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
	out := map[string]any{"skills": results}
	switch {
	case replaceOld:
		if err := agents.RemoveOldSkill(old); err != nil {
			return nil, err
		}
		a.ui.Note(i18n.T("skill.old_replaced", len(old)))
		out["old_skill_removed"] = old
	case len(old) > 0:
		// the person said no: theirs to keep, but they should know
		a.ui.Warn(i18n.T("skill.old_kept", len(old)))
		out["old_skill_kept"] = old
	}
	return out, nil
}

// oldSkillTargets says where the four skills go when only the old single
// skill is installed: the same agents, in the repo unless every old copy
// lives in the home folder.
func oldSkillTargets(root string, old []agents.OldSkill) ([]agents.Agent, agents.Scope) {
	seen := map[agents.Agent]bool{}
	var list []agents.Agent
	add := func(x agents.Agent) {
		if !seen[x] {
			seen[x] = true
			list = append(list, x)
		}
	}
	sc := agents.ScopeUser
	for _, o := range old {
		p := filepath.ToSlash(o.Path)
		switch {
		case strings.Contains(p, "/.claude/"):
			add(agents.Claude)
		case strings.Contains(p, "opencode/"):
			add(agents.OpenCode)
		default: // rapibase.md and its AGENTS.md line
			add(agents.Other)
		}
		if strings.HasPrefix(filepath.Clean(o.Path), filepath.Clean(root)+string(filepath.Separator)) {
			sc = agents.ScopeProject
		}
	}
	return list, sc
}

// --- mcp ---

func (a *app) mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mcp",
		Short:   i18n.T("cmd.mcp.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase mcp add", "rapibase mcp add --agent cursor", "rapibase mcp remove"),
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
		slug, sub := "", ""
		if a.flags.project != "" {
			p, err := a.findProject(a.flags.project)
			if err != nil {
				return nil, err
			}
			slug, sub = p.Slug, p.Subdomain
		} else if link, _ := project.Find(a.cwd); link != nil {
			slug = link.Slug
			// the subdomain finds the instance's own MCP even under another
			// name; without a session the name is enough
			if a.sess != nil && a.sess.LoggedIn() {
				if p, err := a.findProject(strconv.FormatInt(link.ProjectID, 10)); err == nil {
					sub = p.Subdomain
				}
			}
		}
		res, err := a.addMCP(list, sc, slug, sub)
		if err != nil {
			return nil, cerr.As(err).With("mcp", res)
		}
		return map[string]any{"mcp": res}, nil
	})

	var ro agentOpts
	remove := &cobra.Command{
		Use: "remove [name]", Aliases: []string{"rm"},
		Short: i18n.T("cmd.mcp.remove.short"), Long: i18n.T("cmd.mcp.remove.long"), Args: cobra.MaximumNArgs(1),
		Example: examples("rapibase mcp remove", "rapibase mcp remove rapibase-my-app-1a2b3c", "rapibase mcp remove --scope user --agent codex"),
	}
	remove.Flags().StringArrayVar(&ro.agents, "agent", nil, i18n.T("flag.agent_remove"))
	remove.Flags().StringVar(&ro.scope, "scope", "project", i18n.T("flag.scope_remove"))
	remove.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		sc, ok := agents.ParseScope(ro.scope)
		if !ok {
			return nil, cerr.New(cerr.Usage, i18n.T("agent.bad_scope", ro.scope))
		}
		list, err := agentList(ro.agents)
		if err != nil {
			return nil, err
		}
		var found []agents.MCPEntry
		for _, x := range list {
			for _, e := range agents.FindMCP(x, a.root(), a.cwd, a.accountMCP(), sc == agents.ScopeUser) {
				if len(args) == 0 || e.Name == args[0] {
					found = append(found, e)
				}
			}
		}
		if len(found) == 0 {
			a.ui.Println(i18n.T("mcp.none_found"))
			return map[string]any{"removed": []agents.MCPEntry{}}, nil
		}
		a.ui.Println(i18n.T("mcp.found", len(found)))
		a.printMCP(found)
		ok, err = a.sure(i18n.T("mcp.remove_confirm", len(found)), i18n.T("mcp.remove_needs_yes", len(found)))
		if err != nil {
			return nil, cerr.As(err).With("found", found)
		}
		if !ok {
			return nil, cerr.New(cerr.Usage, i18n.T("deploy.cancelled"))
		}
		removed, manual, err := agents.RemoveMCP(found)
		for _, e := range removed {
			a.ui.Success(i18n.T("mcp.removed", e.Name, e.Where()))
		}
		if len(manual) > 0 {
			a.ui.Note(i18n.T("mcp.remove_manual"))
			for _, m := range manual {
				a.ui.Println("    " + a.ui.Cyan(m))
			}
		}
		if err != nil {
			return nil, err
		}
		if len(removed) > 0 {
			a.ui.Note(i18n.T("mcp.restart_after_remove"))
		}
		if removed == nil {
			removed = []agents.MCPEntry{}
		}
		return map[string]any{"removed": removed, "manual": manual}, nil
	})
	cmd.AddCommand(add, remove)
	return cmd
}

// agentList parses --agent for the commands that look in every agent when
// none is given.
func agentList(vals []string) ([]agents.Agent, error) {
	var out []agents.Agent
	seen := map[agents.Agent]bool{}
	for _, s := range vals {
		for _, part := range strings.Split(s, ",") {
			x, ok := agents.Parse(part)
			if !ok || x == agents.None {
				return nil, cerr.New(cerr.Usage, i18n.T("agent.bad", part, agentNames()))
			}
			if !seen[x] {
				seen[x] = true
				out = append(out, x)
			}
		}
	}
	if len(out) == 0 {
		return []agents.Agent{agents.Claude, agents.Cursor, agents.OpenCode, agents.Codex, agents.Hermes}, nil
	}
	return out, nil
}

func (a *app) accountMCP() string { return strings.TrimRight(a.host, "/") + "/mcp" }

// printMCP lists MCP servers without their keys: only that one is saved.
func (a *app) printMCP(es []agents.MCPEntry) {
	for _, e := range es {
		line := "    " + e.Name + "  " + a.ui.Dim(e.Agent.Label()+" · "+e.Where())
		if e.Token {
			line += a.ui.Dim(" · " + i18n.T("mcp.legacy_token"))
		}
		a.ui.Println(line)
	}
}

// sure asks before deleting something the person may still use (spec
// §9.1): a question with a TTY, --yes without one (missing_flag otherwise).
func (a *app) sure(question, needsYes string) (bool, error) {
	if a.flags.yes {
		return true, nil
	}
	if !a.ui.Interactive {
		return false, a.missing(needsYes, "--yes")
	}
	return a.ui.Confirm(question, true)
}

// addMCP registers the MCP for each agent. A failure with one agent does not
// stop the others: what is left to do is printed (and in the JSON).
//
// Older Rapibase servers of the same project (the instance's own MCP,
// Connect's rapibase-<slug>) are replaced after a yes, or --yes without a
// TTY. Without it the new one is not added, so the agent never gets the
// same tools twice, and the error that comes back says to use --yes.
func (a *app) addMCP(list []agents.Agent, sc agents.Scope, slug, sub string) ([]*agents.MCPResult, error) {
	out := []*agents.MCPResult{}
	var needYes error
	for _, x := range list {
		o := agents.MCPOptions{
			Agent: x, Scope: sc, Root: a.root(), Cwd: a.cwd, AccountURL: a.accountMCP(),
			Slug: slug, Sub: sub, Interactive: a.ui.Interactive,
		}
		var replaced []string
		if legacy := agents.LegacyMCP(o); len(legacy) > 0 {
			a.ui.Warn(i18n.T("mcp.legacy_found", x.Label(), len(legacy)))
			a.printMCP(legacy)
			ok, err := a.sure(i18n.T("mcp.legacy_confirm"), i18n.T("mcp.legacy_needs_yes", x.Label()))
			if err != nil && cerr.As(err).Code != cerr.MissingFlag {
				return out, err
			}
			if !ok {
				out = append(out, &agents.MCPResult{Agent: x, Name: agents.ServerName, Scope: sc, Legacy: legacy})
				if err != nil {
					needYes = err
				} else {
					a.ui.Note(i18n.T("mcp.legacy_kept", x.Label()))
				}
				continue
			}
			removed, manual, err := agents.RemoveMCP(legacy)
			for _, e := range removed {
				replaced = append(replaced, e.Name)
				a.ui.Success(i18n.T("mcp.removed", e.Name, e.Where()))
			}
			if len(manual) > 0 {
				a.ui.Note(i18n.T("mcp.remove_manual"))
				for _, m := range manual {
					a.ui.Println("    " + a.ui.Cyan(m))
				}
			}
			if err != nil {
				a.ui.Warn(i18n.T("mcp.failed", x.Label(), err))
			}
		}
		r, err := agents.AddMCP(o)
		if r == nil {
			a.ui.Warn(i18n.T("mcp.failed", x.Label(), err))
			continue
		}
		r.Replaced = replaced
		out = append(out, r)
		switch {
		case err != nil:
			a.ui.Warn(i18n.T("mcp.failed", x.Label(), err))
		case r.Unchanged && r.Name != agents.ServerName:
			a.ui.Success(i18n.T("mcp.already_as", x.Label(), r.Name))
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
		if r.Registered && ((!r.Unchanged && r.RestartNeeded) || len(replaced) > 0) {
			a.ui.Note(i18n.T("mcp.restart_" + string(x)))
		}
	}
	return out, needYes
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
		// waitRunning returns no project on error: keep p for the link
		running, err := a.waitRunning(p)
		if err != nil {
			if e := cerr.As(err); e.Code == cerr.Provisioning {
				// link anyway, so `status --wait` and a second init pick it up
				_, _ = a.linkTo(p, force)
				e.With("project", (&target{Project: p}).view()).With("hint", "rapibase status --wait && rapibase init")
				return nil, e
			}
			return nil, err
		}
		p = running
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
					e := cerr.As(err)
					if e.Code != cerr.MissingFlag {
						return nil, err
					}
					// the old skill needs --yes: the project is already made and
					// linked, so say it and finish the rest instead of failing here
					a.ui.Warn(e.Message)
					a.ui.Note(i18n.T("skill.old_retry"))
					out["skill_error"] = map[string]any{"code": e.Code, "message": e.Message, "hint": "rapibase skill install --yes"}
				} else {
					out["skills"] = res["skills"]
					for _, k := range []string{"old_skill_removed", "old_skill_kept"} {
						if v, ok := res[k]; ok {
							out[k] = v
						}
					}
				}
			}
			if len(list) > 0 && !noMCP {
				res, err := a.addMCP(list, sc, p.Slug, p.Subdomain)
				out["mcp"] = res
				if err != nil {
					// older MCP servers of this project and no --yes: like the
					// old skill, say it and finish the rest
					e := cerr.As(err)
					a.ui.Warn(e.Message)
					a.ui.Note(i18n.T("mcp.legacy_retry"))
					out["mcp_error"] = map[string]any{"code": e.Code, "message": e.Message, "hint": "rapibase mcp add --yes"}
				}
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
		p, err := a.createProject(me, name, org, region, false)
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
	// the menu comes back when creating is cancelled (Esc) or the plan
	// doesn't allow it there, so another organization can be chosen
	for {
		opts, idx := projectOptions(ps)
		canCreate := me.CanCreate()
		if canCreate {
			opts = append([]ui.Option{newProjectOption(orgChoices(me, ps))}, opts...)
		}
		if firstSelectable(opts) < 0 {
			if !canCreate {
				return nil, false, cerr.New(cerr.Forbidden, i18n.T("create.no_scope")).With("action_url", me.Connection.URL)
			}
			// no projects and nowhere to create one: the server says why
			p, err := a.createProject(me, "", org, region, false)
			return p, err == nil, err
		}
		i, err := a.ui.Select(i18n.T("init.ask_project"), opts, firstSelectable(opts))
		if err != nil {
			return nil, false, err
		}
		if canCreate {
			if i == 0 {
				p, err := a.createProject(me, "", org, region, true)
				if errors.Is(err, ui.ErrBack) {
					continue
				}
				// not there (free one taken, Team without its plan or
				// full): say why and choose again, without pushing to a
				// paid plan (§6.3)
				if err != nil {
					if e := cerr.As(err); e.Code == cerr.PlanLimit || e.Code == cerr.Forbidden {
						a.ui.Warn(e.Message)
						org = ""
						continue
					}
					return nil, false, err
				}
				return p, true, nil
			}
			i--
		}
		return &ps[idx[i]], false, nil
	}
}
