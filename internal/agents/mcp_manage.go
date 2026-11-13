package agents

// Finding, replacing and removing Rapibase MCP servers that are already in
// an agent's config.
//
// Before the CLI there were two other ways to connect a project, and both
// are still around in people's configs:
//   - the instance's own MCP, `rapibase-<slug>` → https://<sub>.<host>/mcp
//     with an `Authorization: Bearer …` header saved in plain text
//     (Claude Code keeps it in ~/.claude.json, local scope);
//   - Connect's account MCP, `rapibase-<slug>` → <host>/mcp?project=<slug>.
//
// Left next to the CLI's `rapibase`, the agent gets two sets of tools for
// the same project. So `mcp add` finds them first, and replaces them after
// a yes (spec §7, step 9).

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// MCPEntry is a Rapibase MCP server found in an agent's config.
type MCPEntry struct {
	Agent Agent  `json:"agent"`
	Name  string `json:"name"`
	URL   string `json:"url,omitempty"`
	Scope Scope  `json:"scope"`
	File  string `json:"file"`
	// Token: the entry keeps an Authorization header (a key in plain text)
	Token bool `json:"token,omitempty"`

	path string // the config file
	key  string // "mcpServers" or "mcp"
	dir  string // ~/.claude.json, local scope: the folder it belongs to
	bin  string // codex / hermes: removed with their own command
}

// place is one config where an agent keeps MCP servers.
type place struct {
	agent   Agent
	scope   Scope
	path    string
	display string
	key     string
	// claudeDir: the project folder inside ~/.claude.json (local scope);
	// claudeJSON with an empty claudeDir is its user scope
	claudeJSON bool
	claudeDir  string
}

func claudeJSONPath() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return filepath.Join(d, ".claude.json")
	}
	return filepath.Join(home(), ".claude.json")
}

// places are where an agent keeps MCP servers for this folder: the repo
// (project scope, and Claude Code's local scope) or the home folder (user).
func places(a Agent, root string, dirs []string, user bool) []place {
	cj := claudeJSONPath()
	switch a {
	case Claude:
		if user {
			return []place{{agent: a, scope: ScopeUser, path: cj, display: tilde(cj), key: "mcpServers", claudeJSON: true}}
		}
		ps := []place{{agent: a, scope: ScopeProject, path: filepath.Join(root, ".mcp.json"), display: ".mcp.json", key: "mcpServers"}}
		for _, d := range dirs {
			ps = append(ps, place{agent: a, scope: ScopeLocal, path: cj, display: tilde(cj), key: "mcpServers", claudeJSON: true, claudeDir: d})
		}
		return ps
	case Cursor:
		if user {
			p := filepath.Join(home(), ".cursor", "mcp.json")
			return []place{{agent: a, scope: ScopeUser, path: p, display: tilde(p), key: "mcpServers"}}
		}
		return []place{{agent: a, scope: ScopeProject, path: filepath.Join(root, ".cursor", "mcp.json"), display: ".cursor/mcp.json", key: "mcpServers"}}
	case OpenCode:
		if user {
			p := filepath.Join(home(), ".config", "opencode", "opencode.json")
			return []place{{agent: a, scope: ScopeUser, path: p, display: tilde(p), key: "mcp"}}
		}
		return []place{{agent: a, scope: ScopeProject, path: filepath.Join(root, "opencode.json"), display: "opencode.json", key: "mcp"}}
	}
	return nil
}

