package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rapibase/cli/internal/skills"
)

// isolate points the home folder and Claude Code's config to a temp folder
// and empties PATH, so no real config or binary (claude, codex) is touched.
func isolate(t *testing.T) (home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	t.Setenv("PATH", "")
	return home
}

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(es []MCPEntry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Name)
	}
	return out
}

func TestAddMCPReportsTheNameAlreadyThere(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	p := filepath.Join(root, "opencode.json")
	orig := `{"mcp":{"rapibase-shop":{"type":"remote","url":"https://rapibase.com/mcp?project=shop"}}}`
	write(t, p, orig)
	res, err := AddMCP(MCPOptions{Agent: OpenCode, Scope: ScopeProject, Root: root, AccountURL: "https://rapibase.com/mcp", Slug: "shop"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unchanged || res.Name != "rapibase-shop" || res.File != "opencode.json" || !res.Pinned {
		t.Fatalf("the entry from Connect with the same URL counts as there, under its name: %+v", res)
	}
	if b, _ := os.ReadFile(p); string(b) != orig {
		t.Errorf("the file was rewritten:\n%s", b)
	}
}

func TestLegacyMCPFindsTheOlderOnesOfThisProject(t *testing.T) {
	home := isolate(t)
	root := filepath.Join(t.TempDir(), "Shop")
	// Claude Code keeps the folder with forward slashes, and on Windows the
	// case may not match the real one
	key := filepath.ToSlash(root)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	cj, _ := json.Marshal(map[string]any{
		"projects": map[string]any{
			key: map[string]any{"mcpServers": map[string]any{
				"rapibase-shop-1a2b3c": map[string]any{"type": "http", "url": "https://shop.rapibase.com/mcp",
					"headers": map[string]any{"Authorization": "Bearer s3cr3t"}},
				"notes": map[string]any{"type": "http", "url": "https://notes.example/mcp"},
			}},
			"/somewhere/else": map[string]any{"mcpServers": map[string]any{
				"rapibase-other-9z9z9z": map[string]any{"type": "http", "url": "https://other.rapibase.com/mcp"},
			}},
		},
		"mcpServers": map[string]any{"rapibase": map[string]any{"type": "http", "url": "https://rapibase.com/mcp"}},
	})
	write(t, filepath.Join(home, ".claude.json"), string(cj))
	write(t, filepath.Join(root, ".mcp.json"), `{
  "mcpServers": {
    "github": {"type": "http", "url": "https://gh.example/mcp"},
    "rapibase": {"type": "http", "url": "https://rapibase.com/mcp?project=old"},
    "shop": {"type": "http", "url": "https://shop.rapibase.com/mcp"},
    "rapibase-another-4d4d4d": {"type": "http", "url": "https://rapibase.com/mcp?project=another-4d4d4d"}
  }
}`)
	o := MCPOptions{Agent: Claude, Scope: ScopeProject, Root: root, AccountURL: "https://rapibase.com/mcp", Slug: "shop-1a2b3c", Sub: "shop"}

	legacy := LegacyMCP(o)
	got := strings.Join(names(legacy), ",")
	// the instance's MCP in the local scope (by name) and in .mcp.json (by
	// host); not the rapibase that gets rewritten, not the user scope, not
	// another project, not someone else's server
	if got != "shop,rapibase-shop-1a2b3c" && got != "rapibase-shop-1a2b3c,shop" {
		t.Fatalf("legacy = %q", got)
	}
	for _, e := range legacy {
		if e.Name == "rapibase-shop-1a2b3c" && (!e.Token || e.Scope != ScopeLocal) {
			t.Errorf("local entry: %+v", e)
		}
	}
	if b, _ := json.Marshal(legacy); strings.Contains(string(b), "s3cr3t") {
		t.Fatal("the key ended up in the JSON")
	}

	removed, manual, err := RemoveMCP(legacy)
	if err != nil {
		t.Fatal(err)
	}
	// .mcp.json is rewritten by the CLI; ~/.claude.json goes through
	// `claude mcp remove`, which isn't on PATH here: the command comes back
	if len(removed) != 1 || removed[0].Name != "shop" {
		t.Fatalf("removed = %v", names(removed))
	}
	if len(manual) != 1 || !strings.Contains(manual[0], "claude mcp remove --scope local rapibase-shop-1a2b3c") || strings.Contains(manual[0], "s3cr3t") {
		t.Fatalf("manual = %q", manual)
	}
	b, _ := os.ReadFile(filepath.Join(root, ".mcp.json"))
	s := string(b)
	if strings.Contains(s, `"shop"`) || !strings.Contains(s, `"github"`) || !strings.Contains(s, `"rapibase-another-4d4d4d"`) ||
		strings.Index(s, `"github"`) > strings.Index(s, `"rapibase"`) {
		t.Fatalf(".mcp.json after removing:\n%s", s)
	}

	// adding now rewrites rapibase in place, pinned to this project
	res, err := AddMCP(o)
	if err != nil || !res.Registered || res.Unchanged || res.URL != "https://rapibase.com/mcp?project=shop-1a2b3c" {
		t.Fatalf("add: %+v %v", res, err)
	}
}

func TestLegacyMCPLocalRapibaseShadowsTheProjectOne(t *testing.T) {
	home := isolate(t)
	root := t.TempDir()
	cj, _ := json.Marshal(map[string]any{"projects": map[string]any{
		filepath.ToSlash(root): map[string]any{"mcpServers": map[string]any{
			"rapibase": map[string]any{"type": "http", "url": "https://rapibase.com/mcp?project=old-1a1a1a"},
		}},
	}})
	write(t, filepath.Join(home, ".claude.json"), string(cj))
	o := MCPOptions{Agent: Claude, Scope: ScopeProject, Root: root, AccountURL: "https://rapibase.com/mcp", Slug: "new-2b2b2b"}
	if got := names(LegacyMCP(o)); len(got) != 1 || got[0] != "rapibase" {
		t.Fatalf("a rapibase in the local scope wins over .mcp.json: %v", got)
	}
	// the same URL in the local scope is the MCP already there
	o.Slug = "old-1a1a1a"
	res, err := AddMCP(o)
	if err != nil || !res.Unchanged || res.Scope != ScopeLocal || len(LegacyMCP(o)) != 0 {
		t.Fatalf("same URL: %+v %v", res, err)
	}
}

func TestFindMCPOnlyRapibase(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	write(t, filepath.Join(root, ".cursor", "mcp.json"), `{"mcpServers":{"rapibase":{"url":"https://rapibase.com/mcp?project=a"},"x":{"url":"https://x.example/mcp"},"mine":{"url":"https://app.rapibase.com/mcp"}}}`)
	got := names(FindMCP(Cursor, root, root, "https://rapibase.com/mcp", false))
	if strings.Join(got, ",") != "mine,rapibase" {
		t.Fatalf("found %v", got)
	}
	if got := FindMCP(Cursor, root, root, "https://rapibase.com/mcp", true); len(got) != 0 {
		t.Fatalf("user scope must not see the repo: %v", names(got))
	}
}

func TestFindSkillsAndUninstall(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	write(t, filepath.Join(root, "AGENTS.md"), "# My rules\n\nBe nice.\n")
	for _, a := range []Agent{Claude, Other} {
		if _, err := InstallSkills(a, ScopeProject, root, false); err != nil {
			t.Fatal(err)
		}
	}
	found := FindSkills(root, false, nil)
	if want := 2*len(skills.Names()) + 1; len(found) != want {
		t.Fatalf("found %d, want %d: %+v", len(found), want, found)
	}
	if got := FindSkills(root, true, nil); len(got) != 0 {
		t.Fatalf("nothing in the home folder: %+v", got)
	}
	if err := RemoveOldSkill(found); err != nil {
		t.Fatal(err)
	}
	if again := FindSkills(root, false, nil); len(again) != 0 {
		t.Fatalf("left: %+v", again)
	}
	b, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if string(b) != "# My rules\n\nBe nice.\n" {
		t.Fatalf("AGENTS.md: %q", b)
	}
	// no empty .claude/skills or .agents/skills left behind
	for _, d := range []string{".claude", ".agents"} {
		if _, err := os.Stat(filepath.Join(root, d)); !os.IsNotExist(err) {
			t.Errorf("%s left empty: %v", d, err)
		}
	}
}

// skill uninstall --agent: only that agent's folder; a folder with other
// things in it stays.
func TestFindSkillsOneAgent(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	for _, a := range []Agent{Claude, Codex} {
		if _, err := InstallSkills(a, ScopeProject, root, false); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(root, ".claude", "settings.json"), `{}`)
	found := FindSkills(root, false, []Agent{Claude})
	if len(found) != len(skills.Names()) {
		t.Fatalf("only Claude's skills: %+v", found)
	}
	for _, f := range found {
		if !strings.Contains(filepath.ToSlash(f.Path), "/.claude/skills/") {
			t.Fatalf("not Claude's: %s", f.Path)
		}
	}
	if err := RemoveOldSkill(found); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills")); !os.IsNotExist(err) {
		t.Error(".claude/skills left empty")
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "settings.json")); err != nil {
		t.Error(".claude had settings.json: it stays")
	}
	if left := FindSkills(root, false, []Agent{Codex}); len(left) == 0 {
		t.Error("Codex's skills should still be there")
	}
}

