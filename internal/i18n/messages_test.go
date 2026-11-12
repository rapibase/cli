package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	literalKeyRe = regexp.MustCompile(`\bT\("([a-z0-9_.]+)"`)
	verbRe       = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z]`)
)

// dynamicKeys are built at run time ("prefix"+value); every value the code
// can produce is listed here.
var dynamicKeys = []string{
	"status.running", "status.provisioning", "status.paused", "status.error", "status.deleting",
	"svc.bad_binary_windows", "svc.bad_binary_macos", "svc.bad_binary_linux", "svc.bad_binary_unknown",
	"mcp.restart_claude", "mcp.restart_cursor", "mcp.restart_opencode", "mcp.restart_codex",
	"mcp.restart_hermes", "mcp.restart_other",
	"free.db_bytes", "free.storage_bytes", "free.egress_bytes",
}

// TestEveryKeyExists reads every T("…") in the code: a missing key would
// print the key itself to the user.
func TestEveryKeyExists(t *testing.T) {
	used := map[string]bool{}
	err := filepath.WalkDir("..", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range literalKeyRe.FindAllStringSubmatch(string(b), -1) {
			if strings.HasSuffix(m[1], ".") || strings.HasSuffix(m[1], "_") {
				continue // a prefix: its keys are in dynamicKeys or checked in package cli
			}
			used[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(used) < 100 {
		t.Fatalf("only %d keys found: is the walk reading the code?", len(used))
	}
	for _, k := range dynamicKeys {
		used[k] = true
	}
	var missing []string
	for k := range used {
		if !Has(k) {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%d key(s) without a message:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// TestTranslations: the three languages exist and take the same arguments
// in the same order (a %d in English and %s in Spanish would print junk).
func TestTranslations(t *testing.T) {
	for _, k := range Keys() {
		tr := Triple(k)
		for i, s := range tr {
			if strings.TrimSpace(s) == "" {
				t.Errorf("%s: language %d is empty", k, i)
			}
		}
		en := verbRe.FindAllString(strings.ReplaceAll(tr[0], "%%", ""), -1)
		for i, s := range tr[1:] {
			other := verbRe.FindAllString(strings.ReplaceAll(s, "%%", ""), -1)
			if strings.Join(en, " ") != strings.Join(other, " ") {
				t.Errorf("%s: verbs differ between en %v and %s %v", k, en, []string{"es", "pt"}[i], other)
			}
		}
	}
}

// TestSpanishUsesTu: product text in Spanish uses "tú", not "vos".
func TestSpanishUsesTu(t *testing.T) {
	// \b is ASCII-only in Go: "borrá" would match inside "borrándose"
	voseo := regexp.MustCompile(`(?i)(?:^|[^\p{L}])(tenés|querés|podés|corré|pasá|usá|mirá|probá|elegí|abrí|confirmá|creá|vinculá|publicá|agregá|borrá|decí)(?:[^\p{L}]|$)`)
	for _, k := range Keys() {
		if m := voseo.FindString(Triple(k)[1]); m != "" {
			t.Errorf("%s: %q is voseo; use tú", k, strings.TrimSpace(m))
		}
	}
}

func TestParse(t *testing.T) {
	for in, want := range map[string]Lang{"es_AR.UTF-8": ES, "pt-BR": PT, "en_US": EN, "ES": ES} {
		if got, ok := Parse(in); !ok || got != want {
			t.Errorf("Parse(%q) = %v, %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "C", "POSIX", "fr_FR"} {
		if _, ok := Parse(in); ok {
			t.Errorf("Parse(%q) should not match", in)
		}
	}
}

func TestTFallsBackToEnglishAndKey(t *testing.T) {
	defer Set(Current())
	Set(ES)
	if got := T("login.done", "a@b"); got != "Sesión iniciada como a@b" {
		t.Errorf("es: %q", got)
	}
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("missing key: %q", got)
	}
}
