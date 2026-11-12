// Package skills embeds the Rapibase Agent Skills (from rapibase/skills,
// pinned to a tag) so `skill install` never downloads anything (spec §9.1).
package skills

import (
	"bufio"
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed all:files
var embedded embed.FS

// Names lists the embedded skills (rapibase-backend, rapibase-deploy…).
func Names() []string {
	entries, _ := fs.ReadDir(embedded, "files")
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			if _, err := fs.Stat(embedded, path.Join("files", e.Name(), "SKILL.md")); err == nil {
				out = append(out, e.Name())
			}
		}
	}
	sort.Strings(out)
	return out
}

// Version of an embedded skill (metadata.version of its SKILL.md).
func Version(name string) string {
	b, err := fs.ReadFile(embedded, path.Join("files", name, "SKILL.md"))
	if err != nil {
		return ""
	}
	_, v := frontmatter(b)
	return v
}

// Write copies an embedded skill to dest/<name>/, replacing what was there
// (files of an older version that no longer exist are removed).
func Write(dest, name string) error {
	target := filepath.Join(dest, name)
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	root := path.Join("files", name)
	return fs.WalkDir(embedded, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		out := filepath.Join(target, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		b, err := fs.ReadFile(embedded, p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o644)
	})
}

// Installed reads dir/SKILL.md and returns its name and metadata.version.
func Installed(dir string) (name, version string, ok bool) {
	b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return "", "", false
	}
	name, version = frontmatter(b)
	return name, version, name != ""
}

// frontmatter extracts name and metadata.version from a SKILL.md. It is a
// tiny reader for the two fields the CLI needs, not a YAML parser.
func frontmatter(b []byte) (name, version string) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	inFM, inMeta := false, false
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "---" {
			if inFM {
				break
			}
			inFM = true
			continue
		}
		if !inFM {
			continue
		}
		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
		key, val, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch {
		case !indented && key == "name":
			name = val
			inMeta = false
		case !indented && key == "metadata":
			inMeta = true
		case !indented:
			inMeta = false
		case inMeta && key == "version":
			version = val
		}
	}
	return name, version
}

// Newer reports whether version a is newer than b (semver, missing = 0).
func Newer(a, b string) bool {
	pa, pb := parse(a), parse(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parse(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	for i, p := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(p)
	}
	return out
}
