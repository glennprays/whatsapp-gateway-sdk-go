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

// Client is the main SDK client for interacting with the WhatsApp Gateway API.
// It provides methods for authentication, messaging, webhook management, and session control.
//
// The client is safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	userAgent  string
}

// NewClient creates a new SDK client with the given options.
//
// The client can be configured with functional options:
//
//	client := waga.NewClient(
//	    waga.WithBaseURL("https://api.example.com"),
//	    waga.WithTimeout(60*time.Second),
//	    waga.WithToken("existing-jwt-token"),
//	)
//
// By default, the client uses:
//   - Base URL: http://localhost:3000/api/v1
//   - Timeout: 30 seconds
//   - User-Agent: WhatsApp-Gateway-SDK-Go/1.0
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

// SetToken sets the JWT token for authentication.
// Use this method if you already have a valid token from a previous registration.
// The token will be used for all subsequent API requests that require authentication.
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetToken returns the current JWT token stored in the client.
// Returns an empty string if no token has been set.
func (c *Client) GetToken() string {
	return c.token
}

// Register registers a new phone number with the WhatsApp Gateway service.
// It retrieves and stores a JWT token for subsequent API requests.
//
// The phoneNumber should be in international format without the "+" symbol
// (e.g., "6281234567890"). The secretKey is provided by the gateway service.
//
// After successful registration, the token is automatically stored in the client
// and used for all authenticated requests.
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

// GetQRCode generates a QR code for WhatsApp login.
// The format parameter specifies the response format:
//   - "json": returns a base64-encoded PNG image
//   - "html": returns an HTML img tag with the embedded QR code
//
// The QR code can be displayed to users for scanning with the WhatsApp mobile app
// to link their account. The QR code expires after the duration specified in ExpiresIn.
//
// Requires authentication (call Register or SetToken first).
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

// GetPairCode generates a pair code for WhatsApp login.
// The pair code is an 8-character code that can be used to link the device
// without scanning a QR code. Users can enter the code in WhatsApp > Linked Devices.
//
// The pair code expires after the duration specified in ExpiresIn.
//
// Requires authentication (call Register or SetToken first).
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

// GetLoginStatus returns the current WhatsApp session status.
// It checks whether the WhatsApp session is authenticated and active.
//
// Requires authentication (call Register or SetToken first).
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

// Logout logs out from the current WhatsApp session.
// This disconnects the WhatsApp account and invalidates the session.
// After logging out, you'll need to authenticate again using GetQRCode or GetPairCode.
//
// Requires authentication (call Register or SetToken first).
func (c *Client) Logout(ctx context.Context) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPost, "/logout", nil, &resp, true)
}

// Reconnect attempts to reconnect to the existing WhatsApp session.
// Use this method if the connection to WhatsApp has been lost but the session
// is still valid. It attempts to restore the connection without requiring re-authentication.
//
// Requires authentication (call Register or SetToken first).
func (c *Client) Reconnect(ctx context.Context) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPost, "/session/reconnect", nil, &resp, true)
}

// SendText sends a text message to the specified recipient.
//
// The msisdn parameter should be the recipient's phone number in WhatsApp JID format.
// Use the FormatMSISDN() helper function to convert a phone number to the correct format:
//
//	msisdn := waga.FormatMSISDN("6281234567890")  // "6281234567890@s.whatsapp.net"
//
// For group messages, use the group ID in the format "groupId@g.us".
//
// Requires authentication (call Register or SetToken first).
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

// SendImage sends an image message to the specified recipient.
//
// The msisdn parameter should be the recipient's phone number in WhatsApp JID format.
// The image parameter is an io.Reader containing the image data (JPEG, PNG, etc.).
// The caption parameter is optional text to accompany the image.
// The isViewOnce parameter, if true, sends the image as a view-once media that disappears
// after being viewed.
//
// Example:
//
//	file, _ := os.Open("image.jpg")
//	defer file.Close()
//	resp, err := client.SendImage(ctx, msisdn, file, "Check this out!", false)
//
// Requires authentication (call Register or SetToken first).
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

// EditMessage edits a previously sent message.
// It replaces the message content with newMessage.
//
// The msisdn is the recipient's phone number, and messageID is the ID of the message
// to edit (returned by SendText or SendImage).
//
// Requires authentication (call Register or SetToken first).
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

// DeleteMessage deletes a previously sent message.
// The message will be removed from the chat for both sender and recipient.
//
// The msisdn is the recipient's phone number, and messageID is the ID of the message
// to delete (returned by SendText or SendImage).
//
// Requires authentication (call Register or SetToken first).
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

