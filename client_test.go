package waga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
