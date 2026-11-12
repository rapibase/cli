package agents

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rapibase/cli/internal/skills"
)

// SkillResult is what `skill install` did for one agent.
type SkillResult struct {
	Agent     Agent    `json:"agent"`
	Dir       string   `json:"dir"`
	Scope     Scope    `json:"scope"`
	Installed []string `json:"installed"`
	Skipped   []string `json:"skipped,omitempty"` // a newer version was already there
	AgentsMD  bool     `json:"agents_md_updated,omitempty"`
	Excluded  bool     `json:"git_excluded,omitempty"`
}

// agentsLine is what agents without native skills read in AGENTS.md.
const agentsLine = "- Rapibase (backend, deploys, functions, logs): read the matching skill in .agents/skills/rapibase-*/SKILL.md before working on it."

// InstallSkills writes the embedded skills where the agent reads them.
// force rewrites even when the installed copy is newer.
func InstallSkills(a Agent, sc Scope, root string, force bool) (*SkillResult, error) {
	t := SkillDir(a, sc, root)
	res := &SkillResult{Agent: a, Dir: t.DisplayPath, Scope: t.ScopeUsed}
	if err := os.MkdirAll(t.Dir, 0o755); err != nil {
		return nil, err
	}
	for _, name := range skills.Names() {
		if _, v, ok := skills.Installed(filepath.Join(t.Dir, name)); ok && !force && skills.Newer(v, skills.Version(name)) {
			res.Skipped = append(res.Skipped, name)
			continue
		}
		if err := skills.Write(t.Dir, name); err != nil {
			return nil, err
		}
		res.Installed = append(res.Installed, name)
	}
	if t.AgentsLine {
		changed, err := ensureAgentsLine(root)
		if err != nil {
			return nil, err
		}
		res.AgentsMD = changed
	}
	if t.ScopeUsed == ScopeLocal {
		var paths []string
		for _, n := range skills.Names() {
			paths = append(paths, "/"+filepath.ToSlash(filepath.Join(t.DisplayPath, n))+"/")
		}
		changed, err := GitExclude(root, paths)
		if err != nil {
			return nil, err
		}
		res.Excluded = changed
	}
	return res, nil
}

// UpdateResult is what `skill update` found and did.
type UpdateResult struct {
	Updated []string `json:"updated"`
	Current []string `json:"current"`
}

// UpdateSkills rewrites every installed Rapibase skill older than the
// embedded one, in every place an agent reads them (repo and user).
func UpdateSkills(root string) (*UpdateResult, error) {
	res := &UpdateResult{Updated: []string{}, Current: []string{}}
	seen := map[string]bool{}
	for _, a := range All {
		for _, sc := range []Scope{ScopeProject, ScopeUser} {
			t := SkillDir(a, sc, root)
			if seen[t.Dir] {
				continue
			}
			seen[t.Dir] = true
			for _, name := range skills.Names() {
				dir := filepath.Join(t.Dir, name)
				_, v, ok := skills.Installed(dir)
				if !ok {
					continue
				}
				where := t.DisplayPath + "/" + name
				if skills.Newer(skills.Version(name), v) {
					if err := skills.Write(t.Dir, name); err != nil {
						return nil, err
					}
					res.Updated = append(res.Updated, where)
				} else {
					res.Current = append(res.Current, where)
				}
			}
		}
	}
	return res, nil
}

// --- the old single skill (name: rapibase) ---

// OldSkill is a leftover of the single `rapibase` skill that Connect used to
// install. It is replaced by the four skills so two guides that disagree
// never live side by side (spec §9.1).
type OldSkill struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // dir | file | agents_line
}

// FindOldSkill lists the old installs in the repo and in the user folders.
func FindOldSkill(root string) []OldSkill {
	var out []OldSkill
	dirs := []string{
		filepath.Join(root, ".claude", "skills", "rapibase"),
		filepath.Join(root, ".opencode", "skill", "rapibase"),
		filepath.Join(home(), ".claude", "skills", "rapibase"),
		filepath.Join(home(), ".config", "opencode", "skill", "rapibase"),
	}
	for _, d := range dirs {
		if name, _, ok := skills.Installed(d); ok && name == "rapibase" {
			out = append(out, OldSkill{Path: d, Kind: "dir"})
		}
	}
	if _, err := os.Stat(filepath.Join(root, "rapibase.md")); err == nil {
		out = append(out, OldSkill{Path: filepath.Join(root, "rapibase.md"), Kind: "file"})
	}
	if b, err := os.ReadFile(filepath.Join(root, "AGENTS.md")); err == nil && strings.Contains(string(b), "rapibase.md") {
		out = append(out, OldSkill{Path: filepath.Join(root, "AGENTS.md"), Kind: "agents_line"})
	}
	return out
}

// RemoveOldSkill deletes what FindOldSkill found.
func RemoveOldSkill(old []OldSkill) error {
	for _, o := range old {
		switch o.Kind {
		case "dir":
			if err := os.RemoveAll(o.Path); err != nil {
				return err
			}
		case "file":
			if err := os.Remove(o.Path); err != nil && !os.IsNotExist(err) {
				return err
			}
		case "agents_line":
			b, err := os.ReadFile(o.Path)
			if err != nil {
				continue
			}
			var keep []string
			for _, line := range strings.Split(string(b), "\n") {
				if !strings.Contains(line, "rapibase.md") {
					keep = append(keep, line)
				}
			}
			if err := os.WriteFile(o.Path, []byte(strings.Join(keep, "\n")), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- AGENTS.md and .git/info/exclude ---

func ensureAgentsLine(root string) (bool, error) {
	p := filepath.Join(root, "AGENTS.md")
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), ".agents/skills/rapibase-") {
		return false, nil
	}
	s := string(b)
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	if len(s) > 0 {
		s += "\n"
	}
	s += agentsLine + "\n"
	return true, os.WriteFile(p, []byte(s), 0o644)
}

// GitExclude adds paths to .git/info/exclude (local to this clone), the way
// Connect does for the "only me" scope, instead of touching the team's
// .gitignore.
func GitExclude(root string, paths []string) (bool, error) {
	gitDir := filepath.Join(root, ".git")
	fi, err := os.Stat(gitDir)
	if err != nil || !fi.IsDir() {
		return false, nil // not a repo (or a worktree): nothing to exclude
	}
	p := filepath.Join(gitDir, "info", "exclude")
	b, _ := os.ReadFile(p)
	have := map[string]bool{}
	for _, line := range strings.Split(string(b), "\n") {
		have[strings.TrimSpace(line)] = true
	}
	s := string(b)
	changed := false
	for _, x := range paths {
		if have[x] {
			continue
		}
		if len(s) > 0 && !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		s += x + "\n"
		changed = true
	}
	if !changed {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(p, []byte(s), 0o644)
}
