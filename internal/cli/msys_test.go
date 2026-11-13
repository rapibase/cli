package cli

import "testing"

func TestUnMSYS(t *testing.T) {
	old := msysRoots
	defer func() { msysRoots = old }()
	msysRoots = func() []string { return []string{"C:/Program Files/Git"} }

	for in, want := range map[string]string{
		"C:/Program Files/Git/api/v1/health":  "/api/v1/health",
		`C:\Program Files\Git\api\v1\health`:  "/api/v1/health",
		"c:/program files/git/functions/hola": "/functions/hola",
		"C:/Program Files/Git/":               "/",
		"C:/Program Files/Git":                "/",
		"C:/Program Files/Github/api":         "C:/Program Files/Github/api", // another folder
		"C:/Users/ana/dist":                   "C:/Users/ana/dist",
		"/api/v1/health":                      "/api/v1/health",
		"./dist":                              "./dist",
		"":                                    "",
	} {
		if got := unMSYS(in); got != want {
			t.Errorf("unMSYS(%q) = %q, want %q", in, got, want)
		}
	}

	// a root it couldn't find (a portable Git, no EXEPATH): an API path has
	// no drive letter, so it keeps from /api/ on
	msysRoots = func() []string { return nil }
	for in, want := range map[string]string{
		"D:/tools/msys64/api/v1/rest/notes": "/api/v1/rest/notes",
		"/api/v1/rest/notes":                "/api/v1/rest/notes",
		"D:/tools/msys64/functions/x":       "D:/tools/msys64/functions/x",
	} {
		if got := unMSYSAPIPath(in); got != want {
			t.Errorf("unMSYSAPIPath(%q) = %q, want %q", in, got, want)
		}
	}
}
