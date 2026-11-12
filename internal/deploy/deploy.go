// Package deploy packs what the CLI publishes (spec §8.1 and §8.4): the
// static site as a zip and services as a tgz, with the server limits checked
// before uploading and secrets never included.
package deploy

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"debug/elf"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Server limits for the static site (deploys.go in the control plane).
const (
	MaxZipBytes       = 50 << 20
	MaxExtractedBytes = 200 << 20
	MaxFiles          = 3000
	// Services: the archive limit of services.go.
	MaxServiceBytes = 100 << 20
)

// Detection is the result of looking for the build folder.
type Detection struct {
	Dir       string // relative to the project folder
	Framework string // vite, next, astro, cra, sveltekit, nuxt, static
	Build     string // suggested build command when Dir is missing
	SSR       bool   // Next.js with a server: not a static site
}

// DetectDir guesses the folder to publish from package.json (spec §8.1).
func DetectDir(root string) Detection {
	pkg := readPackage(root)
	has := func(name string) bool {
		_, a := pkg.Deps[name]
		_, b := pkg.DevDeps[name]
		return a || b
	}
	build := "npm run build"
	if _, ok := pkg.Scripts["build"]; !ok {
		build = ""
	}
	switch {
	case has("next"):
		if nextStaticExport(root) {
			return Detection{Dir: "out", Framework: "next", Build: build}
		}
		return Detection{Framework: "next", SSR: true, Build: build}
	case has("nuxt"):
		return Detection{Dir: filepath.Join(".output", "public"), Framework: "nuxt", Build: "npx nuxi generate"}
	case has("@sveltejs/kit"):
		return Detection{Dir: "build", Framework: "sveltekit", Build: build}
	case has("astro"):
		return Detection{Dir: "dist", Framework: "astro", Build: build}
	case has("vite"):
		return Detection{Dir: "dist", Framework: "vite", Build: build}
	case has("react-scripts"):
		return Detection{Dir: "build", Framework: "cra", Build: build}
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err == nil {
		return Detection{Dir: ".", Framework: "static"}
	}
	for _, d := range []string{"dist", "build", "out", "public"} {
		if _, err := os.Stat(filepath.Join(root, d, "index.html")); err == nil {
			return Detection{Dir: d, Framework: "static"}
		}
	}
	return Detection{Build: build}
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
	Deps    map[string]string `json:"dependencies"`
	DevDeps map[string]string `json:"devDependencies"`
}

func readPackage(root string) packageJSON {
	var p packageJSON
	if b, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
		_ = json.Unmarshal(b, &p)
	}
	return p
}

var nextExportRe = regexp.MustCompile(`output\s*:\s*['"]export['"]`)

func nextStaticExport(root string) bool {
	for _, f := range []string{"next.config.js", "next.config.mjs", "next.config.ts", "next.config.cjs"} {
		if b, err := os.ReadFile(filepath.Join(root, f)); err == nil && nextExportRe.Match(b) {
			return true
		}
	}
	return false
}

// --- what never gets published ---

// forbidden reports files that must never be in a public site or uploaded
// service: env files, keys, VCS and dependency folders.
func forbidden(rel string, isDir bool) bool {
	base := path.Base(rel)
	if isDir {
		return base == ".git" || base == "node_modules"
	}
	switch {
	case base == ".env" || strings.HasPrefix(base, ".env.") && base != ".env.example":
		return true
	case strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".p12") || strings.HasSuffix(base, ".pfx"):
		return true
	case base == "id_rsa" || base == "id_ed25519" || base == ".npmrc" || base == ".netrc":
		return true
	}
	return false
}

// Exclusion matches --exclude patterns (path.Match on the relative path
// and on the base name; "dir/" excludes a folder).
type Exclusion []string

func (x Exclusion) match(rel string, isDir bool) bool {
	for _, p := range x {
		p = filepath.ToSlash(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/") {
			if isDir && (rel == strings.TrimSuffix(p, "/") || path.Base(rel) == strings.TrimSuffix(p, "/")) {
				return true
			}
			continue
		}
		if ok, _ := path.Match(p, rel); ok {
			return true
		}
		if ok, _ := path.Match(p, path.Base(rel)); ok {
			return true
		}
	}
	return false
}

// Entry is a file to pack.
type Entry struct {
	Rel  string // slash-separated path inside the archive
	Abs  string
	Size int64
	Mode fs.FileMode
}

// Scan walks dir and returns the files to publish, or the list of
// forbidden files found (which the caller reports and aborts on).
func Scan(dir string, excl Exclusion, skipDeps bool) (files []Entry, blocked []string, err error) {
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if excl.match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".rapibase" { // the folder link: ours, not secret, never published
				return filepath.SkipDir
			}
			if skipDeps && (base == "node_modules" || base == ".venv" || base == "venv" || base == "__pycache__" || base == ".git" || base == ".rapibase") {
				return filepath.SkipDir
			}
			if forbidden(rel, true) {
				blocked = append(blocked, rel+"/")
				return filepath.SkipDir
			}
			return nil
		}
		if forbidden(rel, false) {
			blocked = append(blocked, rel)
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // symlinks are skipped: they could point outside
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, Entry{Rel: rel, Abs: p, Size: info.Size(), Mode: info.Mode()})
		return nil
	})
	sort.Strings(blocked)
	return files, blocked, err
}

