package waga

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// Client is the main SDK client for interacting with the WhatsApp Gateway API
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	userAgent  string
}

// NewClient creates a new SDK client with the given options
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:   DefaultBaseURL,
		userAgent: DefaultUserAgent,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// SetToken sets the JWT token for authentication
// Use this if you already have a valid token
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetToken returns the current JWT token
func (c *Client) GetToken() string {
	return c.token
}

// Register registers a new phone number and retrieves a JWT token
// The token is automatically stored in the client for subsequent requests
func (c *Client) Register(ctx context.Context, phoneNumber, secretKey string) (*RegisterResponse, error) {
	reqBody := RegisterRequest{
		PhoneNumber: phoneNumber,
		SecretKey:   secretKey,
	}

	var resp RegisterResponse
	if err := c.doRequest(ctx, http.MethodPost, "/register", reqBody, &resp, false); err != nil {
		return nil, err
	}

	// Store token for subsequent requests
	c.token = resp.Token
	return &resp, nil
}

// GetQRCode generates a QR code for WhatsApp login
// Format can be "json" (returns base64) or "html" (returns HTML img tag)
func (c *Client) GetQRCode(ctx context.Context, format string) (*LoginQrResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	var resp LoginQrResponse
	if err := c.doRequest(ctx, http.MethodPost, "/login/qr_code/"+format, nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPairCode generates a pair code for WhatsApp login
func (c *Client) GetPairCode(ctx context.Context) (*LoginPairResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	var resp LoginPairResponse
	if err := c.doRequest(ctx, http.MethodPost, "/login/pair_code", nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetLoginStatus returns the current WhatsApp session status
func (c *Client) GetLoginStatus(ctx context.Context) (*LoginStatus, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	var resp LoginStatus
	if err := c.doRequest(ctx, http.MethodGet, "/login/status", nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logout logs out from the current WhatsApp session
func (c *Client) Logout(ctx context.Context) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPost, "/logout", nil, &resp, true)
}

// Reconnect attempts to reconnect to the existing WhatsApp session
func (c *Client) Reconnect(ctx context.Context) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPost, "/session/reconnect", nil, &resp, true)
}

// SendText sends a text message to the specified recipient
// msisdn should be in format: "6281234567890@s.whatsapp.net" for individual or group ID
func (c *Client) SendText(ctx context.Context, msisdn, message string) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	reqBody := SendMessageTextRequest{
		Msisdn:  msisdn,
		Message: message,
	}

	var resp SendMessageResponse
	if err := c.doRequest(ctx, http.MethodPost, "/message/text", reqBody, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SendImage sends an image message to the specified recipient
// msisdn should be in format: "6281234567890@s.whatsapp.net" for individual or group ID
// image is an io.Reader containing the image data
// caption and isViewOnce are optional
func (c *Client) SendImage(ctx context.Context, msisdn string, image io.Reader, caption string, isViewOnce bool) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add msisdn field
	if err := writer.WriteField("msisdn", msisdn); err != nil {
		return nil, fmt.Errorf("failed to write msisdn field: %w", err)
	}

	// Add caption if provided
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return nil, fmt.Errorf("failed to write caption field: %w", err)
		}
	}

	// Add is_view_once if true
	if isViewOnce {
		if err := writer.WriteField("is_view_once", "true"); err != nil {
			return nil, fmt.Errorf("failed to write is_view_once field: %w", err)
		}
	}

	// Add image file
	part, err := writer.CreateFormFile("image", "image.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create image form file: %w", err)
	}
	if _, err := io.Copy(part, image); err != nil {
		return nil, fmt.Errorf("failed to copy image data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	url := c.baseURL + "/message/image"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseError(body, resp.StatusCode)
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// EditMessage edits a previously sent message
func (c *Client) EditMessage(ctx context.Context, msisdn, messageID, newMessage string) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	reqBody := MessageEditRequest{
		Msisdn:     msisdn,
		MessageId:  messageID,
		NewMessage: newMessage,
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPut, "/message", reqBody, &resp, true)
}

// DeleteMessage deletes a previously sent message
func (c *Client) DeleteMessage(ctx context.Context, msisdn, messageID string) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	reqBody := MessageDeleteRequest{
		Msisdn:    msisdn,
		MessageId: messageID,
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodDelete, "/message", reqBody, &resp, true)
}

// React adds a reaction to a previously sent message
func (c *Client) React(ctx context.Context, msisdn, messageID, emoji string) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	reqBody := MessageReactRequest{
		Msisdn:    msisdn,
		MessageId: messageID,
		Emoji:     emoji,
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPost, "/message/react", reqBody, &resp, true)
}

// RegisterWebhook registers a URL to receive webhook events
// hmacSecret is optional but recommended for signature verification
func (c *Client) RegisterWebhook(ctx context.Context, url, hmacSecret string) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	reqBody := WebhookRegisterRequest{
		Url:        url,
		HmacSecret: &hmacSecret,
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPost, "/webhook", reqBody, &resp, true)
}

// GetWebhook returns the currently registered webhook URL
func (c *Client) GetWebhook(ctx context.Context) (*WebhookResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	var resp WebhookResponse
	if err := c.doRequest(ctx, http.MethodGet, "/webhook", nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UnregisterWebhook removes the currently registered webhook URL
func (c *Client) UnregisterWebhook(ctx context.Context) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodDelete, "/webhook", nil, &resp, true)
}

// Health checks the health status of the gateway service
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.doRequest(ctx, http.MethodGet, "/health", nil, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// doRequest performs an HTTP request and unmarshals the response
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}, requireAuth bool) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if requireAuth {
		if c.token == "" {
			return ErrNotAuthenticated
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseError(respBody, resp.StatusCode)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

func (c *Client) checkAuth() error {
	if c.token == "" {
		return ErrNotAuthenticated
	}
	return nil
}

// Additional response types

// LoginStatus represents the response from GetLoginStatus
type LoginStatus struct {
	Authenticated bool `json:"authenticated"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool `json:"success"`
}

// WebhookResponse represents the response from GetWebhook
type WebhookResponse struct {
	URL string `json:"url"`
}

// HealthResponse represents the response from Health
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// Outgoing event types
const (
	EventMessageQueued   = OutgoingEventMessageMessageQueued
	EventMessageSent     = OutgoingEventMessageMessageSent
	EventMessageFailed   = OutgoingEventMessageMessageFailed
	EventMessageIncoming = IncomingEventMessageMessageIncoming
)

// Incoming message types
const (
	MessageTypeText     = IncomingMessageTypeText
	MessageTypeImage    = IncomingMessageTypeImage
	MessageTypeVideo    = IncomingMessageTypeVideo
	MessageTypeAudio    = IncomingMessageTypeAudio
	MessageTypeDocument = IncomingMessageTypeDocument
)

// Helper functions for working with MSISDNs

// FormatMSISDN formats a phone number to WhatsApp JID format
// e.g., "6281234567890" -> "6281234567890@s.whatsapp.net"
func FormatMSISDN(phoneNumber string) string {
	if strings.Contains(phoneNumber, "@") {
		return phoneNumber
	}
	return phoneNumber + "@s.whatsapp.net"
}

// FormatGroupID formats a group ID to WhatsApp group JID format
// e.g., "1234567890" -> "1234567890@g.us"
func FormatGroupID(groupID string) string {
	if strings.Contains(groupID, "@") {
		return groupID
	}
	return groupID + "@g.us"
}
