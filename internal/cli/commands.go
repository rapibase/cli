package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/build"
	"github.com/rapibase/cli/internal/i18n"
)

// helpGroups is the order of the root help (spec §3.1).
var helpGroups = []struct {
	title string
	cmds  []string
}{
	{"help.group.start", []string{"init", "login", "logout", "whoami"}},
	{"help.group.project", []string{"projects", "projects create", "link", "status", "env pull", "open"}},
	{"help.group.publish", []string{"deploy [dir]", "deploys", "functions deploy", "services deploy", "logs"}},
	{"help.group.operate", []string{"sql", "tables", "api", "rls", "config", "restart"}},
	{"help.group.agent", []string{"skill install", "skill update", "skill uninstall", "mcp add", "mcp remove"}},
}

// helpEnv are the environment variables the CLI reads (spec §4.7).
var helpEnv = []struct{ name, key string }{
	{"RAPIBASE_TOKEN", "env.token"},
	{"RAPIBASE_DEPLOY_TOKEN", "env.deploy_token"},
	{"RAPIBASE_PROJECT", "env.project"},
	{"RAPIBASE_LANG", "env.lang"},
	{"RAPIBASE_NO_UPDATE_CHECK", "env.no_update_check"},
	{"RAPIBASE_API_URL", "env.api_url"},
}

func (a *app) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "rapibase",
		Short:         i18n.T("tagline"),
		Version:       build.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := a.setup(cmd); err != nil {
				return err
			}
			return a.checkVersion(cmd)
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	pf := root.PersistentFlags()
	pf.StringVarP(&a.flags.project, "project", "p", "", i18n.T("flag.project"))
	pf.BoolVar(&a.flags.json, "json", false, i18n.T("flag.json"))
	pf.BoolVarP(&a.flags.yes, "yes", "y", false, i18n.T("flag.yes"))
	pf.StringVar(&a.flags.lang, "lang", "", i18n.T("flag.lang"))
	pf.BoolVar(&a.flags.noBrowser, "no-browser", false, i18n.T("flag.no_browser"))
	pf.BoolVar(&a.flags.debug, "debug", false, i18n.T("flag.debug"))
	root.Flags().BoolP("version", "v", false, i18n.T("flag.version"))
	root.Flags().BoolP("help", "h", false, i18n.T("flag.help"))

	root.AddCommand(
		a.initCmd(),
		a.loginCmd(), a.logoutCmd(), a.whoamiCmd(),
		a.projectsCmd(), a.linkCmd(), a.unlinkCmd(), a.statusCmd(), a.envCmd(), a.openCmd(),
		a.deployCmd(), a.deploysCmd(), a.functionsCmd(), a.servicesCmd(), a.logsCmd(),
		a.sqlCmd(), a.tablesCmd(), a.apiCmd(), a.rlsCmd(), a.configCmd(), a.restartCmd(),
		a.skillCmd(), a.mcpCmd(),
	)
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		if cmd == root {
			rootHelp(cmd.OutOrStdout(), root)
			return
		}
		cmdHelp(cmd.OutOrStdout(), cmd)
	})
	root.SetUsageFunc(func(cmd *cobra.Command) error {
		cmdHelp(cmd.ErrOrStderr(), cmd)
		return nil
	})
	root.SuggestionsMinimumDistance = 2
	root.CompletionOptions.DisableDefaultCmd = true
	translateHelpFlag(root)
	return root
}

// translateHelpFlag replaces cobra's "help for <name>" in every command.
func translateHelpFlag(c *cobra.Command) {
	c.InitDefaultHelpFlag()
	if f := c.Flags().Lookup("help"); f != nil {
		f.Usage = i18n.T("flag.help")
	}
	for _, s := range c.Commands() {
		translateHelpFlag(s)
	}
}

// rootHelp prints the grouped help of spec §3.1 in the current language.
func rootHelp(w io.Writer, root *cobra.Command) {
	fmt.Fprintf(w, "rapibase — %s\n\n", i18n.T("tagline"))
	fmt.Fprintf(w, "%s: rapibase <%s> [%s]\n", i18n.T("help.usage"), i18n.T("help.command"), i18n.T("help.options"))
	for _, g := range helpGroups {
		fmt.Fprintf(w, "\n%s\n", i18n.T(g.title))
		for _, name := range g.cmds {
			fmt.Fprintf(w, "  %-20s %s\n", name, i18n.T("cmd."+strings.Fields(name)[0]+keySuffix(name)+".short"))
		}
	}
	fmt.Fprintf(w, "\n%s\n", i18n.T("help.global"))
	for _, f := range []struct{ flag, key string }{
		{"-p, --project <slug|id>", "flag.project"},
		{"    --json", "flag.json"},
		{"-y, --yes", "flag.yes"},
		{"    --lang <es|en|pt>", "flag.lang"},
		{"-h, --help", "flag.help_root"},
		{"-v, --version", "flag.version"},
	} {
		fmt.Fprintf(w, "  %-25s %s\n", f.flag, i18n.T(f.key))
	}
	fmt.Fprintf(w, "\n%s\n", i18n.T("help.env"))
	for _, e := range helpEnv {
		fmt.Fprintf(w, "  %-25s %s\n", e.name, i18n.T(e.key))
	}
	fmt.Fprintf(w, "\n%s\n", i18n.T("help.examples"))
	fmt.Fprintln(w, "  npx rapibase@latest init")
	fmt.Fprintln(w, "  rapibase deploy ./dist")
	fmt.Fprintln(w, "  rapibase env pull")
	fmt.Fprintf(w, "\n%s rapibase.com/cli\n", i18n.T("help.more"))
}

// keySuffix maps "projects create" → ".create", "deploy [dir]" → "".
func keySuffix(name string) string {
	f := strings.Fields(name)
	if len(f) > 1 && !strings.HasPrefix(f[1], "[") {
		return "." + f[1]
	}
	return ""
}

// cmdHelp is the help of one command: usage, description, subcommands,
// flags and examples, with translated headings.
func cmdHelp(w io.Writer, cmd *cobra.Command) {
	desc := cmd.Long
	if desc == "" {
		desc = cmd.Short
	}
	fmt.Fprintln(w, desc)
	fmt.Fprintf(w, "\n%s:\n  %s\n", i18n.T("help.usage"), cmd.UseLine())
	if subs := cmd.Commands(); len(subs) > 0 {
		fmt.Fprintf(w, "\n%s:\n", i18n.T("help.commands"))
		for _, s := range subs {
			if !s.Hidden {
				fmt.Fprintf(w, "  %-20s %s\n", s.Name(), s.Short)
			}
		}
	}
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(w, "\n%s:\n%s", i18n.T("help.flags"), cmd.LocalFlags().FlagUsages())
	}
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(w, "\n%s:\n%s", i18n.T("help.global"), cmd.InheritedFlags().FlagUsages())
	}
	if cmd.Example != "" {
		fmt.Fprintf(w, "\n%s:\n%s\n", i18n.T("help.examples"), cmd.Example)
	}
}

// examples indents example lines.
func examples(lines ...string) string {
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}
