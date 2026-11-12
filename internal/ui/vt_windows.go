//go:build windows

package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT turns on ANSI escape processing in the Windows console (cmd and
// PowerShell 5.1 need it; Windows Terminal already has it). If it can't,
// the CLI prints without colors.
func enableVT() bool {
	ok := true
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		h := windows.Handle(f.Fd())
		var mode uint32
		if windows.GetConsoleMode(h, &mode) != nil {
			ok = false
			continue
		}
		if windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil {
			ok = false
		}
	}
	return ok
}