// Stats summarizes an archive.
type Stats struct {
	Files     int   `json:"files"`
	Bytes     int64 `json:"bytes"`
	ZipBytes  int64 `json:"archive_bytes"`
	HasIndex  bool  `json:"-"`
	Truncated bool  `json:"-"`
}

// ErrLimit is a server limit the archive would break.
type ErrLimit struct{ What, Got, Max string }

func (e *ErrLimit) Error() string { return fmt.Sprintf("%s: %s > %s", e.What, e.Got, e.Max) }

// ZipSite builds the site zip in memory, checking the server limits.
func ZipSite(files []Entry) ([]byte, Stats, error) {
	var st Stats
	st.Files = len(files)
	if st.Files > MaxFiles {
		return nil, st, &ErrLimit{"files", fmt.Sprint(st.Files), fmt.Sprint(MaxFiles)}
	}
	for _, f := range files {
		st.Bytes += f.Size
		if f.Rel == "index.html" {
			st.HasIndex = true
		}
	}
	if st.Bytes > MaxExtractedBytes {
		return nil, st, &ErrLimit{"size", HumanBytes(st.Bytes), HumanBytes(MaxExtractedBytes)}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		h := &zip.FileHeader{Name: f.Rel, Method: zip.Deflate}
		h.SetMode(0o644)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return nil, st, err
		}
		if err := copyFile(w, f.Abs); err != nil {
			return nil, st, err
		}
		if buf.Len() > MaxZipBytes {
			return nil, st, &ErrLimit{"zip", "> " + HumanBytes(MaxZipBytes), HumanBytes(MaxZipBytes)}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, st, err
	}
	st.ZipBytes = int64(buf.Len())
	if st.ZipBytes > MaxZipBytes {
		return nil, st, &ErrLimit{"zip", HumanBytes(st.ZipBytes), HumanBytes(MaxZipBytes)}
	}
	return buf.Bytes(), st, nil
}

// TarService packs a service folder as tgz. Executable bits are kept so a
// binary inside the folder still runs.
func TarService(files []Entry) ([]byte, Stats, error) {
	var st Stats
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range files {
		mode := int64(0o644)
		if f.Mode&0o111 != 0 || isELF(f.Abs) {
			mode = 0o755
		}
		if err := tw.WriteHeader(&tar.Header{Name: f.Rel, Mode: mode, Size: f.Size, Typeflag: tar.TypeReg}); err != nil {
			return nil, st, err
		}
		if err := copyFile(tw, f.Abs); err != nil {
			return nil, st, err
		}
		st.Files++
		st.Bytes += f.Size
		if buf.Len() > MaxServiceBytes {
			return nil, st, &ErrLimit{"archive", "> " + HumanBytes(MaxServiceBytes), HumanBytes(MaxServiceBytes)}
		}
	}
	if err := tw.Close(); err != nil {
		return nil, st, err
	}
	if err := gz.Close(); err != nil {
		return nil, st, err
	}
	st.ZipBytes = int64(buf.Len())
	return buf.Bytes(), st, nil
}

func copyFile(w io.Writer, p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// --- binaries for services ---

// ErrNotLinuxAMD64 explains what a binary is when it is not a Linux x86-64
// ELF (the server only runs those).
type ErrNotLinuxAMD64 struct{ Kind string }

func (e *ErrNotLinuxAMD64) Error() string { return e.Kind }

// CheckBinary verifies a Linux x86-64 ELF and says what it is otherwise.
func CheckBinary(p string) error {
	f, err := elf.Open(p)
	if err != nil {
		head := make([]byte, 4)
		if fh, err := os.Open(p); err == nil {
			_, _ = io.ReadFull(fh, head)
			fh.Close()
		}
		switch {
		case head[0] == 'M' && head[1] == 'Z':
			return &ErrNotLinuxAMD64{"windows"}
		case bytes.Equal(head, []byte{0xcf, 0xfa, 0xed, 0xfe}) || bytes.Equal(head, []byte{0xca, 0xfe, 0xba, 0xbe}):
			return &ErrNotLinuxAMD64{"macos"}
		}
		return &ErrNotLinuxAMD64{"unknown"}
	}
	defer f.Close()
	if f.Machine != elf.EM_X86_64 {
		return &ErrNotLinuxAMD64{"linux-" + strings.ToLower(strings.TrimPrefix(f.Machine.String(), "EM_"))}
	}
	return nil
}

func isELF(p string) bool {
	fh, err := os.Open(p)
	if err != nil {
		return false
	}
	defer fh.Close()
	head := make([]byte, 4)
	if _, err := io.ReadFull(fh, head); err != nil {
		return false
	}
	return bytes.Equal(head, []byte{0x7f, 'E', 'L', 'F'})
}

// --- git rules (spec §8.1) ---

// GitDirty reports uncommitted changes in the repo of dir (false outside a
// repo or without git).
func GitDirty(dir string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

// ActionsWorkflow returns the path of the Rapibase deploy workflow if the
// repo has one: then publishing happens with git push.
func ActionsWorkflow(repoRoot string) string {
	if repoRoot == "" {
		return ""
	}
	for _, n := range []string{"rapibase-deploy.yml", "rapibase-deploy.yaml"} {
		p := filepath.Join(repoRoot, ".github", "workflows", n)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// HumanBytes formats a size.
func HumanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// IsLimit reports a server limit error.
func IsLimit(err error) (*ErrLimit, bool) {
	var l *ErrLimit
	return l, errors.As(err, &l)
}
