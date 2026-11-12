package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rapibase/cli/internal/skills"
)

func TestMergeServerKeepsOthersAndOrder(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".mcp.json")
	orig := `{
  "mcpServers": {
    "github": {"type": "http", "url": "https://gh.example/mcp"},
    "zeta": {"command": "z"}
  },
  "other": true
}`
	if err := os.WriteFile(p, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{"type": "http", "url": "https://rapibase.com/mcp?project=shop"}
	changed, err := mergeServer(p, "mcpServers", ServerName, entry, nil)
	if err != nil || !changed {
		t.Fatalf("first merge: %v %v", changed, err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if !(strings.Index(s, `"github"`) < strings.Index(s, `"zeta"`) && strings.Index(s, `"zeta"`) < strings.Index(s, `"rapibase"`)) {
		t.Errorf("order of the servers changed:\n%s", s)
	}
	if !strings.Contains(s, `"other": true`) {
		t.Errorf("other top-level keys must stay:\n%s", s)
	}
	var v struct {
		Servers map[string]map[string]any `json:"mcpServers"`
		Other   bool                      `json:"other"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("invalid JSON written: %v", err)
	}
	if v.Servers["rapibase"]["url"] != "https://rapibase.com/mcp?project=shop" || len(v.Servers) != 3 || !v.Other {
		t.Errorf("written: %+v", v)
	}
	// idempotent
	if changed, err := mergeServer(p, "mcpServers", ServerName, entry, nil); err != nil || changed {
		t.Errorf("second merge must not change the file: %v %v", changed, err)
	}
}

func TestMergeServerConnectEntryCounts(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "opencode.json")
	_ = os.WriteFile(p, []byte(`{"mcp":{"rapibase-shop":{"type":"remote","url":"https://rapibase.com/mcp?project=shop"}}}`), 0o644)
	entry := map[string]any{"type": "remote", "url": "https://rapibase.com/mcp?project=shop"}
	if changed, err := mergeServer(p, "mcp", ServerName, entry, nil); err != nil || changed {
		t.Errorf("an entry from Connect with the same URL counts as registered: %v %v", changed, err)
	}
}

func TestMergeServerRefusesJSONC(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "opencode.json")
	jsonc := "{\n  // my config\n  \"mcp\": {}\n}\n"
	_ = os.WriteFile(p, []byte(jsonc), 0o644)
	if _, err := mergeServer(p, "mcp", ServerName, map[string]any{"url": "u"}, nil); err != errNotJSON {
		t.Fatalf("JSONC must not be rewritten: %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != jsonc {
		t.Error("the file was modified")
	}
}

func TestMergeServerNewFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "opencode.json")
	top := map[string]any{"$schema": "https://opencode.ai/config.json"}
	if _, err := mergeServer(p, "mcp", ServerName, map[string]any{"type": "remote", "url": "u"}, top); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"$schema"`) || !strings.Contains(string(b), `"rapibase"`) {
		t.Errorf("new file: %s", b)
	}
}

func TestParseAndScope(t *testing.T) {
	if a, ok := Parse(" Claude-Code "); !ok || a != Claude {
		t.Errorf("claude-code → %v %v", a, ok)
	}
	if _, ok := Parse("vscode"); ok {
		t.Error("unknown agent accepted")
	}
	if _, ok := ParseScope("global"); ok {
		t.Error("unknown scope accepted")
	}
}

func TestSkillDir(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		a     Agent
		sc    Scope
		rel   string
		line  bool
		scope Scope
	}{
		{Claude, ScopeProject, ".claude/skills", false, ScopeProject},
		{OpenCode, ScopeLocal, ".opencode/skill", false, ScopeLocal},
		{Codex, ScopeProject, ".agents/skills", false, ScopeProject},
		{Cursor, ScopeUser, ".agents/skills", true, ScopeProject}, // Cursor has no user skills
		{Other, ScopeProject, ".agents/skills", true, ScopeProject},
	}
	for _, c := range cases {
		got := SkillDir(c.a, c.sc, root)
		if got.Dir != filepath.Join(root, filepath.FromSlash(c.rel)) || got.AgentsLine != c.line || got.ScopeUsed != c.scope {
			t.Errorf("%s/%s → %+v", c.a, c.sc, got)
		}
	}
	if h := SkillDir(Hermes, ScopeProject, root); h.ScopeUsed != ScopeUser || !strings.HasPrefix(h.Dir, HermesHome()) {
		t.Errorf("hermes reads skills only from its home: %+v", h)
	}
}

func TestInstallSkillsAndOldSkill(t *testing.T) {
	if len(skills.Names()) == 0 {
		t.Skip("no embedded skills (run tools/syncskills)")
	}
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755)
	// the old single skill and its AGENTS.md line
	old := filepath.Join(root, ".claude", "skills", "rapibase")
	_ = os.MkdirAll(old, 0o755)
	_ = os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("---\nname: rapibase\n---\n"), 0o644)

	res, err := InstallSkills(Claude, ScopeLocal, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Installed) != len(skills.Names()) || !res.Excluded {
		t.Errorf("install: %+v", res)
	}
	ex, _ := os.ReadFile(filepath.Join(root, ".git", "info", "exclude"))
	if !strings.Contains(string(ex), "/.claude/skills/rapibase-backend/") {
		t.Errorf(".git/info/exclude: %s", ex)
	}
	found := FindOldSkill(root)
	if len(found) != 1 || found[0].Kind != "dir" {
		t.Fatalf("old skill: %+v", found)
	}
	if err := RemoveOldSkill(found); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("the old skill must be removed")
	}
	// the new ones are not mistaken for the old one
	if again := FindOldSkill(root); len(again) != 0 {
		t.Errorf("after removal: %+v", again)
	}

	// AGENTS.md line for agents without native skills, written once
	for i := 0; i < 2; i++ {
		if _, err := InstallSkills(Other, ScopeProject, root, false); err != nil {
			t.Fatal(err)
		}
	}
	md, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if strings.Count(string(md), ".agents/skills/rapibase-") != 1 {
		t.Errorf("AGENTS.md: %s", md)
	}
}
