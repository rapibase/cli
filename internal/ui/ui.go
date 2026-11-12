// Package ui is the CLI's output and prompts (spec §4.1 and §4.2).
//
// Human mode: short lines with ✔/✖, colors and spinners only on a TTY,
// NO_COLOR respected. JSON mode: nothing on stdout until the single final
// object; warnings still go to stderr. Prompts exist only in interactive mode
// (stdin and stdout are TTYs, no CI, no --json, no --yes): every question has
// a flag, and without a TTY a missing value is an error, never a prompt.
package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/rapibase/cli/internal/i18n"
)

// UI is the output channel of one CLI run.
type UI struct {
	JSON        bool
	Interactive bool
	Color       bool
	// StderrTTY: spinners and in-place lines only when stderr is a terminal.
	StderrTTY bool
	Out       io.Writer
	Err       io.Writer
	in        *bufio.Reader
}

// New builds the UI from the global flags and the environment.
func New(jsonOut, yes bool) *UI {
	stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
	stdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))
	stderrTTY := term.IsTerminal(int(os.Stderr.Fd()))
	u := &UI{
		JSON:      jsonOut,
		StderrTTY: stderrTTY,
		Out:       os.Stdout,
		Err:       os.Stderr,
		in:        bufio.NewReader(os.Stdin),
	}
	u.Interactive = stdinTTY && stdoutTTY && !IsCI() && !jsonOut && !yes
	u.Color = stdoutTTY && stderrTTY && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && enableVT()
	return u
}

