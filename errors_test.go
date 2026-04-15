package waga

import (
	"encoding/json"
	"testing"
)

// TestParseError_ValidErrorResponse tests parsing a valid error response with code and message
func TestParseError_ValidErrorResponse(t *testing.T) {
	body := []byte(`{"code": 404, "error": "resource not found"}`)
	err := parseError(body, 404)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}

	if sdkErr.Code != 404 {
		t.Errorf("expected code 404, got %d", sdkErr.Code)
	}
	if sdkErr.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got '%s'", sdkErr.Message)
	}
}

// TestParseError_MalformedJSON tests parsing with malformed JSON (fallback to raw body)
func TestParseError_MalformedJSON(t *testing.T) {
	body := []byte(`this is not valid json`)
	err := parseError(body, 500)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}

	if sdkErr.Code != 500 {
		t.Errorf("expected code 500, got %d", sdkErr.Code)
	}
	if sdkErr.Message != string(body) {
		t.Errorf("expected message to be raw body, got '%s'", sdkErr.Message)
	}
}

// TestParseError_EmptyBody tests parsing with empty response body
func TestParseError_EmptyBody(t *testing.T) {
	body := []byte{}
	err := parseError(body, 503)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}

	if sdkErr.Code != 503 {
		t.Errorf("expected code 503, got %d", sdkErr.Code)
	}
}

// TestParseError_MissingCodeField tests error response with missing code field
func TestParseError_MissingCodeField(t *testing.T) {
	body := []byte(`{"error": "something went wrong"}`)
	err := parseError(body, 0)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}

	// Should use statusCode from parameter when code is 0
	if sdkErr.Code != 0 {
		t.Errorf("expected code 0, got %d", sdkErr.Code)
	}
	if sdkErr.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got '%s'", sdkErr.Message)
	}
}

// TestParseError_MissingMessageField tests error response with missing message field
func TestParseError_MissingMessageField(t *testing.T) {
	body := []byte(`{"code": 400}`)
	err := parseError(body, 0)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}

	if sdkErr.Code != 400 {
		t.Errorf("expected code 400, got %d", sdkErr.Code)
	}
}

