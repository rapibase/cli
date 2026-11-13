// Package cli wires the commands (spec §3) on top of cobra.
//
// Every command returns (data, error). Main prints the single JSON object in
// --json mode, or the human error with what to do next, and maps the error
// code to the exit code (spec §4.3).
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/agents"
	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/auth"
	"github.com/rapibase/cli/internal/build"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/config"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/ui"
)

// globalFlags are the options every command accepts (spec §3.1).
type globalFlags struct {
	project   string
	json      bool
	yes       bool
	lang      string
	noBrowser bool
	debug     bool
}

// app is the state of one run.
type app struct {
	ctx   context.Context
	flags globalFlags
	ui    *ui.UI
	host  string
	api   *api.Client
	sess  *auth.Session
	cwd   string

	projects []api.Project // GET /api/projects, cached for the run
	// draftName: the name typed for a new project, offered again when
	// init comes back to its menu
	draftName string

	data any
	err  error
}

// Main runs the CLI and returns the exit code.
func Main(args []string) int {
	i18n.Detect(preScanLang(args))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	a := &app{ctx: ctx, host: config.APIURL()}
	a.cwd, _ = os.Getwd()
	root := a.rootCmd()
	root.SetArgs(args)
	if err := root.ExecuteContext(ctx); err != nil {
		var ce *cerr.Error
		if errors.As(err, &ce) {
			a.err = err // from setup (bad --lang, unreadable credentials…)
		} else {
			// cobra errors: unknown command or flag, wrong number of args
			a.err = a.usageError(root, args, err)
		}
	}
	if a.ui == nil { // failed before PersistentPreRun (bad flag)
		a.ui = ui.New(hasFlag(args, "--json"), hasFlag(args, "--yes") || hasFlag(args, "-y"))
	}
	if a.err != nil {
		return a.fail(a.err)
	}
	if a.ui.JSON && a.data != nil {
		a.ui.Result(a.data)
	}
	return 0
}

// preScanLang reads --lang before cobra parses, so help texts come out in
// the right language.
func preScanLang(args []string) string {
	for i, x := range args {
		if v, ok := strings.CutPrefix(x, "--lang="); ok {
			return v
		}
		if x == "--lang" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func hasFlag(args []string, f string) bool {
	for _, x := range args {
		if x == f || strings.HasPrefix(x, f+"=") {
			return true
		}
	}
	return false
}

// setup runs before every command: UI, HTTP client, session.
func (a *app) setup(cmd *cobra.Command) error {
	if a.flags.lang != "" {
		if l, ok := i18n.Parse(a.flags.lang); ok {
			i18n.Set(l)
		} else {
			return cerr.New(cerr.Usage, i18n.T("err.bad_lang", a.flags.lang))
		}
	}
	a.ui = ui.New(a.flags.json, a.flags.yes)
	a.api = api.New(a.host)
	a.api.CI = ui.IsCI()
	if r := agents.Running(); r != "" {
		a.api.Agent = string(r)
	}
	if a.flags.debug {
		a.api.Debug = os.Stderr
	}
	sess, err := auth.Load(a.host, api.New(a.host), allowsDeployToken(cmd))
	if err != nil {
		return err
	}
	sess.Client.Agent, sess.Client.CI, sess.Client.Debug = a.api.Agent, a.api.CI, a.api.Debug
	a.sess = sess
	a.api.Auth = sess
	return nil
}

// allowsDeployToken: RAPIBASE_DEPLOY_TOKEN reaches what the control plane
// accepts it for: the site's deploys and the services (deploy, activate,
// restart, update, rm and the list). Functions, SQL, the API, logs and
// config need a personal token.
func allowsDeployToken(cmd *cobra.Command) bool {
	p := cmd.CommandPath()
	return p == "rapibase deploy" || p == "rapibase deploys" || strings.HasPrefix(p, "rapibase deploys ") ||
		p == "rapibase services" || strings.HasPrefix(p, "rapibase services ")
}

// run adapts a command function to cobra: results are kept for Main.
func (a *app) run(fn func(cmd *cobra.Command, args []string) (any, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		a.data, a.err = fn(cmd, args)
		return nil
	}
}

// fail prints the error (human or JSON) and returns the exit code.
func (a *app) fail(err error) int {
	if errors.Is(err, context.Canceled) {
		if !a.ui.JSON {
			fmt.Fprintln(a.ui.Err)
		}
		return 130
	}
	e := cerr.As(err)
	if a.ui.JSON {
		a.ui.ErrorJSON(e.Code, e.Message, e.Extra)
		return cerr.ExitCode(e.Code)
	}
	w := a.ui.Err
	fmt.Fprintln(w, a.ui.Red("✖")+" "+e.Message)
	switch e.Code {
	case cerr.LoginPending:
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  "+i18n.T("login.open_link"))
		fmt.Fprintln(w, "    "+a.ui.Cyan(e.Str("verification_uri_complete")))
		fmt.Fprintln(w, "    "+i18n.T("login.code", a.ui.Bold(e.Str("user_code"))))
	}
	if list, ok := e.Extra["projects"].([]projectChoice); ok && len(list) > 0 {
		fmt.Fprintln(w, "  "+i18n.T("err.available_projects"))
		for _, p := range list {
			fmt.Fprintf(w, "    %s  %s\n", p.Slug, a.ui.Dim(p.Org))
		}
	}
	if flags, ok := e.Extra["missing"].([]string); ok && len(flags) > 0 {
		fmt.Fprintln(w, "  "+i18n.T("err.missing_flags", strings.Join(flags, ", ")))
	}
	if u := e.Str("action_url"); u != "" {
		fmt.Fprintln(w, "  → "+a.ui.Cyan(u))
	}
	if h := e.Str("hint"); h != "" {
		fmt.Fprintln(w, "  "+h)
	}
	if ra, ok := e.Extra["retry_after"]; ok {
		fmt.Fprintln(w, "  "+i18n.T("err.retry_after", fmt.Sprint(ra)))
	}
	return cerr.ExitCode(e.Code)
}

// usageError turns cobra's errors into a usage error, with the closest
// command for a typo ("did you mean deploy?").
func (a *app) usageError(root *cobra.Command, args []string, err error) error {
	msg := err.Error()
	if strings.HasPrefix(msg, "unknown command") && len(args) > 0 {
		e := cerr.New(cerr.Usage, i18n.T("err.unknown_command", args[0]))
		if s := root.SuggestionsFor(args[0]); len(s) > 0 {
			e.With("hint", i18n.T("err.did_you_mean", s[0]))
		} else {
			e.With("hint", i18n.T("err.see_help"))
		}
		return e
	}
	return cerr.New(cerr.Usage, msg).With("hint", i18n.T("err.see_help"))
}

// --- helpers used by the commands ---

// missing fails with missing_flag in non-interactive mode (spec §4.1).
func (a *app) missing(msg string, flags ...string) *cerr.Error {
	return cerr.New(cerr.MissingFlag, msg).With("missing", flags)
}

// wait budget: without a TTY nothing blocks more than 90 s (spec §2.2).
func (a *app) waitBudget(interactive time.Duration) time.Duration {
	if a.ui.Interactive {
		return interactive
	}
	return 90 * time.Second
}

func (a *app) version() string { return build.Version }