// IsCI reports a CI environment (CI=true/1, or the usual CI variables).
func IsCI() bool {
	switch strings.ToLower(os.Getenv("CI")) {
	case "1", "true", "yes":
		return true
	}
	for _, k := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE", "CIRCLECI", "JENKINS_URL", "TF_BUILD"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// --- colors ---

func (u *UI) paint(code, s string) string {
	if !u.Color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (u *UI) Bold(s string) string  { return u.paint("1", s) }
func (u *UI) Dim(s string) string   { return u.paint("2", s) }
func (u *UI) Green(s string) string { return u.paint("32", s) }
func (u *UI) Red(s string) string   { return u.paint("31", s) }
func (u *UI) Cyan(s string) string  { return u.paint("36", s) }
func (u *UI) Yellow(s string) string {
	return u.paint("33", s)
}

// --- lines ---

// Println writes a human line to stdout (nothing in JSON mode).
func (u *UI) Println(a ...any) {
	if !u.JSON {
		fmt.Fprintln(u.Out, a...)
	}
}

// Printf writes formatted human output to stdout (nothing in JSON mode).
func (u *UI) Printf(format string, a ...any) {
	if !u.JSON {
		fmt.Fprintf(u.Out, format, a...)
	}
}

// Success prints "✔ msg".
func (u *UI) Success(msg string) {
	if !u.JSON {
		fmt.Fprintln(u.Out, u.Green("✔")+" "+msg)
	}
}

// Note prints an indented secondary line.
func (u *UI) Note(msg string) {
	if !u.JSON {
		fmt.Fprintln(u.Out, "  "+u.Dim(msg))
	}
}

// Warn prints "! msg" on stderr, also in JSON mode: warnings never go to
// stdout, so the JSON stays parseable.
func (u *UI) Warn(msg string) {
	fmt.Fprintln(u.Err, u.Yellow("!")+" "+msg)
}

// Progress prints a progress line on stderr (human mode only).
func (u *UI) Progress(msg string) {
	if !u.JSON {
		fmt.Fprintln(u.Err, u.Dim(msg))
	}
}

// Target prints the "→ project (org)" line every changing command starts
// with (spec §4.5.5), on stderr so it never mixes with data.
func (u *UI) Target(slug, org string) {
	if u.JSON {
		return
	}
	line := "→ " + u.Bold(slug)
	if org != "" {
		line += " " + u.Dim("("+org+")")
	}
	fmt.Fprintln(u.Err, line)
}

// --- JSON ---

// Result writes the final JSON object for a successful command.
func (u *UI) Result(data any) {
	writeJSON(u.Out, map[string]any{"ok": true, "data": data})
}

// ErrorJSON writes the final JSON object for a failed command.
func (u *UI) ErrorJSON(code, msg string, extra map[string]any) {
	obj := map[string]any{"code": code, "message": msg}
	for k, v := range extra {
		obj[k] = v
	}
	writeJSON(u.Out, map[string]any{"ok": false, "error": obj})
}

func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// PrettyJSON prints any value as indented JSON on stdout (human mode).
func (u *UI) PrettyJSON(v any) {
	enc := json.NewEncoder(u.Out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// --- prompts (interactive only) ---

// Option is a choice in Select. Header options are group titles, not
// selectable.
type Option struct {
	Label  string
	Hint   string
	Header bool
}

// Select shows a numbered list and returns the index of the chosen option.
func (u *UI) Select(prompt string, opts []Option, def int) (int, error) {
	fmt.Fprintln(u.Err, u.Bold("? ")+prompt)
	index := map[int]int{}
	n := 0
	for i, o := range opts {
		if o.Header {
			fmt.Fprintln(u.Err, "  "+u.Dim(o.Label))
			continue
		}
		n++
		index[n] = i
		mark := " "
		if i == def {
			mark = u.Cyan("›")
		}
		line := fmt.Sprintf(" %s %2d) %s", mark, n, o.Label)
		if o.Hint != "" {
			line += "  " + u.Dim(o.Hint)
		}
		fmt.Fprintln(u.Err, line)
	}
	defNum := 0
	for num, i := range index {
		if i == def {
			defNum = num
		}
	}
	for {
		ask := i18n.T("ui.choose", n)
		if defNum > 0 {
			ask += " [" + strconv.Itoa(defNum) + "]"
		}
		fmt.Fprint(u.Err, "  "+ask+": ")
		line, err := u.readLine()
		if err != nil {
			return 0, err
		}
		if line == "" && defNum > 0 {
			return index[defNum], nil
		}
		if k, err := strconv.Atoi(line); err == nil {
			if i, ok := index[k]; ok {
				return i, nil
			}
		}
		fmt.Fprintln(u.Err, "  "+u.Red(i18n.T("ui.invalid_choice")))
	}
}

// Input asks for a line of text, with a default.
func (u *UI) Input(prompt, def string, valid func(string) string) (string, error) {
	for {
		q := u.Bold("? ") + prompt
		if def != "" {
			q += " " + u.Dim("("+def+")")
		}
		fmt.Fprint(u.Err, q+" › ")
		line, err := u.readLine()
		if err != nil {
			return "", err
		}
		if line == "" {
			line = def
		}
		if valid != nil {
			if msg := valid(line); msg != "" {
				fmt.Fprintln(u.Err, "  "+u.Red(msg))
				continue
			}
		}
		return line, nil
	}
}

// Confirm asks a yes/no question.
func (u *UI) Confirm(prompt string, def bool) (bool, error) {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	for {
		fmt.Fprint(u.Err, u.Bold("? ")+prompt+" "+u.Dim("("+hint+")")+" ")
		line, err := u.readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(line) {
		case "":
			return def, nil
		case "y", "yes", "s", "si", "sí", "sim":
			return true, nil
		case "n", "no", "não", "nao":
			return false, nil
		}
	}
}

func (u *UI) readLine() (string, error) {
	line, err := u.in.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// --- spinner ---

// Spinner shows an animated line on stderr while work runs (TTY human mode
// only) and returns the function that stops it.
func (u *UI) Spinner(msg string) (stop func()) {
	if u.JSON || !u.StderrTTY {
		if !u.JSON {
			fmt.Fprintln(u.Err, u.Dim(msg))
		}
		return func() {}
	}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(90 * time.Millisecond)
		defer t.Stop()
		for i := 0; ; i++ {
			fmt.Fprintf(u.Err, "\r%s %s", u.Cyan(frames[i%len(frames)]), msg)
			select {
			case <-done:
				fmt.Fprint(u.Err, "\r\x1b[2K")
				return
			case <-t.C:
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			wg.Wait()
		})
	}
}
