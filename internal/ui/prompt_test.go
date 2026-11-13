package ui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
)

// chunks is a terminal: each Read returns one key press (or one paste).
type chunks []string

func (c *chunks) Read(p []byte) (int, error) {
	if len(*c) == 0 {
		return 0, io.EOF
	}
	n := copy(p, (*c)[0])
	(*c)[0] = (*c)[0][n:]
	if (*c)[0] == "" {
		*c = (*c)[1:]
	}
	return n, nil
}

// keysUI is a UI whose prompts take the arrows, reading the key presses in.
func keysUI(t *testing.T, in ...string) (*UI, *bytes.Buffer) {
	t.Helper()
	oldRaw, oldSize := rawMode, termSize
	rawMode = func() (func(), error) { return func() {}, nil }
	termSize = func() (int, int) { return 60, 20 }
	t.Cleanup(func() { rawMode, termSize = oldRaw, oldSize })
	var out bytes.Buffer
	u := &UI{Interactive: true, Out: &out, Err: &out}
	c := chunks(in)
	u.SetInput(&c)
	u.keys = true
	return u, &out
}

const (
	up    = "\x1b[A"
	down  = "\x1b[B"
	enter = "\r"
	esc   = "\x1b"
	bs    = "\x7f"
)

var projects = []Option{
	{Label: "Crear uno nuevo", Hint: "en CSIBEINGS"},
	{Label: "CSIBEINGS", Header: true},
	{Label: "biyuya-c5a9bd", Hint: "Team · corriendo"},
	{Label: "deobra-51f0e9", Hint: "Team · corriendo"},
	{Label: "Personal", Header: true},
	{Label: "satelites-3a746d", Hint: "Gratis · corriendo"},
}

func TestSelectWithArrows(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		def  int
		want int
	}{
		{"enter takes the default", []string{enter}, 3, 3},
		{"down skips the header", []string{down, enter}, 0, 2},
		{"up from the first wraps to the last", []string{up, enter}, 0, 5},
		{"typing filters", []string{"s", "a", "t", "e", enter}, 0, 5},
		{"a paste filters too", []string{"sate", enter}, 0, 5},
		{"the filter also matches the group", []string{"csib", down, enter}, 0, 3},
		{"backspace edits the filter", []string{"satx", bs, bs, bs, bs, up, enter}, 0, 3},
		{"esc clears the filter first", []string{"zzz", esc, down, enter}, 0, 2},
		{"end", []string{"\x1b[F", enter}, 0, 5},
		{"page down stops at the last", []string{"\x1b[6~", "\x1b[6~", enter}, 0, 5},
		{"enter with nothing matching does nothing", []string{"zzz", enter, "\x15", enter}, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, _ := keysUI(t, c.in...)
			got, err := u.Select("¿Qué proyecto?", projects, c.def)
			if err != nil || got != c.want {
				t.Fatalf("got %d, %v; want %d", got, err, c.want)
			}
		})
	}
}

func TestSelectSkipsDisabled(t *testing.T) {
	opts := []Option{
		{Label: "Personal", Hint: "ya usaste tu proyecto gratis: satelites", Disabled: true},
		{Label: "CSIBEINGS", Hint: "Team"},
		{Label: "Otra", Hint: "Team"},
	}
	u, out := keysUI(t, enter)
	if got, _ := u.Select("Organización", opts, 0); got != 1 {
		t.Fatalf("default on a disabled option: got %d, want 1", got)
	}
	if !strings.Contains(out.String(), "ya usaste tu proyecto gratis") {
		t.Fatal("the disabled option doesn't show why")
	}
	u, _ = keysUI(t, up, enter)
	if got, _ := u.Select("Organización", opts, 1); got != 2 {
		t.Fatalf("up from the first enabled: got %d, want 2 (wraps past the disabled one)", got)
	}
}

func TestSelectBackAndCancel(t *testing.T) {
	u, _ := keysUI(t, esc)
	if _, err := u.SelectBack("Organización", projects, 0); !errors.Is(err, ErrBack) {
		t.Fatalf("SelectBack + esc: %v, want ErrBack", err)
	}
	u, _ = keysUI(t, esc, enter)
	if got, err := u.Select("¿Qué proyecto?", projects, 0); err != nil || got != 0 {
		t.Fatalf("Select ignores esc: got %d, %v", got, err)
	}
	u, _ = keysUI(t, "\x03")
	if _, err := u.Select("¿Qué proyecto?", projects, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("ctrl+c: %v, want context.Canceled", err)
	}
}

