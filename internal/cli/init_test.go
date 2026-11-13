package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/i18n"
	"github.com/rapibase/cli/internal/ui"
)

func meWithTeam() *api.Me {
	return &api.Me{
		Orgs: []api.Org{
			{ID: 1, Name: "Personal", Kind: "personal", Role: "owner"},
			{ID: 3, Name: "CSIBEINGS", Kind: "team", Role: "owner"},
		},
		Connection: api.Connection{AccountScopes: []string{"projects:create"}},
	}
}

var freeProject = api.Project{ID: 7, OrgID: 1, OrgName: "Personal", OrgKind: "personal", Slug: "satelites-3a746d", Plan: "free", Status: "running"}

// "Create a new one" says where the project can go, without pushing to a
// paid plan (spec §6.3).
func TestNewProjectOption(t *testing.T) {
	defer i18n.Set(i18n.Current())
	i18n.Set(i18n.ES)
	me := meWithTeam()

	o := newProjectOption(orgChoices(me, nil))
	if o.Disabled || o.Hint != "Gratis" {
		t.Errorf("free one available: %+v", o)
	}

	cs := orgChoices(me, []api.Project{freeProject})
	if cs[0].ok || !strings.Contains(cs[0].hint, "satelites-3a746d") || !cs[1].ok {
		t.Errorf("free one taken, Personal should be disabled with its project: %+v", cs)
	}
	if o := newProjectOption(cs); o.Disabled || o.Hint != "en CSIBEINGS" {
		t.Errorf("free one taken, with a Team: %+v", o)
	}

	me.Orgs = me.Orgs[:1]
	o = newProjectOption(orgChoices(me, []api.Project{freeProject}))
	if !o.Disabled || o.Hint != "ya usaste tu proyecto gratis: satelites-3a746d" {
		t.Errorf("free one taken, no Team: %+v", o)
	}
}

// The bug of 2026-11-12: creating in Personal failed (free one taken) and
// init only offered the existing projects. Now it goes back to the menu
// and the Team can be chosen.
func TestInitGoesBackToTheMenuWhenTheOrgCantTakeIt(t *testing.T) {
	defer i18n.Set(i18n.Current())
	i18n.Set(i18n.ES)
	var posts []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/projects":
			// a session that doesn't reach the free project: the CLI can't
			// know beforehand, the server says it
			json.NewEncoder(w).Encode([]api.Project{{ID: 20, OrgID: 3, OrgName: "CSIBEINGS", OrgKind: "team", Slug: "biyuya-c5a9bd", Plan: "team", Status: "running"}})
		case "GET /api/regions":
			w.Write([]byte("[]"))
		case "POST /api/projects":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			posts = append(posts, body)
			if body["org_id"] == float64(1) {
				w.WriteHeader(403)
				w.Write([]byte(`{"error":"ya tienes tu proyecto gratis; usa ese proyecto o bórralo desde la consola para crear otro","code":"plan_limit","project":"satelites-3a746d"}`))
				return
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(api.Project{ID: 99, OrgID: 3, OrgName: "CSIBEINGS", Slug: "test-rapi-cli-1a2b3c", Status: "provisioning"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cwd := filepath.Join(t.TempDir(), "test-rapi-cli")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	u := ui.New(false, false)
	u.Out, u.Err = &errOut, &errOut
	u.Interactive = true
	// menu: create · name: default · org: Personal → plan_limit
	// menu: create · name: the same · org: CSIBEINGS → created
	u.SetInput(strings.NewReader("1\n\n1\n1\n\n2\n"))
	a := &app{ctx: context.Background(), ui: u, cwd: cwd, api: api.New(srv.URL)}

	p, created, err := a.initProject(meWithTeam(), "", "", "")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, errOut.String())
	}
	if !created || p.Slug != "test-rapi-cli-1a2b3c" {
		t.Fatalf("got %+v, created=%v", p, created)
	}
	if len(posts) != 2 || posts[0]["org_id"] != float64(1) || posts[1]["org_id"] != float64(3) {
		t.Fatalf("requests: %v", posts)
	}
	if posts[1]["name"] != "test-rapi-cli" {
		t.Fatalf("the second try lost the name: %v", posts[1]["name"])
	}
	if !strings.Contains(errOut.String(), "ya tienes tu proyecto gratis") {
		t.Fatalf("no warning:\n%s", errOut.String())
	}
}
