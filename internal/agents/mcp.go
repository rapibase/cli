package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ServerName is the MCP server name the CLI registers.
const ServerName = "rapibase"

// MCPOptions says what to register.
type MCPOptions struct {
	Agent       Agent
	Scope       Scope
	Root        string // repo (or folder) root
	AccountURL  string // <control plane>/mcp
	Slug        string // project to pin to (repo-level configs)
	Sub         string // its subdomain: finds the instance's own MCP
	Cwd         string // where the CLI runs (Claude Code's local scope may use it)
	Interactive bool   // may run interactive logins (codex, hermes)
}

// MCPResult is what `mcp add` did. It never contains a token: the agent
// does its own OAuth the first time (spec §7, step 9).
type MCPResult struct {
	Agent         Agent  `json:"agent"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Pinned        bool   `json:"pinned"`
	Scope         Scope  `json:"scope"`
	Registered    bool   `json:"registered"`
	Unchanged     bool   `json:"unchanged,omitempty"`
	File          string `json:"file,omitempty"`
	Command       string `json:"command,omitempty"`
	Manual        string `json:"manual,omitempty"` // what the person still has to run or paste
	Login         string `json:"login,omitempty"`  // command that completes the OAuth
	RestartNeeded bool   `json:"restart_needed"`
	// Replaced: older Rapibase servers of the same project that were removed
	Replaced []string `json:"replaced,omitempty"`
	// Legacy: the older servers that were left (no --yes, or the person said
	// no); the new one is then not added, so the tools don't show up twice
	Legacy []MCPEntry `json:"legacy,omitempty"`
}

// url pins repo-level configs to the project (?project=<slug>, spec
// §4.5.11); user-level configs stay unpinned and the skill tells the agent to
// pass the project of the folder.
func (o MCPOptions) url(pinned bool) string {
	if pinned && o.Slug != "" {
		return o.AccountURL + "?project=" + o.Slug
	}
	return o.AccountURL
}

// AddMCP registers the Rapibase MCP in the agent's config. A server that
// already points to the same URL, under any name, is left as it is and
// reported with its own name.
func AddMCP(o MCPOptions) (*MCPResult, error) {
	if e := sameMCP(o); e != nil {
		return &MCPResult{
			Agent: o.Agent, Name: e.Name, URL: e.URL, Pinned: strings.Contains(e.URL, "?project="), Scope: e.Scope,
			File: e.File, Registered: true, Unchanged: true,
		}, nil
	}
	switch o.Agent {
	case Claude:
		return addClaude(o)
	case Cursor:
		return addJSON(o, filepath.Join(".cursor", "mcp.json"), filepath.Join(home(), ".cursor", "mcp.json"), "mcpServers",
			func(url string) map[string]any { return map[string]any{"url": url} }, nil)
	case OpenCode:
		return addJSON(o, "opencode.json", filepath.Join(home(), ".config", "opencode", "opencode.json"), "mcp",
			func(url string) map[string]any { return map[string]any{"type": "remote", "url": url} },
			map[string]any{"$schema": "https://opencode.ai/config.json"})
	case Codex:
		return addCodex(o)
	case Hermes:
		return addHermes(o)
	}
	url := o.AccountURL
	cfg, _ := json.MarshalIndent(map[string]any{"mcpServers": map[string]any{ServerName: map[string]any{"type": "http", "url": url}}}, "", "  ")
	return &MCPResult{Agent: o.Agent, Name: ServerName, URL: url, Scope: ScopeUser, Manual: string(cfg)}, nil
}

// --- Claude Code ---

func addClaude(o MCPOptions) (*MCPResult, error) {
	if o.Scope == ScopeProject {
		// --scope project writes .mcp.json: writing it directly needs no
		// binary and is idempotent
		return addJSON(o, ".mcp.json", "", "mcpServers",
			func(url string) map[string]any { return map[string]any{"type": "http", "url": url} }, nil)
	}
	pinned := o.Scope == ScopeLocal
	url := o.url(pinned)
	res := &MCPResult{Agent: Claude, Name: ServerName, URL: url, Pinned: pinned, Scope: o.Scope, RestartNeeded: true}
	args := []string{"mcp", "add", "--transport", "http", "--scope", string(o.Scope), ServerName, url}
	res.Command = "claude " + strings.Join(quoteArgs(args), " ")
	if _, err := exec.LookPath("claude"); err != nil {
		res.Manual = res.Command
		return res, nil
	}
	_, _ = run(30*time.Second, "claude", "mcp", "remove", "--scope", string(o.Scope), ServerName)
	if out, err := run(60*time.Second, "claude", args...); err != nil {
		res.Manual = res.Command
		return res, fmt.Errorf("claude mcp add: %v: %s", err, strings.TrimSpace(out))
	}
	res.Registered = true
	return res, nil
}

// --- Codex ---

func addCodex(o MCPOptions) (*MCPResult, error) {
	url := o.AccountURL // config per user: never pinned
	res := &MCPResult{Agent: Codex, Name: ServerName, URL: url, Scope: ScopeUser, RestartNeeded: true}
	add := []string{"mcp", "add", ServerName, "--url", url}
	res.Command = "codex " + strings.Join(quoteArgs(add), " ")
	login := "codex mcp login " + ServerName
	if _, err := exec.LookPath("codex"); err != nil {
		res.Manual = res.Command + "\n" + login
		return res, nil
	}
	if out, err := run(20*time.Second, "codex", "mcp", "get", ServerName); err == nil && strings.Contains(out, url) {
		res.Registered, res.Unchanged, res.RestartNeeded = true, true, false
		return res, nil
	}
	_, _ = run(20*time.Second, "codex", "mcp", "remove", ServerName)
	if out, err := run(60*time.Second, "codex", add...); err != nil {
		res.Manual = res.Command + "\n" + login
		return res, fmt.Errorf("codex mcp add: %v: %s", err, strings.TrimSpace(out))
	}
	res.Registered = true
	if o.Interactive {
		if err := runInteractive("codex", "mcp", "login", ServerName); err != nil {
			res.Login = login
		}
	} else {
		res.Login = login
	}
	return res, nil
}

// --- Hermes ---

func addHermes(o MCPOptions) (*MCPResult, error) {
	url := o.AccountURL
	res := &MCPResult{Agent: Hermes, Name: ServerName, URL: url, Scope: ScopeUser, RestartNeeded: true}
	args := []string{"mcp", "add", ServerName, "--url", url, "--auth", "oauth"}
	res.Command = "hermes " + strings.Join(quoteArgs(args), " ")
	if _, err := exec.LookPath("hermes"); err != nil {
		res.Manual = res.Command
		return res, nil
	}
	if out, err := run(30*time.Second, "hermes", "mcp", "list"); err == nil && strings.Contains(out, url) {
		res.Registered, res.Unchanged, res.RestartNeeded = true, true, false
		return res, nil
	}
	// `hermes mcp add` connects and runs the OAuth right away: it needs the
	// person at the keyboard
	if !o.Interactive {
		res.Manual = res.Command
		return res, nil
	}
	if err := runInteractive("hermes", args...); err != nil {
		res.Manual = res.Command
		return res, nil
	}
	res.Registered = true
	return res, nil
}

// --- JSON configs (.mcp.json, .cursor/mcp.json, opencode.json) ---

// errNotJSON: the file has comments (JSONC) or is broken; the CLI does not
// rewrite it and prints what to paste instead.
var errNotJSON = errors.New("not plain JSON")

func addJSON(o MCPOptions, repoRel, userPath, key string, entry func(string) map[string]any, newFileTop map[string]any) (*MCPResult, error) {
	pinned := o.Scope != ScopeUser || userPath == ""
	path := filepath.Join(o.Root, repoRel)
	display := filepath.ToSlash(repoRel)
	scope := o.Scope
	if !pinned {
		path, display = userPath, tilde(userPath)
	}
	if scope == ScopeUser && pinned {
		scope = ScopeProject
	}
	url := o.url(pinned)
	res := &MCPResult{Agent: o.Agent, Name: ServerName, URL: url, Pinned: pinned, Scope: scope, File: display, RestartNeeded: true}
	changed, err := mergeServer(path, key, ServerName, entry(url), newFileTop)
	if errors.Is(err, errNotJSON) {
		snippet, _ := json.MarshalIndent(map[string]any{key: map[string]any{ServerName: entry(url)}}, "", "  ")
		res.Manual = string(snippet)
		return res, nil
	}
	if err != nil {
		return nil, err
	}
	res.Registered = true
	if !changed {
		res.Unchanged, res.RestartNeeded = true, false
	}
	if scope == ScopeLocal {
		_, _ = GitExclude(o.Root, []string{"/" + display})
	}
	return res, nil
}

// mergeServer adds or updates key.name in a JSON file, keeping every other
// server and the order of the keys.
func mergeServer(path, key, name string, entry map[string]any, newFileTop map[string]any) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	top := &ordered{}
	if len(bytes.TrimSpace(b)) > 0 {
		if err := top.UnmarshalJSON(b); err != nil {
			return false, errNotJSON
		}
	} else {
		for k, v := range newFileTop {
			raw, _ := json.Marshal(v)
			top.set(k, raw)
		}
	}
	servers := &ordered{}
	if raw, ok := top.get(key); ok {
		if err := servers.UnmarshalJSON(raw); err != nil {
			return false, errNotJSON
		}
	}
	want, _ := json.Marshal(entry)
	wantURL, _ := entry["url"].(string)
	if raw, ok := servers.get(name); ok {
		var cur map[string]any
		_ = json.Unmarshal(raw, &cur)
		if u, _ := cur["url"].(string); u == wantURL {
			return false, nil
		}
	}
	servers.set(name, want)
	sb, _ := servers.MarshalJSON()
	top.set(key, sb)
	return true, writeJSON(path, top)
}

// writeJSON writes an ordered object indented, the way the agents do.
func writeJSON(path string, top *ordered) error {
	out, _ := top.MarshalJSON()
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, out, "", "  "); err != nil {
		return err
	}
	pretty.WriteByte('\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, pretty.Bytes(), 0o644)
}

// ordered is a JSON object that keeps its key order.
type ordered struct {
	keys []string
	vals map[string]json.RawMessage
}

func (o *ordered) get(k string) (json.RawMessage, bool) {
	v, ok := o.vals[k]
	return v, ok
}

func (o *ordered) set(k string, v json.RawMessage) {
	if o.vals == nil {
		o.vals = map[string]json.RawMessage{}
	}
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

func (o *ordered) del(k string) bool {
	if _, ok := o.vals[k]; !ok {
		return false
	}
	delete(o.vals, k)
	for i, x := range o.keys {
		if x == k {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
	return true
}

func (o *ordered) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	t, err := dec.Token()
	if err != nil || t != json.Delim('{') {
		return errNotJSON
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return err
		}
		k, _ := kt.(string)
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return err
		}
		o.set(k, v)
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	if dec.More() {
		return errNotJSON
	}
	return nil
}

func (o *ordered) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		b.Write(kb)
		b.WriteByte(':')
		b.Write(o.vals[k])
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// --- running other CLIs ---

func run(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stderr, os.Stderr
	return cmd.Run()
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " ?&") {
			out[i] = `"` + a + `"`
		} else {
			out[i] = a
		}
	}
	return out
}
