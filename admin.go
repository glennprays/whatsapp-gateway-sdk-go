package waga

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// AdminClient is a separate, opt-in client for the gateway's operator-only admin
// plane. That plane is cross-tenant, bearer-gated by the gateway's
// ADMIN_API_SECRET, and served at the server ROOT path (not under /api/v1) — so
// it is kept off the tenant-facing Client to make accidental use impossible.
//
// Construct it with NewAdminClient and the admin secret:
//
//	admin := waga.NewAdminClient(
//	    waga.WithBaseURL("https://gateway.example.com"), // server ROOT
//	    waga.WithAdminSecret(os.Getenv("ADMIN_API_SECRET")),
//	)
//
// The admin secret is sent as "Authorization: Bearer <secret>".
type AdminClient struct {
	c *Client
}

// NewAdminClient creates an admin-plane client. Unlike NewClient it defaults its
// base URL to the server ROOT (DefaultAdminBaseURL); override it with
// WithBaseURL to point at your gateway origin. Supply the admin secret via
// WithAdminSecret (or WithToken).
func NewAdminClient(opts ...Option) *AdminClient {
	c := &Client{
		baseURL:   DefaultAdminBaseURL,
		userAgent: DefaultUserAgent,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return &AdminClient{c: c}
}

// WithAdminSecret sets the operator admin secret sent as the Bearer credential
// on admin-plane requests. It is an alias for WithToken, named for admin usage.
func WithAdminSecret(secret string) Option {
	return WithToken(secret)
}

// SessionInventoryItem is the per-instance view of one account: masked phone,
// honest lifecycle state, and where the state was observed.
type SessionInventoryItem struct {
	PhoneMasked  string     `json:"phone_masked"`
	State        string     `json:"state"`  // connected|disconnected|never_paired|logged_out|banned
	Source       string     `json:"source"` // in-memory | store
	Reason       string     `json:"reason,omitempty"`
	LastSeen     time.Time  `json:"last_seen"`
	BanExpiresAt *time.Time `json:"ban_expires_at,omitempty"`
}

// SessionInventory is the GET /admin/sessions response. Instance identifies the
// reporting node (hostname); the view is per-instance by design.
type SessionInventory struct {
	Instance string                 `json:"instance"`
	Count    int                    `json:"count"`
	Sessions []SessionInventoryItem `json:"sessions"`
}

// AdminSessionResponse is the GET /admin/sessions/{phone} response: one account's
// session plus the reporting instance hostname.
type AdminSessionResponse struct {
	Instance string               `json:"instance"`
	Session  SessionInventoryItem `json:"session"`
}

// HealthComponent is one dependency's health in a readiness probe.
type HealthComponent struct {
	Connected bool `json:"connected"`
	Enabled   bool `json:"enabled,omitempty"` // queue only
}

// HealthReadyResponse is the GET /health/ready body (returned for both 200
// "ready" and 503 "not_ready"). Inspect Status to branch.
type HealthReadyResponse struct {
	Status    string           `json:"status"` // "ready" | "not_ready"
	Timestamp string           `json:"timestamp"`
	Database  *HealthComponent `json:"database,omitempty"`
	Queue     *HealthComponent `json:"queue,omitempty"`
}

// Sessions returns the reporting instance's session inventory (masked phones,
// honest states). An empty node returns Count 0, never an error.
func (a *AdminClient) Sessions(ctx context.Context) (*SessionInventory, error) {
	var resp SessionInventory
	if err := a.c.doRequest(ctx, http.MethodGet, "/admin/sessions", nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Session returns one account's session state by bare phone number. An unknown
// account returns ErrNotFound (404).
func (a *AdminClient) Session(ctx context.Context, phone string) (*AdminSessionResponse, error) {
	var resp AdminSessionResponse
	if err := a.c.doRequest(ctx, http.MethodGet, "/admin/sessions/"+url.PathEscape(phone), nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Live is the liveness probe (root /health/live). It always returns 200 for a
// running process and requires no admin secret.
func (a *AdminClient) Live(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := a.c.doRequest(ctx, http.MethodGet, "/health/live", nil, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Ready is the readiness probe (root /health/ready). It requires no admin
// secret and returns the structured body for both "ready" (HTTP 200) and
// "not_ready" (HTTP 503) — inspect the returned Status rather than treating 503
// as an error. Any other status (or a transport failure) is returned as an error.
func (a *AdminClient) Ready(ctx context.Context) (*HealthReadyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.c.baseURL+"/health/ready", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", a.c.userAgent)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}

	resp, err := a.c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	// 200 (ready) and 503 (not ready) both carry the structured body.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return nil, parseError(body, resp.StatusCode, resp.Header.Get(TraceIDHeader))
	}

	var result HealthReadyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}
