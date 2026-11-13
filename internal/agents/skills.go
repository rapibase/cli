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

// FindSkills lists every Rapibase skill installed for this folder (the
// repo's agent folders) or, with user, in the home folder: the four skills,
// the old single one and the lines in AGENTS.md that point to them. It is
// what `skill uninstall` removes. only limits it to some agents (nil: all);
// the old single skill is not tied to an agent, so it comes only with nil.
func FindSkills(root string, user bool, only []Agent) []OldSkill {
	var out []OldSkill
	seen := map[string]bool{}
	list := only
	if list == nil {
		list = All
	}
	agentsLine := only == nil
	for _, a := range list {
		t := SkillDir(a, ScopeProject, root)
		if user {
			t = SkillDir(a, ScopeUser, root)
		}
		if (t.ScopeUsed == ScopeUser) != user || seen[t.Dir] {
			continue
		}
		seen[t.Dir] = true
		if filepath.Base(filepath.Dir(t.Dir)) == ".agents" {
			agentsLine = true
		}
		for _, name := range skills.Names() {
			d := filepath.Join(t.Dir, name)
			if n, _, ok := skills.Installed(d); ok && n == name {
				out = append(out, OldSkill{Path: d, Kind: "dir"})
			}
		}
	}
	inRepo := func(p string) bool {
		return strings.HasPrefix(filepath.Clean(p), filepath.Clean(root)+string(filepath.Separator))
	}
	if only == nil {
		for _, o := range FindOldSkill(root) {
			if inRepo(o.Path) != user {
				out = append(out, o)
			}
		}
	}
	if !user && agentsLine {
		p := filepath.Join(root, "AGENTS.md")
		if b, err := os.ReadFile(p); err == nil && strings.Contains(string(b), ".agents/skills/rapibase-") {
			out = append(out, OldSkill{Path: p, Kind: "skills_line"})
		}
	}
	return out
}

// RemoveOldSkill deletes what FindOldSkill (or FindSkills) found.
func RemoveOldSkill(old []OldSkill) error {
	for _, o := range old {
		switch o.Kind {
		case "skills_line":
			if err := dropLines(o.Path, ".agents/skills/rapibase-"); err != nil {
				return err
			}
			continue
		case "dir":
			if err := os.RemoveAll(o.Path); err != nil {
				return err
			}
			// .claude/skills (and .claude) left empty go too
			pruneEmpty(filepath.Dir(o.Path), 2)
		case "file":
			if err := os.Remove(o.Path); err != nil && !os.IsNotExist(err) {
				return err
			}
		case "agents_line":
			if err := dropLines(o.Path, "rapibase.md"); err != nil {
				return err
			}
		}
	}
	return nil
}

// pruneEmpty removes dir if it is empty, and then its parents, at most
// levels folders up. A folder with anything in it stays.
func pruneEmpty(dir string, levels int) {
	for i := 0; i < levels; i++ {
		if os.Remove(dir) != nil { // fails unless empty
			return
		}
		dir = filepath.Dir(dir)
	}
}

// dropLines removes the lines of a file that contain s; a blank line left
// at the end by the removal goes too.
func dropLines(path, s string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var keep []string
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, s) {
			keep = append(keep, line)
		}
	}
	out := strings.Join(keep, "\n")
	if strings.HasSuffix(out, "\n\n") {
		out = strings.TrimRight(out, "\n") + "\n"
	}
	return os.WriteFile(path, []byte(out), 0o644)
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
