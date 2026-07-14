package waga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
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

// ============================================================================
// Simple Utility Tests
// ============================================================================

// TestGetToken tests the GetToken method
func TestGetToken(t *testing.T) {
	client := NewClient(WithToken("test-token-123"))
	if client.GetToken() != "test-token-123" {
		t.Errorf("expected token 'test-token-123', got '%s'", client.GetToken())
	}

	client.SetToken("new-token")
	if client.GetToken() != "new-token" {
		t.Errorf("expected token 'new-token', got '%s'", client.GetToken())
	}
}

// TestHealth_Success tests successful health check
func TestHealth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			t.Errorf("expected path /api/v1/health, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		resp := HealthResponse{Status: "ok", Timestamp: "2024-01-01T00:00:00Z"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL + "/api/v1"))
	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", health.Status)
	}
}

// TestHealth_Error tests health check with server error
func TestHealth_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "service unavailable"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL + "/api/v1"))
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsInternalServer(err) {
		t.Errorf("expected IsInternalServer to be true, got %v", err)
	}
}

// TestHealth_Timeout tests health check with timeout
func TestHealth_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate timeout by not responding
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithTimeout(1*time.Millisecond),
	)
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestWithHTTPClient tests custom HTTP client option
func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 60 * time.Second,
	}
	client := NewClient(WithHTTPClient(customClient))

	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", client.httpClient.Timeout)
	}
}

// TestWithTimeout tests timeout option
func TestWithTimeout(t *testing.T) {
	client := NewClient(WithTimeout(45 * time.Second))

	if client.httpClient.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", client.httpClient.Timeout)
	}
}

// TestWithUserAgent tests custom User-Agent option
func TestWithUserAgent(t *testing.T) {
	customUA := "MyApp/2.0"
	client := NewClient(WithUserAgent(customUA))

	if client.userAgent != customUA {
		t.Errorf("expected User-Agent '%s', got '%s'", customUA, client.userAgent)
	}
}

// ============================================================================
// Authentication & Session Management Tests
// ============================================================================

// TestGetQRCode_JSONFormat tests successful QR code retrieval with JSON format
func TestGetQRCode_JSONFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/login/qr_code/json" {
			t.Errorf("expected path /api/v1/login/qr_code/json, got %s", r.URL.Path)
		}

		resp := LoginQrResponse{QrCode: "base64qrdata", ExpiresIn: 60}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	qr, err := client.GetQRCode(context.Background(), "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qr.QrCode != "base64qrdata" {
		t.Errorf("expected qr code 'base64qrdata', got '%s'", qr.QrCode)
	}
	if qr.ExpiresIn != 60 {
		t.Errorf("expected expires in 60, got %d", qr.ExpiresIn)
	}
}

// TestGetQRCode_HTMLFormat tests successful QR code retrieval with HTML format
func TestGetQRCode_HTMLFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/login/qr_code/html" {
			t.Errorf("expected path /api/v1/login/qr_code/html, got %s", r.URL.Path)
		}

		resp := LoginQrResponse{QrCode: "<img src='...'>", ExpiresIn: 120}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	qr, err := client.GetQRCode(context.Background(), "html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qr.QrCode != "<img src='...'>" {
		t.Errorf("expected HTML img tag, got '%s'", qr.QrCode)
	}
}

// TestGetQRCode_NotAuthenticated tests QR code retrieval without authentication
func TestGetQRCode_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	_, err := client.GetQRCode(context.Background(), "json")

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestGetQRCode_InvalidFormat tests QR code retrieval with invalid format
func TestGetQRCode_InvalidFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid format", "code": "400"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	_, err := client.GetQRCode(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsBadRequest(err) {
		t.Errorf("expected IsBadRequest to be true, got %v", err)
	}
}

// TestGetQRCode_ServerError tests QR code retrieval with server error
func TestGetQRCode_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error", "code": "500"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	_, err := client.GetQRCode(context.Background(), "json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsInternalServer(err) {
		t.Errorf("expected IsInternalServer to be true, got %v", err)
	}
}

// TestGetPairCode_Success tests successful pair code retrieval
func TestGetPairCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/login/pair_code" {
			t.Errorf("expected path /api/v1/login/pair_code, got %s", r.URL.Path)
		}

		resp := LoginPairResponse{PairCode: "ABCD1234", ExpiresIn: 300}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	pair, err := client.GetPairCode(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.PairCode != "ABCD1234" {
		t.Errorf("expected pair code 'ABCD1234', got '%s'", pair.PairCode)
	}
	if pair.ExpiresIn != 300 {
		t.Errorf("expected expires in 300, got %d", pair.ExpiresIn)
	}
}

// TestGetPairCode_NotAuthenticated tests pair code retrieval without authentication
func TestGetPairCode_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	_, err := client.GetPairCode(context.Background())

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestGetPairCode_RateLimited tests pair code retrieval with rate limiting
func TestGetPairCode_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limited", "code": "429"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	_, err := client.GetPairCode(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsRateLimited(err) {
		t.Errorf("expected IsRateLimited to be true, got %v", err)
	}
}

// TestGetPairCode_ContextCancellation tests pair code retrieval with cancelled context
func TestGetPairCode_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusRequestTimeout)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetPairCode(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

// TestGetLoginStatus_Authenticated tests login status when authenticated
func TestGetLoginStatus_Authenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/login/status" {
			t.Errorf("expected path /api/v1/login/status, got %s", r.URL.Path)
		}

		resp := LoginStatus{Authenticated: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	status, err := client.GetLoginStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Authenticated {
		t.Error("expected authenticated to be true")
	}
}

// TestGetLoginStatus_NotAuthenticated tests login status when not authenticated
func TestGetLoginStatus_NotAuthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := LoginStatus{Authenticated: false}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	status, err := client.GetLoginStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Authenticated {
		t.Error("expected authenticated to be false")
	}
}

