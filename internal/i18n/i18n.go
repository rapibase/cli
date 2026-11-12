// Package i18n picks the CLI language (spec §4.4) and translates messages.
//
// Order: --lang > RAPIBASE_LANG > LC_ALL / LC_MESSAGES / LANG > the Windows
// user locale > en. Messages live in messages.go as {en, es, pt} triples.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

// Lang is a supported language.
type Lang string

const (
	EN Lang = "en"
	ES Lang = "es"
	PT Lang = "pt"
)

var current = EN

// Current is the active language.
func Current() Lang { return current }

// Set fixes the active language.
func Set(l Lang) { current = l }

// Parse turns "es", "es_AR.UTF-8", "pt-BR"… into a supported language.
func Parse(s string) (Lang, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "c" || s == "posix" {
		return "", false
	}
	switch {
	case strings.HasPrefix(s, "es"):
		return ES, true
	case strings.HasPrefix(s, "pt"):
		return PT, true
	case strings.HasPrefix(s, "en"):
		return EN, true
	}
	return "", false
}

// Detect resolves the language from the --lang flag value and the
// environment, and makes it current.
func Detect(flag string) Lang {
	for _, v := range []string{flag, os.Getenv("RAPIBASE_LANG"), os.Getenv("LC_ALL"), os.Getenv("LC_MESSAGES"), os.Getenv("LANG")} {
		if l, ok := Parse(v); ok {
			current = l
			return l
		}
	}
	if l, ok := Parse(systemLocale()); ok {
		current = l
		return l
	}
	current = EN
	return EN
}

// T translates a message key and formats it with args (fmt.Sprintf).
// A missing key returns the key itself, which the tests catch.
func T(key string, args ...any) string {
	m, ok := messages[key]
	if !ok {
		if len(args) > 0 {
			return key + " " + fmt.Sprint(args...)
		}
		return key
	}
	s := m[0]
	switch current {
	case ES:
		if m[1] != "" {
			s = m[1]
		}
	case PT:
		if m[2] != "" {
			s = m[2]
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// Has reports whether a key exists (tests).
func Has(key string) bool { _, ok := messages[key]; return ok }

// Keys lists every key (tests).
func Keys() []string {
	out := make([]string, 0, len(messages))
	for k := range messages {
		out = append(out, k)
	}
	return out
}

// Triple returns the three translations of a key (tests).
func Triple(key string) [3]string { return messages[key] }

// AcceptLanguage is the header value for the control plane, so server
// errors come back in the same language.
func AcceptLanguage() string { return string(current) }