// TestParseError_HTTPStatusCodes tests parsing errors with various HTTP status codes
func TestParseError_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"bad request", 400, `{"code": 400, "error": "bad request"}`},
		{"unauthorized", 401, `{"code": 401, "error": "unauthorized"}`},
		{"forbidden", 403, `{"code": 403, "error": "forbidden"}`},
		{"not found", 404, `{"code": 404, "error": "not found"}`},
		{"conflict", 409, `{"code": 409, "error": "conflict"}`},
		{"rate limited", 429, `{"code": 429, "error": "rate limited"}`},
		{"internal server error", 500, `{"code": 500, "error": "internal server error"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseError([]byte(tt.body), tt.statusCode)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			sdkErr, ok := err.(*SDKError)
			if !ok {
				t.Fatalf("expected *SDKError, got %T", err)
			}

			if sdkErr.Code != tt.statusCode {
				t.Errorf("expected code %d, got %d", tt.statusCode, sdkErr.Code)
			}
		})
	}
}

// TestSDKError_Is_SameCode tests that Is returns true for same code
func TestSDKError_Is_SameCode(t *testing.T) {
	err1 := &SDKError{Code: 404, Message: "not found"}
	err2 := &SDKError{Code: 404, Message: "different message"}

	if !err1.Is(err2) {
		t.Error("expected Is to return true for same code")
	}
}

// TestSDKError_Is_DifferentCode tests that Is returns false for different code
func TestSDKError_Is_DifferentCode(t *testing.T) {
	err1 := &SDKError{Code: 404, Message: "not found"}
	err2 := &SDKError{Code: 500, Message: "internal server error"}

	if err1.Is(err2) {
		t.Error("expected Is to return false for different code")
	}
}

// TestSDKError_Is_NonSDKErrorTarget tests that Is returns false for non-SDKError target
func TestSDKError_Is_NonSDKErrorTarget(t *testing.T) {
	sdkErr := &SDKError{Code: 404, Message: "not found"}
	standardErr := &SDKError{Code: 500, Message: "different type comparison"}

	if sdkErr.Is(standardErr) {
		t.Error("expected Is to return false for different error code")
	}

	// Test with completely different error type
	genericErr := NewSDKError(500, "server error")
	if sdkErr.Is(genericErr) {
		t.Error("expected Is to return false for different error")
	}
}

// TestSDKError_Is_PredefinedErrors tests comparison with predefined errors
func TestSDKError_Is_PredefinedErrors(t *testing.T) {
	customErr := &SDKError{Code: 404, Message: "custom not found"}

	if !customErr.Is(ErrNotFound) {
		t.Error("expected custom 404 error to match ErrNotFound")
	}

	// The Is method compares codes, so this should work
	if !ErrNotFound.Is(customErr) {
		t.Error("expected ErrNotFound.Is to work with custom 404 error")
	}
}

// TestIsBadRequest tests the IsBadRequest error checker
func TestIsBadRequest(t *testing.T) {
	err := ErrBadRequest
	if !IsBadRequest(err) {
		t.Error("expected IsBadRequest to return true for ErrBadRequest")
	}

	otherErr := &SDKError{Code: 404, Message: "not found"}
	if IsBadRequest(otherErr) {
		t.Error("expected IsBadRequest to return false for 404 error")
	}
}

// TestIsNotFound tests the IsNotFound error checker
func TestIsNotFound(t *testing.T) {
	err := ErrNotFound
	if !IsNotFound(err) {
		t.Error("expected IsNotFound to return true for ErrNotFound")
	}

	otherErr := &SDKError{Code: 400, Message: "bad request"}
	if IsNotFound(otherErr) {
		t.Error("expected IsNotFound to return false for 400 error")
	}
}

// TestIsRateLimited tests the IsRateLimited error checker
func TestIsRateLimited(t *testing.T) {
	err := ErrRateLimited
	if !IsRateLimited(err) {
		t.Error("expected IsRateLimited to return true for ErrRateLimited")
	}

	otherErr := &SDKError{Code: 404, Message: "not found"}
	if IsRateLimited(otherErr) {
		t.Error("expected IsRateLimited to return false for 404 error")
	}
}

// TestIsForbidden tests the IsForbidden error checker
func TestIsForbidden(t *testing.T) {
	err := ErrForbidden
	if !IsForbidden(err) {
		t.Error("expected IsForbidden to return true for ErrForbidden")
	}

	otherErr := &SDKError{Code: 404, Message: "not found"}
	if IsForbidden(otherErr) {
		t.Error("expected IsForbidden to return false for 404 error")
	}
}

// TestIsConflict tests the IsConflict error checker
func TestIsConflict(t *testing.T) {
	err := ErrConflict
	if !IsConflict(err) {
		t.Error("expected IsConflict to return true for ErrConflict")
	}

	otherErr := &SDKError{Code: 404, Message: "not found"}
	if IsConflict(otherErr) {
		t.Error("expected IsConflict to return false for 404 error")
	}
}

// TestIsInternalServer tests the IsInternalServer error checker
func TestIsInternalServer(t *testing.T) {
	err := ErrInternalServer
	if !IsInternalServer(err) {
		t.Error("expected IsInternalServer to return true for ErrInternalServer")
	}

	otherErr := &SDKError{Code: 404, Message: "not found"}
	if IsInternalServer(otherErr) {
		t.Error("expected IsInternalServer to return false for 404 error")
	}
}

// TestNewSDKError tests creating a new SDKError
func TestNewSDKError_Create(t *testing.T) {
	err := NewSDKError(418, "I'm a teapot")

	if err.Code != 418 {
		t.Errorf("expected code 418, got %d", err.Code)
	}
	if err.Message != "I'm a teapot" {
		t.Errorf("expected message 'I'm a teapot', got '%s'", err.Message)
	}
}

// TestNewSDKError_ErrorString tests the Error() string format
func TestNewSDKError_ErrorString(t *testing.T) {
	err := NewSDKError(404, "resource not found")
	expected := "sdk error: code=404, message=resource not found"

	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

// TestSDKError_JSONRoundTrip tests JSON marshaling and unmarshaling of SDKError
func TestSDKError_JSONRoundTrip(t *testing.T) {
	original := &SDKError{Code: 403, Message: "access denied"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var restored SDKError
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if restored.Code != original.Code {
		t.Errorf("expected code %d, got %d", original.Code, restored.Code)
	}
	if restored.Message != original.Message {
		t.Errorf("expected message '%s', got '%s'", original.Message, restored.Message)
	}
}

// TestIsUnauthorized tests the IsUnauthorized error checker
func TestIsUnauthorized(t *testing.T) {
	err := ErrUnauthorized
	if !IsUnauthorized(err) {
		t.Error("expected IsUnauthorized to return true for ErrUnauthorized")
	}

	otherErr := &SDKError{Code: 404, Message: "not found"}
	if IsUnauthorized(otherErr) {
		t.Error("expected IsUnauthorized to return false for 404 error")
	}

	// Test with custom 401 error
	custom401 := &SDKError{Code: 401, Message: "custom unauthorized"}
	if !IsUnauthorized(custom401) {
		t.Error("expected IsUnauthorized to return true for custom 401 error")
	}
}
