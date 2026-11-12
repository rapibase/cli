//go:build !windows

package i18n

// systemLocale: on Unix the locale comes from LC_ALL / LC_MESSAGES / LANG,
// already read by Detect.
func systemLocale() string { return "" }
