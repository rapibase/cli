package cli

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/envfile"
	"github.com/rapibase/cli/internal/i18n"
)

func (a *app) envCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: i18n.T("cmd.env.short"),
		Args:  cobra.NoArgs,
	}
	var framework, file string
	pull := &cobra.Command{
		Use:     "pull",
		Short:   i18n.T("cmd.env.pull.short"),
		Long:    i18n.T("cmd.env.pull.long"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase env pull", "rapibase env pull --framework vite", "rapibase env pull --project my-app-staging --file .env.staging"),
	}
	pull.Flags().StringVar(&framework, "framework", "", i18n.T("flag.framework", strings.Join(envfile.FrameworkNames(), "|")))
	pull.Flags().StringVar(&file, "file", ".env", i18n.T("flag.env_file"))
	pull.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		res, err := a.envPull(t, file, framework)
		if err != nil {
			return nil, err
		}
		return map[string]any{"project": t.view(), "project_source": t.Source, "env_file": rel(a.cwd, res.File),
			"env_vars": res.Vars, "framework": res.Framework, "gitignore_updated": res.Gitignore}, nil
	})
	cmd.AddCommand(pull)
	return cmd
}

// envPull downloads the keys and writes them. Values never reach the
// terminal, the JSON or the debug log (spec §8.2).
func (a *app) envPull(t *target, file, framework string) (*envfile.Result, error) {
	if framework != "" {
		if _, ok := envfile.Frameworks[framework]; !ok {
			return nil, cerr.New(cerr.Usage, i18n.T("env.bad_framework", framework, strings.Join(envfile.FrameworkNames(), ", ")))
		}
	}
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(a.cwd, file)
	}
	if framework == "" {
		framework = envfile.DetectFramework(filepath.Dir(path))
	}
	// checked before asking for the keys: a tracked .env would end up in git
	if abs, _ := filepath.Abs(path); envfile.Tracked(abs) {
		return nil, trackedErr(file)
	}
	var out struct {
		Env map[string]string `json:"env"`
	}
	if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString()+"/env", nil, &out); err != nil {
		return nil, err
	}
	res, err := envfile.Write(path, out.Env, framework)
	if errors.Is(err, envfile.ErrTracked) {
		return nil, trackedErr(file)
	}
	if err != nil {
		return nil, err
	}
	a.ui.Success(i18n.T("env.written", rel(a.cwd, res.File), strings.Join(res.Vars, ", ")))
	if res.Gitignore {
		a.ui.Note(i18n.T("env.gitignored", filepath.Base(res.File)))
	}
	if res.Framework != "" {
		a.ui.Note(i18n.T("env.framework", res.Framework, envfile.Frameworks[res.Framework]))
	}
	return res, nil
}

func trackedErr(file string) error {
	return cerr.New(cerr.Blocked, i18n.T("env.tracked", file)).
		With("hint", "git rm --cached "+file+"  &&  git commit -m \"Stop tracking "+file+"\"")
}