// sameDir compares folders the way the OS does (case-insensitive on Windows).
func sameDir(a, b string) bool {
	a = strings.TrimRight(filepath.ToSlash(filepath.Clean(a)), "/")
	b = strings.TrimRight(filepath.ToSlash(filepath.Clean(b)), "/")
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// isRapibase: the name is ours, or the URL is the account MCP or an
// instance's MCP (a subdomain of the account host).
func isRapibase(name, rawURL, accountHost string) bool {
	if name == ServerName || strings.HasPrefix(name, ServerName+"-") {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil || accountHost == "" || !strings.HasPrefix(u.Path, "/mcp") {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return h == accountHost || strings.HasSuffix(h, "."+accountHost)
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// entries reads the Rapibase servers of a place.
func (p place) entries(accountHost string) []MCPEntry {
	b, err := os.ReadFile(p.path)
	if err != nil {
		return nil
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil {
		return nil
	}
	// the servers by folder ("" outside ~/.claude.json's projects). Claude
	// Code keys a project by the folder as typed, so the same one can show up
	// twice on Windows ("Desktop" and "desktop"): every match counts.
	raws := map[string]json.RawMessage{"": top[p.key]}
	if p.claudeJSON && p.claudeDir != "" {
		raws = map[string]json.RawMessage{}
		var projects map[string]map[string]json.RawMessage
		_ = json.Unmarshal(top["projects"], &projects)
		for k, v := range projects {
			if sameDir(k, p.claudeDir) {
				raws[k] = v["mcpServers"]
			}
		}
	}
	var out []MCPEntry
	for dir, raw := range raws {
		var servers map[string]struct {
			URL     string         `json:"url"`
			Headers map[string]any `json:"headers"`
		}
		if json.Unmarshal(raw, &servers) != nil {
			continue
		}
		for name, s := range servers {
			if !isRapibase(name, s.URL, accountHost) {
				continue
			}
			_, auth := s.Headers["Authorization"]
			if !auth {
				_, auth = s.Headers["authorization"]
			}
			out = append(out, MCPEntry{
				Agent: p.agent, Name: name, URL: s.URL, Scope: p.scope, File: p.display, Token: auth,
				path: p.path, key: p.key, dir: dir,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].dir < out[j].dir
	})
	return out
}

func (o MCPOptions) accountHost() string { return hostOf(o.AccountURL) }

func (o MCPOptions) dirs() []string {
	ds := []string{o.Root}
	if o.Cwd != "" && !sameDir(o.Cwd, o.Root) {
		ds = append(ds, o.Cwd)
	}
	return ds
}

// target is where AddMCP writes for these options, and the URL it writes.
func (o MCPOptions) target() (Scope, string) {
	switch o.Agent {
	case Claude:
		return o.Scope, o.url(o.Scope != ScopeUser)
	case Cursor, OpenCode:
		if o.Scope == ScopeUser {
			return ScopeUser, o.url(false)
		}
		return ScopeProject, o.url(true)
	}
	return ScopeUser, o.AccountURL
}

func (o MCPOptions) found() []MCPEntry {
	user := o.Scope == ScopeUser
	var out []MCPEntry
	for _, p := range places(o.Agent, o.Root, o.dirs(), user) {
		out = append(out, p.entries(o.accountHost())...)
	}
	return out
}

// sameMCP is an entry that already does what AddMCP would write (same
// URL), under any name and in any of this folder's places: nothing to add.
func sameMCP(o MCPOptions) *MCPEntry {
	_, want := o.target()
	for _, e := range o.found() {
		if e.URL == want {
			e := e
			return &e
		}
	}
	return nil
}

// LegacyMCP lists the servers of the same project that AddMCP would leave
// next to its own: the instance's MCP (by its subdomain), Connect's
// `rapibase-<slug>`, and a `rapibase` in another scope that would win over
// the new one (Claude Code: local beats project). The `rapibase` AddMCP
// rewrites in place is not one of them.
func LegacyMCP(o MCPOptions) []MCPEntry {
	scope, want := o.target()
	host := o.accountHost()
	var out []MCPEntry
	for _, e := range o.found() {
		if e.URL == want || (e.Name == ServerName && e.Scope == scope) {
			continue
		}
		mine := e.Name == ServerName ||
			(o.Slug != "" && e.Name == ServerName+"-"+o.Slug) ||
			(o.Sub != "" && host != "" && hostOf(e.URL) == strings.ToLower(o.Sub)+"."+host)
		if mine {
			out = append(out, e)
		}
	}
	return out
}

// FindMCP lists every Rapibase server an agent has for this folder (repo
// places), or in the person's home with user. Codex and Hermes keep theirs
// per user only, so they show up with user.
func FindMCP(a Agent, root, cwd, accountURL string, user bool) []MCPEntry {
	o := MCPOptions{Agent: a, Root: root, Cwd: cwd, AccountURL: accountURL}
	var out []MCPEntry
	for _, p := range places(a, root, o.dirs(), user) {
		out = append(out, p.entries(o.accountHost())...)
	}
	if !user {
		return out
	}
	switch a {
	case Codex:
		if _, err := exec.LookPath("codex"); err == nil {
			if _, err := run(20*time.Second, "codex", "mcp", "get", ServerName); err == nil {
				out = append(out, MCPEntry{Agent: a, Name: ServerName, Scope: ScopeUser, File: "~/.codex/config.toml", bin: "codex"})
			}
		}
	case Hermes:
		if _, err := exec.LookPath("hermes"); err == nil {
			if list, err := run(30*time.Second, "hermes", "mcp", "list"); err == nil && strings.Contains(list, ServerName) {
				out = append(out, MCPEntry{Agent: a, Name: ServerName, Scope: ScopeUser, File: "hermes", bin: "hermes"})
			}
		}
	}
	return out
}

// RemoveMCP deletes the entries. What it can't do itself (Claude Code's
// ~/.claude.json without the `claude` binary, a command that failed, a
// config with comments) comes back as what the person has to run or edit.
func RemoveMCP(entries []MCPEntry) (removed []MCPEntry, manual []string, err error) {
	for _, e := range entries {
		switch {
		case e.bin != "":
			if _, rerr := run(30*time.Second, e.bin, "mcp", "remove", e.Name); rerr != nil {
				manual = append(manual, e.bin+" mcp remove "+e.Name)
				continue
			}
		case e.Agent == Claude && e.path == claudeJSONPath():
			if m := removeClaude(e); m != "" {
				manual = append(manual, m)
				continue
			}
		default:
			if _, rerr := removeServer(e.path, e.key, e.Name); rerr != nil {
				if errors.Is(rerr, errNotJSON) {
					manual = append(manual, e.File+" → "+e.key+"."+e.Name)
					continue
				}
				return removed, manual, rerr
			}
		}
		removed = append(removed, e)
	}
	return removed, manual, nil
}

// Where is the config file and, when it says something, the scope.
func (e MCPEntry) Where() string {
	if e.Scope == ScopeLocal {
		return e.File + " (local)"
	}
	return e.File
}

// removeClaude removes a server from ~/.claude.json with Claude Code's own
// command (it may be running and rewrites that file), and checks it's gone.
func removeClaude(e MCPEntry) string {
	cmd := "claude mcp remove --scope " + string(e.Scope) + " " + e.Name
	if e.Scope == ScopeLocal {
		cmd = "cd " + quoteArgs([]string{filepath.FromSlash(e.dir)})[0] + " && " + cmd
	}
	if _, err := exec.LookPath("claude"); err != nil {
		return cmd
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "claude", "mcp", "remove", "--scope", string(e.Scope), e.Name)
	if e.Scope == ScopeLocal {
		c.Dir = filepath.FromSlash(e.dir)
	}
	_, _ = c.CombinedOutput()
	p := place{agent: Claude, scope: e.Scope, path: e.path, claudeJSON: true, claudeDir: e.dir}
	for _, left := range p.entries("") {
		if left.Name == e.Name && left.dir == e.dir {
			return cmd
		}
	}
	return ""
}

// removeServer deletes key.name from a JSON config, keeping everything else
// and the order of the keys.
func removeServer(path, key, name string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	top := &ordered{}
	if err := top.UnmarshalJSON(b); err != nil {
		return false, errNotJSON
	}
	raw, ok := top.get(key)
	if !ok {
		return false, nil
	}
	servers := &ordered{}
	if err := servers.UnmarshalJSON(raw); err != nil {
		return false, errNotJSON
	}
	if !servers.del(name) {
		return false, nil
	}
	sb, _ := servers.MarshalJSON()
	top.set(key, sb)
	return true, writeJSON(path, top)
}
