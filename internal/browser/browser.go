// Package browser opens URLs in the person's browser, only on a local
// desktop (spec §4.1): never over SSH, in CI or with --no-browser. Then the
// CLI prints the URL instead.
package browser

import (
	"os"
	"os/exec"
	"runtime"
)

// Allowed reports whether opening a browser makes sense here.
func Allowed(noBrowser, ci bool) bool {
	if noBrowser || ci {
		return false
	}
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CLIENT") != "" {
		return false
	}
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return false
	}
	return true
}

// Open tries to open url; it returns false if it could not.
func Open(url string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// rundll32 avoids cmd's quoting problems with & in URLs
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return false
	}
	go func() { _ = cmd.Wait() }()
	return true
}
