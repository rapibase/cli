package cli

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rapibase/cli/internal/agents"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/skills"
	"github.com/rapibase/cli/internal/ui"
)

// skillApp is an app in a fresh repo with its own home folder. answer != ""
// means a TTY that answers the questions with it.
func skillApp(t *testing.T, yes bool, answer string) (*app, string, string) {
	t.Helper()
	if len(skills.Names()) == 0 {
		t.Skip("no embedded skills (run tools/syncskills)")
	}
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o755)
	u := ui.New(false, yes)
	u.Out, u.Err = io.Discard, io.Discard
	u.Interactive = answer != ""
	u.SetInput(strings.NewReader(answer))
	return &app{ui: u, cwd: root, flags: globalFlags{yes: yes}}, root, home
}

func oldSkill(t *testing.T, base string) string {
	t.Helper()
	dir := filepath.Join(base, ".claude", "skills", "rapibase")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: rapibase\n---\nmy notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// L11: the old skill is never deleted without a yes (spec §9.1).
func TestInstallSkillsAsksBeforeReplacingTheOldOne(t *testing.T) {
	newSkill := func(root string) string { return filepath.Join(root, ".claude", "skills", "rapibase-backend") }

	t.Run("no TTY, no --yes: fails and touches nothing", func(t *testing.T) {
		a, root, _ := skillApp(t, false, "")
		old := oldSkill(t, root)
		_, err := a.installSkills([]agents.Agent{agents.Claude}, agents.ScopeProject, false)
		if err == nil || cerr.As(err).Code != cerr.MissingFlag {
			t.Fatalf("want missing_flag, got %v", err)
		}
		if !exists(old) || exists(newSkill(root)) {
			t.Error("nothing may change without --yes")
		}
	})

	t.Run("--yes replaces it", func(t *testing.T) {
		a, root, _ := skillApp(t, true, "")
		old := oldSkill(t, root)
		out, err := a.installSkills([]agents.Agent{agents.Claude}, agents.ScopeProject, false)
		if err != nil {
			t.Fatal(err)
		}
		if exists(old) || !exists(newSkill(root)) || out["old_skill_removed"] == nil {
			t.Errorf("want the old one replaced: %v", out)
		}
	})

	t.Run("TTY, enter: replaces it", func(t *testing.T) {
		a, root, _ := skillApp(t, false, "\n")
		old := oldSkill(t, root)
		if _, err := a.installSkills([]agents.Agent{agents.Claude}, agents.ScopeProject, false); err != nil {
			t.Fatal(err)
		}
		if exists(old) || !exists(newSkill(root)) {
			t.Error("the default answer replaces it")
		}
	})

	t.Run("TTY, no: installs and keeps it", func(t *testing.T) {
		a, root, _ := skillApp(t, false, "n\n")
		old := oldSkill(t, root)
		out, err := a.installSkills([]agents.Agent{agents.Claude}, agents.ScopeProject, false)
		if err != nil {
			t.Fatal(err)
		}
		if !exists(old) || !exists(newSkill(root)) || out["old_skill_kept"] == nil {
			t.Errorf("want both: %v", out)
		}
	})

	t.Run("the home copy is left alone by a project install", func(t *testing.T) {
		a, root, home := skillApp(t, false, "")
		old := oldSkill(t, home)
		if _, err := a.installSkills([]agents.Agent{agents.Claude}, agents.ScopeProject, false); err != nil {
			t.Fatalf("no question for a copy it does not touch: %v", err)
		}
		if !exists(old) || !exists(newSkill(root)) {
			t.Error("other repos keep their guide")
		}
	})
}

// `skill update` with only the old skill installs the new ones where it was.
func TestOldSkillTargets(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	list, sc := oldSkillTargets(root, []agents.OldSkill{
		{Path: filepath.Join(root, ".claude", "skills", "rapibase"), Kind: "dir"},
		{Path: filepath.Join(home, ".config", "opencode", "skill", "rapibase"), Kind: "dir"},
		{Path: filepath.Join(root, "rapibase.md"), Kind: "file"},
		{Path: filepath.Join(root, "AGENTS.md"), Kind: "agents_line"},
	})
	if want := []agents.Agent{agents.Claude, agents.OpenCode, agents.Other}; !reflect.DeepEqual(list, want) || sc != agents.ScopeProject {
		t.Errorf("repo: %v %v", list, sc)
	}

	list, sc = oldSkillTargets(root, []agents.OldSkill{
		{Path: filepath.Join(home, ".claude", "skills", "rapibase"), Kind: "dir"},
	})
	if !reflect.DeepEqual(list, []agents.Agent{agents.Claude}) || sc != agents.ScopeUser {
		t.Errorf("home only: %v %v", list, sc)
	}
}
