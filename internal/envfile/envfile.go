// Package envfile writes the project keys into .env (spec §8.2).
//
// Rules: the values are never printed; the file is added to .gitignore
// before it is written; if it is already tracked by git nothing is written;
// RAPIBASE_* variables are merged and every other line is kept as-is; a
// .env.example with the names goes next to it; public framework prefixes
// (VITE_, NEXT_PUBLIC_…) are added only for the URL and the anon key, never
// for the service key or the database URL.
package envfile

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rapibase/cli/internal/i18n"
)

// Order of the base variables in new files.
var Base = []string{"RAPIBASE_URL", "RAPIBASE_ANON_KEY", "RAPIBASE_SERVICE_KEY", "RAPIBASE_DB_URL", "RAPIBASE_PROJECT_ID"}

// Public is what may carry a public prefix: it ends up in the browser.
var Public = []string{"RAPIBASE_URL", "RAPIBASE_ANON_KEY"}

// Frameworks and their public prefix.
var Frameworks = map[string]string{
	"vite":      "VITE_",
	"next":      "NEXT_PUBLIC_",
	"astro":     "PUBLIC_",
	"sveltekit": "PUBLIC_",
	"expo":      "EXPO_PUBLIC_",
}

// FrameworkNames lists the accepted --framework values.
func FrameworkNames() []string {
	out := make([]string, 0, len(Frameworks))
	for k := range Frameworks {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// DetectFramework reads package.json in dir and guesses the framework
// ("" if none of the known ones).
func DetectFramework(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Deps    map[string]string `json:"dependencies"`
		DevDeps map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return ""
	}
	has := func(name string) bool {
		_, a := pkg.Deps[name]
		_, b := pkg.DevDeps[name]
		return a || b
	}
	switch {
	case has("next"):
		return "next"
	case has("expo"):
		return "expo"
	case has("@sveltejs/kit"):
		return "sveltekit"
	case has("astro"):
		return "astro"
	case has("vite"):
		return "vite"
	}
	return ""
}

// ErrTracked means the env file is tracked by git: writing keys there would
// commit them.
var ErrTracked = errors.New("tracked by git")

// Result says what was written.
type Result struct {
	File      string   `json:"file"`
	Vars      []string `json:"vars"`
	Example   string   `json:"example,omitempty"`
	Gitignore bool     `json:"gitignore_updated"`
	Framework string   `json:"framework,omitempty"`
}

// Write merges env into path. root is the folder whose .gitignore is used
// (the repo root or the folder of the file).
func Write(path string, env map[string]string, framework string) (*Result, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(abs)
	if Tracked(abs) {
		return nil, ErrTracked
	}
	vars := map[string]string{}
	order := []string{}
	for _, k := range Base {
		if v, ok := env[k]; ok {
			vars[k] = v
			order = append(order, k)
		}
	}
	if prefix := Frameworks[framework]; prefix != "" {
		for _, k := range Public {
			if v, ok := env[k]; ok {
				vars[prefix+k] = v
				order = append(order, prefix+k)
			}
		}
	}

	added, err := EnsureIgnored(dir, filepath.Base(abs))
	if err != nil {
		return nil, err
	}
	old, _ := os.ReadFile(abs)
	if err := os.WriteFile(abs, merge(old, vars, order), 0o600); err != nil {
		return nil, err
	}
	_ = os.Chmod(abs, 0o600)

	res := &Result{File: abs, Vars: order, Gitignore: added, Framework: framework}
	// .env.example next to .env (for .env.local and friends, too)
	example := filepath.Join(dir, ".env.example")
	if ex, err := writeExample(example, order); err == nil && ex {
		res.Example = example
	}
	return res, nil
}

// merge replaces KEY=… lines for the given vars, keeps every other line and
// appends the missing ones at the end.
func merge(old []byte, vars map[string]string, order []string) []byte {
	var out bytes.Buffer
	seen := map[string]bool{}
	if len(old) > 0 {
		sc := bufio.NewScanner(bytes.NewReader(old))
		sc.Buffer(make([]byte, 64*1024), 4<<20)
		for sc.Scan() {
			line := sc.Text()
			key := keyOf(line)
			if v, ok := vars[key]; ok {
				if !seen[key] {
					out.WriteString(key + "=" + quote(v) + "\n")
					seen[key] = true
				}
				continue
			}
			out.WriteString(line + "\n")
		}
	}
	missing := []string{}
	for _, k := range order {
		if !seen[k] {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		if out.Len() > 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n\n")) {
			out.WriteString("\n")
		}
		out.WriteString("# " + i18n.T("env.file_header") + "\n")
		for _, k := range missing {
			out.WriteString(k + "=" + quote(vars[k]) + "\n")
		}
	}
	return out.Bytes()
}

func keyOf(line string) string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "export ")
	if s == "" || strings.HasPrefix(s, "#") {
		return ""
	}
	if i := strings.IndexByte(s, '='); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return ""
}

// quote leaves simple values bare (the Rapibase values always are: URLs,
// JWTs, hex) and double-quotes anything with spaces, # or quotes.
func quote(v string) string {
	if strings.ContainsAny(v, " \t#\"'`") {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v) + `"`
	}
	return v
}

// writeExample adds the names (without values) to .env.example. It returns
// whether the file changed.
func writeExample(path string, names []string) (bool, error) {
	old, _ := os.ReadFile(path)
	have := map[string]bool{}
	for _, line := range strings.Split(string(old), "\n") {
		if k := keyOf(line); k != "" {
			have[k] = true
		}
	}
	var add []string
	for _, k := range names {
		if !have[k] {
			add = append(add, k+"=")
		}
	}
	if len(add) == 0 {
		return false, nil
	}
	var b strings.Builder
	b.Write(old)
	if len(old) > 0 && !strings.HasSuffix(string(old), "\n") {
		b.WriteString("\n")
	}
	if len(old) == 0 {
		b.WriteString("# Rapibase: fill with `rapibase env pull` (writes .env, never committed).\n")
	}
	b.WriteString(strings.Join(add, "\n") + "\n")
	return true, os.WriteFile(path, []byte(b.String()), 0o644)
}

// EnsureIgnored adds name to dir/.gitignore if git would not ignore it
// already. It returns whether .gitignore changed.
func EnsureIgnored(dir, name string) (bool, error) {
	if ignoredByGit(dir, name) {
		return false, nil
	}
	gi := filepath.Join(dir, ".gitignore")
	old, _ := os.ReadFile(gi)
	for _, line := range strings.Split(string(old), "\n") {
		l := strings.TrimSpace(line)
		if l == name || l == "/"+name {
			return false, nil
		}
	}
	var b strings.Builder
	b.Write(old)
	if len(old) > 0 && !strings.HasSuffix(string(old), "\n") {
		b.WriteString("\n")
	}
	b.WriteString(name + "\n")
	return true, os.WriteFile(gi, []byte(b.String()), 0o644)
}

// ignoredByGit asks git (if available) whether dir/name is already ignored.
func ignoredByGit(dir, name string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	cmd := exec.Command("git", "check-ignore", "-q", name)
	cmd.Dir = dir
	return cmd.Run() == nil
}

// Tracked reports whether git tracks the file (git ls-files
// --error-unmatch). Without git or outside a repo it is false.
func Tracked(abs string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	cmd := exec.Command("git", "ls-files", "--error-unmatch", filepath.Base(abs))
	cmd.Dir = filepath.Dir(abs)
	return cmd.Run() == nil
}
