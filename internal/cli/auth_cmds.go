package cli

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/auth"
	"github.com/rapibase/cli/internal/browser"
	"github.com/rapibase/cli/internal/build"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/config"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/skills"
	"github.com/rapibase/cli/internal/ui"
)

func (a *app) loginCmd() *cobra.Command {
	var wait bool
	var ref string
	cmd := &cobra.Command{
		Use:     "login",
		Short:   i18n.T("cmd.login.short"),
		Long:    i18n.T("cmd.login.long"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase login", "rapibase login --ref ABC123", "rapibase login --wait --json"),
	}
	cmd.Flags().BoolVar(&wait, "wait", false, i18n.T("flag.wait_login"))
	cmd.Flags().StringVar(&ref, "ref", "", i18n.T("flag.ref"))
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		if a.sess.Source() == auth.SourceEnv {
			return nil, cerr.New(cerr.Usage, i18n.T("login.env_token"))
		}
		if a.sess.Source() == auth.SourceLogin {
			me, err := a.me()
			if err == nil {
				a.ui.Success(i18n.T("login.already", maskEmail(me.Email)))
				return map[string]any{"email": me.Email, "connection": me.Connection.Name, "already": true}, nil
			}
			if ce := cerr.As(err); ce.Code != cerr.SessionExpired && ce.Code != cerr.NeedsLogin {
				return nil, err
			}
		}
		me, err := a.login(ref)
		if err != nil {
			return nil, err
		}
		return map[string]any{"email": me.Email, "connection": me.Connection.Name}, nil
	})
	return cmd
}

// ensureLogin logs in if there is no valid session (used by init).
func (a *app) ensureLogin(ref string) (*api.Me, error) {
	if a.sess.LoggedIn() {
		me, err := a.me()
		if err == nil {
			return me, nil
		}
		if ce := cerr.As(err); ce.Code != cerr.SessionExpired && ce.Code != cerr.NeedsLogin {
			return nil, err
		}
		if a.sess.Source() != auth.SourceLogin {
			return nil, err // a bad RAPIBASE_TOKEN is not fixed by a browser login
		}
	}
	return a.login(ref)
}

// login runs the device flow (spec §5.1). With a TTY it waits for the
// approval; without one it never waits for a person before showing what
// they need: the first run ends with login_pending (exit 10) and running the
// same command again keeps polling the same code for up to 90 s.
func (a *app) login(ref string) (*api.Me, error) {
	p, isNew, err := a.sess.StartDevice(a.ctx, ref)
	if err != nil {
		return nil, err
	}
	opened := false
	if isNew && browser.Allowed(a.flags.noBrowser, ui.IsCI()) {
		opened = browser.Open(p.VerificationURIComplete)
	}
	pending := func() error {
		return cerr.New(cerr.LoginPending, i18n.T("login.pending")).
			With("verification_uri", p.VerificationURI).
			With("verification_uri_complete", p.VerificationURIComplete).
			With("user_code", p.UserCode).
			With("expires_in", int(time.Until(p.ExpiresAt).Seconds())).
			With("browser_opened", opened)
	}
	// without a TTY a new code is shown before any waiting: the agent only
	// sees the output when the command ends. Running it again (or `login
	// --wait`) waits up to 90 s for the same code.
	if !a.ui.Interactive && isNew {
		return nil, pending()
	}

	if a.ui.Interactive {
		a.ui.Println()
		if opened {
			a.ui.Println("  " + i18n.T("login.opened"))
		} else {
			a.ui.Println("  " + i18n.T("login.open_link"))
		}
		a.ui.Println("    " + a.ui.Cyan(p.VerificationURI) + "   " + i18n.T("login.code", a.ui.Bold(p.UserCode)))
		a.ui.Println()
	}
	stop := a.ui.Spinner(i18n.T("login.waiting"))
	budget := time.Until(p.ExpiresAt)
	if !a.ui.Interactive {
		budget = 90 * time.Second
	}
	st, err := a.sess.Poll(a.ctx, p, budget)
	stop()
	if err != nil {
		return nil, err
	}
	switch st {
	case auth.StillPending:
		return nil, pending()
	case auth.Denied:
		return nil, cerr.New(cerr.NeedsLogin, i18n.T("login.denied"))
	case auth.Expired:
		return nil, cerr.New(cerr.NeedsLogin, i18n.T("login.expired")).With("hint", i18n.T("login.expired_hint"))
	}
	me, err := a.me()
	if err != nil {
		return nil, err
	}
	a.sess.SetEmail(me.Email)
	a.ui.Success(i18n.T("login.done", maskEmail(me.Email)))
	return me, nil
}

