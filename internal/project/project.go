// Package project handles the folder link, .rapibase/project.json (spec
// §4.5 and §4.6). It holds the id and the slug, never keys, and is meant to
// be committed: whoever clones the repo (a teammate, the CI) already points
// at the right project.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Dir and File are the link location inside a folder.
const (
	Dir  = ".rapibase"
	File = "project.json"
)

// Link is the content of .rapibase/project.json. Version lets the file grow
// (named targets in phase 4) without breaking committed ones.
type Link struct {
	Version   int    `json:"version"`
	ProjectID int64  `json:"project_id"`
	Slug      string `json:"slug"`
	OrgID     int64  `json:"org_id,omitempty"`
}

// Found is a link and where it was found.
type Found struct {
	Link
	Path string // absolute path of project.json
	Root string // the folder that holds .rapibase/
}

// RepoRoot returns the git root of start ("" if not in a repo).
func RepoRoot(start string) string {
	dir := start
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && (fi.IsDir() || fi.Mode().IsRegular()) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Find returns the nearest link from start upwards, without going past the
// repo root (spec §4.5.2). Outside a repo it stops at the home folder, so a
// stray link high up never captures unrelated folders.
func Find(start string) (*Found, error) {
	start, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	stop := RepoRoot(start)
	if stop == "" {
		if home, err := os.UserHomeDir(); err == nil && isWithin(start, home) {
			stop = home
		}
	}
	dir := start
	for {
		p := filepath.Join(dir, Dir, File)
		if b, err := os.ReadFile(p); err == nil {
			var l Link
			if err := json.Unmarshal(b, &l); err != nil || l.ProjectID == 0 {
				return nil, fmt.Errorf("%s: %w", p, errInvalid)
			}
			return &Found{Link: l, Path: p, Root: dir}, nil
		}
		if dir == stop {
			return nil, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

var errInvalid = errors.New("invalid project link (expected {\"version\":1,\"project_id\":…})")

// Write creates or replaces .rapibase/project.json in dir.
func Write(dir string, l Link) (string, error) {
	l.Version = 1
	if err := os.MkdirAll(filepath.Join(dir, Dir), 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, Dir, File)
	return p, os.WriteFile(p, append(b, '\n'), 0o644)
}

// Remove deletes the link (and .rapibase/ if it ends up empty).
func Remove(f *Found) error {
	if err := os.Remove(f.Path); err != nil {
		return err
	}
	_ = os.Remove(filepath.Dir(f.Path)) // only succeeds if empty
	return nil
}

func isWithin(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	return err == nil && rel != ".." && !startsWithDotDot(rel)
}

func startsWithDotDot(rel string) bool {
	return len(rel) >= 3 && rel[:2] == ".." && (rel[2] == '/' || rel[2] == '\\')
}
