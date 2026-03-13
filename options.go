package waga

import (
	"net/http"
	"time"
)

// Option configures the SDK client
type Option func(*Client)

// WithBaseURL sets the base URL for the API
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithToken sets a pre-existing JWT token
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithUserAgent sets a custom User-Agent header
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// Default options
const (
	DefaultBaseURL   = "http://localhost:3000/api/v1"
	DefaultTimeout   = 30 * time.Second
	DefaultUserAgent = "WhatsApp-Gateway-SDK-Go/1.0"
)
