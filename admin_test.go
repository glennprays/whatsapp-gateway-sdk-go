package waga

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/sessions" {
			t.Errorf("expected /admin/sessions (root, no /api/v1), got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer admin-secret" {
			t.Errorf("expected admin bearer, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SessionInventory{
			Instance: "gw-1", Count: 1,
			Sessions: []SessionInventoryItem{{PhoneMasked: "62800xxxx", State: "connected", Source: "in-memory"}},
		})
	}))
	defer server.Close()

	admin := NewAdminClient(WithBaseURL(server.URL), WithAdminSecret("admin-secret"))
	inv, err := admin.Sessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Instance != "gw-1" || inv.Count != 1 || inv.Sessions[0].State != "connected" {
		t.Errorf("unexpected inventory: %+v", inv)
	}
}

func TestAdminSession_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/sessions/628" {
			t.Errorf("expected /admin/sessions/628, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "not found", "code": 404})
	}))
	defer server.Close()

	admin := NewAdminClient(WithBaseURL(server.URL), WithAdminSecret("admin-secret"))
	_, err := admin.Session(context.Background(), "628")
	if !IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAdminSession_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AdminSessionResponse{
			Instance: "gw-1",
			Session:  SessionInventoryItem{PhoneMasked: "62800xxxx", State: "banned"},
		})
	}))
	defer server.Close()

	admin := NewAdminClient(WithBaseURL(server.URL), WithAdminSecret("admin-secret"))
	resp, err := admin.Session(context.Background(), "628")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Instance != "gw-1" || resp.Session.State != "banned" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestAdminLive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health/live" {
			t.Errorf("expected /health/live (root), got %s", r.URL.Path)
		}
		// Live requires no admin secret.
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no auth header on live probe, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	}))
	defer server.Close()

	// No admin secret configured — Live must still work.
	admin := NewAdminClient(WithBaseURL(server.URL))
	resp, err := admin.Live(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "alive" {
		t.Errorf("expected status alive, got %q", resp.Status)
	}
}

func TestAdminReady(t *testing.T) {
	var status int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health/ready" {
			t.Errorf("expected /health/ready (root), got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		body := HealthReadyResponse{Status: "ready", Database: &HealthComponent{Connected: true}}
		if status == http.StatusServiceUnavailable {
			body = HealthReadyResponse{Status: "not_ready", Database: &HealthComponent{Connected: false}}
		}
		json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	admin := NewAdminClient(WithBaseURL(server.URL))

	// 200 -> ready, no error.
	status = http.StatusOK
	resp, err := admin.Ready(context.Background())
	if err != nil || resp.Status != "ready" {
		t.Fatalf("ready 200: %v %+v", err, resp)
	}

	// 503 -> not_ready, body still returned (no error).
	status = http.StatusServiceUnavailable
	resp, err = admin.Ready(context.Background())
	if err != nil {
		t.Fatalf("ready 503 should not error, got %v", err)
	}
	if resp.Status != "not_ready" || resp.Database.Connected {
		t.Errorf("unexpected not_ready body: %+v", resp)
	}
}

func TestAdminSessions_RequiresSecret(t *testing.T) {
	// No secret configured -> admin (authenticated) calls fail before any request.
	admin := NewAdminClient(WithBaseURL("http://example.invalid"))
	if _, err := admin.Sessions(context.Background()); err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated without admin secret, got %v", err)
	}
}
