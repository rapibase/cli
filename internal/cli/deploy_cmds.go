package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/deploy"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/project"
	"github.com/rapibase/cli/internal/ui"
)

func (a *app) deployCmd() *cobra.Command {
	var exclude []string
	var allowDirty, force bool
	cmd := &cobra.Command{
		Use:     "deploy [dir]",
		Short:   i18n.T("cmd.deploy.short"),
		Long:    i18n.T("cmd.deploy.long"),
		Args:    cobra.MaximumNArgs(1),
		Example: examples("rapibase deploy", "rapibase deploy ./dist", "rapibase deploy . --exclude 'drafts/'"),
	}
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, i18n.T("flag.exclude"))
	cmd.Flags().BoolVar(&allowDirty, "allow-dirty", false, i18n.T("flag.allow_dirty"))
	cmd.Flags().BoolVar(&force, "force", false, i18n.T("flag.force_deploy"))
	cmd.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		dir, err := a.siteDir(args)
		if err != nil {
			return nil, err
		}
		// the skill's rule made code (spec §8.1). In CI the workflow IS the
		// deploy, so neither check applies there.
		if !ui.IsCI() {
			if wf := deploy.ActionsWorkflow(project.RepoRoot(a.cwd)); wf != "" && !force {
				if !a.ui.Interactive {
					return nil, cerr.New(cerr.Blocked, i18n.T("deploy.has_workflow", rel(a.cwd, wf))).With("hint", i18n.T("deploy.has_workflow_hint"))
				}
				a.ui.Warn(i18n.T("deploy.has_workflow", rel(a.cwd, wf)))
				if ok, _ := a.ui.Confirm(i18n.T("deploy.confirm_anyway"), false); !ok {
					return nil, cerr.New(cerr.Usage, i18n.T("deploy.cancelled"))
				}
			}
			if deploy.GitDirty(a.cwd) && !allowDirty {
				if !a.ui.Interactive {
					return nil, cerr.New(cerr.Blocked, i18n.T("deploy.dirty")).With("hint", i18n.T("deploy.dirty_hint"))
				}
				a.ui.Warn(i18n.T("deploy.dirty"))
			}
		}
		files, blocked, err := deploy.Scan(dir, deploy.Exclusion(exclude), false)
		if err != nil {
			return nil, err
		}
		if len(blocked) > 0 {
			return nil, cerr.New(cerr.Blocked, i18n.T("deploy.secrets", rel(a.cwd, dir), strings.Join(blocked, ", "))).
				With("files", blocked).With("hint", i18n.T("deploy.secrets_hint"))
		}
		zipped, st, err := deploy.ZipSite(files)
		if err != nil {
			if l, ok := deploy.IsLimit(err); ok {
				return nil, cerr.New(cerr.PlanLimit, i18n.T("deploy.limit", l.What, l.Got, l.Max))
			}
			return nil, err
		}
		if !st.HasIndex {
			return nil, cerr.New(cerr.Usage, i18n.T("deploy.no_index", rel(a.cwd, dir)))
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		body, ct := multipartBody("site", "site.zip", zipped, map[string]string{"source": "cli"})
		stop := a.ui.Spinner(i18n.T("deploy.uploading", st.Files, deploy.HumanBytes(st.ZipBytes)))
		var out struct {
			ReleaseID string `json:"release_id"`
			URL       string `json:"url"`
			Files     int    `json:"files"`
		}
		err = a.api.Do(a.ctx, "POST", "/api/projects/"+t.IDString()+"/deploys", api.BytesBody(body, ct), &out)
		stop()
		if err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("deploy.done", a.ui.Cyan(out.URL)))
		a.ui.Note(i18n.T("deploy.release", out.ReleaseID))
		return map[string]any{"project": t.view(), "project_source": t.Source, "url": out.URL,
			"release_id": out.ReleaseID, "files": st.Files, "bytes": st.Bytes, "dir": rel(a.cwd, dir)}, nil
	})
	return cmd
}

// siteDir resolves the folder to publish: the argument, or the build
// folder of the framework (spec §8.1).
func (a *app) siteDir(args []string) (string, error) {
	if len(args) == 1 {
		d := args[0]
		if !filepath.IsAbs(d) {
			d = filepath.Join(a.cwd, d)
		}
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			return "", cerr.New(cerr.Usage, i18n.T("deploy.no_dir", args[0]))
		}
		return d, nil
	}
	det := deploy.DetectDir(a.cwd)
	if det.SSR {
		return "", cerr.New(cerr.Usage, i18n.T("deploy.next_ssr")).With("hint", "rapibase services deploy web --path /")
	}
	if det.Dir == "" {
		return "", cerr.New(cerr.MissingFlag, i18n.T("deploy.no_detect")).With("missing", []string{"[dir]"})
	}
	d := filepath.Join(a.cwd, det.Dir)
	if _, err := os.Stat(d); err != nil {
		e := cerr.New(cerr.Usage, i18n.T("deploy.not_built", det.Dir))
		if det.Build != "" {
			e.With("hint", i18n.T("deploy.build_first", det.Build))
		}
		return "", e
	}
	a.ui.Progress(i18n.T("deploy.detected", det.Dir, det.Framework))
	return d, nil
}

func multipartBody(field, filename string, data []byte, fields map[string]string) ([]byte, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	fw, _ := w.CreateFormFile(field, filename)
	_, _ = io.Copy(fw, bytes.NewReader(data))
	_ = w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

// --- deploys (releases and rollback) ---

func (a *app) deploysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploys",
		Short:   i18n.T("cmd.deploys.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase deploys", "rapibase deploys activate 20261112-1530-ab12"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		var out struct {
			Deploys []struct {
				ID        string `json:"id"`
				Files     int    `json:"files"`
				SizeBytes int64  `json:"size_bytes"`
				Source    string `json:"source"`
				CreatedAt string `json:"created_at"`
				Active    bool   `json:"active"`
			} `json:"deploys"`
			URL string `json:"url"`
		}
		if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString()+"/deploys", nil, &out); err != nil {
			return nil, err
		}
		if len(out.Deploys) == 0 {
			a.ui.Println(i18n.T("deploys.none"))
		}
		for _, d := range out.Deploys {
			mark := " "
			if d.Active {
				mark = a.ui.Green("●")
			}
			a.ui.Printf(" %s %s  %s\n", mark, d.ID, a.ui.Dim(fmt.Sprintf("%s · %d · %s · %s", shortDate(d.CreatedAt), d.Files, deploy.HumanBytes(d.SizeBytes), d.Source)))
		}
		return map[string]any{"project": t.view(), "deploys": out.Deploys, "url": out.URL}, nil
	})
	activate := &cobra.Command{
		Use:     "activate <release_id>",
		Short:   i18n.T("cmd.deploys.activate.short"),
		Args:    cobra.ExactArgs(1),
		Example: examples("rapibase deploys activate 20261112-1530-ab12"),
	}
	activate.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		var out map[string]any
		if err := a.api.Do(a.ctx, "POST", "/api/projects/"+t.IDString()+"/deploys/"+args[0]+"/activate", nil, &out); err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("deploys.activated", args[0]))
		return map[string]any{"project": t.view(), "release_id": args[0], "result": out}, nil
	})
	cmd.AddCommand(activate)
	return cmd
}

// printRaw prints a JSON response in human mode. With --json it prints
// nothing: the response goes inside the single final object.
func (a *app) printRaw(raw json.RawMessage) {
	if a.ui.JSON {
		return
	}
	var v any
	if json.Unmarshal(raw, &v) == nil {
		a.ui.PrettyJSON(v)
		return
	}
	a.ui.Println(string(raw))
}