func (a *app) logoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logout",
		Short:   i18n.T("cmd.logout.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase logout"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		if a.sess.Source() == auth.SourceEnv || a.sess.Source() == auth.SourceDeploy {
			return nil, cerr.New(cerr.Usage, i18n.T("logout.env_token", string(a.sess.Source())))
		}
		if !a.sess.LoggedIn() {
			a.ui.Success(i18n.T("logout.none"))
			return map[string]any{"logged_out": true, "revoked": false}, nil
		}
		revoked, err := a.sess.Logout(a.ctx)
		if err != nil {
			return nil, err
		}
		if revoked {
			a.ui.Success(i18n.T("logout.done"))
		} else {
			a.ui.Success(i18n.T("logout.local_only"))
			a.ui.Note(config.APIURL() + "/app/connections")
		}
		return map[string]any{"logged_out": true, "revoked": revoked}, nil
	})
	return cmd
}

func (a *app) whoamiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "whoami",
		Short:   i18n.T("cmd.whoami.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase whoami", "rapibase whoami --json"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		me, err := a.me()
		if err != nil {
			return nil, err
		}
		ps, err := a.listProjects()
		if err != nil {
			return nil, err
		}
		a.ui.Printf("%s %s\n", i18n.T("whoami.account"), a.ui.Bold(maskEmail(me.Email)))
		a.ui.Printf("%s %s %s\n", i18n.T("whoami.session"), me.Connection.Name, a.ui.Dim("("+sourceLabel(a.sess.Source())+")"))
		scope := i18n.T("whoami.some_projects", len(ps))
		if me.Connection.AllProjects {
			scope = i18n.T("whoami.all_projects", modeLabel(me.Connection.DefaultMode))
		}
		a.ui.Printf("%s %s\n", i18n.T("whoami.reach"), scope)
		if me.CanCreate() {
			a.ui.Note(i18n.T("whoami.can_create"))
		}
		for _, p := range ps {
			a.ui.Printf("  %s %s %s\n", p.Slug, a.ui.Dim(p.OrgName), a.ui.Dim(modeLabel(p.Mode)))
		}
		a.ui.Note(i18n.T("whoami.manage", me.Connection.URL))
		type proj struct {
			Slug string `json:"slug"`
			ID   int64  `json:"id"`
			Org  string `json:"org"`
			Mode string `json:"mode"`
		}
		list := []proj{}
		for _, p := range ps {
			list = append(list, proj{p.Slug, p.ID, p.OrgName, p.Mode})
		}
		return map[string]any{
			"email":           maskEmail(me.Email),
			"source":          string(a.sess.Source()),
			"connection":      me.Connection,
			"projects":        list,
			"create_projects": me.CanCreate(),
		}, nil
	})
	return cmd
}

// me is GET /api/me.
func (a *app) me() (*api.Me, error) {
	if a.sess.IsDeployToken() {
		return nil, cerr.New(cerr.Forbidden, i18n.T("auth.deploy_only"))
	}
	var me api.Me
	if err := a.api.Do(a.ctx, "GET", "/api/me", nil, &me); err != nil {
		return nil, err
	}
	return &me, nil
}

// maskEmail turns joel@example.com into j•••@example.com (spec §3.2).
func maskEmail(e string) string {
	user, domain, ok := strings.Cut(e, "@")
	if !ok || user == "" {
		return e
	}
	return user[:1] + "•••@" + domain
}

func sourceLabel(s auth.Source) string {
	switch s {
	case auth.SourceLogin:
		return i18n.T("whoami.src_login")
	case auth.SourceEnv:
		return "RAPIBASE_TOKEN"
	case auth.SourceDeploy:
		return "RAPIBASE_DEPLOY_TOKEN"
	}
	return "-"
}

func modeLabel(m string) string {
	switch m {
	case "write":
		return i18n.T("mode.write")
	case "read":
		return i18n.T("mode.read")
	}
	return m
}

// --- version check (spec §10.4) ---

// checkVersion stops incompatible versions (exit 8) and, at most once a
// day, tells the person there is a newer one. It never updates by itself
// and stays quiet in CI, with --json and with RAPIBASE_NO_UPDATE_CHECK.
func (a *app) checkVersion(cmd *cobra.Command) error {
	if build.Dev() || cmd.Name() == "help" || cmd.Name() == "rapibase" {
		return nil
	}
	hc := config.LoadHost(a.host)
	if time.Since(hc.CheckedAt) > 24*time.Hour {
		ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
		var v api.Version
		err := a.api.Do(ctx, "GET", "/api/cli/version", nil, &v)
		cancel()
		if err == nil {
			hc.CheckedAt, hc.Latest, hc.MinSupported = time.Now(), v.Latest, v.MinSupported
			_ = config.SaveHost(a.host, hc)
		}
	}
	if hc.MinSupported != "" && skills.Newer(hc.MinSupported, build.Version) {
		return cerr.New(cerr.UpgradeRequired, i18n.T("version.unsupported", build.Version, hc.MinSupported)).
			With("hint", "npx rapibase@latest "+cmd.Name()+"  ·  npm i -g rapibase@latest").
			With("min_supported", hc.MinSupported)
	}
	quiet := a.ui.JSON || ui.IsCI() || os.Getenv("RAPIBASE_NO_UPDATE_CHECK") != ""
	if !quiet && hc.Latest != "" && skills.Newer(hc.Latest, build.Version) {
		a.ui.Warn(i18n.T("version.newer", hc.Latest, build.Version))
	}
	return nil
}
