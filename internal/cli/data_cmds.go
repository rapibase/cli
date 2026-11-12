package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/i18n"
)

// --- sql ---

func (a *app) sqlCmd() *cobra.Command {
	var file, paramsJSON string
	var params []string
	var timeout float64
	cmd := &cobra.Command{
		Use:   "sql [query]",
		Short: i18n.T("cmd.sql.short"),
		Long:  i18n.T("cmd.sql.long"),
		Args:  cobra.MaximumNArgs(1),
		Example: examples(
			`rapibase sql "select count(*) from orders"`,
			`rapibase sql "select * from orders where status = $1" --param paid`,
			"rapibase sql --file migrations/001_init.sql --timeout 300",
			"cat seed.sql | rapibase sql -",
		),
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", i18n.T("flag.sql_file"))
	cmd.Flags().StringArrayVar(&params, "param", nil, i18n.T("flag.sql_param"))
	cmd.Flags().StringVar(&paramsJSON, "params", "", i18n.T("flag.sql_params"))
	cmd.Flags().Float64Var(&timeout, "timeout", 0, i18n.T("flag.sql_timeout"))
	cmd.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		query, err := a.readQuery(args, file)
		if err != nil {
			return nil, err
		}
		body := map[string]any{"sql": query}
		switch {
		case paramsJSON != "" && len(params) > 0:
			return nil, cerr.New(cerr.Usage, i18n.T("sql.params_both"))
		case paramsJSON != "":
			var ps []any
			if err := json.Unmarshal([]byte(paramsJSON), &ps); err != nil {
				return nil, cerr.New(cerr.Usage, i18n.T("sql.bad_params"))
			}
			body["params"] = ps
		case len(params) > 0:
			ps := make([]any, len(params))
			for i, p := range params {
				ps[i] = scalar(p)
			}
			body["params"] = ps
		}
		if timeout > 0 {
			body["timeout_seconds"] = timeout
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		stop := a.ui.Spinner(i18n.T("sql.running"))
		var out struct {
			Result json.RawMessage `json:"result"`
		}
		err = a.api.Do(a.ctx, "POST", "/api/projects/"+t.IDString()+"/sql", api.JSONBody(body), &out)
		stop()
		if err != nil {
			return nil, err
		}
		var res struct {
			Columns      []string         `json:"columns"`
			Rows         []map[string]any `json:"rows"`
			RowsAffected int64            `json:"rows_affected"`
			Duration     string           `json:"duration"`
		}
		if json.Unmarshal(out.Result, &res) != nil {
			a.printRaw(out.Result)
			return map[string]any{"project": t.view(), "result": out.Result}, nil
		}
		if len(res.Columns) > 0 {
			a.printTable(res.Columns, res.Rows)
			a.ui.Note(i18n.T("sql.rows", len(res.Rows), res.Duration))
		} else {
			a.ui.Success(i18n.T("sql.affected", res.RowsAffected, res.Duration))
		}
		if res.Rows == nil {
			res.Rows = []map[string]any{}
		}
		return map[string]any{"project": t.view(), "columns": res.Columns, "rows": res.Rows,
			"rows_affected": res.RowsAffected, "duration": res.Duration}, nil
	})
	return cmd
}

// readQuery takes the SQL from the argument, --file, or stdin, only with an
// explicit "-": agent harnesses often leave stdin open without writing to
// it, and reading it by default would hang the command.
func (a *app) readQuery(args []string, file string) (string, error) {
	var q string
	switch {
	case file != "" && len(args) == 1:
		return "", cerr.New(cerr.Usage, i18n.T("sql.query_and_file"))
	case file != "":
		b, err := os.ReadFile(file)
		if err != nil {
			return "", cerr.New(cerr.Usage, i18n.T("sql.no_file", file))
		}
		q = string(b)
	case len(args) == 1 && args[0] != "-":
		q = args[0]
	case len(args) == 1 && args[0] == "-":
		if term.IsTerminal(int(os.Stdin.Fd())) {
			a.ui.Progress(i18n.T("sql.reading_stdin"))
		}
		b, err := io.ReadAll(io.LimitReader(os.Stdin, 4<<20))
		if err != nil {
			return "", err
		}
		q = string(b)
	}
	if strings.TrimSpace(q) == "" {
		return "", a.missing(i18n.T("sql.empty"), "[query]", "--file")
	}
	return q, nil
}

// scalar reads a --param: numbers, true/false and null go as such; the
// rest as text (to force text, use --params '["123"]').
func scalar(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	var n json.Number
	if d := json.NewDecoder(strings.NewReader(s)); d.Decode(&n) == nil && !d.More() && n.String() == s {
		return n
	}
	return s
}