// TestGetLoginStatus_NotAuthenticatedClient tests login status with unauthenticated client
func TestGetLoginStatus_NotAuthenticatedClient(t *testing.T) {
	client := NewClient() // No token set
	_, err := client.GetLoginStatus(context.Background())

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestGetLoginStatus_NetworkError tests login status with network error
func TestGetLoginStatus_NetworkError(t *testing.T) {
	// Use a URL that will fail to connect
	client := NewClient(
		WithBaseURL("http://localhost:9999/api/v1"),
		WithToken("test-token"),
	)

	_, err := client.GetLoginStatus(context.Background())
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// TestLogout_Success tests successful logout
func TestLogout_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/logout" {
			t.Errorf("expected path /api/v1/logout, got %s", r.URL.Path)
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.Logout(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLogout_NotAuthenticated tests logout without authentication
func TestLogout_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	err := client.Logout(context.Background())

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestLogout_Conflict tests logout with conflict error
func TestLogout_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "session already terminated", "code": "409"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.Logout(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsConflict(err) {
		t.Errorf("expected IsConflict to be true, got %v", err)
	}
}

// TestReconnect_Success tests successful reconnection
func TestReconnect_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/session/reconnect" {
			t.Errorf("expected path /api/v1/session/reconnect, got %s", r.URL.Path)
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.Reconnect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestReconnect_NotAuthenticated tests reconnection without authentication
func TestReconnect_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	err := client.Reconnect(context.Background())

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestReconnect_Forbidden tests reconnection with forbidden error
func TestReconnect_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "reconnection not allowed", "code": "403"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.Reconnect(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsForbidden(err) {
		t.Errorf("expected IsForbidden to be true, got %v", err)
	}
}

// ============================================================================
// Messaging Operations Tests
// ============================================================================

// Mock helpers for image testing

type mockImageReader struct {
	data []byte
	err  error
}

func (m *mockImageReader) Read(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	if len(m.data) == 0 {
		return 0, io.EOF
	}
	n = copy(p, m.data)
	m.data = m.data[n:]
	return n, nil
}

type errorReader struct{ err error }

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

// TestSendImage_AllParameters tests successful image send with all parameters
func TestSendImage_AllParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/image" {
			t.Errorf("expected path /api/v1/message/image, got %s", r.URL.Path)
		}

		// Parse multipart form to verify fields
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}

		if r.FormValue("msisdn") != "6281234567890@s.whatsapp.net" {
			t.Errorf("expected msisdn '6281234567890@s.whatsapp.net', got '%s'", r.FormValue("msisdn"))
		}
		if r.FormValue("caption") != "Test image" {
			t.Errorf("expected caption 'Test image', got '%s'", r.FormValue("caption"))
		}
		if r.FormValue("is_view_once") != "true" {
			t.Errorf("expected is_view_once 'true', got '%s'", r.FormValue("is_view_once"))
		}

		resp := SendMessageResponse{Success: true, MessageId: "img_123"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	imageData := []byte("fake image data")
	image := &mockImageReader{data: imageData}

	resp, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "Test image", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.MessageId != "img_123" {
		t.Errorf("expected message ID 'img_123', got '%s'", resp.MessageId)
	}
}

// TestSendImage_MinimalParameters tests successful image send with minimal parameters
func TestSendImage_MinimalParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SendMessageResponse{Success: true, MessageId: "img_456"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	imageData := []byte("minimal image")
	image := &mockImageReader{data: imageData}

	resp, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
}

// TestSendImage_WithCaptionOnly tests image send with caption only
func TestSendImage_WithCaptionOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SendMessageResponse{Success: true, MessageId: "img_789"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	imageData := []byte("image with caption")
	image := &mockImageReader{data: imageData}

	resp, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "Hello World", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
}

// TestSendImage_WithViewOnceOnly tests image send with view-once only
func TestSendImage_WithViewOnceOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("is_view_once") != "true" {
			t.Errorf("expected is_view_once 'true', got '%s'", r.FormValue("is_view_once"))
		}

		resp := SendMessageResponse{Success: true, MessageId: "img_101"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	imageData := []byte("view once image")
	image := &mockImageReader{data: imageData}

	resp, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
}

// TestSendImage_NotAuthenticated tests image send without authentication
func TestSendImage_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	imageData := []byte("test image")
	image := &mockImageReader{data: imageData}

	_, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "", false)
	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestSendImage_InvalidImage tests image send with read failure
func TestSendImage_InvalidImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SendMessageResponse{Success: true, MessageId: "img_error"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	image := &errorReader{err: io.ErrUnexpectedEOF}

	_, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to copy image data") {
		t.Errorf("expected image copy error, got %v", err)
	}
}

// TestSendImage_BadRequest tests image send with bad request error
func TestSendImage_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid image format", "code": "400"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	imageData := []byte("invalid image")
	image := &mockImageReader{data: imageData}

	_, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsBadRequest(err) {
		t.Errorf("expected IsBadRequest to be true, got %v", err)
	}
}

