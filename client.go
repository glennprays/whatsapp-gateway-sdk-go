package waga

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Client is the main SDK client for interacting with the WhatsApp Gateway API.
// It provides methods for authentication, messaging, webhook management, and session control.
//
// The client is safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex // guards token
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
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// GetToken returns the current JWT token stored in the client.
// Returns an empty string if no token has been set.
func (c *Client) GetToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
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
	c.SetToken(resp.Token)
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
func (c *Client) SendText(ctx context.Context, msisdn, message string, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)
	reqBody := SendMessageTextRequest{
		Chat:          cfg.chat,
		Msisdn:        msisdn,
		Message:       message,
		ReplyToID:     cfg.replyToID,
		ReplyToSender: cfg.replyToSender,
		ReplyToText:   cfg.replyToText,
		Mentions:      cfg.mentions,
	}

	var resp SendMessageResponse
	if err := c.doRequest(ctx, http.MethodPost, "/message/text", reqBody, &resp, true, cfg.headers()...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// writeSendContext writes the shared chat/reply/mentions multipart form fields
// carried by every media send. mentions are written as repeated fields (one
// "mentions" part per entry) because the gateway binds them to a []string;
// comma-joining would be parsed as a single mention.
func writeSendContext(w *multipart.Writer, cfg sendConfig) error {
	fields := []struct{ name, value string }{
		{"chat", cfg.chat},
		{"reply_to_id", cfg.replyToID},
		{"reply_to_sender", cfg.replyToSender},
		{"reply_to_text", cfg.replyToText},
	}
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		if err := w.WriteField(f.name, f.value); err != nil {
			return fmt.Errorf("failed to write %s field: %w", f.name, err)
		}
	}
	for _, m := range cfg.mentions {
		if err := w.WriteField("mentions", m); err != nil {
			return fmt.Errorf("failed to write mentions field: %w", err)
		}
	}
	return nil
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
func (c *Client) SendImage(ctx context.Context, msisdn string, image io.Reader, caption string, isViewOnce bool, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add msisdn field
	if err := writer.WriteField("msisdn", msisdn); err != nil {
		return nil, fmt.Errorf("failed to write msisdn field: %w", err)
	}

	// Add chat/reply/mentions context
	if err := writeSendContext(writer, cfg); err != nil {
		return nil, err
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
	req.Header.Set("Authorization", "Bearer "+c.GetToken())
	req.Header.Set("User-Agent", c.userAgent)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}
	setHeaders(req, cfg.headers())

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
		return nil, parseError(body, resp.StatusCode, resp.Header.Get(TraceIDHeader))
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// SendLocation sends a location message to the specified recipient.
//
// The msisdn parameter should be the recipient's phone number in WhatsApp JID format.
// The latitude and longitude specify the geographic coordinates.
// The name and address parameters are optional metadata for the location pin.
//
// Requires authentication (call Register or SetToken first).
func (c *Client) SendLocation(ctx context.Context, msisdn string, latitude, longitude float64, name, address string, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)
	reqBody := SendLocationMessageRequest{
		Chat:          cfg.chat,
		Msisdn:        msisdn,
		Latitude:      latitude,
		Longitude:     longitude,
		Name:          name,
		Address:       address,
		ReplyToID:     cfg.replyToID,
		ReplyToSender: cfg.replyToSender,
		ReplyToText:   cfg.replyToText,
		Mentions:      cfg.mentions,
	}

	var resp SendMessageResponse
	if err := c.doRequest(ctx, http.MethodPost, "/message/location", reqBody, &resp, true, cfg.headers()...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SendPoll sends a poll message to the specified recipient.
//
// The msisdn parameter should be the recipient's phone number in WhatsApp JID format.
// The question is the poll question text, and options is the list of answer choices.
// The selectableCount limits how many options a user can select (0 means no limit).
//
// Requires authentication (call Register or SetToken first).
func (c *Client) SendPoll(ctx context.Context, msisdn, question string, options []string, selectableCount int, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)
	reqBody := SendPollMessageRequest{
		Chat:            cfg.chat,
		Msisdn:          msisdn,
		Question:        question,
		Options:         options,
		SelectableCount: selectableCount,
		ReplyToID:       cfg.replyToID,
		ReplyToSender:   cfg.replyToSender,
		ReplyToText:     cfg.replyToText,
		Mentions:        cfg.mentions,
	}

	var resp SendMessageResponse
	if err := c.doRequest(ctx, http.MethodPost, "/message/poll", reqBody, &resp, true, cfg.headers()...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SendSticker sends a sticker message to the specified recipient.
//
// The msisdn parameter should be the recipient's phone number in WhatsApp JID format.
// The sticker parameter is an io.Reader containing the sticker data (WebP format).
//
// Example:
//
//	file, _ := os.Open("sticker.webp")
//	defer file.Close()
//	resp, err := client.SendSticker(ctx, msisdn, file)
//
// Requires authentication (call Register or SetToken first).
func (c *Client) SendSticker(ctx context.Context, msisdn string, sticker io.Reader, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add msisdn field
	if err := writer.WriteField("msisdn", msisdn); err != nil {
		return nil, fmt.Errorf("failed to write msisdn field: %w", err)
	}

	// Add chat/reply/mentions context
	if err := writeSendContext(writer, cfg); err != nil {
		return nil, err
	}

	// Add sticker file
	part, err := writer.CreateFormFile("sticker", "sticker.webp")
	if err != nil {
		return nil, fmt.Errorf("failed to create sticker form file: %w", err)
	}
	if _, err := io.Copy(part, sticker); err != nil {
		return nil, fmt.Errorf("failed to copy sticker data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	url := c.baseURL + "/message/sticker"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.GetToken())
	req.Header.Set("User-Agent", c.userAgent)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}
	setHeaders(req, cfg.headers())

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
		return nil, parseError(body, resp.StatusCode, resp.Header.Get(TraceIDHeader))
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// SendAudio sends an audio message to the specified recipient.
//
// The audio parameter is an io.Reader containing the audio file bytes.
// If isPTT is true, WhatsApp renders the audio as a voice note bubble.
// If isViewOnce is true, the media is sent as view-once.
func (c *Client) SendAudio(ctx context.Context, msisdn string, audio io.Reader, isPTT, isViewOnce bool, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("msisdn", msisdn); err != nil {
		return nil, fmt.Errorf("failed to write msisdn field: %w", err)
	}
	if err := writeSendContext(writer, cfg); err != nil {
		return nil, err
	}
	if isPTT {
		if err := writer.WriteField("is_ptt", "true"); err != nil {
			return nil, fmt.Errorf("failed to write is_ptt field: %w", err)
		}
	}
	if isViewOnce {
		if err := writer.WriteField("is_view_once", "true"); err != nil {
			return nil, fmt.Errorf("failed to write is_view_once field: %w", err)
		}
	}

	part, err := writer.CreateFormFile("audio", "audio.ogg")
	if err != nil {
		return nil, fmt.Errorf("failed to create audio form file: %w", err)
	}
	if _, err := io.Copy(part, audio); err != nil {
		return nil, fmt.Errorf("failed to copy audio data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/message/audio", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.GetToken())
	req.Header.Set("User-Agent", c.userAgent)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}
	setHeaders(req, cfg.headers())

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
		return nil, parseError(body, resp.StatusCode, resp.Header.Get(TraceIDHeader))
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// SendVideo sends a video message to the specified recipient.
//
// The video parameter is an io.Reader containing the video file bytes.
// caption is optional. isGif toggles GIF-like rendering. isViewOnce controls
// view-once behavior.
func (c *Client) SendVideo(ctx context.Context, msisdn string, video io.Reader, caption string, isGif, isViewOnce bool, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("msisdn", msisdn); err != nil {
		return nil, fmt.Errorf("failed to write msisdn field: %w", err)
	}
	if err := writeSendContext(writer, cfg); err != nil {
		return nil, err
	}
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return nil, fmt.Errorf("failed to write caption field: %w", err)
		}
	}
	if isGif {
		if err := writer.WriteField("is_gif", "true"); err != nil {
			return nil, fmt.Errorf("failed to write is_gif field: %w", err)
		}
	}
	if isViewOnce {
		if err := writer.WriteField("is_view_once", "true"); err != nil {
			return nil, fmt.Errorf("failed to write is_view_once field: %w", err)
		}
	}

	part, err := writer.CreateFormFile("video", "video.mp4")
	if err != nil {
		return nil, fmt.Errorf("failed to create video form file: %w", err)
	}
	if _, err := io.Copy(part, video); err != nil {
		return nil, fmt.Errorf("failed to copy video data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/message/video", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.GetToken())
	req.Header.Set("User-Agent", c.userAgent)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}
	setHeaders(req, cfg.headers())

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
		return nil, parseError(body, resp.StatusCode, resp.Header.Get(TraceIDHeader))
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// SendDocument sends a document message to the specified recipient.
//
// fileName and caption are optional. When fileName is empty, the gateway uses
// its own default naming.
func (c *Client) SendDocument(ctx context.Context, msisdn string, document io.Reader, fileName, caption string, opts ...SendOption) (*SendMessageResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	cfg := newSendConfig(opts)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("msisdn", msisdn); err != nil {
		return nil, fmt.Errorf("failed to write msisdn field: %w", err)
	}
	if err := writeSendContext(writer, cfg); err != nil {
		return nil, err
	}
	if fileName != "" {
		if err := writer.WriteField("file_name", fileName); err != nil {
			return nil, fmt.Errorf("failed to write file_name field: %w", err)
		}
	}
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return nil, fmt.Errorf("failed to write caption field: %w", err)
		}
	}

	part, err := writer.CreateFormFile("document", "document.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to create document form file: %w", err)
	}
	if _, err := io.Copy(part, document); err != nil {
		return nil, fmt.Errorf("failed to copy document data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/message/document", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.GetToken())
	req.Header.Set("User-Agent", c.userAgent)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}
	setHeaders(req, cfg.headers())

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
		return nil, parseError(body, resp.StatusCode, resp.Header.Get(TraceIDHeader))
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
func (c *Client) React(ctx context.Context, msisdn, messageID, emoji string, senderMsisdn ...string) error {
	if err := c.checkAuth(); err != nil {
		return err
	}

	reqBody := MessageReactRequest{
		Msisdn:    msisdn,
		MessageId: messageID,
		Emoji:     emoji,
	}
	if len(senderMsisdn) > 0 && senderMsisdn[0] != "" {
		reqBody.SenderMsisdn = senderMsisdn[0]
	}

	var resp SuccessResponse
	return c.doRequest(ctx, http.MethodPost, "/message/react", reqBody, &resp, true)
}

// CheckContact validates whether a recipient is registered on WhatsApp.
//
// The msisdn argument can be a plain phone number or WhatsApp JID; the gateway
// normalizes it and returns canonical JID details.
func (c *Client) CheckContact(ctx context.Context, msisdn string) (*ContactCheckResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	var resp ContactCheckResponse
	path := "/contact/check?msisdn=" + url.QueryEscape(msisdn)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetIncomingMessages fetches the most recent incoming messages buffered by the
// gateway for the authenticated session, newest first.
//
// The limit parameter caps the number of messages returned. If limit <= 0, the
// gateway substitutes its default (10). Values above the gateway's maximum (50)
// are clamped server-side.
//
// Requires authentication (call Register or SetToken first).
func (c *Client) GetIncomingMessages(ctx context.Context, limit int) (*IncomingMessagesResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/message/incoming?limit=%d", limit)
	var resp IncomingMessagesResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetJobStatus fetches the status of an asynchronously queued message job.
//
// When the gateway runs in queue mode, send endpoints return 202 Accepted
// with a job ID instead of a message ID. Use this method to poll the job
// until its Status is "completed" or "failed".
//
// Requires authentication (call Register or SetToken first).
func (c *Client) GetJobStatus(ctx context.Context, jobID string) (*JobStatusResponse, error) {
	if err := c.checkAuth(); err != nil {
		return nil, err
	}

	var resp JobStatusResponse
	if err := c.doRequest(ctx, http.MethodGet, "/message/job/"+url.PathEscape(jobID), nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
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
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}, requireAuth bool, headers ...reqHeader) error {
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
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(TraceIDHeader, traceID)
	}
	setHeaders(req, headers)

	if requireAuth {
		token := c.GetToken()
		if token == "" {
			return ErrNotAuthenticated
		}
		req.Header.Set("Authorization", "Bearer "+token)
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
		return parseError(respBody, resp.StatusCode, resp.Header.Get(TraceIDHeader))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

func (c *Client) checkAuth() error {
	if c.GetToken() == "" {
		return ErrNotAuthenticated
	}
	return nil
}

// setHeaders applies extra request headers (e.g. Idempotency-Key) to req.
func setHeaders(req *http.Request, headers []reqHeader) {
	for _, h := range headers {
		req.Header.Set(h.key, h.value)
	}
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
