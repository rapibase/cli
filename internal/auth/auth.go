// Package auth is the CLI session: the device flow (RFC 8628, spec §5.1),
// token refresh and the tokens from the environment (spec §5.3).
//
// Each machine registers its own OAuth client ("Rapibase CLI · <hostname>")
// the first time, so it shows up in Connections with its name and can be
// revoked on its own. The client_id is kept after logout: logging in again
// from the same machine reuses the same connection.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/config"
	"github.com/rapibase/cli/internal/i18n"
)

// Token prefixes (spec §5.3).
const (
	PrefixPersonal = "rbpat_"
	PrefixDeploy   = "rbdep_"
)

// Source is where the token of this run comes from.
type Source string

const (
	SourceNone   Source = ""
	SourceLogin  Source = "login"          // credentials.json
	SourceEnv    Source = "RAPIBASE_TOKEN" // personal or OAuth token
	SourceDeploy Source = "RAPIBASE_DEPLOY_TOKEN"
)

// Session implements api.TokenSource for one control plane.
type Session struct {
	Host   string
	Client *api.Client // unauthenticated client for /oauth/*

	mu     sync.Mutex
	source Source
	static string
	sess   *config.Session
}

// Load resolves the token: RAPIBASE_TOKEN > credentials.json. The deploy
// token is only used when allowDeploy is true (deploy and deploys).
func Load(host string, c *api.Client, allowDeploy bool) (*Session, error) {
	s := &Session{Host: host, Client: c}
	if t := strings.TrimSpace(os.Getenv("RAPIBASE_TOKEN")); t != "" {
		s.source, s.static = SourceEnv, t
		return s, nil
	}
	if allowDeploy {
		if t := strings.TrimSpace(os.Getenv("RAPIBASE_DEPLOY_TOKEN")); t != "" {
			s.source, s.static = SourceDeploy, t
			return s, nil
		}
	}
	sess, err := config.LoadSession(host)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if sess != nil && sess.AccessToken != "" {
		s.source, s.sess = SourceLogin, sess
	}
	return s, nil
}

// Source says where the token comes from (SourceNone = not logged in).
func (s *Session) Source() Source { return s.source }

// Email is the email saved at login ("" with env tokens).
func (s *Session) Email() string {
	if s.sess != nil {
		return s.sess.Email
	}
	return ""
}

// LoggedIn reports whether there is any token.
func (s *Session) LoggedIn() bool { return s.source != SourceNone }

// IsDeployToken reports a project deploy token (it only publishes).
func (s *Session) IsDeployToken() bool { return s.source == SourceDeploy }

// Token returns a valid access token, refreshing it a minute before it
// expires.
func (s *Session) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.source {
	case SourceNone:
		return "", cerr.New(cerr.NeedsLogin, i18n.T("auth.needs_login"))
	case SourceEnv, SourceDeploy:
		return s.static, nil
	}
	if time.Until(s.sess.ExpiresAt) < time.Minute {
		if err := s.refreshLocked(ctx); err != nil {
			return "", err
		}
	}
	return s.sess.AccessToken, nil
}

// Refresh is called after a 401.
func (s *Session) Refresh(ctx context.Context) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.source {
	case SourceNone:
		return false, cerr.New(cerr.NeedsLogin, i18n.T("auth.needs_login"))
	case SourceEnv:
		return false, cerr.New(cerr.SessionExpired, i18n.T("auth.env_token_rejected"))
	case SourceDeploy:
		return false, cerr.New(cerr.Forbidden, i18n.T("auth.deploy_token_rejected"))
	}
	if err := s.refreshLocked(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// refreshLocked rotates the refresh token. Another process may have rotated
// it first (two commands at once): then the file already has the new pair.
func (s *Session) refreshLocked(ctx context.Context) error {
	tok, err := s.tokenRequest(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {s.sess.RefreshToken},
		"client_id":     {s.sess.ClientID},
	})
	if err != nil {
		var oe *oauthError
		if errors.As(err, &oe) && oe.Code == "invalid_grant" {
			if fresh, ferr := config.LoadSession(s.Host); ferr == nil && fresh != nil && fresh.RefreshToken != s.sess.RefreshToken {
				s.sess = fresh
				if time.Until(fresh.ExpiresAt) > time.Minute {
					return nil
				}
				return s.refreshLocked(ctx)
			}
			_ = config.ClearSession(s.Host)
			s.source, s.sess = SourceNone, nil
			return cerr.New(cerr.SessionExpired, i18n.T("auth.session_expired"))
		}
		return err
	}
	s.sess.AccessToken = tok.AccessToken
	s.sess.RefreshToken = tok.RefreshToken
	s.sess.ExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return config.SaveSession(s.Host, s.sess)
}

// --- OAuth plumbing ---

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type oauthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
	Status      int    `json:"-"`
}

func (e *oauthError) Error() string {
	if e.Description != "" {
		return e.Code + ": " + e.Description
	}
	return e.Code
}

func (s *Session) tokenRequest(ctx context.Context, form url.Values) (*tokenResponse, error) {
	res, err := s.Client.Raw(ctx, "POST", "/oauth/token", api.FormBody(form), false)
	if err != nil {
		return nil, err
	}
	if res.Status != 200 {
		oe := &oauthError{Status: res.Status}
		_ = json.Unmarshal(res.Body, oe)
		if oe.Code == "" {
			return nil, api.ErrorFrom(res)
		}
		return nil, oe
	}
	var tok tokenResponse
	if err := json.Unmarshal(res.Body, &tok); err != nil || tok.AccessToken == "" {
		return nil, cerr.New(cerr.ServerError, i18n.T("api.bad_response", res.Status))
	}
	return &tok, nil
}

// Hostname is the machine name shown in the browser and in Connections.
func Hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	if i := strings.IndexByte(h, '.'); i > 0 {
		h = h[:i]
	}
	return h
}

// OSName is a friendly OS name for the approval screen.
func OSName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	}
	return runtime.GOOS
}

// ensureClient registers this machine's OAuth client once.
func (s *Session) ensureClient(ctx context.Context) (string, error) {
	hc := config.LoadHost(s.Host)
	name := "Rapibase CLI · " + Hostname()
	if hc.ClientID != "" && hc.ClientName == name {
		return hc.ClientID, nil
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	body := map[string]any{
		"client_name":                name,
		"grant_types":                []string{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		"token_endpoint_auth_method": "none",
	}
	res, err := s.Client.Raw(ctx, "POST", "/oauth/register", api.JSONBody(body), false)
	if err != nil {
		return "", err
	}
	if res.Status != 201 && res.Status != 200 {
		oe := &oauthError{Status: res.Status}
		_ = json.Unmarshal(res.Body, oe)
		if res.Status == 429 {
			return "", cerr.New(cerr.RateLimited, i18n.T("auth.register_rate_limited"))
		}
		return "", cerr.New(cerr.ServerError, i18n.T("auth.register_failed", oe.Error()))
	}
	if err := json.Unmarshal(res.Body, &out); err != nil || out.ClientID == "" {
		return "", cerr.New(cerr.ServerError, i18n.T("api.bad_response", res.Status))
	}
	hc.ClientID, hc.ClientName = out.ClientID, name
	if err := config.SaveHost(s.Host, hc); err != nil {
		return "", err
	}
	return out.ClientID, nil
}
