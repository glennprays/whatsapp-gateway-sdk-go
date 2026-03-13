package waga

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// SDKError represents an error returned by the API
type SDKError struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
}

func (e *SDKError) Error() string {
	return fmt.Sprintf("sdk error: code=%d, message=%s", e.Code, e.Message)
}

// Is implements errors.Is interface for comparison
func (e *SDKError) Is(target error) bool {
	t, ok := target.(*SDKError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Common API errors
var (
	ErrUnauthorized     = &SDKError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrBadRequest       = &SDKError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrForbidden        = &SDKError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrNotFound         = &SDKError{Code: http.StatusNotFound, Message: "not found"}
	ErrConflict         = &SDKError{Code: http.StatusConflict, Message: "conflict"}
	ErrRateLimited      = &SDKError{Code: http.StatusTooManyRequests, Message: "rate limited"}
	ErrInternalServer   = &SDKError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrNotAuthenticated = errors.New("client not authenticated, call Register() or SetToken() first")
)

// parseError attempts to parse an error response from the API
func parseError(body []byte, statusCode int) error {
	var apiErr SDKError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return a generic one
		apiErr = SDKError{
			Code:    statusCode,
			Message: string(body),
		}
	}
	if apiErr.Code == 0 {
		apiErr.Code = statusCode
	}
	return &apiErr
}

// NewSDKError creates a new SDKError with the given code and message
func NewSDKError(code int, message string) *SDKError {
	return &SDKError{Code: code, Message: message}
}

// IsUnauthorized checks if the error is an unauthorized error
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsBadRequest checks if the error is a bad request error
func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsRateLimited checks if the error is a rate limit error
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}
