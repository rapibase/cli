package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindNearestWithinRepo(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	web := filepath.Join(repo, "apps", "web")
	admin := filepath.Join(repo, "apps", "admin", "src")
	for _, d := range []string{web, admin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Write(repo, Link{ProjectID: 1, Slug: "root"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(web, Link{ProjectID: 2, Slug: "web"}); err != nil {
		t.Fatal(err)
	}

	// the nearest one wins (monorepo)
	if f, err := Find(web); err != nil || f == nil || f.Slug != "web" || f.Version != 1 {
		t.Fatalf("Find(web) = %+v, %v", f, err)
	}
	// a subfolder without its own link uses the root's
	if f, err := Find(admin); err != nil || f == nil || f.Slug != "root" || f.Root != repo {
		t.Fatalf("Find(admin) = %+v, %v", f, err)
	}
	if RepoRoot(admin) != repo {
		t.Errorf("RepoRoot(admin) = %q", RepoRoot(admin))
	}

	// a link above the repo root is never used
	outer := t.TempDir()
	inner := filepath.Join(outer, "repo")
	_ = os.MkdirAll(filepath.Join(inner, ".git"), 0o755)
	_, _ = Write(outer, Link{ProjectID: 9, Slug: "outside"})
	if f, err := Find(inner); err != nil || f != nil {
		t.Fatalf("a link outside the repo must not be found: %+v, %v", f, err)
	}

	// Remove deletes the file and the empty folder
	f, _ := Find(web)
	if err := Remove(f); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(web, Dir)); !os.IsNotExist(err) {
		t.Error(".rapibase should be gone")
	}
}

func TestFindInvalidLink(t *testing.T) {
	repo := t.TempDir()
	_ = os.Mkdir(filepath.Join(repo, ".git"), 0o755)
	_ = os.MkdirAll(filepath.Join(repo, Dir), 0o755)
	_ = os.WriteFile(filepath.Join(repo, Dir, File), []byte(`{"slug":"x"}`), 0o644)
	if _, err := Find(repo); err == nil {
		t.Fatal("a link without project_id must be an error")
	}
}