// An .mcp.json that only held Rapibase goes; one with more stays.
func TestRemoveServerEmptyFile(t *testing.T) {
	root := t.TempDir()
	only := filepath.Join(root, ".mcp.json")
	write(t, only, `{"mcpServers":{"rapibase":{"type":"http","url":"https://rapibase.com/mcp"}}}`)
	if ok, err := removeServer(only, "mcpServers", "rapibase"); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if _, err := os.Stat(only); !os.IsNotExist(err) {
		t.Error(".mcp.json with nothing left should go")
	}
	cursor := filepath.Join(root, ".cursor", "mcp.json")
	write(t, cursor, `{"mcpServers":{"rapibase":{"url":"https://rapibase.com/mcp"}}}`)
	if _, err := removeServer(cursor, "mcpServers", "rapibase"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor")); !os.IsNotExist(err) {
		t.Error(".cursor only had mcp.json: it goes")
	}
	more := filepath.Join(root, "opencode.json")
	write(t, more, `{"$schema":"x","mcp":{"rapibase":{"type":"remote"}}}`)
	if _, err := removeServer(more, "mcp", "rapibase"); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(more); err != nil || !strings.Contains(string(b), "$schema") {
		t.Errorf("opencode.json has more config: it stays (%s, %v)", b, err)
	}
	if root == "" || filepath.Dir(only) != root {
		t.Fatal("the repo folder itself is never removed")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("root removed")
	}
}

// Claude Code keys a project by the folder as typed: on Windows the same
// folder can show up as ".../Desktop/app" and ".../desktop/app".
func TestFindMCPSameFolderTwiceInClaudeJSON(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("folders are case-sensitive here")
	}
	home := isolate(t)
	root := filepath.Join(t.TempDir(), "App")
	lower := strings.ToLower(filepath.ToSlash(root))
	cj, _ := json.Marshal(map[string]any{"projects": map[string]any{
		filepath.ToSlash(root): map[string]any{"mcpServers": map[string]any{}},
		lower: map[string]any{"mcpServers": map[string]any{
			"rapibase-app-1a2b3c": map[string]any{"type": "http", "url": "https://rapibase.com/mcp?project=app-1a2b3c"},
		}},
	}})
	write(t, filepath.Join(home, ".claude.json"), string(cj))
	got := FindMCP(Claude, root, root, "https://rapibase.com/mcp", false)
	if len(got) != 1 || got[0].Name != "rapibase-app-1a2b3c" || got[0].dir != lower {
		t.Fatalf("found %+v", got)
	}
}
