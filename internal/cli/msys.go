package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Git Bash (MSYS2) rewrites every argument that looks like a POSIX path
// before a Windows program sees it: `rapibase api /api/v1/health` arrives as
// "C:/Program Files/Git/api/v1/health", and `--path /api` as
// "C:/Program Files/Git/api". Claude Code runs its shell on Git Bash on
// Windows, so every agent there hit it. A path of the project's API or a
// route of a service is never a Windows path, so the CLI undoes the rewrite
// for those arguments (and only those: `deploy ./dist` or `-F file=@/c/x.png`
// are real files and the rewrite is what makes them work).

// msysRoots are the Windows folders MSYS maps "/" to, from the PATH Git Bash
// hands to programs (".../Git/usr/bin", ".../Git/mingw64/bin") and EXEPATH.
// Tests replace it.
var msysRoots = func() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	var roots []string
	add := func(p string) {
		p = strings.TrimRight(winSlash(p), "/")
		if p == "" {
			return
		}
		for _, r := range roots {
			if strings.EqualFold(r, p) {
				return
			}
		}
		roots = append(roots, p)
	}
	if p := os.Getenv("EXEPATH"); p != "" {
		add(p)
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		d := strings.TrimRight(winSlash(dir), "/")
		low := strings.ToLower(d)
		for _, suffix := range []string{"/usr/bin", "/mingw64/bin", "/mingw32/bin", "/clangarm64/bin", "/ucrt64/bin"} {
			if strings.HasSuffix(low, suffix) {
				add(d[:len(d)-len(suffix)])
				break
			}
		}
	}
	return roots
}

// unMSYS turns "C:/Program Files/Git/api/v1/x" back into "/api/v1/x" when it
// starts with an MSYS root; anything else comes back as it was.
func unMSYS(s string) string {
	if len(s) < 3 || s[1] != ':' || (s[2] != '/' && s[2] != '\\') {
		return s
	}
	norm := winSlash(s)
	for _, root := range msysRoots() {
		if len(norm) >= len(root) && strings.EqualFold(norm[:len(root)], root) {
			rest := norm[len(root):]
			if rest == "" {
				return "/"
			}
			if rest[0] == '/' {
				return rest
			}
		}
	}
	return s
}

// unMSYSAPIPath also covers a root it couldn't find: an API path never has a
// drive letter, so "X:/…/api/v1/x" keeps from its last "/api/" on.
func unMSYSAPIPath(s string) string {
	if u := unMSYS(s); u != s {
		return u
	}
	if len(s) > 2 && s[1] == ':' && (s[2] == '/' || s[2] == '\\') {
		norm := winSlash(s)
		if i := strings.LastIndex(norm, "/api/"); i > 0 {
			return norm[i:]
		}
	}
	return s
}

// winSlash is filepath.ToSlash for Windows paths on any OS (the tests run
// on Linux too).
func winSlash(s string) string { return strings.ReplaceAll(s, `\`, "/") }
