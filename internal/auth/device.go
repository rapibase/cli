package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"github.com/rapibase/cli/internal/api"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/config"
	"github.com/rapibase/cli/internal/i18n"
)

// PollStatus is the outcome of waiting for an approval.
type PollStatus int

const (
	Approved PollStatus = iota
	StillPending
	Denied
	Expired
)

// StartDevice returns the pending login if there is a valid one (same
// referral code), or asks the control plane for a new code.
func (s *Session) StartDevice(ctx context.Context, ref string) (*config.Pending, bool, error) {
	if p, _ := config.LoadPending(s.Host); p != nil && p.Ref == ref && time.Until(p.ExpiresAt) > 20*time.Second {
		return p, false, nil
	}
	clientID, err := s.ensureClient(ctx)
	if err != nil {
		return nil, false, err
	}
	form := url.Values{"client_id": {clientID}, "hostname": {Hostname()}, "os": {OSName()}}
	if ref != "" {
		form.Set("ref", ref)
	}
	res, err := s.Client.Raw(ctx, "POST", "/oauth/device_authorization", api.FormBody(form), false)
	if err != nil {
		return nil, false, err
	}
	if res.Status != 200 {
		oe := &oauthError{Status: res.Status}
		_ = json.Unmarshal(res.Body, oe)
		switch {
		case res.Status == 429:
			return nil, false, cerr.New(cerr.RateLimited, i18n.T("auth.device_rate_limited")).With("retry_after", 600)
		case oe.Code == "invalid_client" || oe.Code == "unauthorized_client":
			// the client was deleted on the server: register again next time
			hc := config.LoadHost(s.Host)
			hc.ClientID, hc.ClientName = "", ""
			_ = config.SaveHost(s.Host, hc)
			return nil, false, cerr.New(cerr.ServerError, i18n.T("auth.client_reset"))
		}
		return nil, false, cerr.New(cerr.ServerError, i18n.T("auth.device_failed", oe.Error()))
	}
	var out struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(res.Body, &out); err != nil || out.DeviceCode == "" {
		return nil, false, cerr.New(cerr.ServerError, i18n.T("api.bad_response", res.Status))
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	p := &config.Pending{
		ClientID:                clientID,
		DeviceCode:              out.DeviceCode,
		UserCode:                out.UserCode,
		VerificationURI:         out.VerificationURI,
		VerificationURIComplete: out.VerificationURIComplete,
		Interval:                out.Interval,
		ExpiresAt:               time.Now().Add(time.Duration(out.ExpiresIn) * time.Second),
		Ref:                     ref,
	}
	if err := config.SavePending(s.Host, p); err != nil {
		return nil, false, err
	}
	return p, true, nil
}

// Poll asks the token endpoint until the login is approved, denied or
// expired, or until maxWait passes (then StillPending). A zero maxWait
// checks once.
func (s *Session) Poll(ctx context.Context, p *config.Pending, maxWait time.Duration) (PollStatus, error) {
	deadline := time.Now().Add(maxWait)
	interval := time.Duration(p.Interval) * time.Second
	for {
		tok, err := s.tokenRequest(ctx, url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {p.DeviceCode},
			"client_id":   {p.ClientID},
		})
		if err == nil {
			sess := &config.Session{
				ClientID:     p.ClientID,
				AccessToken:  tok.AccessToken,
				RefreshToken: tok.RefreshToken,
				ExpiresAt:    time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
			}
			if err := config.SaveSession(s.Host, sess); err != nil {
				return 0, err
			}
			s.mu.Lock()
			s.source, s.sess, s.static = SourceLogin, sess, ""
			s.mu.Unlock()
			return Approved, nil
		}
		var oe *oauthError
		if !errors.As(err, &oe) {
			return 0, err
		}
		switch oe.Code {
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "access_denied":
			_ = config.SavePending(s.Host, nil)
			return Denied, nil
		default: // expired_token, invalid_grant (used or unknown)
			_ = config.SavePending(s.Host, nil)
			return Expired, nil
		}
		if time.Now().Add(interval).After(deadline) || time.Now().After(p.ExpiresAt) {
			if time.Now().After(p.ExpiresAt) {
				_ = config.SavePending(s.Host, nil)
				return Expired, nil
			}
			return StillPending, nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// SetEmail stores the account email in the session file (for whoami
// without network and the "Signed in as" line).
func (s *Session) SetEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess != nil && s.sess.Email != email {
		s.sess.Email = email
		_ = config.SaveSession(s.Host, s.sess)
	}
}

// Logout revokes the tokens on the server (best effort) and deletes them
// from this machine. Env tokens are not touched: they are not ours.
func (s *Session) Logout(ctx context.Context) (revoked bool, err error) {
	s.mu.Lock()
	sess := s.sess
	s.mu.Unlock()
	if sess == nil {
		_ = config.SavePending(s.Host, nil)
		return false, nil
	}
	revoked = true
	for _, t := range []string{sess.AccessToken, sess.RefreshToken} {
		if t == "" {
			continue
		}
		res, err := s.Client.Raw(ctx, "POST", "/oauth/revoke", api.FormBody(url.Values{"token": {t}}), false)
		if err != nil || res.Status != 200 {
			revoked = false
		}
	}
	if err := config.ClearSession(s.Host); err != nil {
		return revoked, err
	}
	s.mu.Lock()
	s.source, s.sess = SourceNone, nil
	s.mu.Unlock()
	return revoked, nil
}