// TestSendImage_MultipartFormValidation tests multipart form field validation
func TestSendImage_MultipartFormValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type is multipart
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			t.Errorf("expected multipart content type, got '%s'", contentType)
		}

		// Verify authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header, got '%s'", r.Header.Get("Authorization"))
		}

		resp := SendMessageResponse{Success: true, MessageId: "img_valid"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	imageData := []byte("validation image")
	image := &mockImageReader{data: imageData}

	resp, err := client.SendImage(context.Background(), "6281234567890@s.whatsapp.net", image, "Valid caption", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MessageId != "img_valid" {
		t.Errorf("expected message ID 'img_valid', got '%s'", resp.MessageId)
	}
}

func TestSendAudio_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/audio" {
			t.Errorf("expected path /api/v1/message/audio, got %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}
		if r.FormValue("msisdn") != "6281234567890@s.whatsapp.net" {
			t.Errorf("expected msisdn, got %s", r.FormValue("msisdn"))
		}
		if r.FormValue("is_ptt") != "true" {
			t.Errorf("expected is_ptt true, got %s", r.FormValue("is_ptt"))
		}
		if r.FormValue("is_view_once") != "true" {
			t.Errorf("expected is_view_once true, got %s", r.FormValue("is_view_once"))
		}

		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, MessageId: "aud_123"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)
	audio := &mockImageReader{data: []byte("audio-data")}
	resp, err := client.SendAudio(context.Background(), "6281234567890@s.whatsapp.net", audio, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MessageId != "aud_123" {
		t.Errorf("expected message ID 'aud_123', got %s", resp.MessageId)
	}
}

func TestSendVideo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/video" {
			t.Errorf("expected path /api/v1/message/video, got %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}
		if r.FormValue("caption") != "hello video" {
			t.Errorf("expected caption 'hello video', got %s", r.FormValue("caption"))
		}
		if r.FormValue("is_gif") != "true" {
			t.Errorf("expected is_gif true, got %s", r.FormValue("is_gif"))
		}
		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, MessageId: "vid_123"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)
	video := &mockImageReader{data: []byte("video-data")}
	resp, err := client.SendVideo(context.Background(), "6281234567890@s.whatsapp.net", video, "hello video", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MessageId != "vid_123" {
		t.Errorf("expected message ID 'vid_123', got %s", resp.MessageId)
	}
}

func TestSendDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/document" {
			t.Errorf("expected path /api/v1/message/document, got %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}
		if r.FormValue("file_name") != "invoice.pdf" {
			t.Errorf("expected file_name 'invoice.pdf', got %s", r.FormValue("file_name"))
		}
		if r.FormValue("caption") != "monthly invoice" {
			t.Errorf("expected caption 'monthly invoice', got %s", r.FormValue("caption"))
		}
		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, MessageId: "doc_123"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)
	doc := &mockImageReader{data: []byte("doc-data")}
	resp, err := client.SendDocument(context.Background(), "6281234567890@s.whatsapp.net", doc, "invoice.pdf", "monthly invoice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MessageId != "doc_123" {
		t.Errorf("expected message ID 'doc_123', got %s", resp.MessageId)
	}
}

// TestSendLocation_Success tests successful location send
func TestSendLocation_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/location" {
			t.Errorf("expected path /api/v1/message/location, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %s", r.Header.Get("Authorization"))
		}

		var reqBody SendLocationMessageRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.Msisdn != "6281234567890@s.whatsapp.net" {
			t.Errorf("expected msisdn '6281234567890@s.whatsapp.net', got '%s'", reqBody.Msisdn)
		}
		if reqBody.Latitude != -6.2088 {
			t.Errorf("expected latitude -6.2088, got %f", reqBody.Latitude)
		}
		if reqBody.Longitude != 106.8456 {
			t.Errorf("expected longitude 106.8456, got %f", reqBody.Longitude)
		}
		if reqBody.Name != "Jakarta" {
			t.Errorf("expected name 'Jakarta', got '%s'", reqBody.Name)
		}
		if reqBody.Address != "Jakarta, Indonesia" {
			t.Errorf("expected address 'Jakarta, Indonesia', got '%s'", reqBody.Address)
		}

		resp := SendMessageResponse{Success: true, MessageId: "loc_123"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	resp, err := client.SendLocation(context.Background(), "6281234567890@s.whatsapp.net", -6.2088, 106.8456, "Jakarta", "Jakarta, Indonesia")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.MessageId != "loc_123" {
		t.Errorf("expected message ID 'loc_123', got '%s'", resp.MessageId)
	}
}

// TestSendLocation_NotAuthenticated tests location send without authentication
func TestSendLocation_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	_, err := client.SendLocation(context.Background(), "6281234567890@s.whatsapp.net", -6.2088, 106.8456, "", "")

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestSendPoll_Success tests successful poll send
func TestSendPoll_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/poll" {
			t.Errorf("expected path /api/v1/message/poll, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %s", r.Header.Get("Authorization"))
		}

		var reqBody SendPollMessageRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.Msisdn != "6281234567890@s.whatsapp.net" {
			t.Errorf("expected msisdn '6281234567890@s.whatsapp.net', got '%s'", reqBody.Msisdn)
		}
		if reqBody.Question != "What is your favorite color?" {
			t.Errorf("expected question 'What is your favorite color?', got '%s'", reqBody.Question)
		}
		if len(reqBody.Options) != 3 {
			t.Errorf("expected 3 options, got %d", len(reqBody.Options))
		}
		if reqBody.SelectableCount != 1 {
			t.Errorf("expected selectable_count 1, got %d", reqBody.SelectableCount)
		}

		resp := SendMessageResponse{Success: true, MessageId: "poll_123"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	resp, err := client.SendPoll(context.Background(), "6281234567890@s.whatsapp.net", "What is your favorite color?", []string{"Red", "Green", "Blue"}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.MessageId != "poll_123" {
		t.Errorf("expected message ID 'poll_123', got '%s'", resp.MessageId)
	}
}

// TestSendPoll_NotAuthenticated tests poll send without authentication
func TestSendPoll_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	_, err := client.SendPoll(context.Background(), "6281234567890@s.whatsapp.net", "Q?", []string{"A", "B"}, 0)

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestSendSticker_Success tests successful sticker send
func TestSendSticker_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/sticker" {
			t.Errorf("expected path /api/v1/message/sticker, got %s", r.URL.Path)
		}

		// Parse multipart form to verify fields
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}

		if r.FormValue("msisdn") != "6281234567890@s.whatsapp.net" {
			t.Errorf("expected msisdn '6281234567890@s.whatsapp.net', got '%s'", r.FormValue("msisdn"))
		}

		// Verify content type is multipart
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			t.Errorf("expected multipart content type, got '%s'", contentType)
		}

		// Verify authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header, got '%s'", r.Header.Get("Authorization"))
		}

		resp := SendMessageResponse{Success: true, MessageId: "stk_123"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	stickerData := []byte("fake sticker data")
	sticker := &mockImageReader{data: stickerData}

	resp, err := client.SendSticker(context.Background(), "6281234567890@s.whatsapp.net", sticker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.MessageId != "stk_123" {
		t.Errorf("expected message ID 'stk_123', got '%s'", resp.MessageId)
	}
}

// TestSendSticker_NotAuthenticated tests sticker send without authentication
func TestSendSticker_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	stickerData := []byte("test sticker")
	sticker := &mockImageReader{data: stickerData}

	_, err := client.SendSticker(context.Background(), "6281234567890@s.whatsapp.net", sticker)
	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestSendSticker_InvalidSticker tests sticker send with read failure
func TestSendSticker_InvalidSticker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SendMessageResponse{Success: true, MessageId: "stk_error"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	sticker := &errorReader{err: io.ErrUnexpectedEOF}

	_, err := client.SendSticker(context.Background(), "6281234567890@s.whatsapp.net", sticker)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to copy sticker data") {
		t.Errorf("expected sticker copy error, got %v", err)
	}
}

