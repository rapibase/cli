// Command syncskills copies the skills of rapibase/skills into
// internal/skills/files, where the CLI embeds them (spec §9.1). The release
// pipeline runs it against a tag of rapibase/skills; locally:
//
//	go run ./tools/syncskills -from ../rapibase-skills
//
// Each skill folder is replaced whole, so files removed upstream go away too.
// files/SOURCE records where they came from.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	from := flag.String("from", "../rapibase-skills", "checkout of rapibase/skills")
	to := flag.String("to", "internal/skills/files", "embed folder of the CLI")
	flag.Parse()
	if err := run(*from, *to); err != nil {
		fmt.Fprintln(os.Stderr, "syncskills:", err)
		os.Exit(1)
	}
}

func run(from, to string) error {
	src := filepath.Join(from, "skills")
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	// drop the skills that are there now (README.md and SOURCE stay)
	old, err := os.ReadDir(to)
	if err != nil {
		return err
	}
	for _, e := range old {
		if e.IsDir() {
			if err := os.RemoveAll(filepath.Join(to, e.Name())); err != nil {
				return err
			}
		}
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(src, e.Name(), "SKILL.md")); err != nil {
			continue
		}
		if err := copyDir(filepath.Join(src, e.Name()), filepath.Join(to, e.Name())); err != nil {
			return err
		}
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		return fmt.Errorf("no skills in %s", src)
	}
	rev := gitOut(from, "describe", "--tags", "--always", "--dirty")
	if rev == "" {
		rev = "unknown"
	}
	source := fmt.Sprintf("rapibase/skills %s\n%s\n", rev, strings.Join(names, "\n"))
	if err := os.WriteFile(filepath.Join(to, "SOURCE"), []byte(source), 0o644); err != nil {
		return err
	}
	fmt.Printf("synced %d skills (%s): %s\n", len(names), rev, strings.Join(names, ", "))
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		// LF everywhere: a checkout on Windows must not change the embed
		b = []byte(strings.ReplaceAll(string(b), "\r\n", "\n"))
		return os.WriteFile(out, b, 0o644)
	})
}

func gitOut(dir string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	b, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
