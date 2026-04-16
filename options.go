package waga

import (
	"net/http"
	"time"
)

// Option configures the SDK client.
// Options are applied when creating a new client with NewClient.
type Option func(*Client)

// WithBaseURL sets the base URL for the API.
// Use this option to connect to a different WhatsApp Gateway server.
//
// The default base URL is "http://localhost:3000/api/v1".
//
// Example:
//
//	client := waga.NewClient(waga.WithBaseURL("https://api.example.com"))
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client for making API requests.
// Use this option to customize HTTP behavior such as timeouts, redirects,
// transport settings, or proxy configuration.
//
// Example:
//
//	httpClient := &http.Client{
//	    Timeout: 60 * time.Second,
//	    Transport: &http.Transport{...},
//	}
//	client := waga.NewClient(waga.WithHTTPClient(httpClient))
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithTimeout sets the HTTP client timeout for API requests.
// The timeout applies to the entire HTTP request, including connection time,
// redirects, and reading the response body.
//
// The default timeout is 30 seconds.
//
// Example:
//
//	client := waga.NewClient(waga.WithTimeout(60 * time.Second))
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithToken sets a pre-existing JWT token for authentication.
// Use this option if you already have a valid token from a previous registration
// and want to skip the registration step.
//
// The token will be used for all authenticated API requests.
//
// Example:
//
//	client := waga.NewClient(waga.WithToken("your-jwt-token"))
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithUserAgent sets a custom User-Agent header for API requests.
// Use this option to identify your application in API requests.
//
// The default User-Agent is "WhatsApp-Gateway-SDK-Go/1.0".
//
// Example:
//
//	client := waga.NewClient(waga.WithUserAgent("MyApp/1.0"))
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// Default configuration constants

const (
	// DefaultBaseURL is the default API endpoint
	DefaultBaseURL = "http://localhost:3000/api/v1"
	// DefaultTimeout is the default HTTP request timeout
	DefaultTimeout = 30 * time.Second
	// DefaultUserAgent is the default User-Agent header
	DefaultUserAgent = "WhatsApp-Gateway-SDK-Go/1.0"
)