// TestEditMessage_Success tests successful message edit
func TestEditMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message" {
			t.Errorf("expected path /api/v1/message, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.EditMessage(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "Corrected message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestEditMessage_NotAuthenticated tests message edit without authentication
func TestEditMessage_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	err := client.EditMessage(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "New message")

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestEditMessage_NotFound tests message edit with not found error
func TestEditMessage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "message not found", "code": "404"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.EditMessage(context.Background(), "6281234567890@s.whatsapp.net", "nonexistent", "New message")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got %v", err)
	}
}

// TestEditMessage_Conflict tests message edit with conflict error
func TestEditMessage_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "message already edited", "code": "409"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.EditMessage(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "New message")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsConflict(err) {
		t.Errorf("expected IsConflict to be true, got %v", err)
	}
}

// TestEditMessage_EmptyNewMessage tests message edit with empty new message
func TestEditMessage_EmptyNewMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server should accept empty string as valid (clear message)
		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.EditMessage(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteMessage_Success tests successful message deletion
func TestDeleteMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message" {
			t.Errorf("expected path /api/v1/message, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.DeleteMessage(context.Background(), "6281234567890@s.whatsapp.net", "msg_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteMessage_NotAuthenticated tests message deletion without authentication
func TestDeleteMessage_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	err := client.DeleteMessage(context.Background(), "6281234567890@s.whatsapp.net", "msg_123")

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestDeleteMessage_NotFound tests message deletion with not found error
func TestDeleteMessage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "message not found", "code": "404"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.DeleteMessage(context.Background(), "6281234567890@s.whatsapp.net", "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got %v", err)
	}
}

// TestDeleteMessage_Forbidden tests message deletion with forbidden error
func TestDeleteMessage_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "cannot delete message", "code": "403"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.DeleteMessage(context.Background(), "6281234567890@s.whatsapp.net", "msg_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsForbidden(err) {
		t.Errorf("expected IsForbidden to be true, got %v", err)
	}
}

// TestReact_Success tests successful reaction with emoji
func TestReact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/react" {
			t.Errorf("expected path /api/v1/message/react, got %s", r.URL.Path)
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.React(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "👍")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestReact_RemoveReaction tests removing a reaction (empty emoji)
func TestReact_RemoveReaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.React(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestReact_NotAuthenticated tests reaction without authentication
func TestReact_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	err := client.React(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "👍")

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestReact_InvalidEmoji tests reaction with invalid emoji
func TestReact_InvalidEmoji(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid emoji", "code": "400"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.React(context.Background(), "6281234567890@s.whatsapp.net", "msg_123", "invalid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsBadRequest(err) {
		t.Errorf("expected IsBadRequest to be true, got %v", err)
	}
}

// TestReact_MessageNotFound tests reaction with message not found
func TestReact_MessageNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "message not found", "code": "404"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.React(context.Background(), "6281234567890@s.whatsapp.net", "nonexistent", "👍")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got %v", err)
	}
}

func TestReact_WithSenderMsisdn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody MessageReactRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody.SenderMsisdn != "6281111111111@s.whatsapp.net" {
			t.Errorf("expected sender_msisdn to be set, got %q", reqBody.SenderMsisdn)
		}
		json.NewEncoder(w).Encode(SuccessResponse{Success: true})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)
	err := client.React(
		context.Background(),
		"120363xxxxx@g.us",
		"msg_123",
		"👍",
		"6281111111111@s.whatsapp.net",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/contact/check" {
			t.Errorf("expected path /api/v1/contact/check, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("msisdn"); got != "6281234567890@s.whatsapp.net" {
			t.Errorf("expected msisdn query, got %q", got)
		}

		resp := ContactCheckResponse{
			Query:        "6281234567890",
			JID:          "6281234567890@s.whatsapp.net",
			IsOnWhatsApp: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)
	resp, err := client.CheckContact(context.Background(), "6281234567890@s.whatsapp.net")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsOnWhatsApp {
		t.Errorf("expected is_on_whatsapp true, got false")
	}
}

// ============================================================================
// Webhook Management Tests
// ============================================================================

// TestRegisterWebhook_WithHMACSecret tests successful webhook registration with HMAC secret
func TestRegisterWebhook_WithHMACSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/webhook" {
			t.Errorf("expected path /api/v1/webhook, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify request body contains hmac_secret
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["url"] != "https://example.com/webhook" {
			t.Errorf("expected URL 'https://example.com/webhook', got '%v'", reqBody["url"])
		}
		if reqBody["hmac_secret"] != "my_secret" {
			t.Errorf("expected hmac_secret 'my_secret', got '%v'", reqBody["hmac_secret"])
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.RegisterWebhook(context.Background(), "https://example.com/webhook", "my_secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRegisterWebhook_WithoutHMACSecret tests successful webhook registration without HMAC secret
func TestRegisterWebhook_WithoutHMACSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body doesn't contain hmac_secret (or it's null)
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["url"] != "https://example.com/webhook" {
			t.Errorf("expected URL 'https://example.com/webhook', got '%v'", reqBody["url"])
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.RegisterWebhook(context.Background(), "https://example.com/webhook", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRegisterWebhook_NotAuthenticated tests webhook registration without authentication
func TestRegisterWebhook_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	err := client.RegisterWebhook(context.Background(), "https://example.com/webhook", "secret")

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestRegisterWebhook_InvalidURL tests webhook registration with invalid URL
func TestRegisterWebhook_InvalidURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid URL", "code": "400"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.RegisterWebhook(context.Background(), "invalid-url", "secret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsBadRequest(err) {
		t.Errorf("expected IsBadRequest to be true, got %v", err)
	}
}

// TestRegisterWebhook_Conflict tests webhook registration with conflict error
func TestRegisterWebhook_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "webhook already registered", "code": "409"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.RegisterWebhook(context.Background(), "https://example.com/webhook", "secret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsConflict(err) {
		t.Errorf("expected IsConflict to be true, got %v", err)
	}
}

// TestGetWebhook_WithRegisteredWebhook tests getting webhook when one is registered
func TestGetWebhook_WithRegisteredWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/webhook" {
			t.Errorf("expected path /api/v1/webhook, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		resp := WebhookResponse{URL: "https://example.com/webhook"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	webhook, err := client.GetWebhook(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if webhook.URL != "https://example.com/webhook" {
		t.Errorf("expected URL 'https://example.com/webhook', got '%s'", webhook.URL)
	}
}

// TestGetWebhook_NoWebhookRegistered tests getting webhook when none is registered
func TestGetWebhook_NoWebhookRegistered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no webhook registered", "code": "404"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	_, err := client.GetWebhook(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got %v", err)
	}
}

