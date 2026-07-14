package waga

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// SDKError represents an error returned by the WhatsApp Gateway API.
// It contains both an HTTP status code and a human-readable error message.
type SDKError struct {
	// Code is the HTTP status code or API error code
	Code int `json:"code"`
	// Message is the error message describing what went wrong
	Message string `json:"error"`
	// TraceID is the gateway's X-Trace-ID for this request, if the response
	// carried one. Use it to correlate the failure with gateway logs.
	TraceID string `json:"-"`
}

func (e *SDKError) Error() string {
	if e.TraceID != "" {
		return fmt.Sprintf("sdk error: code=%d, message=%s, trace_id=%s", e.Code, e.Message, e.TraceID)
	}
	return fmt.Sprintf("sdk error: code=%d, message=%s", e.Code, e.Message)
}

// Is implements the errors.Is interface for error comparison.
// It allows SDKError instances to be compared using errors.Is().
//
// Example:
//
//	if errors.Is(err, waga.ErrUnauthorized) {
//	    // Handle unauthorized error
//	}
func (e *SDKError) Is(target error) bool {
	t, ok := target.(*SDKError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Common API errors
// These predefined errors can be used for comparison with errors.Is().
var (
	// ErrUnauthorized is returned when authentication fails (401)
	ErrUnauthorized = &SDKError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	// ErrBadRequest is returned for invalid requests (400)
	ErrBadRequest = &SDKError{Code: http.StatusBadRequest, Message: "bad request"}
	// ErrForbidden is returned when access is denied (403)
	ErrForbidden = &SDKError{Code: http.StatusForbidden, Message: "forbidden"}
	// ErrNotFound is returned when a resource is not found (404)
	ErrNotFound = &SDKError{Code: http.StatusNotFound, Message: "not found"}
	// ErrConflict is returned for conflicting requests (409)
	ErrConflict = &SDKError{Code: http.StatusConflict, Message: "conflict"}
	// ErrGone is returned when a resource is no longer available (410), e.g.
	// GetGroupInviteInfo on a revoked invite link.
	ErrGone = &SDKError{Code: http.StatusGone, Message: "gone"}
	// ErrRateLimited is returned when rate limit is exceeded (429)
	ErrRateLimited = &SDKError{Code: http.StatusTooManyRequests, Message: "rate limited"}
	// ErrNotModified is returned by GetAvatar when the caller-supplied prior
	// avatar id (If-None-Match) still matches — the picture is unchanged (304).
	ErrNotModified = &SDKError{Code: http.StatusNotModified, Message: "not modified"}
	// ErrInternalServer is returned for server errors (500)
	ErrInternalServer = &SDKError{Code: http.StatusInternalServerError, Message: "internal server error"}
	// ErrInvalidSignature is returned when webhook signature verification fails
	ErrInvalidSignature = errors.New("invalid webhook signature")
	// ErrUnknownWebhookEvent is returned by ParseWebhook when the payload's
	// event field is not a recognized webhook event type.
	ErrUnknownWebhookEvent = errors.New("unknown webhook event")
	// ErrNotAuthenticated is returned when trying to use an authenticated method without setting a token
	ErrNotAuthenticated = errors.New("client not authenticated, call Register() or SetToken() first")
)

// parseError attempts to parse an error response from the API.
// If the response body contains a valid error JSON, it returns an SDKError.
// Otherwise, it returns a generic SDKError with the status code and raw body.
func parseError(body []byte, statusCode int, traceID string) error {
	var apiErr SDKError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return a generic one
		apiErr = SDKError{
			Code:    statusCode,
			Message: string(body),
		}
	}
	// Tolerate gateway builds that serialize the error as the Go field names
	// {"Status","Message"} instead of the documented {"code","error"} shape, so
	// the human-readable message is not silently dropped against an older gateway.
	if apiErr.Message == "" || apiErr.Code == 0 {
		var alt struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &alt) == nil {
			if apiErr.Message == "" {
				apiErr.Message = alt.Message
			}
			if apiErr.Code == 0 {
				apiErr.Code = alt.Status
			}
		}
	}
	if apiErr.Code == 0 {
		apiErr.Code = statusCode
	}
	apiErr.TraceID = traceID
	return &apiErr
}

// NewSDKError creates a new SDKError with the given code and message.
// Use this function to create custom SDK errors with specific codes and messages.
//
// Example:
//
//	err := waga.NewSDKError(418, "I'm a teapot")
func NewSDKError(code int, message string) *SDKError {
	return &SDKError{Code: code, Message: message}
}

// IsUnauthorized checks if the error is an unauthorized error (HTTP 401).
// Returns true if err matches ErrUnauthorized.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsBadRequest checks if the error is a bad request error (HTTP 400).
// Returns true if err matches ErrBadRequest.
func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

// IsNotFound checks if the error is a not found error (HTTP 404).
// Returns true if err matches ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsRateLimited checks if the error is a rate limit error (HTTP 429).
// Returns true if err matches ErrRateLimited.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsForbidden checks if the error is a forbidden error (HTTP 403).
// Returns true if err matches ErrForbidden.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsConflict checks if the error is a conflict error (HTTP 409).
// Returns true if err matches ErrConflict.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsGone checks if the error is a gone error (HTTP 410).
// Returns true if err matches ErrGone.
func IsGone(err error) bool {
	return errors.Is(err, ErrGone)
}

// IsInternalServer checks if the error is an internal server error (HTTP 500).
// Returns true if err matches ErrInternalServer.
func IsInternalServer(err error) bool {
	return errors.Is(err, ErrInternalServer)
}
