package cli

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/i18n"
)

// rawKey catches a message key printed as-is (a missing translation).
var rawKey = regexp.MustCompile(`\b(cmd|flag|help|err|login|status|deploy|svc|fn|sql|api|rls|init|agent|skill|mcp)\.[a-z_]+(\.[a-z_]+)*\b`)
var cobraEnglish = regexp.MustCompile(`help for [a-z]`)

// every command and subcommand, with its help, in the three languages: a
// missing key, a duplicated flag shorthand or a broken template shows here.
func TestHelpInEveryLanguage(t *testing.T) {
	defer i18n.Set(i18n.Current())
	for _, lang := range []i18n.Lang{i18n.EN, i18n.ES, i18n.PT} {
		i18n.Set(lang)
		a := &app{}
		root := a.rootCmd()
		var paths [][]string
		var walk func(c *cobra.Command, prefix []string)
		walk = func(c *cobra.Command, prefix []string) {
			for _, s := range c.Commands() {
				p := append(append([]string{}, prefix...), s.Name())
				paths = append(paths, p)
				walk(s, p)
			}
		}
		walk(root, nil)
		paths = append(paths, []string{})
		if len(paths) < 30 {
			t.Fatalf("only %d commands found", len(paths))
		}
		for _, p := range paths {
			a := &app{}
			root := a.rootCmd()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs(append(append([]string{}, p...), "--help"))
			if err := root.Execute(); err != nil {
				t.Errorf("%s %v --help: %v", lang, p, err)
				continue
			}
			if lang != i18n.EN && cobraEnglish.MatchString(out.String()) {
				t.Errorf("%s %v --help has cobra's English help flag", lang, p)
			}
			if m := rawKey.FindString(out.String()); m != "" {
				t.Errorf("%s %v --help prints the key %q", lang, p, m)
			}
			if out.Len() < 40 {
				t.Errorf("%s %v --help is almost empty: %q", lang, p, out.String())
			}
		}
	}
}

// the root help lists every group with a description in each language
func TestRootHelpGroups(t *testing.T) {
	defer i18n.Set(i18n.Current())
	for _, lang := range []i18n.Lang{i18n.EN, i18n.ES, i18n.PT} {
		i18n.Set(lang)
		for _, g := range helpGroups {
			if !i18n.Has(g.title) {
				t.Errorf("%s: missing %s", lang, g.title)
			}
			for _, name := range g.cmds {
				k := "cmd." + strings.Fields(name)[0] + keySuffix(name) + ".short"
				if !i18n.Has(k) {
					t.Errorf("%s: missing %s", lang, k)
				}
			}
		}
	}
}

func TestAPIPath(t *testing.T) {
	ok := map[string]string{
		"/api/v1/rest/products":           "v1/rest/products",
		"api/v1/rest/products?limit=5":    "v1/rest/products?limit=5",
		"v1/health":                       "v1/health",
		"https://x.rapibase.com/api/v1/a": "v1/a",
	}
	for in, want := range ok {
		if got, err := apiPath(in); err != nil || got != want {
			t.Errorf("apiPath(%q) = %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "/api/", "/api/../admin", "/api/v1//x", `/api/v1\x`} {
		if _, err := apiPath(bad); err == nil {
			t.Errorf("apiPath(%q) should fail", bad)
		}
	}
}

func TestScalar(t *testing.T) {
	cases := map[string]any{
		"42": json.Number("42"), "-1.5": json.Number("-1.5"), "true": true, "null": nil,
		"paid": "paid", "007": "007", "1e3x": "1e3x", "": "",
	}
	for in, want := range cases {
		if got := scalar(in); got != want {
			t.Errorf("scalar(%q) = %#v, want %#v", in, got, want)
		}
	}
}

func TestFormBody(t *testing.T) {
	if _, _, err := formBody([]string{"novalue"}); err == nil {
		t.Error("a field without = must fail")
	}
	b, ct, err := formBody([]string{"path=folder/"})
	if err != nil || !strings.HasPrefix(ct, "multipart/form-data") || !bytes.Contains(b, []byte("folder/")) {
		t.Errorf("form: %v %q", err, ct)
	}
}