// printTable prints rows as aligned columns; long cells are cut.
func (a *app) printTable(cols []string, rows []map[string]any) {
	const maxCell = 40
	cell := func(v any) string {
		var s string
		switch x := v.(type) {
		case nil:
			s = "NULL"
		case string:
			s = x
		default:
			b, _ := json.Marshal(x)
			s = string(b)
		}
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", "↵")
		if utf8.RuneCountInString(s) > maxCell {
			s = string([]rune(s)[:maxCell-1]) + "…"
		}
		return s
	}
	width := make([]int, len(cols))
	grid := make([][]string, len(rows))
	for i, c := range cols {
		width[i] = utf8.RuneCountInString(c)
	}
	for r, row := range rows {
		grid[r] = make([]string, len(cols))
		for i, c := range cols {
			grid[r][i] = cell(row[c])
			if n := utf8.RuneCountInString(grid[r][i]); n > width[i] {
				width[i] = n
			}
		}
	}
	line := func(vals []string, paint func(string) string) {
		var b strings.Builder
		for i, v := range vals {
			if i > 0 {
				b.WriteString("  ")
			}
			b.WriteString(paint(v))
			if i < len(vals)-1 {
				b.WriteString(strings.Repeat(" ", width[i]-utf8.RuneCountInString(v)))
			}
		}
		a.ui.Println(b.String())
	}
	line(cols, a.ui.Bold)
	for _, r := range grid {
		line(r, func(s string) string { return s })
	}
}

// --- api ---

var apiMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}

func (a *app) apiCmd() *cobra.Command {
	var method, data, contentType, output string
	var forms []string
	cmd := &cobra.Command{
		Use:   "api [get|post|put|patch|delete] <path>",
		Short: i18n.T("cmd.api.short"),
		Long:  i18n.T("cmd.api.long"),
		Args:  cobra.RangeArgs(1, 2),
		Example: examples(
			`rapibase api get "/api/v1/rest/products?limit=5"`,
			`rapibase api post /api/v1/rest/products --data '{"name":"Mate","price":10}'`,
			"rapibase api post /api/v1/storage/photos/upload -F file=@cat.png -F path=pets/cat.png",
			"rapibase api get /api/v1/storage/photos/object/pets/cat.png -o cat.png",
		),
	}
	cmd.Flags().StringVarP(&method, "method", "X", "", i18n.T("flag.api_method"))
	cmd.Flags().StringVarP(&data, "data", "d", "", i18n.T("flag.api_data"))
	cmd.Flags().StringArrayVarP(&forms, "form", "F", nil, i18n.T("flag.api_form"))
	cmd.Flags().StringVar(&contentType, "content-type", "", i18n.T("flag.api_content_type"))
	cmd.Flags().StringVarP(&output, "output", "o", "", i18n.T("flag.api_output"))
	cmd.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		// `api post <path>` (the spec's form) or `api <path> -X POST`
		if len(args) == 2 {
			if method != "" && !strings.EqualFold(method, args[0]) {
				return nil, cerr.New(cerr.Usage, i18n.T("api.two_methods", args[0], method))
			}
			method, args = args[0], args[1:]
		}
		rest, err := apiPath(args[0])
		if err != nil {
			return nil, err
		}
		if data != "" && len(forms) > 0 {
			return nil, cerr.New(cerr.Usage, i18n.T("api.data_and_form"))
		}
		m := strings.ToUpper(method)
		if m == "" {
			m = "GET"
			if data != "" || len(forms) > 0 {
				m = "POST"
			}
		}
		if !apiMethods[m] {
			return nil, cerr.New(cerr.Usage, i18n.T("api.bad_method", method))
		}
		var body api.Body
		switch {
		case len(forms) > 0:
			b, ct, err := formBody(forms)
			if err != nil {
				return nil, err
			}
			body = api.BytesBody(b, ct)
		case data != "":
			b, err := readData(data)
			if err != nil {
				return nil, err
			}
			ct := contentType
			if ct == "" {
				ct = "application/octet-stream"
				if json.Valid(b) {
					ct = "application/json"
				} else if strings.HasPrefix(data, "@") {
					if t := mime.TypeByExtension(filepath.Ext(data[1:])); t != "" {
						ct = t
					}
				}
			}
			body = api.BytesBody(b, ct)
		}
		t, err := a.resolve(m != "GET")
		if err != nil {
			return nil, err
		}
		res, err := a.api.Raw(a.ctx, m, "/api/projects/"+t.IDString()+"/api/"+rest, body, true)
		if err != nil {
			return nil, err
		}
		ct := res.Header.Get("Content-Type")
		isJSON := strings.Contains(ct, "json") && json.Valid(res.Body)
		// the control plane's own errors (project_not_running, forbidden…)
		// carry a code; the instance's go back as they came
		if res.Status >= 300 {
			if !isInstanceError(res) {
				return nil, api.ErrorFrom(res)
			}
			e := cerr.New(cerr.Failed, i18n.T("api.status", res.Status, http.StatusText(res.Status))).With("status", res.Status)
			if isJSON {
				e.With("body", json.RawMessage(res.Body))
				if !a.ui.JSON {
					a.printRaw(res.Body)
				}
			} else if len(res.Body) > 0 {
				e.With("body", string(res.Body))
				if !a.ui.JSON {
					fmt.Fprintln(a.ui.Err, string(res.Body))
				}
			}
			return nil, e
		}
		out := map[string]any{"project": t.view(), "status": res.Status, "content_type": ct}
		switch {
		case output != "":
			if err := os.WriteFile(output, res.Body, 0o644); err != nil {
				return nil, err
			}
			a.ui.Success(i18n.T("api.saved", output, len(res.Body)))
			out["output"] = output
			out["bytes"] = len(res.Body)
		case isJSON:
			a.printRaw(res.Body)
			out["body"] = json.RawMessage(res.Body)
		case utf8.Valid(res.Body):
			a.ui.Println(string(res.Body))
			out["body"] = string(res.Body)
		default:
			return nil, cerr.New(cerr.Usage, i18n.T("api.binary", len(res.Body))).With("hint", "-o <file>")
		}
		return out, nil
	})
	return cmd
}

