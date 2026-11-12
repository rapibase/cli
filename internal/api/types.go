package api

import (
	"encoding/json"
	"strconv"
)

// Project is a project as the control plane shows it to a token: without
// keys, with its URLs and the mode this session has on it.
type Project struct {
	ID             int64           `json:"id"`
	OrgID          int64           `json:"org_id"`
	OrgName        string          `json:"org_name,omitempty"`
	OrgKind        string          `json:"org_kind,omitempty"`
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	Subdomain      string          `json:"subdomain"`
	Region         string          `json:"region"`
	Plan           string          `json:"plan"`
	Status         string          `json:"status"`
	Error          string          `json:"error,omitempty"`
	Mode           string          `json:"mode,omitempty"`
	ActiveDeployID string          `json:"active_deploy_id,omitempty"`
	CustomDomain   string          `json:"custom_domain,omitempty"`
	AppURL         string          `json:"app_url,omitempty"`
	DashboardURL   string          `json:"dashboard_url,omitempty"`
	ConsoleURL     string          `json:"console_url,omitempty"`
	Free           json.RawMessage `json:"free,omitempty"`
	Paid           json.RawMessage `json:"paid,omitempty"`
}

// IDString is the id as text.
func (p *Project) IDString() string { return strconv.FormatInt(p.ID, 10) }

// Org is an organization of the person.
type Org struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"` // personal | team
	Role string `json:"role"`
}

// Connection is the session's connection (what `whoami` shows).
type Connection struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	AllProjects   bool     `json:"all_projects"`
	DefaultMode   string   `json:"default_mode"`
	AccountScopes []string `json:"account_scopes"`
	URL           string   `json:"url"`
}

// Me is GET /api/me with a token.
type Me struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	Orgs          []Org      `json:"orgs"`
	Connection    Connection `json:"connection"`
}

// CanCreate reports whether the session may create free projects.
func (m *Me) CanCreate() bool {
	for _, s := range m.Connection.AccountScopes {
		if s == "projects:create" {
			return true
		}
	}
	return false
}

// Region is an available region.
type Region struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Version is GET /api/cli/version.
type Version struct {
	Latest       string `json:"latest"`
	MinSupported string `json:"min_supported"`
}