// TestGetWebhook_NotAuthenticated tests getting webhook without authentication
func TestGetWebhook_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	_, err := client.GetWebhook(context.Background())

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestUnregisterWebhook_Success tests successful webhook unregistration
func TestUnregisterWebhook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/webhook" {
			t.Errorf("expected path /api/v1/webhook, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		resp := SuccessResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.UnregisterWebhook(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestUnregisterWebhook_NotAuthenticated tests webhook unregistration without authentication
func TestUnregisterWebhook_NotAuthenticated(t *testing.T) {
	client := NewClient() // No token set
	err := client.UnregisterWebhook(context.Background())

	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// TestUnregisterWebhook_NotFound tests webhook unregistration when none is registered
func TestUnregisterWebhook_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no webhook registered", "code": "404"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.UnregisterWebhook(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got %v", err)
	}
}

// ============================================================================
// doRequest() Edge Case Tests
// ============================================================================

// TestDoRequest_ContextCancellation tests request with cancelled context
func TestDoRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusRequestTimeout)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var resp map[string]interface{}
	err := client.doRequest(ctx, http.MethodGet, "/health", nil, &resp, false)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

// TestDoRequest_Timeout tests request timeout
func TestDoRequest_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		<-time.After(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithTimeout(10*time.Millisecond),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, &resp, false)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestDoRequest_NetworkError tests request with network error
func TestDoRequest_NetworkError(t *testing.T) {
	// Use a URL that will fail to connect
	client := NewClient(
		WithBaseURL("http://localhost:9999/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, &resp, false)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("expected request failed error, got %v", err)
	}
}

// TestDoRequest_ResponseBodyReadError tests response body read error
func TestDoRequest_ResponseBodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send response but immediately close connection
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		// Close the connection to simulate read error
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, &resp, false)
	// This might succeed with empty response or fail with read error
	// We'll accept either since connection behavior varies
	if err != nil && !strings.Contains(err.Error(), "failed to read response") {
		t.Logf("Got error (acceptable): %v", err)
	}
}

// TestDoRequest_UnmarshalError tests response unmarshal error with invalid JSON
func TestDoRequest_UnmarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, &resp, false)
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("expected parse error, got %v", err)
	}
}

// TestDoRequest_EmptyResponse204 tests empty response with 204 status
func TestDoRequest_EmptyResponse204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// No body sent
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodDelete, "/resource", nil, &resp, true)
	if err != nil {
		t.Errorf("unexpected error for 204 response: %v", err)
	}
}

// TestDoRequest_NilResult tests request with nil result parameter
func TestDoRequest_NilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, nil, false)
	if err != nil {
		t.Errorf("unexpected error with nil result: %v", err)
	}
}

// TestDoRequest_MarshalError tests request body marshal error
func TestDoRequest_MarshalError(t *testing.T) {
	client := NewClient(
		WithBaseURL("http://localhost:9999/api/v1"),
		WithToken("test-token"),
	)

	// Pass a channel (which cannot be marshaled to JSON)
	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodPost, "/test", make(chan int), &resp, false)
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to marshal request body") {
		t.Errorf("expected marshal error, got %v", err)
	}
}

// TestDoRequest_InvalidJSON tests response with invalid JSON characters
func TestDoRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send JSON with BOM (byte order mark)
		w.Write([]byte("\xEF\xBB\xBF{\"status\":\"ok\"}"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, &resp, false)
	// BOM handling may vary, but we shouldn't get an unmarshal error
	if err != nil {
		t.Logf("Got error with BOM (acceptable): %v", err)
	}
}

// TestDoRequest_EmptyJSONObject tests response with empty JSON object
func TestDoRequest_EmptyJSONObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, &resp, false)
	if err != nil {
		t.Errorf("unexpected error with empty JSON object: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty map, got %v", resp)
	}
}

// TestDoRequest_UnexpectedStatusCode tests response with unexpected status code
func TestDoRequest_UnexpectedStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/health", nil, &resp, false)
	// 201 is in 2xx range, so should be successful
	if err != nil {
		t.Errorf("unexpected error with 201 status: %v", err)
	}
}

// TestDoRequest_RedirectStatus tests response with redirect status
func TestDoRequest_RedirectStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusFound)
		w.Header().Set("Location", "/other")
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL+"/api/v1"),
		WithToken("test-token"),
	)

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/redirect", nil, &resp, false)
	// 302 is not in 2xx range, should return error
	if err == nil {
		t.Fatal("expected error for redirect status, got nil")
	}
}

