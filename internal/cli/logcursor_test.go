package cli

import (
	"reflect"
	"testing"
)

func TestLogCursor(t *testing.T) {
	const (
		t1 = "2026-11-12T10:00:00.000000001Z"
		t2 = "2026-11-12T10:00:00.000000002Z"
		t3 = "2026-11-12T10:00:01.500000000Z"
	)
	var c logCursor
	if c.since() != "" {
		t.Fatal("since before any line")
	}
	first := []string{t1 + " GET /health 200", t2 + " GET /health 200", t2 + " GET /health 200"}
	if got := c.fresh(first); !reflect.DeepEqual(got, first) {
		t.Fatalf("first batch: %v", got)
	}
	if c.since() != t2 {
		t.Fatalf("since = %q", c.since())
	}

	// the server answers from t2 included: the two lines at t2 were shown,
	// the third one at t2 is new, and identical lines in a row both count
	next := []string{t2 + " GET /health 200", t2 + " GET /health 200", t2 + " GET /health 200", t3 + " boom", t3 + " boom"}
	want := []string{t2 + " GET /health 200", t3 + " boom", t3 + " boom"}
	if got := c.fresh(next); !reflect.DeepEqual(got, want) {
		t.Fatalf("next batch: %v", got)
	}
	// an old server ignores since and sends the tail again: nothing repeats
	if got := c.fresh(append(first, next...)); len(got) != 0 {
		t.Fatalf("repeated tail: %v", got)
	}
	// a line without timestamp goes with the one before it
	if got := c.fresh([]string{t3 + " boom", t3 + " boom", "  at main.go:12", "2026-11-12T10:00:02Z ok"}); !reflect.DeepEqual(got, []string{"  at main.go:12", "2026-11-12T10:00:02Z ok"}) {
		t.Fatalf("continuation line: %v", got)
	}

	// gap: a full batch that starts after the cursor lost lines
	if !c.gap([]string{"2026-11-12T10:00:05Z a", "2026-11-12T10:00:05Z b"}, 2) {
		t.Error("full batch after the cursor is a gap")
	}
	if c.gap([]string{"2026-11-12T10:00:02Z ok", "2026-11-12T10:00:05Z b"}, 2) {
		t.Error("a batch that starts at the cursor is not a gap")
	}
	if c.gap([]string{"2026-11-12T10:00:05Z a"}, 2) {
		t.Error("a batch that is not full is not a gap")
	}
}
