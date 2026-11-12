//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

// systemLocale reads the Windows user locale ("es-AR", "pt-BR"…).
func systemLocale() string {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
	if proc.Find() != nil {
		return ""
	}
	buf := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
