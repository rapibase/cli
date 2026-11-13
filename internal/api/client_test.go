package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeAuth struct{ refreshed int }

func (f *fakeAuth) Token(context.Context) (string, error) { return "personal", nil }
func (f *fakeAuth) Refresh(context.Context) (bool, error) {
	f.refreshed++
	return true, nil
}

func TestRawSendsTheHeadersAndKeepsTheSessionOnUpstream401(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		switch r.URL.Path {
		case "/upstream":
			// the instance said no to the visitor: not the CLI's session
			w.Header().Set("X-Rapibase-Upstream", "1")
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()
	auth := &fakeAuth{}
	c := New(srv.URL)
	c.Auth = auth

	res, err := c.Raw(context.Background(), "GET", "/upstream", nil, true, "X-Rapibase-As", "user", "X-Rapibase-User-JWT", "eyJ.u")
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("X-Rapibase-As") != "user" || got.Get("X-Rapibase-User-JWT") != "eyJ.u" || got.Get("Authorization") != "Bearer personal" {
		t.Fatalf("headers: %v", got)
	}
	if res.Status != http.StatusUnauthorized || !res.Upstream() || auth.refreshed != 0 {
		t.Fatalf("a 401 from the instance must not refresh the session: status %d, refreshed %d", res.Status, auth.refreshed)
	}

	// a 401 of the control plane itself does refresh
	if _, err := c.Raw(context.Background(), "GET", "/own", nil, true); err != nil {
		t.Fatal(err)
	}
	if auth.refreshed != 1 {
		t.Fatalf("refreshed %d", auth.refreshed)
	}
}
