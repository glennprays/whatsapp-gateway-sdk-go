package waga

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("expected client to be created")
	}
	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL %s, got %s", DefaultBaseURL, client.baseURL)
	}
}

func TestNewClientWithOptions(t *testing.T) {
	customURL := "https://custom.example.com/api/v1"
	client := NewClient(
		WithBaseURL(customURL),
		WithToken("test-token"),
	)

	if client.baseURL != customURL {
		t.Errorf("expected baseURL %s, got %s", customURL, client.baseURL)
	}
	if client.token != "test-token" {
		t.Errorf("expected token test-token, got %s", client.token)
	}
}

func TestSetToken(t *testing.T) {
	client := NewClient()
	client.SetToken("new-token")
	if client.token != "new-token" {
		t.Errorf("expected token new-token, got %s", client.token)
	}
}

func TestRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/register" {
			t.Errorf("expected path /api/v1/register, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := RegisterResponse{Token: "test-jwt-token"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL + "/api/v1"))
	resp, err := client.Register(context.Background(), "6281234567890", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "test-jwt-token" {
		t.Errorf("expected token test-jwt-token, got %s", resp.Token)
	}
	if client.token != "test-jwt-token" {
		t.Errorf("client token not set after registration")
	}
}

func TestSendText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/text" {
			t.Errorf("expected path /api/v1/message/text, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %s", r.Header.Get("Authorization"))
		}

		resp := SendMessageResponse{Success: true, MessageId: "msg_123"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	resp, err := client.SendText(context.Background(), "6281234567890@s.whatsapp.net", "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.MessageId != "msg_123" {
		t.Errorf("expected message ID msg_123, got %s", resp.MessageId)
	}
}

func TestNotAuthenticated(t *testing.T) {
	client := NewClient()
	_, err := client.SendText(context.Background(), "test", "test")
	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestErrorParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid token",
			"code":  401,
		})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	_, err := client.SendText(context.Background(), "test", "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsUnauthorized(err) {
		t.Errorf("expected IsUnauthorized to be true, got %v", err)
	}
}

func TestFormatMSISDN(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"6281234567890", "6281234567890@s.whatsapp.net"},
		{"6281234567890@s.whatsapp.net", "6281234567890@s.whatsapp.net"},
	}

	for _, tt := range tests {
		result := FormatMSISDN(tt.input)
		if result != tt.expected {
			t.Errorf("FormatMSISDN(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatGroupID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1234567890", "1234567890@g.us"},
		{"1234567890@g.us", "1234567890@g.us"},
	}

	for _, tt := range tests {
		result := FormatGroupID(tt.input)
		if result != tt.expected {
			t.Errorf("FormatGroupID(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestSDKError(t *testing.T) {
	err := &SDKError{Code: 404, Message: "not found"}
	if err.Error() != "sdk error: code=404, message=not found" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}
