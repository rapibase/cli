package ui

// Prompts driven by keys: arrows to move, typing to filter, Esc to go back.
// They need a terminal that takes raw mode and VT sequences (every Windows 10+
// console, macOS and Linux terminals); otherwise the numbered prompts of
// ui.go are used.
//
// Each prompt draws a block of lines and redraws it in place on every key.
// Lines are clipped to the width so they never wrap: moving up as many rows
// as were drawn always lands on the first one. In raw mode a Unix terminal
// stops turning "\n" into "\r\n", so the block uses "\r\n".

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/rapibase/cli/internal/i18n"
)

// ErrBack is what SelectBack and InputBack return when the person presses
// Esc: the caller goes back to the previous question.
var ErrBack = errors.New("back")

// rawMode puts stdin in raw mode and returns the function that restores it.
var rawMode = func() (func(), error) {
	fd := int(os.Stdin.Fd())
	st, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() { _ = term.Restore(fd, st) }, nil
}

// termSize is the size of the terminal the prompts draw on (80x24 if unknown).
var termSize = func() (int, int) {
	w, h, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

// maxRows is the most option rows a Select shows at once; longer lists scroll.
const maxRows = 14

// --- keys ---

type key int

const (
	keyOther key = iota
	keyRune
	keyEnter
	keyUp
	keyDown
	keyLeft
	keyRight
	keyHome
	keyEnd
	keyPgUp
	keyPgDn
	keyTab
	keyBackTab
	keyBackspace
	keyClear
	keyEsc
	keyCancel
)

// readKey reads one key press from stdin in raw mode. An Esc with nothing
// right behind it is the Esc key; with "[" or "O" behind, a sequence (arrows,
// Home, End…). Both Windows (VT input) and Unix terminals deliver a whole
// sequence in a single read.
func (u *UI) readKey() (key, rune, error) {
	r, _, err := u.in.ReadRune()
	if err != nil {
		return keyOther, 0, err
	}
	switch r {
	case '\r', '\n':
		return keyEnter, 0, nil
	case 0x03, 0x04: // Ctrl+C, Ctrl+D
		return keyCancel, 0, nil
	case '\t':
		return keyTab, 0, nil
	case 0x7f, 0x08:
		return keyBackspace, 0, nil
	case 0x15: // Ctrl+U
		return keyClear, 0, nil
	case 0x10: // Ctrl+P
		return keyUp, 0, nil
	case 0x0e: // Ctrl+N
		return keyDown, 0, nil
	case 0x1b:
		if u.in.Buffered() == 0 {
			return keyEsc, 0, nil
		}
		b, _ := u.in.ReadByte()
		if b != '[' && b != 'O' {
			return keyOther, 0, nil // Alt+key
		}
		var params []byte
		for u.in.Buffered() > 0 {
			c, _ := u.in.ReadByte()
			if c >= 0x40 && c <= 0x7e {
				return csiKey(c, string(params)), 0, nil
			}
			params = append(params, c)
		}
		return keyOther, 0, nil
	}
	if r < 0x20 || r == utf8.RuneError {
		return keyOther, 0, nil
	}
	return keyRune, r, nil
}

func csiKey(final byte, params string) key {
	switch final {
	case 'A':
		return keyUp
	case 'B':
		return keyDown
	case 'C':
		return keyRight
	case 'D':
		return keyLeft
	case 'H':
		return keyHome
	case 'F':
		return keyEnd
	case 'Z':
		return keyBackTab
	case '~':
		switch params {
		case "1", "7":
			return keyHome
		case "4", "8":
			return keyEnd
		case "5":
			return keyPgUp
		case "6":
			return keyPgDn
		}
	}
	return keyOther
}

// --- drawing ---

// frame is a block of lines redrawn in place.
type frame struct {
	u   *UI
	row int // rows from the first line of the block down to the cursor
}

// draw replaces the block with lines. With curRow >= 0 the cursor ends on
// that line at column curCol; otherwise below the block.
func (f *frame) draw(lines []string, curRow, curCol int) {
	var b strings.Builder
	f.home(&b)
	for _, l := range lines {
		b.WriteString(l)
		b.WriteString("\r\n")
	}
	f.row = len(lines)
	if curRow >= 0 && curRow < len(lines) {
		fmt.Fprintf(&b, "\x1b[%dA", len(lines)-curRow)
		if curCol > 0 {
			fmt.Fprintf(&b, "\x1b[%dC", curCol)
		}
		f.row = curRow
	}
	fmt.Fprint(f.u.Err, b.String())
}

// end clears the block and leaves line in its place (nothing if "").
func (f *frame) end(line string) {
	var b strings.Builder
	f.home(&b)
	if line != "" {
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	f.row = 0
	fmt.Fprint(f.u.Err, b.String())
}

func (f *frame) home(b *strings.Builder) {
	if f.row > 0 {
		fmt.Fprintf(b, "\x1b[%dA", f.row)
	}
	b.WriteString("\r\x1b[J")
}

const (
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
)

func width(s string) int { return utf8.RuneCountInString(s) }

// clip cuts s to n columns, with "…" when it doesn't fit.
func clip(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if width(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func (u *UI) question(prompt string) string {
	return u.Bold(u.Cyan("?")) + " " + u.Bold(prompt)
}

// answered is the line a prompt leaves behind: "✔ prompt answer".
func (u *UI) answered(prompt, answer string) string {
	return u.Green("✔") + " " + prompt + " " + u.Cyan(answer)
}

// --- Select ---

type selectState struct {
	opts   []Option
	depth  []int    // 1 under a header
	group  []string // header of each option
	filter []rune
	cur    int // option under the cursor; -1 when none is selectable
	top    int // first row shown when the list scrolls
}

func newSelect(opts []Option, def int) *selectState {
	s := &selectState{opts: opts, depth: make([]int, len(opts)), group: make([]string, len(opts)), cur: def}
	header := -1
	for i, o := range opts {
		if o.Header {
			header = i
			continue
		}
		if header >= 0 {
			s.depth[i] = 1
			s.group[i] = opts[header].Label
		}
	}
	s.fix(s.rows())
	return s
}

func (s *selectState) selectable(i int) bool {
	return i >= 0 && i < len(s.opts) && !s.opts[i].Header && !s.opts[i].Disabled
}

// matches: the filter is in the label or in the group name.
func (s *selectState) matches(i int) bool {
	if len(s.filter) == 0 {
		return true
	}
	f := strings.ToLower(string(s.filter))
	return strings.Contains(strings.ToLower(s.opts[i].Label), f) ||
		strings.Contains(strings.ToLower(s.group[i]), f)
}

// rows are the options shown with the current filter; a header only if some
// option of its group is shown.
func (s *selectState) rows() []int {
	var out []int
	header := -1
	for i, o := range s.opts {
		if o.Header {
			header = i
			continue
		}
		if !s.matches(i) {
			continue
		}
		if header >= 0 {
			out = append(out, header)
			header = -1
		}
		out = append(out, i)
	}
	return out
}

func (s *selectState) choices(rows []int) []int {
	var out []int
	for _, i := range rows {
		if s.selectable(i) {
			out = append(out, i)
		}
	}
	return out
}

// fix keeps the cursor on a visible, selectable option.
func (s *selectState) fix(rows []int) {
	cs := s.choices(rows)
	for _, i := range cs {
		if i == s.cur {
			return
		}
	}
	s.cur = -1
	if len(cs) > 0 {
		s.cur = cs[0]
	}
}

// move steps the cursor; wrap goes around the ends (arrows), otherwise it
// stops there (page keys).
func (s *selectState) move(rows []int, delta int, wrap bool) {
	cs := s.choices(rows)
	if len(cs) == 0 {
		s.cur = -1
		return
	}
	pos := 0
	for j, i := range cs {
		if i == s.cur {
			pos = j
		}
	}
	pos += delta
	switch {
	case wrap:
		pos = ((pos % len(cs)) + len(cs)) % len(cs)
	case pos < 0:
		pos = 0
	case pos >= len(cs):
		pos = len(cs) - 1
	}
	s.cur = cs[pos]
}

func (s *selectState) render(u *UI, prompt string, back bool) []string {
	w, h := termSize()
	room := w - 1
	rows := s.rows()

	head := u.question(clip(prompt, room-2))
	if len(s.filter) > 0 {
		head += " " + u.Dim("›") + " " + clip(string(s.filter), room-width(prompt)-5)
	}
	lines := []string{head}
	if len(rows) == 0 {
		lines = append(lines, "  "+u.Dim(clip(i18n.T("ui.no_match", string(s.filter)), room-2)))
	}

	// scrolling: show at most `show` rows, keeping the cursor in sight
	show := min(maxRows, h-3)
	if show < 3 {
		show = 3
	}
	first, last := 0, len(rows)
	scrolls := len(rows) > show
	if scrolls {
		at := 0
		for j, i := range rows {
			if i == s.cur {
				at = j
			}
		}
		if at < s.top {
			s.top = at
			if s.top > 0 && s.opts[rows[s.top-1]].Header {
				s.top--
			}
		}
		if at >= s.top+show {
			s.top = at - show + 1
		}
		s.top = max(0, min(s.top, len(rows)-show))
		first, last = s.top, s.top+show
	}

	// hints start in the same column at every depth
	labelW := 0
	for _, i := range rows {
		if !s.opts[i].Header {
			labelW = max(labelW, 2*s.depth[i]+width(s.opts[i].Label))
		}
	}
	labelW = min(labelW, 40)

	for _, i := range rows[first:last] {
		o := s.opts[i]
		if o.Header {
			lines = append(lines, "  "+u.Dim(clip(o.Label, room-2)))
			continue
		}
		indent := strings.Repeat("  ", s.depth[i])
		avail := room - len(indent) - 2
		lw := max(1, min(labelW-len(indent), avail))
		label := clip(o.Label, lw)
		pad := strings.Repeat(" ", max(0, lw-width(label)))
		pointer := "  "
		switch {
		case i == s.cur:
			pointer = u.Cyan("›") + " "
			label = u.Cyan(label)
		case o.Disabled:
			label = u.Dim(label)
		}
		line := indent + pointer + label
		if hint := clip(o.Hint, avail-lw-2); hint != "" {
			line += pad + "  " + u.Dim(hint)
		}
		lines = append(lines, line)
	}

	footer := i18n.T("ui.keys")
	if back {
		footer += i18n.T("ui.keys_back")
	}
	if scrolls {
		cs := s.choices(rows)
		pos := 0
		for j, i := range cs {
			if i == s.cur {
				pos = j + 1
			}
		}
		footer = fmt.Sprintf("%d/%d · ", pos, len(cs)) + footer
	}
	return append(lines, "  "+u.Dim(clip(footer, room-2)))
}

func (u *UI) selectKeys(prompt string, opts []Option, def int, back bool) (int, error) {
	restore, err := rawMode()
	if err != nil {
		return u.selectNumbered(prompt, opts, def)
	}
	defer restore()
	fmt.Fprint(u.Err, hideCursor)
	defer fmt.Fprint(u.Err, showCursor)

	s := newSelect(opts, def)
	f := &frame{u: u}
	for {
		f.draw(s.render(u, prompt, back), -1, 0)
		k, r, err := u.readKey()
		if err != nil {
			f.end("")
			return 0, err
		}
		rows := s.rows()
		page := max(1, min(maxRows, len(rows))-1)
		switch k {
		case keyUp, keyBackTab:
			s.move(rows, -1, true)
		case keyDown, keyTab:
			s.move(rows, 1, true)
		case keyPgUp:
			s.move(rows, -page, false)
		case keyPgDn:
			s.move(rows, page, false)
		case keyHome:
			s.move(rows, -len(rows), false)
		case keyEnd:
			s.move(rows, len(rows), false)
		case keyEnter:
			if s.selectable(s.cur) {
				f.end(u.answered(prompt, s.opts[s.cur].Label))
				return s.cur, nil
			}
		case keyEsc:
			switch {
			case len(s.filter) > 0:
				s.filter = nil
			case back:
				f.end("")
				return 0, ErrBack
			}
		case keyCancel:
			f.end("")
			return 0, context.Canceled
		case keyBackspace:
			if len(s.filter) > 0 {
				s.filter = s.filter[:len(s.filter)-1]
			}
		case keyClear:
			s.filter = nil
		case keyRune:
			s.filter = append(s.filter, r)
		}
		s.fix(s.rows())
	}
}

// --- Confirm ---

func (u *UI) confirmKeys(prompt string, def bool) (bool, error) {
	restore, err := rawMode()
	if err != nil {
		return u.confirmLine(prompt, def)
	}
	defer restore()
	fmt.Fprint(u.Err, hideCursor)
	defer fmt.Fprint(u.Err, showCursor)

	yes, no := i18n.T("ui.yes"), i18n.T("ui.no")
	v := def
	f := &frame{u: u}
	done := func(v bool) (bool, error) {
		a := no
		if v {
			a = yes
		}
		f.end(u.answered(prompt, a))
		return v, nil
	}
	for {
		y, n := u.Dim("○ "+yes), u.Dim("○ "+no)
		if v {
			y = u.Cyan("● " + yes)
		} else {
			n = u.Cyan("● " + no)
		}
		f.draw([]string{u.question(prompt) + " " + u.Dim("›") + " " + y + "  " + n}, -1, 0)
		k, r, err := u.readKey()
		if err != nil {
			f.end("")
			return false, err
		}
		switch k {
		case keyLeft, keyRight, keyUp, keyDown, keyTab, keyBackTab:
			v = !v
		case keyEnter:
			return done(v)
		case keyCancel:
			f.end("")
			return false, context.Canceled
		case keyRune:
			switch r {
			case 'y', 'Y', 's', 'S':
				return done(true)
			case 'n', 'N':
				return done(false)
			}
		}
	}
}

// --- Input ---

func (u *UI) inputKeys(prompt, def string, valid func(string) string, back bool) (string, error) {
	restore, err := rawMode()
	if err != nil {
		return u.inputLine(prompt, def, valid)
	}
	defer restore()
	defer fmt.Fprint(u.Err, showCursor)

	var val []rune
	msg := ""
	f := &frame{u: u}
	for {
		w, _ := termSize()
		room := w - 1
		head := u.question(prompt) + " " + u.Dim("›") + " "
		col := width(prompt) + 5
		var shown string
		if len(val) == 0 {
			shown = u.Dim(clip(def, room-col))
		} else {
			s := string(val)
			if n := room - col - 1; width(s) > n && n > 1 {
				r := []rune(s)
				s = "…" + string(r[len(r)-n+1:])
			}
			shown = s
			col += width(s)
		}
		lines := []string{head + shown}
		switch {
		case msg != "":
			lines = append(lines, "  "+u.Red(clip(msg, room-2)))
		case back:
			lines = append(lines, "  "+u.Dim(clip(i18n.T("ui.keys_input_back"), room-2)))
		}
		f.draw(lines, 0, col)
		k, r, err := u.readKey()
		if err != nil {
			f.end("")
			return "", err
		}
		switch k {
		case keyEnter:
			v := strings.TrimSpace(string(val))
			if v == "" {
				v = def
			}
			if valid != nil {
				if m := valid(v); m != "" {
					msg = m
					continue
				}
			}
			f.end(u.answered(prompt, v))
			return v, nil
		case keyBackspace:
			if len(val) > 0 {
				val = val[:len(val)-1]
			}
		case keyClear:
			val = nil
		case keyTab, keyRight:
			// take the default to edit it
			if len(val) == 0 {
				val = []rune(def)
			}
		case keyEsc:
			if back {
				f.end("")
				return "", ErrBack
			}
			val = nil
		case keyCancel:
			f.end("")
			return "", context.Canceled
		case keyRune:
			val = append(val, r)
		}
		msg = ""
	}
}
