// Package build holds values stamped at release time (GoReleaser ldflags).
package build

// Version is the CLI version. Releases set it with
// -ldflags "-X github.com/rapibase/cli/internal/build.Version=0.1.0".
var Version = "0.0.0-dev"

// Dev reports whether this is a local build without a release version.
func Dev() bool { return Version == "0.0.0-dev" }