// React adds a reaction emoji to a previously sent message.
// The emoji will appear on the message for all participants in the chat.
//
// The msisdn is the recipient's phone number, messageID is the ID of the message
// to react to, and emoji is the emoji character(s) to use (e.g., "👍", "❤️").
//
// Requires authentication (call Register or SetToken first).
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

// RegisterWebhook registers a URL to receive webhook events.
// Webhooks allow you to receive real-time notifications for incoming and outgoing messages.
//
// The url parameter is the endpoint where webhook events will be sent.
// The hmacSecret parameter is optional but highly recommended for verifying webhook
// signatures. If provided, all webhook payloads will include an X-Webhook-Signature
// header that can be verified using the WebhookVerifier.
//
// Requires authentication (call Register or SetToken first).
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

// GetWebhook returns the currently registered webhook URL.
// Use this to check if a webhook is configured and retrieve its URL.
//
// Requires authentication (call Register or SetToken first).
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

// UnregisterWebhook removes the currently registered webhook URL.
// After calling this method, webhook events will no longer be sent to the previously
// configured endpoint.
//
// Requires authentication (call Register or SetToken first).
func (c *Client) UnregisterWebhook(ctx context.Context) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodDelete, "/webhook", nil, &resp, true)
}

// Health checks the health status of the gateway service.
// This method does not require authentication and can be used to verify
// that the service is running and accessible.
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

// LoginStatus represents the response from GetLoginStatus.
// It indicates whether the WhatsApp session is currently authenticated.
type LoginStatus struct {
	// Authenticated is true if the session is active and authenticated
	Authenticated bool `json:"authenticated"`
}

// SuccessResponse represents a generic success response from API operations.
type SuccessResponse struct {
	// Success indicates whether the operation completed successfully
	Success bool `json:"success"`
}

// WebhookResponse represents the response from GetWebhook.
type WebhookResponse struct {
	// URL is the currently registered webhook endpoint
	URL string `json:"url"`
}

// HealthResponse represents the response from Health.
type HealthResponse struct {
	// Status is the health status of the service (e.g., "ok", "degraded")
	Status string `json:"status"`
	// Timestamp is the ISO 8601 timestamp of the health check
	Timestamp string `json:"timestamp"`
}

// Outgoing event type constants for convenience
const (
	// EventMessageQueued is emitted when a message is added to the queue
	EventMessageQueued = OutgoingEventMessageMessageQueued
	// EventMessageSent is emitted when a message is successfully sent
	EventMessageSent = OutgoingEventMessageMessageSent
	// EventMessageFailed is emitted when a message fails to send
	EventMessageFailed = OutgoingEventMessageMessageFailed
	// EventMessageIncoming is emitted when a message is received
	EventMessageIncoming = IncomingEventMessageMessageIncoming
)

// Incoming message type constants for convenience
const (
	// MessageTypeText represents a text message
	MessageTypeText = IncomingMessageTypeText
	// MessageTypeImage represents an image message
	MessageTypeImage = IncomingMessageTypeImage
	// MessageTypeVideo represents a video message
	MessageTypeVideo = IncomingMessageTypeVideo
	// MessageTypeAudio represents an audio message
	MessageTypeAudio = IncomingMessageTypeAudio
	// MessageTypeDocument represents a document message
	MessageTypeDocument = IncomingMessageTypeDocument
)

// Helper functions for working with MSISDNs

// FormatMSISDN formats a phone number to WhatsApp JID format for individual contacts.
// If the phoneNumber already contains "@", it is returned as-is.
// Otherwise, "@s.whatsapp.net" is appended to create a valid JID.
//
// Example:
//
//	msisdn := waga.FormatMSISDN("6281234567890")  // "6281234567890@s.whatsapp.net"
func FormatMSISDN(phoneNumber string) string {
	if strings.Contains(phoneNumber, "@") {
		return phoneNumber
	}
	return phoneNumber + "@s.whatsapp.net"
}

// FormatGroupID formats a group ID to WhatsApp group JID format.
// If the groupID already contains "@", it is returned as-is.
// Otherwise, "@g.us" is appended to create a valid group JID.
//
// Example:
//
//	groupID := waga.FormatGroupID("1234567890")  // "1234567890@g.us"
func FormatGroupID(groupID string) string {
	if strings.Contains(groupID, "@") {
		return groupID
	}
	return groupID + "@g.us"
}
