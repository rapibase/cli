package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedSkills(t *testing.T) {
	names := Names()
	want := []string{"rapibase-backend", "rapibase-deploy", "rapibase-functions", "rapibase-observe"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("embedded skills %v, want %v (run go run ./tools/syncskills)", names, want)
	}
	for _, n := range names {
		if v := Version(n); v == "" {
			t.Errorf("%s has no metadata.version", n)
		}
	}
}

func TestWriteAndInstalled(t *testing.T) {
	dest := t.TempDir()
	// a file of an older version must not survive the rewrite
	stale := filepath.Join(dest, "rapibase-backend", "references", "gone.md")
	_ = os.MkdirAll(filepath.Dir(stale), 0o755)
	_ = os.WriteFile(stale, []byte("x"), 0o644)
	if err := Write(dest, "rapibase-backend"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale file kept")
	}
	name, v, ok := Installed(filepath.Join(dest, "rapibase-backend"))
	if !ok || name != "rapibase-backend" || v != Version("rapibase-backend") {
		t.Errorf("Installed = %q %q %v", name, v, ok)
	}
	if _, err := os.Stat(filepath.Join(dest, "rapibase-backend", "references", "rls.md")); err != nil {
		t.Error("references not written")
	}
}

func TestFrontmatter(t *testing.T) {
	md := "---\r\nname: rapibase-deploy\r\ndescription: \"x: y\"\r\nmetadata:\r\n  author: rapibase\r\n  version: \"1.2.3\"\r\nlicense: MIT\r\n---\r\n# body\nversion: 9\n"
	name, v := frontmatter([]byte(md))
	if name != "rapibase-deploy" || v != "1.2.3" {
		t.Errorf("frontmatter = %q %q", name, v)
	}
}

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"1.0.1", "1.0.0", true},
		{"1.10.0", "1.9.9", true},
		{"v2.0.0", "1.99.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0-rc.1", "1.0.0", false},
		{"", "0.0.1", false},
	} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}
