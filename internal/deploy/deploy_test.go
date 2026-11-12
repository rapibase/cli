package deploy

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, body := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDetectDir(t *testing.T) {
	cases := []struct {
		pkg, extra string
		want       Detection
	}{
		{`{"devDependencies":{"vite":"6"},"scripts":{"build":"vite build"}}`, "", Detection{Dir: "dist", Framework: "vite", Build: "npm run build"}},
		{`{"dependencies":{"next":"15"}}`, "", Detection{Framework: "next", SSR: true}},
		{`{"dependencies":{"next":"15"}}`, "next.config.mjs", Detection{Dir: "out", Framework: "next"}},
		{`{"dependencies":{"react-scripts":"5"}}`, "", Detection{Dir: "build", Framework: "cra"}},
	}
	for _, c := range cases {
		dir := t.TempDir()
		write(t, dir, map[string]string{"package.json": c.pkg})
		if c.extra != "" {
			write(t, dir, map[string]string{c.extra: `export default { output: 'export' }`})
		}
		if got := DetectDir(dir); got != c.want {
			t.Errorf("%s %s → %+v, want %+v", c.pkg, c.extra, got, c.want)
		}
	}
	// plain static site
	dir := t.TempDir()
	write(t, dir, map[string]string{"public/index.html": "<h1>hi</h1>"})
	if got := DetectDir(dir); got.Dir != "public" || got.Framework != "static" {
		t.Errorf("static: %+v", got)
	}
}

func TestScanBlocksSecretsAndExcludes(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{
		"index.html":             "x",
		"assets/app.js":          "x",
		".env":                   "SECRET=1",
		".env.production":        "SECRET=1",
		".env.example":           "SECRET=",
		"certs/server.key":       "k",
		"drafts/wip.html":        "x",
		"node_modules/a/i.js":    "x",
		".rapibase/project.json": `{"project_id":1}`,
	})
	files, blocked, err := Scan(dir, Exclusion{"drafts/"}, false)
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, f := range files {
		rels = append(rels, f.Rel)
	}
	sort.Strings(rels)
	if strings.Join(rels, ",") != ".env.example,assets/app.js,index.html" {
		t.Errorf("files: %v", rels)
	}
	if strings.Join(blocked, ",") != ".env,.env.production,certs/server.key,node_modules/" {
		t.Errorf("blocked: %v", blocked)
	}

	// services skip dependency folders instead of blocking them
	_, blocked, _ = Scan(dir, nil, true)
	for _, b := range blocked {
		if b == "node_modules/" {
			t.Error("services must skip node_modules, not block it")
		}
	}
}

func TestZipSite(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{"index.html": "<h1>hi</h1>", "a/b.css": "body{}"})
	files, _, _ := Scan(dir, nil, false)
	b, st, err := ZipSite(files)
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasIndex || st.Files != 2 {
		t.Errorf("stats: %+v", st)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "a/b.css,index.html" {
		t.Errorf("zip has %v (paths must use /)", names)
	}
}

func TestTarServiceKeepsExecBit(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{"main.py": "print(1)", "server": "\x7fELF fake"})
	files, _, _ := Scan(dir, nil, true)
	b, st, err := TarService(files)
	if err != nil || st.Files != 2 {
		t.Fatalf("%v %+v", err, st)
	}
	gz, _ := gzip.NewReader(bytes.NewReader(b))
	tr := tar.NewReader(gz)
	modes := map[string]int64{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		modes[h.Name] = h.Mode
	}
	if modes["server"] != 0o755 || modes["main.py"] != 0o644 {
		t.Errorf("modes: %v", modes)
	}
}

func TestCheckBinary(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{"win.exe": "MZ\x90\x00rest", "text": "hello"})
	if err := CheckBinary(filepath.Join(dir, "win.exe")); err == nil || err.Error() != "windows" {
		t.Errorf("windows binary: %v", err)
	}
	if err := CheckBinary(filepath.Join(dir, "text")); err == nil || err.Error() != "unknown" {
		t.Errorf("text file: %v", err)
	}
}

func TestActionsWorkflow(t *testing.T) {
	repo := t.TempDir()
	if ActionsWorkflow(repo) != "" {
		t.Error("no workflow yet")
	}
	write(t, repo, map[string]string{".github/workflows/rapibase-deploy.yml": "on: push"})
	if ActionsWorkflow(repo) == "" {
		t.Error("the workflow must be found")
	}
}
