// Package config owns the CLI's files outside the repo (spec §4.6):
//
//	<config>/rapibase/credentials.json  tokens, pending login (0600)
//	<config>/rapibase/config.json       client_id per host, version check
//
// <config> is os.UserConfigDir(): %AppData% on Windows (per-user ACL),
// ~/Library/Application Support on macOS and ~/.config on Linux. Both files
// are keyed by control plane URL, so RAPIBASE_API_URL pointed at a local
// control plane never mixes its session with the real one.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultAPIURL is the control plane.
const DefaultAPIURL = "https://rapibase.com"

// APIURL is the control plane URL (RAPIBASE_API_URL overrides it).
func APIURL() string {
	if v := strings.TrimSpace(os.Getenv("RAPIBASE_API_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return DefaultAPIURL
}

// Dir is <config>/rapibase.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "rapibase"), nil
}

// --- credentials.json ---

// Session is a logged-in CLI session for one control plane.
type Session struct {
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Email        string    `json:"email,omitempty"`
}

// Pending is a device login waiting for the person to approve it. It is
// kept so that running the same command again keeps polling the same code
// instead of showing a new one (spec §2.2).
type Pending struct {
	ClientID                string    `json:"client_id"`
	DeviceCode              string    `json:"device_code"`
	UserCode                string    `json:"user_code"`
	VerificationURI         string    `json:"verification_uri"`
	VerificationURIComplete string    `json:"verification_uri_complete"`
	Interval                int       `json:"interval"`
	ExpiresAt               time.Time `json:"expires_at"`
	Ref                     string    `json:"ref,omitempty"`
}

type hostCreds struct {
	Session *Session `json:"session,omitempty"`
	Pending *Pending `json:"pending,omitempty"`
}

type credentialsFile struct {
	Version int                   `json:"version"`
	Hosts   map[string]*hostCreds `json:"hosts"`
}

func credentialsPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "credentials.json"), nil
}

func loadCredentials() (*credentialsFile, error) {
	f := &credentialsFile{Version: 1, Hosts: map[string]*hostCreds{}}
	p, err := credentialsPath()
	if err != nil {
		return f, err
	}
	if err := readJSON(p, f); err != nil {
		return f, err
	}
	if f.Hosts == nil {
		f.Hosts = map[string]*hostCreds{}
	}
	return f, nil
}

func (f *credentialsFile) host(h string) *hostCreds {
	c := f.Hosts[h]
	if c == nil {
		c = &hostCreds{}
		f.Hosts[h] = c
	}
	return c
}

func (f *credentialsFile) save() error {
	for h, c := range f.Hosts {
		if c.Session == nil && c.Pending == nil {
			delete(f.Hosts, h)
		}
	}
	p, err := credentialsPath()
	if err != nil {
		return err
	}
	return writeJSON(p, f, 0o600)
}

// LoadSession returns the session for a control plane (nil if none).
func LoadSession(host string) (*Session, error) {
	f, err := loadCredentials()
	if err != nil {
		return nil, err
	}
	if c := f.Hosts[host]; c != nil {
		return c.Session, nil
	}
	return nil, nil
}

// SaveSession stores the session and clears any pending login.
func SaveSession(host string, s *Session) error {
	f, err := loadCredentials()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	c := f.host(host)
	c.Session, c.Pending = s, nil
	return f.save()
}

// ClearSession removes the session (logout).
func ClearSession(host string) error {
	f, err := loadCredentials()
	if err != nil {
		return nil
	}
	if c := f.Hosts[host]; c != nil {
		c.Session, c.Pending = nil, nil
	}
	return f.save()
}

// LoadPending returns the pending device login (nil if none or expired).
func LoadPending(host string) (*Pending, error) {
	f, err := loadCredentials()
	if err != nil {
		return nil, err
	}
	c := f.Hosts[host]
	if c == nil || c.Pending == nil || time.Now().After(c.Pending.ExpiresAt) {
		return nil, nil
	}
	return c.Pending, nil
}

// SavePending stores (or clears, with nil) the pending device login.
func SavePending(host string, p *Pending) error {
	f, err := loadCredentials()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	f.host(host).Pending = p
	return f.save()
}

// --- config.json ---

// HostConfig is what the CLI remembers per control plane that is not a
// secret: the OAuth client of this machine and the last version check.
type HostConfig struct {
	ClientID     string    `json:"client_id,omitempty"`
	ClientName   string    `json:"client_name,omitempty"`
	CheckedAt    time.Time `json:"version_checked_at,omitempty"`
	Latest       string    `json:"latest,omitempty"`
	MinSupported string    `json:"min_supported,omitempty"`
}

type configFile struct {
	Version int                    `json:"version"`
	Hosts   map[string]*HostConfig `json:"hosts"`
}

func configPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// LoadHost returns the host config (empty if none).
func LoadHost(host string) *HostConfig {
	f := &configFile{}
	if p, err := configPath(); err == nil {
		_ = readJSON(p, f)
	}
	if h := f.Hosts[host]; h != nil {
		return h
	}
	return &HostConfig{}
}

// SaveHost stores the host config.
func SaveHost(host string, h *HostConfig) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	f := &configFile{Version: 1}
	_ = readJSON(p, f)
	if f.Hosts == nil {
		f.Hosts = map[string]*HostConfig{}
	}
	f.Version = 1
	f.Hosts[host] = h
	return writeJSON(p, f, 0o600)
}

// --- helpers ---

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// writeJSON writes atomically (temp file + rename) with the given mode.
func writeJSON(path string, v any, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), mode); err != nil {
		return err
	}
	_ = os.Chmod(tmp, mode)
	return os.Rename(tmp, path)
}