var csi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// what stays on screen after a prompt: its answer on one line
func TestSelectLeavesTheAnswer(t *testing.T) {
	u, out := keysUI(t, down, enter)
	if _, err := u.Select("¿Qué proyecto?", projects, 0); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	last := s[strings.LastIndex(s, "\x1b[J")+3:]
	if got := csi.ReplaceAllString(last, ""); got != "✔ ¿Qué proyecto? biyuya-c5a9bd\r\n" {
		t.Fatalf("final line %q", got)
	}
	if !strings.HasSuffix(s, showCursor) {
		t.Fatal("the cursor stays hidden")
	}
}

// no line is wider than the terminal, or redrawing would leave garbage
func TestSelectClipsToTheWidth(t *testing.T) {
	u, out := keysUI(t, enter)
	termSize = func() (int, int) { return 24, 20 }
	if _, err := u.Select("¿Qué proyecto quieres usar ahora?", projects, 0); err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(csi.ReplaceAllString(out.String(), ""), "\r\n") {
		l = strings.TrimPrefix(l, "\r")
		if strings.HasPrefix(l, "✔") {
			continue // the answer is written once, never redrawn
		}
		if n := width(l); n > 23 {
			t.Errorf("line of %d columns: %q", n, l)
		}
	}
}

func TestSelectScrollsLongLists(t *testing.T) {
	var opts []Option
	for i := 0; i < 40; i++ {
		opts = append(opts, Option{Label: "p" + string(rune('a'+i%26)) + string(rune('a'+i/26))})
	}
	var in []string
	for i := 0; i < 30; i++ {
		in = append(in, down)
	}
	u, out := keysUI(t, append(in, enter)...)
	got, err := u.Select("¿Qué proyecto?", opts, 0)
	if err != nil || got != 30 {
		t.Fatalf("got %d, %v", got, err)
	}
	frames := strings.Split(out.String(), "\x1b[J")
	for _, f := range frames {
		if n := strings.Count(f, "\r\n"); n > maxRows+2 {
			t.Fatalf("a frame of %d lines", n)
		}
	}
	if !strings.Contains(out.String(), "31/40") {
		t.Fatal("no position while scrolling")
	}
}

func TestConfirmWithKeys(t *testing.T) {
	for in, want := range map[string]bool{"": true, "\x1b[C": false, "n": false, "s": true, "y": true} {
		u, _ := keysUI(t, in, enter)
		if got, err := u.Confirm("¿Borrar el skill viejo?", true); err != nil || got != want {
			t.Errorf("%q: got %v, %v; want %v", in, got, err, want)
		}
	}
}

func TestInputWithKeys(t *testing.T) {
	valid := func(s string) string {
		if strings.Contains(s, "!") {
			return "sin signos"
		}
		return ""
	}
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{enter}, "test-rapi-cli"},
		{[]string{"mi-app", enter}, "mi-app"},
		{[]string{"\t", "-2", enter}, "test-rapi-cli-2"},
		{[]string{"ab!", enter, "\x7f", enter}, "ab"},
		{[]string{"xyz", "\x15", "ok", enter}, "ok"},
	}
	for _, c := range cases {
		u, out := keysUI(t, c.in...)
		got, err := u.Input("Nombre del proyecto", "test-rapi-cli", valid)
		if err != nil || got != c.want {
			t.Errorf("%q: got %q, %v; want %q", c.in, got, err, c.want)
		}
		if strings.Contains(strings.Join(c.in, ""), "!") && !strings.Contains(out.String(), "sin signos") {
			t.Errorf("%q: the validation message isn't shown", c.in)
		}
	}
	u, _ := keysUI(t, "abc", esc)
	if _, err := u.InputBack("Nombre del proyecto", "x", nil); !errors.Is(err, ErrBack) {
		t.Fatalf("InputBack + esc: %v", err)
	}
}

// without raw mode (an old console), the numbered prompts still work
func TestFallsBackToNumbers(t *testing.T) {
	u, out := keysUI(t, "3\n")
	rawMode = func() (func(), error) { return nil, errors.New("no raw") }
	got, err := u.Select("¿Qué proyecto?", projects, 0)
	if err != nil || got != 3 {
		t.Fatalf("got %d, %v; want 3", got, err)
	}
	if !strings.Contains(out.String(), " 3) deobra-51f0e9") {
		t.Fatalf("not numbered: %q", out.String())
	}
}
