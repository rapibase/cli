package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/deploy"
	"github.com/rapibase/cli/internal/i18n"
)

// --- functions (spec §8.3) ---

var fnNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+\.(ts|js)$`)

func (a *app) functionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "functions",
		Short:   i18n.T("cmd.functions.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase functions deploy", "rapibase functions list", "rapibase functions rm old.ts"),
	}
	var prune bool
	dep := &cobra.Command{
		Use:     "deploy [dir]",
		Short:   i18n.T("cmd.functions.deploy.short"),
		Long:    i18n.T("cmd.functions.deploy.long"),
		Args:    cobra.MaximumNArgs(1),
		Example: examples("rapibase functions deploy", "rapibase functions deploy ./functions --prune --yes"),
	}
	dep.Flags().BoolVar(&prune, "prune", false, i18n.T("flag.prune"))
	dep.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		dir := filepath.Join(a.cwd, "functions")
		if len(args) == 1 {
			dir = args[0]
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(a.cwd, dir)
			}
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, cerr.New(cerr.Usage, i18n.T("fn.no_dir", rel(a.cwd, dir)))
		}
		var names []string
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".ts") || strings.HasSuffix(e.Name(), ".js")) && !strings.HasSuffix(e.Name(), ".d.ts") {
				if !fnNameRe.MatchString(e.Name()) {
					return nil, cerr.New(cerr.Usage, i18n.T("fn.bad_name", e.Name()))
				}
				names = append(names, e.Name())
			}
		}
		if len(names) == 0 {
			return nil, cerr.New(cerr.Usage, i18n.T("fn.none", rel(a.cwd, dir)))
		}
		if len(names) > 50 {
			return nil, cerr.New(cerr.PlanLimit, i18n.T("fn.too_many", len(names)))
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		for _, n := range names {
			b, err := os.ReadFile(filepath.Join(dir, n))
			if err != nil {
				return nil, err
			}
			fw, _ := w.CreateFormFile("file", n)
			_, _ = io.Copy(fw, bytes.NewReader(b))
		}
		_ = w.Close()
		stop := a.ui.Spinner(i18n.T("fn.uploading", len(names)))
		var out fnState
		err = a.api.Do(a.ctx, "POST", "/api/projects/"+t.IDString()+"/functions", api.BytesBody(buf.Bytes(), w.FormDataContentType()), &out)
		stop()
		if err != nil {
			return nil, err
		}
		var pruned []string
		if prune {
			local := map[string]bool{}
			for _, n := range names {
				local[n] = true
			}
			var extra []string
			for _, f := range out.Files {
				if !local[f] {
					extra = append(extra, f)
				}
			}
			if len(extra) > 0 {
				if !a.ui.Interactive && !a.flags.yes {
					return nil, a.missing(i18n.T("fn.prune_needs_yes", strings.Join(extra, ", ")), "--yes")
				}
				if a.ui.Interactive {
					if ok, _ := a.ui.Confirm(i18n.T("fn.prune_confirm", strings.Join(extra, ", ")), false); !ok {
						extra = nil
					}
				}
				for _, f := range extra {
					if err := a.api.Do(a.ctx, "DELETE", "/api/projects/"+t.IDString()+"/functions/"+url.PathEscape(f), nil, &out); err != nil {
						return nil, err
					}
					pruned = append(pruned, f)
				}
			}
		}
		a.reportFunctions(out, names)
		res := map[string]any{"project": t.view(), "uploaded": names, "functions": out.Functions, "errors": out.Errors, "files": out.Files}
		if len(pruned) > 0 {
			res["pruned"] = pruned
		}
		if len(out.Errors) > 0 {
			return nil, cerr.New(cerr.Failed, i18n.T("fn.compile_errors", len(out.Errors))).With("errors", out.Errors)
		}
		return res, nil
	})
	list := &cobra.Command{
		Use:     "list",
		Short:   i18n.T("cmd.functions.list.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase functions list", "rapibase functions list --json"),
	}
	list.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		var out fnState
		if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString()+"/functions", nil, &out); err != nil {
			return nil, err
		}
		a.reportFunctions(out, nil)
		return map[string]any{"project": t.view(), "functions": out.Functions, "errors": out.Errors, "files": out.Files}, nil
	})
	rm := &cobra.Command{
		Use:     "rm <file>",
		Short:   i18n.T("cmd.functions.rm.short"),
		Args:    cobra.ExactArgs(1),
		Example: examples("rapibase functions rm old.ts"),
	}
	rm.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		if !fnNameRe.MatchString(args[0]) {
			return nil, cerr.New(cerr.Usage, i18n.T("fn.bad_name", args[0]))
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		var out fnState
		if err := a.api.Do(a.ctx, "DELETE", "/api/projects/"+t.IDString()+"/functions/"+url.PathEscape(args[0]), nil, &out); err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("fn.removed", args[0]))
		return map[string]any{"project": t.view(), "deleted": args[0], "files": out.Files}, nil
	})
	cmd.AddCommand(dep, list, rm)
	return cmd
}

type fnState struct {
	Functions []map[string]any `json:"functions"`
	Errors    []string         `json:"errors"`
	Files     []string         `json:"files"`
}

func (a *app) reportFunctions(s fnState, uploaded []string) {
	if uploaded != nil {
		a.ui.Success(i18n.T("fn.deployed", len(uploaded), strings.Join(uploaded, ", ")))
	}
	for _, f := range s.Functions {
		name, _ := f["name"].(string)
		kind, _ := f["kind"].(string)
		extra := ""
		for _, k := range []string{"path", "schedule", "method"} {
			if v, ok := f[k].(string); ok && v != "" {
				extra += " " + v
			}
		}
		a.ui.Printf("  %s %s%s\n", name, a.ui.Dim(kind), a.ui.Dim(extra))
	}
	for _, e := range s.Errors {
		a.ui.Warn(e)
	}
}

// --- services (spec §8.4) ---

var serviceNameRe = regexp.MustCompile(`^[a-z][a-z0-9]{0,19}$`)

func (a *app) servicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "services",
		Short:   i18n.T("cmd.services.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase services", "rapibase services deploy api ./api", "rapibase services deploy api --binary ./bin/api"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString()+"/services", nil, &raw); err != nil {
			return nil, err
		}
		rows := serviceRows(raw)
		if len(rows) == 0 {
			a.ui.Println(i18n.T("services.none"))
		}
		for _, s := range rows {
			where, _ := s["url"].(string)
			if where == "" {
				where, _ = s["internal_url"].(string)
			}
			state := ""
			if asleep, _ := s["asleep"].(bool); asleep {
				state = i18n.T("services.asleep")
			}
			a.ui.Printf("  %-12v %s %s\n", s["name"], a.ui.Cyan(where), a.ui.Dim(state))
		}
		return map[string]any{"project": t.view(), "services": raw}, nil
	})

	var binary, mount, domain string
	var subdomain bool
	dep := &cobra.Command{
		Use:     "deploy <name> [dir]",
		Short:   i18n.T("cmd.services.deploy.short"),
		Long:    i18n.T("cmd.services.deploy.long"),
		Args:    cobra.RangeArgs(1, 2),
		Example: examples("rapibase services deploy api ./api", "rapibase services deploy api --binary ./bin/api --path /api", "rapibase services deploy web ./web --path /"),
	}
	dep.Flags().StringVar(&binary, "binary", "", i18n.T("flag.binary"))
	dep.Flags().StringVar(&mount, "path", "", i18n.T("flag.svc_path"))
	dep.Flags().BoolVar(&subdomain, "subdomain", false, i18n.T("flag.svc_subdomain"))
	dep.Flags().StringVar(&domain, "custom-domain", "", i18n.T("flag.svc_domain"))
	dep.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		name := args[0]
		if !serviceNameRe.MatchString(name) {
			return nil, cerr.New(cerr.Usage, i18n.T("svc.bad_name", name))
		}
		var data []byte
		var st deploy.Stats
		switch {
		case binary != "":
			fi, err := os.Stat(binary)
			if err != nil || fi.IsDir() {
				return nil, cerr.New(cerr.Usage, i18n.T("svc.no_binary", binary))
			}
			if err := deploy.CheckBinary(binary); err != nil {
				return nil, cerr.New(cerr.Usage, i18n.T("svc.bad_binary_"+strings.SplitN(err.Error(), "-", 2)[0], binary)).
					With("hint", "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "+filepath.Base(binary)+" .")
			}
			base := filepath.Base(binary)
			files := []deploy.Entry{{Rel: base, Abs: binary, Size: fi.Size(), Mode: 0o755}}
			hasSpec := false
			if len(args) == 2 { // extra files the binary reads
				more, blocked, err := deploy.Scan(args[1], nil, true)
				if err != nil {
					return nil, cerr.New(cerr.Usage, i18n.T("deploy.no_dir", args[1]))
				}
				if len(blocked) > 0 {
					return nil, cerr.New(cerr.Blocked, i18n.T("svc.secrets", strings.Join(blocked, ", "))).
						With("files", blocked).With("hint", i18n.T("svc.secrets_hint"))
				}
				for _, f := range more {
					if f.Rel == base {
						continue // the binary passed with --binary wins
					}
					hasSpec = hasSpec || f.Rel == "rapibase.json"
					files = append(files, f)
				}
			}
			// a package.json or requirements.txt among the extra files would
			// make the server pick another runtime: say it's this binary
			if !hasSpec {
				spec, err := binarySpec(base)
				if err != nil {
					return nil, err
				}
				defer os.Remove(spec.Abs)
				files = append(files, spec)
			}
			if data, st, err = deploy.TarService(files); err != nil {
				if l, ok := deploy.IsLimit(err); ok {
					return nil, cerr.New(cerr.PlanLimit, i18n.T("deploy.limit", l.What, l.Got, l.Max))
				}
				return nil, err
			}
		default:
			dir := a.cwd
			if len(args) == 2 {
				dir = args[1]
				if !filepath.IsAbs(dir) {
					dir = filepath.Join(a.cwd, dir)
				}
			}
			files, blocked, err := deploy.Scan(dir, nil, true)
			if err != nil {
				return nil, cerr.New(cerr.Usage, i18n.T("deploy.no_dir", rel(a.cwd, dir)))
			}
			if len(blocked) > 0 {
				return nil, cerr.New(cerr.Blocked, i18n.T("svc.secrets", strings.Join(blocked, ", "))).
					With("files", blocked).With("hint", i18n.T("svc.secrets_hint"))
			}
			if len(files) == 0 {
				return nil, cerr.New(cerr.Usage, i18n.T("svc.empty", rel(a.cwd, dir)))
			}
			if data, st, err = deploy.TarService(files); err != nil {
				if l, ok := deploy.IsLimit(err); ok {
					return nil, cerr.New(cerr.PlanLimit, i18n.T("deploy.limit", l.What, l.Got, l.Max))
				}
				return nil, err
			}
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		fields := map[string]string{"source": "cli"}
		if cmd.Flags().Changed("path") {
			fields["path"] = unMSYS(mount)
		}
		if cmd.Flags().Changed("subdomain") {
			fields["subdomain"] = fmt.Sprint(subdomain)
		}
		if cmd.Flags().Changed("custom-domain") {
			fields["custom_domain"] = domain
		}
		body, ct := multipartBody("app", "app.tgz", data, fields)
		stop := a.ui.Spinner(i18n.T("svc.deploying", name, st.Files, deploy.HumanBytes(st.ZipBytes)))
		res, err := a.api.Raw(a.ctx, "POST", "/api/projects/"+t.IDString()+"/services/"+name+"/releases", api.BytesBody(body, ct), true)
		stop()
		if err != nil {
			return nil, err
		}
		var out map[string]any
		_ = json.Unmarshal(res.Body, &out)
		if res.Status == 422 { // build or start failed: the previous release keeps serving
			phase, _ := out["phase"].(string)
			msg, _ := out["error"].(string)
			if logTail, _ := out["log"].(string); logTail != "" && !a.ui.JSON {
				fmt.Fprintln(a.ui.Err, a.ui.Dim(logTail))
			}
			e := cerr.New(cerr.Failed, i18n.T("svc.failed", name, phase, msg))
			for k, v := range out {
				e.With(k, v)
			}
			return nil, e
		}
		if res.Status >= 300 {
			return nil, api.ErrorFrom(res)
		}
		u, _ := out["url"].(string)
		a.ui.Success(i18n.T("svc.ready", name, a.ui.Cyan(u)))
		if notes, ok := out["notes"].([]any); ok {
			for _, n := range notes {
				a.ui.Note(fmt.Sprint(n))
			}
		}
		out["project"] = t.view()
		return out, nil
	})

	restart := &cobra.Command{
		Use: "restart <name>", Short: i18n.T("cmd.services.restart.short"), Args: cobra.ExactArgs(1),
		Example: examples("rapibase services restart api"),
	}
	restart.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		return a.simplePost("/services/"+url.PathEscape(args[0])+"/restart", "POST", i18n.T("svc.restarted", args[0]))
	})
	activate := &cobra.Command{
		Use: "activate <name> <release_id>", Short: i18n.T("cmd.services.activate.short"), Args: cobra.ExactArgs(2),
		Example: examples("rapibase services activate api 20261112-1530-ab12"),
	}
	activate.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		return a.simplePost("/services/"+url.PathEscape(args[0])+"/releases/"+url.PathEscape(args[1])+"/activate", "POST",
			i18n.T("svc.activated", args[0], args[1]))
	})
	rm := &cobra.Command{
		Use: "rm <name>", Short: i18n.T("cmd.services.rm.short"), Args: cobra.ExactArgs(1),
		Example: examples("rapibase services rm worker --yes"),
	}
	rm.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		if a.ui.Interactive {
			if ok, _ := a.ui.Confirm(i18n.T("svc.confirm_rm", args[0]), false); !ok {
				return nil, cerr.New(cerr.Usage, i18n.T("deploy.cancelled"))
			}
		} else if !a.flags.yes {
			return nil, a.missing(i18n.T("svc.rm_needs_yes", args[0]), "--yes")
		}
		return a.simplePost("/services/"+url.PathEscape(args[0]), "DELETE", i18n.T("svc.removed", args[0]))
	})
	var upPath, upDomain string
	var upSub bool
	update := &cobra.Command{
		Use: "update <name>", Short: i18n.T("cmd.services.update.short"), Args: cobra.ExactArgs(1),
		Example: examples("rapibase services update api --path /v2", "rapibase services update web --subdomain=false", `rapibase services update api --custom-domain ""`),
	}
	update.Flags().StringVar(&upPath, "path", "", i18n.T("flag.svc_path"))
	update.Flags().BoolVar(&upSub, "subdomain", false, i18n.T("flag.svc_subdomain"))
	update.Flags().StringVar(&upDomain, "custom-domain", "", i18n.T("flag.svc_domain"))
	update.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		body := map[string]any{}
		if cmd.Flags().Changed("path") {
			body["path"] = unMSYS(upPath)
		}
		if cmd.Flags().Changed("subdomain") {
			body["subdomain"] = upSub
		}
		if cmd.Flags().Changed("custom-domain") {
			body["custom_domain"] = upDomain
		}
		if len(body) == 0 {
			return nil, a.missing(i18n.T("svc.update_nothing"), "--path", "--subdomain", "--custom-domain")
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		var out json.RawMessage
		if err := a.api.Do(a.ctx, "PATCH", "/api/projects/"+t.IDString()+"/services/"+url.PathEscape(args[0]), api.JSONBody(body), &out); err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("svc.updated", args[0]))
		return map[string]any{"project": t.view(), "service": args[0], "result": out}, nil
	})
	cmd.AddCommand(dep, update, restart, activate, rm)
	return cmd
}

// binarySpec writes a temporary rapibase.json that runs the binary.
func binarySpec(base string) (deploy.Entry, error) {
	b, _ := json.Marshal(map[string]string{"runtime": "binary", "start": "./" + base})
	f, err := os.CreateTemp("", "rapibase-spec-*.json")
	if err != nil {
		return deploy.Entry{}, err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return deploy.Entry{}, err
	}
	return deploy.Entry{Rel: "rapibase.json", Abs: f.Name(), Size: int64(len(b)), Mode: 0o644}, nil
}

// simplePost calls a project route that changes something and prints one
// success line.
func (a *app) simplePost(path, method, done string) (any, error) {
	t, err := a.resolve(true)
	if err != nil {
		return nil, err
	}
	var out json.RawMessage
	if err := a.api.Do(a.ctx, method, "/api/projects/"+t.IDString()+path, nil, &out); err != nil {
		return nil, err
	}
	a.ui.Success(done)
	res := map[string]any{"project": t.view()}
	if len(out) > 0 {
		res["result"] = out
	}
	return res, nil
}

// --- logs (spec §8.5) ---

func (a *app) logsCmd() *cobra.Command {
	var service, container string
	var tail int
	var follow bool
	cmd := &cobra.Command{
		Use:     "logs",
		Short:   i18n.T("cmd.logs.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase logs", "rapibase logs --service api --tail 50", "rapibase logs --follow", "rapibase logs --container db"),
	}
	cmd.Flags().StringVar(&service, "service", "", i18n.T("flag.log_service"))
	cmd.Flags().StringVar(&container, "container", "", i18n.T("flag.log_container"))
	cmd.Flags().IntVar(&tail, "tail", 200, i18n.T("flag.tail"))
	cmd.Flags().BoolVar(&follow, "follow", false, i18n.T("flag.follow"))
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		which := ""
		switch {
		case service != "":
			which = "svc:" + service
		case container == "db" || container == "storage":
			which = container
		case container != "" && container != "app":
			return nil, cerr.New(cerr.Usage, i18n.T("logs.bad_container", container))
		}
		fetch := func(n int) ([]string, error) {
			q := url.Values{"tail": {fmt.Sprint(n)}}
			if which != "" {
				q.Set("container", which)
			}
			var out struct {
				Logs string `json:"logs"`
			}
			if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString()+"/logs?"+q.Encode(), nil, &out); err != nil {
				return nil, err
			}
			lines := strings.Split(strings.TrimRight(out.Logs, "\n"), "\n")
			if len(lines) == 1 && lines[0] == "" {
				return nil, nil
			}
			return lines, nil
		}
		// the server answers 200 lines to anything above 2000: ask for the most
		// it gives instead of silently getting less
		lines, err := fetch(min(tail, 2000))
		if err != nil {
			return nil, err
		}
		all := append([]string{}, lines...)
		for _, l := range lines {
			a.ui.Println(l)
		}
		if follow {
			// polling every 2 s; lines carry docker timestamps, so the last
			// one seen marks where to continue. Without a TTY it ends at 90 s.
			seen := map[string]bool{}
			for _, l := range lines {
				seen[l] = true
			}
			deadline := time.Now().Add(a.waitBudget(24 * time.Hour))
			for time.Now().Before(deadline) {
				select {
				case <-a.ctx.Done():
					return map[string]any{"project": t.view(), "lines": all}, nil
				case <-time.After(2 * time.Second):
				}
				more, err := fetch(200)
				if err != nil {
					return nil, err
				}
				for _, l := range more {
					if !seen[l] {
						seen[l] = true
						all = append(all, l)
						a.ui.Println(l)
					}
				}
				if len(seen) > 5000 { // keep memory bounded on long follows
					seen = map[string]bool{}
					for _, l := range more {
						seen[l] = true
					}
				}
			}
		}
		return map[string]any{"project": t.view(), "lines": all}, nil
	})
	return cmd
}

// --- config and restart ---

func (a *app) configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   i18n.T("cmd.config.short"),
		Args:    cobra.NoArgs,
		Example: examples("rapibase config get", "rapibase config set SMTP_HOST=smtp.example.org SMTP_PORT=587"),
	}
	get := &cobra.Command{Use: "get", Short: i18n.T("cmd.config.get.short"), Args: cobra.NoArgs}
	get.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := a.api.Do(a.ctx, "GET", "/api/projects/"+t.IDString()+"/config", nil, &raw); err != nil {
			return nil, err
		}
		a.printRaw(raw)
		return map[string]any{"project": t.view(), "config": raw}, nil
	})
	set := &cobra.Command{
		Use: "set KEY=VALUE [KEY=VALUE…]", Short: i18n.T("cmd.config.set.short"), Args: cobra.MinimumNArgs(1),
		Example: examples("rapibase config set AUTH_SIGNUP_ENABLED=true", "rapibase config set CORS_ORIGINS=https://my.app"),
	}
	set.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		env := map[string]string{}
		for _, kv := range args {
			k, v, ok := strings.Cut(kv, "=")
			if !ok || strings.TrimSpace(k) == "" {
				return nil, cerr.New(cerr.Usage, i18n.T("config.bad_pair", kv))
			}
			env[strings.TrimSpace(k)] = v
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := a.api.Do(a.ctx, "POST", "/api/projects/"+t.IDString()+"/config", api.JSONBody(map[string]any{"env": env}), &raw); err != nil {
			return nil, err
		}
		var res struct {
			Rejected []string `json:"rejected"`
		}
		_ = json.Unmarshal(raw, &res)
		rejected := map[string]bool{}
		for _, k := range res.Rejected {
			rejected[k] = true
		}
		keys := make([]string, 0, len(env))
		for k := range env {
			if !rejected[k] {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			a.ui.Success(i18n.T("config.saved", strings.Join(keys, ", ")))
		}
		if len(res.Rejected) > 0 {
			sort.Strings(res.Rejected)
			a.ui.Warn(i18n.T("config.rejected", strings.Join(res.Rejected, ", ")))
		}
		out := map[string]any{"project": t.view(), "set": keys, "result": raw}
		if len(res.Rejected) > 0 {
			out["rejected"] = res.Rejected
		}
		return out, nil
	})
	cmd.AddCommand(get, set)
	return cmd
}

func (a *app) restartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "restart", Short: i18n.T("cmd.restart.short"), Args: cobra.NoArgs,
		Example: examples("rapibase restart"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, _ []string) (any, error) {
		return a.simplePost("/restart", "POST", i18n.T("restart.done"))
	})
	return cmd
}
