package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteMergesAndProtects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	old := "# mine\nOTHER=1\nRAPIBASE_URL=https://old.example\nexport RAPIBASE_ANON_KEY=old\n"
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"RAPIBASE_URL":         "https://app.rapibase.com",
		"RAPIBASE_ANON_KEY":    "anon",
		"RAPIBASE_SERVICE_KEY": "with space",
	}
	res, err := Write(path, env, "vite")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	got := string(b)
	for _, want := range []string{
		"# mine\n", "OTHER=1\n",
		"RAPIBASE_URL=https://app.rapibase.com\n",
		"RAPIBASE_ANON_KEY=anon\n",
		`RAPIBASE_SERVICE_KEY="with space"` + "\n",
		"VITE_RAPIBASE_URL=https://app.rapibase.com\n",
		"VITE_RAPIBASE_ANON_KEY=anon\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf(".env lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "old") {
		t.Errorf("old values must be replaced:\n%s", got)
	}
	if strings.Contains(got, "VITE_RAPIBASE_SERVICE_KEY") {
		t.Error("the service key must never get a public prefix")
	}
	if !res.Gitignore {
		t.Error(".gitignore should have been updated")
	}
	gi, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(gi), ".env\n") {
		t.Errorf(".gitignore: %q", gi)
	}
	ex, _ := os.ReadFile(filepath.Join(dir, ".env.example"))
	if !strings.Contains(string(ex), "RAPIBASE_URL=\n") || strings.Contains(string(ex), "rapibase.com") {
		t.Errorf(".env.example must list names without values: %q", ex)
	}

	// running it again changes nothing in .gitignore and keeps one line per key
	res, err = Write(path, env, "vite")
	if err != nil {
		t.Fatal(err)
	}
	if res.Gitignore {
		t.Error("second run must not touch .gitignore")
	}
	b, _ = os.ReadFile(path)
	if n := strings.Count(string(b), "RAPIBASE_URL="); n != 2 { // plain + VITE_
		t.Errorf("RAPIBASE_URL appears %d times:\n%s", n, b)
	}
}

func TestDetectFramework(t *testing.T) {
	for deps, want := range map[string]string{
		`{"dependencies":{"next":"15"}}`:                    "next",
		`{"devDependencies":{"vite":"6"}}`:                  "vite",
		`{"dependencies":{"@sveltejs/kit":"2","vite":"6"}}`: "sveltekit",
		`{"dependencies":{"expo":"52"}}`:                    "expo",
		`{"dependencies":{"express":"5"}}`:                  "",
	} {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(deps), 0o644)
		if got := DetectFramework(dir); got != want {
			t.Errorf("%s → %q, want %q", deps, got, want)
		}
	}
}

func TestQuote(t *testing.T) {
	for in, want := range map[string]string{
		"plain":       "plain",
		"a b":         `"a b"`,
		`say "hi"`:    `"say \"hi\""`,
		"x#y":         `"x#y"`,
		"https://a.b": "https://a.b",
	} {
		if got := quote(in); got != want {
			t.Errorf("quote(%q) = %q, want %q", in, got, want)
		}
	}
}
