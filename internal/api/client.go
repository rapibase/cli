// Package api is the HTTP client for the Rapibase control plane.
//
// Every request carries the CLI User-Agent (spec §6.5) and Accept-Language
// (server errors come back translated). GETs are retried on network errors
// and 502/503/504; other methods only when the connection was never made, so
// a POST is never sent twice. --debug logs method, path, status and timing,
// never headers or bodies: tokens and keys don't reach the terminal.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rapibase/cli/internal/build"
	"github.com/rapibase/cli/internal/cerr"
	"github.com/rapibase/cli/internal/i18n"
)

// TokenSource gives the bearer token and can refresh it after a 401.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
	// Refresh tries to get a new token after the server rejected the
	// current one. It returns false if there is nothing to refresh.
	Refresh(ctx context.Context) (bool, error)
}

// Client talks to one control plane.
type Client struct {
	Base  string
	Auth  TokenSource
	Debug io.Writer
	Agent string // detected agent for the User-Agent ("none" if unknown)
	CI    bool
	HTTP  *http.Client
}

// New returns a client for base.
func New(base string) *Client {
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: 5 * time.Minute}, Agent: "none"}
}

// UserAgent is rapibase-cli/<version> (<os>/<arch>; agent=<x>; ci=<0|1>).
func (c *Client) UserAgent() string {
	ci := "0"
	if c.CI {
		ci = "1"
	}
	agent := c.Agent
	if agent == "" {
		agent = "none"
	}
	return fmt.Sprintf("rapibase-cli/%s (%s/%s; agent=%s; ci=%s)", build.Version, runtime.GOOS, runtime.GOARCH, agent, ci)
}

// Body builds a request body; it is called again for each retry.
type Body func() (io.Reader, string, error)

// JSONBody encodes v as JSON.
func JSONBody(v any) Body {
	return func() (io.Reader, string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(b), "application/json", nil
	}
}

// BytesBody sends raw bytes with a content type.
func BytesBody(b []byte, contentType string) Body {
	return func() (io.Reader, string, error) { return bytes.NewReader(b), contentType, nil }
}

// FormBody sends an application/x-www-form-urlencoded body.
func FormBody(v url.Values) Body {
	return BytesBody([]byte(v.Encode()), "application/x-www-form-urlencoded")
}

// Response is a raw response, already read.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Do sends a request with auth and decodes a JSON response into out (if
// not nil). Non-2xx responses become *cerr.Error.
func (c *Client) Do(ctx context.Context, method, path string, body Body, out any) error {
	res, err := c.Raw(ctx, method, path, body, true)
	if err != nil {
		return err
	}
	if res.Status >= 300 {
		return ErrorFrom(res)
	}
	if out != nil && len(res.Body) > 0 {
		if err := json.Unmarshal(res.Body, out); err != nil {
			return cerr.New(cerr.ServerError, i18n.T("api.bad_response", res.Status))
		}
	}
	return nil
}

// Raw sends a request and returns the response as-is (any status). With
// auth, a 401 of the control plane triggers one token refresh and a retry
// (a 401 that comes from the project's instance is the answer itself).
// hdr adds headers to the request.
func (c *Client) Raw(ctx context.Context, method, path string, body Body, auth bool, hdr ...string) (*Response, error) {
	res, err := c.send(ctx, method, path, body, auth, hdr)
	if err != nil {
		return nil, err
	}
	if res.Status == http.StatusUnauthorized && auth && c.Auth != nil && !res.Upstream() {
		ok, rerr := c.Auth.Refresh(ctx)
		if rerr != nil {
			return nil, rerr
		}
		if ok {
			return c.send(ctx, method, path, body, auth, hdr)
		}
	}
	return res, nil
}

// Upstream reports a response the control plane passed through from the
// project's instance (X-Rapibase-Upstream), not one of its own.
func (r *Response) Upstream() bool { return r.Header.Get("X-Rapibase-Upstream") != "" }