// isInstanceError tells the control plane's errors (with a code from spec
// §4.3) from the instance's, which go back to the user as they are.
func isInstanceError(res *api.Response) bool {
	var probe struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(res.Body, &probe)
	switch probe.Code {
	case cerr.ProjectNotRunning, cerr.ProjectNotFound, cerr.Forbidden, cerr.SessionExpired,
		cerr.RateLimited, cerr.ServerError:
		return false
	}
	return res.Status != http.StatusUnauthorized
}

// apiPath normalizes "/api/v1/rest/x", "api/v1/rest/x" or "v1/rest/x" to
// "v1/rest/x" (what goes after /api/projects/{id}/api/).
func apiPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		if u, err := url.Parse(p); err == nil {
			p = u.RequestURI()
		}
	}
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimPrefix(p, "api/")
	pathOnly, _, _ := strings.Cut(p, "?")
	if pathOnly == "" || strings.Contains(pathOnly, "..") || strings.Contains(pathOnly, "//") || strings.Contains(pathOnly, `\`) {
		return "", cerr.New(cerr.Usage, i18n.T("api.bad_path", p)).With("hint", "rapibase api /api/v1/rest/<table>")
	}
	return p, nil
}

// readData reads -d: text, @file or - (stdin).
func readData(d string) ([]byte, error) {
	switch {
	case d == "-" || d == "@-":
		return io.ReadAll(io.LimitReader(os.Stdin, 8<<20))
	case strings.HasPrefix(d, "@"):
		b, err := os.ReadFile(d[1:])
		if err != nil {
			return nil, cerr.New(cerr.Usage, i18n.T("api.no_file", d[1:]))
		}
		return b, nil
	}
	return []byte(d), nil
}

// formBody builds a multipart body from -F name=value / name=@file.
func formBody(forms []string) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range forms {
		k, v, ok := strings.Cut(f, "=")
		if !ok || k == "" {
			return nil, "", cerr.New(cerr.Usage, i18n.T("api.bad_form", f))
		}
		if !strings.HasPrefix(v, "@") {
			_ = w.WriteField(k, v)
			continue
		}
		path := v[1:]
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, "", cerr.New(cerr.Usage, i18n.T("api.no_file", path))
		}
		ct := mime.TypeByExtension(filepath.Ext(path))
		if ct == "" {
			ct = http.DetectContentType(b)
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, k, filepath.Base(path)))
		h.Set("Content-Type", ct)
		pw, _ := w.CreatePart(h)
		_, _ = pw.Write(b)
	}
	_ = w.Close()
	return buf.Bytes(), w.FormDataContentType(), nil
}

// --- rls ---

var rlsModeNames = []string{"owner", "authenticated", "public", "read_only", "custom"}

func (a *app) rlsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rls",
		Short: i18n.T("cmd.rls.short"),
		Long:  i18n.T("cmd.rls.long"),
		Args:  cobra.NoArgs,
		Example: examples(
			"rapibase rls",
			"rapibase rls set notes owner --owner-column user_id",
			"rapibase rls set products read_only",
			"rapibase rls disable notes --yes",
		),
	}
	list := func(cmd *cobra.Command, args []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		path := "/api/projects/" + t.IDString() + "/rls"
		if len(args) == 1 {
			path += "?table=" + url.QueryEscape(args[0])
		}
		var out struct {
			Result json.RawMessage `json:"result"`
		}
		if err := a.api.Do(a.ctx, "GET", path, nil, &out); err != nil {
			return nil, err
		}
		if len(args) == 1 {
			a.printRaw(out.Result)
			return map[string]any{"project": t.view(), "table": args[0], "status": out.Result}, nil
		}
		var cfg struct {
			Tables []struct {
				Table       string `json:"table"`
				Mode        string `json:"mode"`
				OwnerColumn string `json:"owner_column"`
			} `json:"tables"`
		}
		_ = json.Unmarshal(out.Result, &cfg)
		if len(cfg.Tables) == 0 {
			a.ui.Println(i18n.T("rls.none"))
		}
		for _, c := range cfg.Tables {
			extra := ""
			if c.OwnerColumn != "" {
				extra = "(" + c.OwnerColumn + ")"
			}
			a.ui.Printf("  %-24s %s %s\n", c.Table, a.ui.Cyan(c.Mode), a.ui.Dim(extra))
		}
		if cfg.Tables == nil {
			return map[string]any{"project": t.view(), "tables": []any{}}, nil
		}
		return map[string]any{"project": t.view(), "tables": cfg.Tables}, nil
	}
	cmd.RunE = a.run(list)
	ls := &cobra.Command{
		Use: "list [table]", Short: i18n.T("cmd.rls.list.short"), Args: cobra.MaximumNArgs(1),
		Example: examples("rapibase rls list", "rapibase rls list notes"),
	}
	ls.RunE = a.run(list)

	var ownerCol string
	set := &cobra.Command{
		Use: "set <table> <mode>", Short: i18n.T("cmd.rls.set.short"), Long: i18n.T("cmd.rls.set.long"),
		Args:    cobra.ExactArgs(2),
		Example: examples("rapibase rls set notes owner --owner-column user_id", "rapibase rls set posts public"),
	}
	set.Flags().StringVar(&ownerCol, "owner-column", "", i18n.T("flag.owner_column"))
	set.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		table, mode := args[0], strings.ToLower(args[1])
		valid := false
		for _, m := range rlsModeNames {
			valid = valid || m == mode
		}
		if !valid {
			return nil, cerr.New(cerr.Usage, i18n.T("rls.bad_mode", mode, strings.Join(rlsModeNames, ", ")))
		}
		if mode == "owner" && ownerCol == "" {
			return nil, a.missing(i18n.T("rls.owner_needs_column"), "--owner-column")
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		body := map[string]any{"mode": mode}
		if ownerCol != "" {
			body["owner_column"] = ownerCol
		}
		var out struct {
			Result json.RawMessage `json:"result"`
		}
		if err := a.api.Do(a.ctx, "PUT", "/api/projects/"+t.IDString()+"/rls/"+url.PathEscape(table), api.JSONBody(body), &out); err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("rls.set", table, mode))
		if mode == "custom" {
			a.ui.Note(i18n.T("rls.custom_note"))
		}
		return map[string]any{"project": t.view(), "table": table, "mode": mode, "status": out.Result}, nil
	})

	disable := &cobra.Command{
		Use: "disable <table>", Short: i18n.T("cmd.rls.disable.short"), Args: cobra.ExactArgs(1),
		Example: examples("rapibase rls disable notes --yes"),
	}
	disable.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		table := args[0]
		// removing protection opens every row to any key: always confirmed
		if a.ui.Interactive && !a.flags.yes {
			if ok, _ := a.ui.Confirm(i18n.T("rls.confirm_disable", table), false); !ok {
				return nil, cerr.New(cerr.Usage, i18n.T("deploy.cancelled"))
			}
		} else if !a.flags.yes {
			return nil, a.missing(i18n.T("rls.disable_needs_yes", table), "--yes")
		}
		t, err := a.resolve(true)
		if err != nil {
			return nil, err
		}
		if err := a.api.Do(a.ctx, "DELETE", "/api/projects/"+t.IDString()+"/rls/"+url.PathEscape(table), nil, nil); err != nil {
			return nil, err
		}
		a.ui.Success(i18n.T("rls.disabled", table))
		return map[string]any{"project": t.view(), "table": table, "disabled": true}, nil
	})
	cmd.AddCommand(ls, set, disable)
	return cmd
}
