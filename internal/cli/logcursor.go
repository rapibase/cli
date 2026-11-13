package cli

import (
	"strings"
	"time"
)

// logCursor marks how far `logs --follow` has shown (spec L5). Every line of
// docker logs --timestamps starts with an RFC 3339 timestamp with
// nanoseconds; the cursor keeps the last one shown and how many lines with
// exactly that timestamp were shown, so the next request can ask the server
// for lines since then and still drop the ones already printed. Two
// identical lines in a row are two lines, and nothing depends on comparing
// text.
type logCursor struct {
	ts   time.Time
	seen int // lines at ts already shown
}

// lineTime is the docker timestamp at the start of a log line.
func lineTime(line string) (time.Time, bool) {
	head, _, _ := strings.Cut(line, " ")
	t, err := time.Parse(time.RFC3339Nano, head)
	return t, err == nil
}

// fresh returns the lines not shown yet and moves the cursor past them.
// Lines without a timestamp belong to the last one seen before them.
func (c *logCursor) fresh(lines []string) []string {
	var out []string
	var cur time.Time
	atCursor := 0
	for _, l := range lines {
		if t, ok := lineTime(l); ok {
			cur = t
		}
		switch {
		case cur.Before(c.ts):
			continue
		case cur.Equal(c.ts):
			atCursor++
			if atCursor <= c.seen {
				continue
			}
			c.seen = atCursor
		default:
			c.ts, c.seen, atCursor = cur, 1, 1
		}
		out = append(out, l)
	}
	return out
}

// since is the value for ?since= (empty before the first line).
func (c *logCursor) since() string {
	if c.ts.IsZero() {
		return ""
	}
	return c.ts.UTC().Format(time.RFC3339Nano)
}

// gap says that a batch did not reach back to the cursor: it came full and
// starts after the last line shown, so lines in between were lost.
func (c *logCursor) gap(lines []string, limit int) bool {
	if c.ts.IsZero() || len(lines) < limit {
		return false
	}
	t, ok := lineTime(lines[0])
	return ok && t.After(c.ts)
}
