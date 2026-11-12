// Package agents knows where each coding agent reads skills and MCP
// servers (spec §7.1), detects which agent is in use, and writes both.
//
// Verified on real installs on 2026-11-12: Claude Code exports CLAUDECODE=1
// to its shell; Codex exports CODEX_SANDBOX or CODEX_MANAGED_BY_NPM/BUN/
// PNPM; OpenCode exports OPENCODE; Hermes exports HERMES_SESSION_ID. Hermes
// reads skills only from its home (<HERMES_HOME>/skills, %LOCALAPPDATA%\hermes
// on Windows) and skills.external_dirs, never from the repo, and adds MCP
// servers with `hermes mcp add <name> --url <url> --auth oauth`. Codex adds
// them with `codex mcp add <name> --url <url>` and logs in with `codex mcp
// login <name>`. Cursor was not available to verify: CURSOR_AGENT is the
// documented variable and .cursor/ the fallback.
package agents

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Agent is a supported coding agent.
type Agent string

const (
	Claude   Agent = "claude"
	Cursor   Agent = "cursor"
	OpenCode Agent = "opencode"
	Codex    Agent = "codex"
	Hermes   Agent = "hermes"
	Other    Agent = "other"
	None     Agent = "none"
)

// All lists the agents in prompt order.
var All = []Agent{Claude, Cursor, OpenCode, Codex, Hermes, Other}

// Parse validates an --agent value.
func Parse(s string) (Agent, bool) {
	a := Agent(strings.ToLower(strings.TrimSpace(s)))
	switch a {
	case "claude-code", "claudecode":
		return Claude, true
	case Claude, Cursor, OpenCode, Codex, Hermes, Other, None:
		return a, true
	}
	return "", false
}

// Label is the human name.
func (a Agent) Label() string {
	switch a {
	case Claude:
		return "Claude Code"
	case Cursor:
		return "Cursor"
	case OpenCode:
		return "OpenCode"
	case Codex:
		return "Codex"
	case Hermes:
		return "Hermes"
	case Other:
		return "AGENTS.md"
	}
	return string(a)
}

// Running is the agent whose shell is running the CLI ("" if none).
func Running() Agent {
	switch {
	case os.Getenv("CLAUDECODE") != "":
		return Claude
	case os.Getenv("CODEX_SANDBOX") != "" || os.Getenv("CODEX_MANAGED_BY_NPM") != "" ||
		os.Getenv("CODEX_MANAGED_BY_BUN") != "" || os.Getenv("CODEX_MANAGED_BY_PNPM") != "":
		return Codex
	case os.Getenv("OPENCODE") != "":
		return OpenCode
	case os.Getenv("HERMES_SESSION_ID") != "":
		return Hermes
	case os.Getenv("CURSOR_AGENT") != "":
		return Cursor
	}
	return ""
}

// FromFiles detects agents by the files of the repo (spec §7, step 7).
func FromFiles(root string) []Agent {
	exists := func(p string) bool { _, err := os.Stat(filepath.Join(root, p)); return err == nil }
	var out []Agent
	if exists(".claude") || exists("CLAUDE.md") {
		out = append(out, Claude)
	}
	if exists(".cursor") {
		out = append(out, Cursor)
	}
	if exists("opencode.json") || exists("opencode.jsonc") || exists(".opencode") {
		out = append(out, OpenCode)
	}
	if exists(".codex") {
		out = append(out, Codex)
	}
	if exists(".hermes") {
		out = append(out, Hermes)
	}
	if len(out) == 0 && (exists("AGENTS.md") || exists(".agents")) {
		out = append(out, Other)
	}
	return out
}

// FromPath lists agents whose binary is installed (only to suggest).
func FromPath() []Agent {
	var out []Agent
	for _, a := range []struct {
		bin   string
		agent Agent
	}{{"claude", Claude}, {"codex", Codex}, {"opencode", OpenCode}, {"hermes", Hermes}} {
		if _, err := exec.LookPath(a.bin); err == nil {
			out = append(out, a.agent)
		}
	}
	return out
}

// Scope is where skills and the MCP go: user (all repos), project (the
// repo, shared with the team) or local (the repo, only this machine).
type Scope string

const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
	ScopeLocal   Scope = "local"
)

// ParseScope validates --scope.
func ParseScope(s string) (Scope, bool) {
	switch sc := Scope(strings.ToLower(strings.TrimSpace(s))); sc {
	case ScopeUser, ScopeProject, ScopeLocal:
		return sc, true
	}
	return "", false
}

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

// HermesHome is HERMES_HOME or the platform default.
func HermesHome() string {
	if h := os.Getenv("HERMES_HOME"); h != "" {
		return h
	}
	if runtime.GOOS == "windows" {
		if la := os.Getenv("LOCALAPPDATA"); la != "" {
			return filepath.Join(la, "hermes")
		}
		return filepath.Join(home(), "AppData", "Local", "hermes")
	}
	return filepath.Join(home(), ".hermes")
}

// SkillTarget is where an agent reads skills.
type SkillTarget struct {
	Dir         string // the folder that holds <skill>/SKILL.md
	AgentsLine  bool   // also add a line to AGENTS.md (agents without native skills)
	Shared      bool   // inside the repo and meant to be committed
	ScopeUsed   Scope  // the scope actually used (Hermes is always user)
	DisplayPath string // for messages: relative inside the repo, ~/… outside
}

// SkillDir resolves the skills folder for an agent and scope.
func SkillDir(a Agent, sc Scope, root string) SkillTarget {
	repo := func(rel string, line bool) SkillTarget {
		return SkillTarget{Dir: filepath.Join(root, rel), AgentsLine: line, Shared: sc == ScopeProject, ScopeUsed: sc, DisplayPath: filepath.ToSlash(rel)}
	}
	user := func(abs string) SkillTarget {
		return SkillTarget{Dir: abs, ScopeUsed: ScopeUser, DisplayPath: tilde(abs)}
	}
	switch a {
	case Claude:
		if sc == ScopeUser {
			return user(filepath.Join(home(), ".claude", "skills"))
		}
		return repo(filepath.Join(".claude", "skills"), false)
	case OpenCode:
		if sc == ScopeUser {
			return user(filepath.Join(home(), ".config", "opencode", "skill"))
		}
		return repo(filepath.Join(".opencode", "skill"), false)
	case Codex:
		if sc == ScopeUser {
			return user(filepath.Join(home(), ".agents", "skills"))
		}
		return repo(filepath.Join(".agents", "skills"), false)
	case Hermes:
		// Hermes does not read skills from the repo
		return user(filepath.Join(HermesHome(), "skills"))
	case Cursor:
		// no user-level skills for Cursor: always the repo
		t := repo(filepath.Join(".agents", "skills"), true)
		if sc == ScopeUser {
			t.ScopeUsed, t.Shared = ScopeProject, true
		}
		return t
	}
	t := repo(filepath.Join(".agents", "skills"), true)
	if sc == ScopeUser {
		t.ScopeUsed, t.Shared = ScopeProject, true
	}
	return t
}

func tilde(p string) string {
	h := home()
	if h != "" && strings.HasPrefix(p, h) {
		return "~" + filepath.ToSlash(strings.TrimPrefix(p, h))
	}
	return filepath.ToSlash(p)
}