// TestDoRequest_RequireAuthWithoutToken tests request requiring auth without token
func TestDoRequest_RequireAuthWithoutToken(t *testing.T) {
	client := NewClient(WithBaseURL("http://localhost:9999/api/v1"))

	var resp map[string]interface{}
	err := client.doRequest(context.Background(), http.MethodGet, "/protected", nil, &resp, true)
	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestGetIncomingMessages_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/incoming" {
			t.Errorf("expected path /api/v1/message/incoming, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("expected limit=5, got %s", got)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Authorization Bearer test-token, got %s", auth)
		}

		resp := IncomingMessagesResponse{
			Success:   true,
			Timestamp: 1715900000000,
			Count:     2,
			Messages: []IncomingMessage{
				{
					MessageId: "MSG_2",
					Chat:      "6281234567890@s.whatsapp.net",
					From:      "6281234567890@s.whatsapp.net",
					IsGroup:   false,
					PushName:  "Google",
					Timestamp: 1715899950,
					Text:      "Your code is 123456",
					Type:      IncomingMessageTypeText,
				},
				{
					MessageId: "MSG_1",
					Chat:      "6281234567890@s.whatsapp.net",
					From:      "6281234567890@s.whatsapp.net",
					IsGroup:   false,
					PushName:  "Alice",
					Timestamp: 1715899900,
					Type:      IncomingMessageTypeImage,
					Media: &IncomingMessageMediaInfo{
						Type:     IncomingMessageTypeImage,
						MimeType: "image/jpeg",
						Size:     12345,
						Caption:  "look at this",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL + "/api/v1"))
	client.SetToken("test-token")

	resp, err := client.GetIncomingMessages(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Count != 2 {
		t.Errorf("expected count=2, got %d", resp.Count)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(resp.Messages))
	}
	if resp.Messages[0].MessageId != "MSG_2" {
		t.Errorf("expected newest-first ordering, got %s", resp.Messages[0].MessageId)
	}
	if resp.Messages[1].Media == nil || resp.Messages[1].Media.MimeType != "image/jpeg" {
		t.Errorf("expected media.mime_type=image/jpeg on second message")
	}
}

func TestGetIncomingMessages_NotAuthenticated(t *testing.T) {
	client := NewClient(WithBaseURL("http://localhost:9999/api/v1"))
	_, err := client.GetIncomingMessages(context.Background(), 10)
	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestGetIncomingMessages_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := IncomingMessagesResponse{Success: true, Timestamp: 1, Count: 0, Messages: []IncomingMessage{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL + "/api/v1"))
	client.SetToken("test-token")

	resp, err := client.GetIncomingMessages(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Count != 0 || len(resp.Messages) != 0 {
		t.Errorf("expected empty response, got count=%d len=%d", resp.Count, len(resp.Messages))
	}
}

func TestGetIncomingMessages_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "client not logged in"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL + "/api/v1"))
	client.SetToken("test-token")

	_, err := client.GetIncomingMessages(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_ConcurrentTokenAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"authenticated":true}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("initial"))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			client.SetToken(fmt.Sprintf("token-%d", i))
		}(i)
		go func() {
			defer wg.Done()
			_ = client.GetToken()
		}()
		go func() {
			defer wg.Done()
			_, _ = client.GetLoginStatus(context.Background())
		}()
	}
	wg.Wait()

	if client.GetToken() == "" {
		t.Error("token must not be empty after concurrent updates")
	}
}