func (c *Client) send(ctx context.Context, method, path string, body Body, auth bool, hdr []string) (*Response, error) {
	target := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		target = c.Base + path
	}
	// GET: three tries (network errors and 502/503/504). Anything else: a
	// second try only if the connection was never made.
	tries := 2
	if method == http.MethodGet {
		tries = 3
	}
	var lastErr error
	for i := 0; i < tries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i) * 600 * time.Millisecond):
			}
		}
		var rd io.Reader
		ct := ""
		if body != nil {
			var err error
			if rd, ct, err = body(); err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, target, rd)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", c.UserAgent())
		req.Header.Set("Accept-Language", i18n.AcceptLanguage())
		req.Header.Set("Accept", "application/json")
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		for i := 0; i+1 < len(hdr); i += 2 {
			req.Header.Set(hdr[i], hdr[i+1])
		}
		if auth && c.Auth != nil {
			tok, err := c.Auth.Token(ctx)
			if err != nil {
				return nil, err
			}
			if tok != "" {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
		}
		start := time.Now()
		res, err := c.HTTP.Do(req)
		if err != nil {
			c.debugf("%s %s → %v (%s)", method, redactPath(target), err, time.Since(start).Round(time.Millisecond))
			lastErr = err
			// a GET is always safe to repeat; anything else only if the
			// connection was never established
			if (method == http.MethodGet || isDialError(err)) && i+1 < tries {
				continue
			}
			return nil, networkError(err)
		}
		data, rerr := io.ReadAll(io.LimitReader(res.Body, 64<<20))
		res.Body.Close()
		c.debugf("%s %s → %d (%s, %d bytes)", method, redactPath(target), res.StatusCode, time.Since(start).Round(time.Millisecond), len(data))
		if rerr != nil {
			lastErr = rerr
			if method == http.MethodGet && i+1 < tries {
				continue
			}
			return nil, networkError(rerr)
		}
		if method == http.MethodGet && i+1 < tries && (res.StatusCode == 502 || res.StatusCode == 503 || res.StatusCode == 504) {
			continue
		}
		return &Response{Status: res.StatusCode, Header: res.Header, Body: data}, nil
	}
	return nil, networkError(lastErr)
}

func (c *Client) debugf(format string, a ...any) {
	if c.Debug != nil {
		fmt.Fprintf(c.Debug, "[debug] "+format+"\n", a...)
	}
}

// redactPath drops the query string: it can carry codes (device, refs).
func redactPath(u string) string {
	if i := strings.IndexByte(u, '?'); i >= 0 {
		return u[:i] + "?…"
	}
	return u
}

func isDialError(err error) bool {
	var op *net.OpError
	if errors.As(err, &op) && op.Op == "dial" {
		return true
	}
	var dns *net.DNSError
	return errors.As(err, &dns)
}

func networkError(err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return cerr.New(cerr.Network, i18n.T("api.network", msg))
}

// ErrorFrom turns an error response into a *cerr.Error. The control plane
// answers {"error": "<translated message>", "code": "<code>", ...extra}; the
// code is inferred from the status when missing.
func ErrorFrom(res *Response) *cerr.Error {
	var body map[string]any
	_ = json.Unmarshal(res.Body, &body)
	msg, _ := body["error"].(string)
	code, _ := body["code"].(string)
	if code == "" {
		code = codeForStatus(res.Status)
	}
	if msg == "" {
		msg = i18n.T("api.http_status", res.Status)
	}
	e := cerr.New(code, msg)
	for k, v := range body {
		if k != "error" && k != "code" {
			e.With(k, v)
		}
	}
	if code == cerr.RateLimited && e.Extra["retry_after"] == nil {
		if s, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil {
			e.With("retry_after", s)
		}
	}
	return e
}

func codeForStatus(status int) string {
	switch {
	case status == 400 || status == 413 || status == 422:
		return cerr.Usage
	case status == 401:
		return cerr.SessionExpired
	case status == 403:
		return cerr.Forbidden
	case status == 404:
		return cerr.ProjectNotFound
	case status == 409:
		return cerr.ProjectNotRunning
	case status == 429:
		return cerr.RateLimited
	}
	return cerr.ServerError
}