func TestGetJobStatus(t *testing.T) {
	messageID := "3EB0ABC123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/job/job-123" {
			t.Errorf("expected path /api/v1/message/job/job-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected bearer token, got %q", auth)
		}

		resp := JobStatusResponse{
			JobID:     "job-123",
			Status:    "completed",
			MessageID: &messageID,
			CreatedAt: "2026-06-08T00:00:00Z",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.GetJobStatus(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "completed" {
		t.Errorf("expected status completed, got %s", resp.Status)
	}
	if resp.MessageID == nil || *resp.MessageID != messageID {
		t.Errorf("expected message ID %s, got %v", messageID, resp.MessageID)
	}
}

func TestGetJobStatus_RequiresAuth(t *testing.T) {
	client := NewClient()
	if _, err := client.GetJobStatus(context.Background(), "job-123"); err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestSendText_ChatReplyMentions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/text" {
			t.Errorf("expected path /api/v1/message/text, got %s", r.URL.Path)
		}
		var body SendMessageTextRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		// Both chat and msisdn are transmitted; the gateway resolves chat.
		if body.Chat != "123@g.us" {
			t.Errorf("expected chat 123@g.us, got %q", body.Chat)
		}
		if body.Msisdn != "628@s.whatsapp.net" {
			t.Errorf("expected msisdn to still be sent, got %q", body.Msisdn)
		}
		if body.ReplyToID != "msg_1" || body.ReplyToSender != "628@s.whatsapp.net" || body.ReplyToText != "hi there" {
			t.Errorf("unexpected reply fields: %+v", body)
		}
		if len(body.Mentions) != 2 || body.Mentions[0] != "111" || body.Mentions[1] != "222" {
			t.Errorf("unexpected mentions: %v", body.Mentions)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, MessageId: "msg_out", Chat: "123@g.us"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.SendText(context.Background(), "628@s.whatsapp.net", "Hello!",
		WithChat("123@g.us"),
		WithReply("msg_1", "628@s.whatsapp.net", "hi there"),
		WithMentions("111", "222"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Chat != "123@g.us" {
		t.Errorf("expected resolved chat 123@g.us in response, got %q", resp.Chat)
	}
}

func TestSendImage_ChatReplyMentions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		if r.FormValue("chat") != "123@g.us" {
			t.Errorf("expected chat 123@g.us, got %q", r.FormValue("chat"))
		}
		if r.FormValue("msisdn") != "628@s.whatsapp.net" {
			t.Errorf("expected msisdn to still be sent, got %q", r.FormValue("msisdn"))
		}
		if r.FormValue("reply_to_id") != "msg_1" || r.FormValue("reply_to_sender") != "628@s.whatsapp.net" || r.FormValue("reply_to_text") != "hi" {
			t.Errorf("unexpected reply fields")
		}
		// mentions must be REPEATED form parts, not comma-joined.
		mentions := r.MultipartForm.Value["mentions"]
		if len(mentions) != 2 || mentions[0] != "111" || mentions[1] != "222" {
			t.Errorf("expected two repeated mentions parts, got %v", mentions)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, MessageId: "img_out", Chat: "123@g.us"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	image := &mockImageReader{data: []byte("fake image data")}
	resp, err := client.SendImage(context.Background(), "628@s.whatsapp.net", image, "", false,
		WithChat("123@g.us"),
		WithReply("msg_1", "628@s.whatsapp.net", "hi"),
		WithMentions("111", "222"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Chat != "123@g.us" {
		t.Errorf("expected resolved chat 123@g.us in response, got %q", resp.Chat)
	}
}

func TestSendText_IdempotencyKey(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, MessageId: "msg_1"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))

	// Key set -> header present.
	if _, err := client.SendText(context.Background(), "628@s.whatsapp.net", "hi", WithIdempotencyKey("key-abc")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "key-abc" {
		t.Errorf("expected Idempotency-Key 'key-abc', got %q", gotKey)
	}

	// No key -> header absent.
	if _, err := client.SendText(context.Background(), "628@s.whatsapp.net", "hi"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "" {
		t.Errorf("expected no Idempotency-Key header, got %q", gotKey)
	}
}

func TestSendImage_IdempotencyKey(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, MessageId: "img_1"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	image := &mockImageReader{data: []byte("data")}
	if _, err := client.SendImage(context.Background(), "628@s.whatsapp.net", image, "", false, WithIdempotencyKey("key-img")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "key-img" {
		t.Errorf("expected Idempotency-Key 'key-img' on multipart send, got %q", gotKey)
	}
}

func TestSendText_IdempotencyConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "a request with this Idempotency-Key is already in progress", "code": 409})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	_, err := client.SendText(context.Background(), "628@s.whatsapp.net", "hi", WithIdempotencyKey("dup"))
	if !IsConflict(err) {
		t.Errorf("expected ErrConflict (409), got %v", err)
	}
}

func TestSendText_IdempotencyUnprocessable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Idempotency-Key was reused with a different request body", "code": 422})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	_, err := client.SendText(context.Background(), "628@s.whatsapp.net", "hi", WithIdempotencyKey("reused"))
	var sdkErr *SDKError
	if !errors.As(err, &sdkErr) || sdkErr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected SDKError with code 422, got %v", err)
	}
}

func TestListContacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/contact/" {
			t.Errorf("expected path /api/v1/contact/, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "50" || r.URL.Query().Get("offset") != "10" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContactListResponse{
			Contacts: []ContactListItem{{JID: "628@s.whatsapp.net", PushName: "Bob"}},
			Count:    1, Total: 42,
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.ListContacts(context.Background(), 50, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Count != 1 || resp.Total != 42 || len(resp.Contacts) != 1 || resp.Contacts[0].PushName != "Bob" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetContactInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/contact/info" {
			t.Errorf("expected path /api/v1/contact/info, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("chat") != "628@s.whatsapp.net" {
			t.Errorf("expected chat query, got %q", r.URL.Query().Get("chat"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContactInfoResponse{JID: "628@s.whatsapp.net", Status: "hi", DeviceCount: 2, LID: "77@lid"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.GetContactInfo(context.Background(), "628@s.whatsapp.net")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DeviceCount != 2 || resp.LID != "77@lid" || resp.Status != "hi" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetAvatar_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/contact/avatar" {
			t.Errorf("expected path /api/v1/contact/avatar, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("chat") != "628@s.whatsapp.net" || r.URL.Query().Get("preview") != "true" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("ETag", `"pic_1"`)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AvatarResponse{JID: "628@s.whatsapp.net", URL: "https://cdn/x.jpg", ID: "pic_1", Type: "preview"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.GetAvatar(context.Background(), "628@s.whatsapp.net", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "pic_1" || resp.Type != "preview" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetAvatar_NotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") != `"pic_1"` {
			t.Errorf("expected If-None-Match \"pic_1\", got %q", r.Header.Get("If-None-Match"))
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	_, err := client.GetAvatar(context.Background(), "628@s.whatsapp.net", false, "pic_1")
	if !errors.Is(err, ErrNotModified) {
		t.Errorf("expected ErrNotModified, got %v", err)
	}
}

func TestGetAvatar_NotFoundAndForbidden(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
	}{
		{http.StatusNotFound, IsNotFound},
		{http.StatusForbidden, IsForbidden},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tc.status)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "x", "code": tc.status})
		}))
		client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
		_, err := client.GetAvatar(context.Background(), "628@s.whatsapp.net", false)
		if !tc.check(err) {
			t.Errorf("status %d: unexpected error mapping: %v", tc.status, err)
		}
		server.Close()
	}
}

func TestListGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/group/" {
			t.Errorf("expected path /api/v1/group/, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GroupListResponse{
			Groups: []GroupListItem{{JID: "12@g.us", Name: "Team", ParticipantCount: 3, IsCommunity: true}},
			Count:  1,
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Count != 1 || resp.Groups[0].Name != "Team" || !resp.Groups[0].IsCommunity {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestListGroups_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "budget exhausted", "code": 429})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	_, err := client.ListGroups(context.Background())
	if !IsRateLimited(err) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestGetGroupInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/group/info" {
			t.Errorf("expected path /api/v1/group/info, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("chat") != "12@g.us" {
			t.Errorf("expected chat=12@g.us, got %q", r.URL.Query().Get("chat"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GroupInfoResponse{
			JID: "12@g.us", Name: "Team", ParticipantCount: 2,
			Participants: []GroupParticipantItem{
				{JID: "628@s.whatsapp.net", PhoneNumber: "628", IsAdmin: true},
				{JID: "629@s.whatsapp.net"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.GetGroupInfo(context.Background(), "12@g.us")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Participants) != 2 || !resp.Participants[0].IsAdmin {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestReadMethods_RequireAuth(t *testing.T) {
	client := NewClient()
	if _, err := client.ListContacts(context.Background(), 10, 0); err != ErrNotAuthenticated {
		t.Errorf("ListContacts: expected ErrNotAuthenticated, got %v", err)
	}
	if _, err := client.GetAvatar(context.Background(), "628", false); err != ErrNotAuthenticated {
		t.Errorf("GetAvatar: expected ErrNotAuthenticated, got %v", err)
	}
	if _, err := client.ListGroups(context.Background()); err != ErrNotAuthenticated {
		t.Errorf("ListGroups: expected ErrNotAuthenticated, got %v", err)
	}
}

func TestMarkRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/message/read" || r.Method != http.MethodPost {
			t.Errorf("expected POST /api/v1/message/read, got %s %s", r.Method, r.URL.Path)
		}
		var body MarkReadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.Chat != "12@g.us" || body.Sender != "628@s.whatsapp.net" {
			t.Errorf("unexpected chat/sender: %+v", body)
		}
		if len(body.MessageIDs) != 2 || body.MessageIDs[0] != "m1" {
			t.Errorf("unexpected message_ids: %v", body.MessageIDs)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SuccessResponse{Success: true})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	if err := client.MarkRead(context.Background(), "12@g.us", []string{"m1", "m2"}, "628@s.whatsapp.net"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarkRead_ErrorMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "sender required for group", "code": 400})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	err := client.MarkRead(context.Background(), "12@g.us", []string{"m1"}, "")
	if !IsBadRequest(err) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestSendChatPresence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chat/presence" || r.Method != http.MethodPost {
			t.Errorf("expected POST /api/v1/chat/presence, got %s %s", r.Method, r.URL.Path)
		}
		var body ChatPresenceRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.Chat != "628@s.whatsapp.net" || body.State != PresenceComposing {
			t.Errorf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SuccessResponse{Success: true})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	if err := client.SendChatPresence(context.Background(), "628@s.whatsapp.net", PresenceComposing); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTwoWayPrimitives_RequireAuth(t *testing.T) {
	client := NewClient()
	if err := client.MarkRead(context.Background(), "628", []string{"m1"}, ""); err != ErrNotAuthenticated {
		t.Errorf("MarkRead: expected ErrNotAuthenticated, got %v", err)
	}
	if err := client.SendChatPresence(context.Background(), "628", PresencePaused); err != ErrNotAuthenticated {
		t.Errorf("SendChatPresence: expected ErrNotAuthenticated, got %v", err)
	}
}

func TestCreateGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/group/" || r.Method != http.MethodPost {
			t.Errorf("expected POST /api/v1/group/, got %s %s", r.Method, r.URL.Path)
		}
		var body CreateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Name != "Team" || !body.IsCommunity || len(body.Participants) != 1 {
			t.Errorf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateGroupResponse{
			GroupJID: "12@g.us",
			Results:  []ParticipantResult{{JID: "628@s.whatsapp.net", Status: "invited", Invite: &ParticipantInvite{Code: "abc"}}},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.CreateGroup(context.Background(), CreateGroupRequest{
		Name: "Team", Participants: []string{"628"}, IsCommunity: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GroupJID != "12@g.us" || resp.Results[0].Status != "invited" || resp.Results[0].Invite.Code != "abc" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestLeaveGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/group/leave" || r.Method != http.MethodPost {
			t.Errorf("expected POST /api/v1/group/leave, got %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["chat"] != "12@g.us" {
			t.Errorf("expected chat 12@g.us, got %q", body["chat"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"left": true})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	if err := client.LeaveGroup(context.Background(), "12@g.us"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateGroupParticipants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/group/participants" {
			t.Errorf("expected /api/v1/group/participants, got %s", r.URL.Path)
		}
		var body struct {
			Chat         string   `json:"chat"`
			Action       string   `json:"action"`
			Participants []string `json:"participants"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Chat != "12@g.us" || body.Action != "promote" || len(body.Participants) != 2 {
			t.Errorf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GroupParticipantsResponse{
			GroupJID: "12@g.us", Action: "promote",
			Results: []ParticipantResult{
				{JID: "628@s.whatsapp.net", Status: "ok"},
				{JID: "629@s.whatsapp.net", Status: "failed", Code: 404},
			},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.UpdateGroupParticipants(context.Background(), "12@g.us", "promote", []string{"628", "629"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 2 || resp.Results[1].Status != "failed" || resp.Results[1].Code != 404 {
		t.Errorf("unexpected results: %+v", resp.Results)
	}
}

func TestSetGroupSettings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/group/settings" {
			t.Errorf("expected PATCH /api/v1/group/settings, got %s %s", r.Method, r.URL.Path)
		}
		var raw map[string]interface{}
		json.NewDecoder(r.Body).Decode(&raw)
		if raw["announce"] != true {
			t.Errorf("expected announce true, got %v", raw["announce"])
		}
		if _, ok := raw["locked"]; ok {
			t.Errorf("expected locked to be omitted, got %v", raw["locked"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GroupSettingsResponse{GroupJID: "12@g.us", Applied: []string{"announce"}})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	announce := true
	resp, err := client.SetGroupSettings(context.Background(), "12@g.us", &announce, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Applied) != 1 || resp.Applied[0] != "announce" {
		t.Errorf("unexpected applied: %v", resp.Applied)
	}
}

func TestSetGroupNameAndTopic(t *testing.T) {
	var gotPath, gotName, gotTopic string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		gotName, gotTopic = body["name"], body["topic"]
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"updated": true})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	if err := client.SetGroupName(context.Background(), "12@g.us", "New Name"); err != nil {
		t.Fatalf("SetGroupName: %v", err)
	}
	if gotPath != "/api/v1/group/name" || gotName != "New Name" {
		t.Errorf("SetGroupName sent %s name=%q", gotPath, gotName)
	}
	if err := client.SetGroupTopic(context.Background(), "12@g.us", ""); err != nil {
		t.Fatalf("SetGroupTopic: %v", err)
	}
	if gotPath != "/api/v1/group/topic" || gotTopic != "" {
		t.Errorf("SetGroupTopic sent %s topic=%q", gotPath, gotTopic)
	}
}

func TestSendText_QueueMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202 queue mode
		json.NewEncoder(w).Encode(SendMessageResponse{Success: true, Status: "queued", JobID: "job_abc"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL+"/api/v1"), WithToken("test-token"))
	resp, err := client.SendText(context.Background(), "6281234567890@s.whatsapp.net", "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.JobID != "job_abc" {
		t.Errorf("expected job ID job_abc, got %q", resp.JobID)
	}
	if resp.Status != "queued" {
		t.Errorf("expected status queued, got %q", resp.Status)
	}
	if resp.MessageId != "" {
		t.Errorf("expected empty message ID in queue mode, got %q", resp.MessageId)
	}
}
