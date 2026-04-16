package waga

// RegisterRequest represents a registration request for a new phone number.
// It contains the phone number and secret key required to authenticate
// with the WhatsApp Gateway service and obtain a JWT token.
type RegisterRequest struct {
	PhoneNumber string `json:"phone_number"`
	SecretKey   string `json:"secret_key"`
}

// RegisterResponse represents the response from a successful registration.
// It contains the JWT token that should be used for subsequent API requests.
type RegisterResponse struct {
	// Token is the JWT authentication token for API requests
	Token string `json:"token"`
}

// LoginQrResponse represents the response when requesting a QR code for login.
// The QR code can be displayed to users for scanning with the WhatsApp mobile app.
type LoginQrResponse struct {
	// QrCode is the base64-encoded QR code image or HTML img tag
	QrCode string `json:"qr_code"`
	// ExpiresIn is the number of seconds until the QR code expires
	ExpiresIn int `json:"expires_in"`
}

// LoginPairResponse represents the response when requesting a pair code for login.
// The pair code can be used to link the WhatsApp account without scanning a QR code.
type LoginPairResponse struct {
	// PairCode is the 8-character code for linking the device
	PairCode string `json:"pair_code"`
	// ExpiresIn is the number of seconds until the pair code expires
	ExpiresIn int `json:"expires_in"`
}

// WebhookRegisterRequest represents a request to register a webhook URL.
// Webhooks receive real-time notifications for message events.
type WebhookRegisterRequest struct {
	// Url is the endpoint URL where webhook events will be sent
	Url string `json:"url"`
	// HmacSecret is the optional secret key used to sign webhook payloads for verification
	HmacSecret *string `json:"hmac_secret,omitempty"`
}

// SendMessageTextRequest represents a request to send a text message.
type SendMessageTextRequest struct {
	// Msisdn is the recipient's phone number in WhatsApp JID format (e.g., "6281234567890@s.whatsapp.net")
	Msisdn string `json:"msisdn"`
	// Message is the text content to send
	Message string `json:"message"`
}

// SendMessageResponse represents the response from sending a message.
type SendMessageResponse struct {
	// Success indicates whether the message was successfully queued
	Success bool `json:"success"`
	// MessageId is the unique identifier for the sent message
	MessageId string `json:"message_id"`
}

// MessageDeleteRequest represents a request to delete a previously sent message.
type MessageDeleteRequest struct {
	// MessageId is the ID of the message to delete
	MessageId string `json:"message_id"`
	// Msisdn is the recipient's phone number in WhatsApp JID format
	Msisdn string `json:"msisdn"`
}

// MessageEditRequest represents a request to edit a previously sent message.
type MessageEditRequest struct {
	// MessageId is the ID of the message to edit
	MessageId string `json:"message_id"`
	// Msisdn is the recipient's phone number in WhatsApp JID format
	Msisdn string `json:"msisdn"`
	// NewMessage is the updated message content
	NewMessage string `json:"new_message"`
}

// MessageReactRequest represents a request to add a reaction to a message.
type MessageReactRequest struct {
	// MessageId is the ID of the message to react to
	MessageId string `json:"message_id"`
	// Msisdn is the recipient's phone number in WhatsApp JID format
	Msisdn string `json:"msisdn"`
	// Emoji is the emoji reaction to add
	Emoji string `json:"emoji"`
}

// OutgoingEventMessage represents the type of outgoing message event.
// These events are sent to webhooks to track message delivery status.
type OutgoingEventMessage string

const (
	// OutgoingEventMessageMessageQueued is emitted when a message is added to the queue
	OutgoingEventMessageMessageQueued OutgoingEventMessage = "message.queued"
	// OutgoingEventMessageMessageSent is emitted when a message is successfully sent
	OutgoingEventMessageMessageSent OutgoingEventMessage = "message.sent"
	// OutgoingEventMessageMessageFailed is emitted when a message fails to send
	OutgoingEventMessageMessageFailed OutgoingEventMessage = "message.failed"
)

// OutgoingWebhookPayload represents the payload sent to webhooks for outgoing message events.
// These events track the delivery status of messages sent through the API.
type OutgoingWebhookPayload struct {
	// Event is the type of event that occurred
	Event OutgoingEventMessage `json:"event"`
	// JobId is the unique identifier for the message job
	JobId string `json:"job_id"`
	// To is the recipient's phone number in WhatsApp JID format
	To string `json:"to"`
	// PhoneNumber is the sender's phone number
	PhoneNumber string `json:"phone_number"`
	// Timestamp is the Unix timestamp of the event
	Timestamp int `json:"timestamp"`
	// MessageId is the unique identifier for the message
	MessageId string `json:"message_id"`
	// Metadata contains additional optional information about the message
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

// IncomingEventMessage represents the type of incoming message event.
type IncomingEventMessage string

const (
	// IncomingEventMessageMessageIncoming is emitted when a new message is received
	IncomingEventMessageMessageIncoming IncomingEventMessage = "message.incoming"
)

// IncomingMessageType represents the type of incoming message content.
type IncomingMessageType string

const (
	// IncomingMessageTypeText represents a text message
	IncomingMessageTypeText IncomingMessageType = "text"
	// IncomingMessageTypeImage represents an image message
	IncomingMessageTypeImage IncomingMessageType = "image"
	// IncomingMessageTypeVideo represents a video message
	IncomingMessageTypeVideo IncomingMessageType = "video"
	// IncomingMessageTypeAudio represents an audio message
	IncomingMessageTypeAudio IncomingMessageType = "audio"
	// IncomingMessageTypeDocument represents a document message
	IncomingMessageTypeDocument IncomingMessageType = "document"
)

// IncomingMessageMediaInfo contains media information for incoming messages with attachments.
// This includes images, videos, audio, and documents.
type IncomingMessageMediaInfo struct {
	// Type is the media type (image, video, audio, or document)
	Type IncomingMessageType `json:"type"`
	// Url is the direct URL to download the media file
	Url string `json:"url"`
	// MimeType is the MIME type of the media file
	MimeType string `json:"mime_type"`
	// Filename is the original filename of the media file (if applicable)
	Filename string `json:"filename,omitempty"`
	// Caption is the text caption accompanying the media (if applicable)
	Caption string `json:"caption,omitempty"`
	// Size is the file size in bytes (if available)
	Size int `json:"size,omitempty"`
}

// IncomingWebhookPayload represents the payload sent to webhooks for incoming message events.
// This contains all information about received messages.
type IncomingWebhookPayload struct {
	// Event is the type of event (always "message.incoming")
	Event IncomingEventMessage `json:"event"`
	// Chat is the chat ID where the message was received
	Chat string `json:"chat"`
	// From is the sender's phone number in WhatsApp JID format
	From string `json:"from"`
	// IsGroup indicates whether the message was received in a group chat
	IsGroup bool `json:"is_group"`
	// MessageId is the unique identifier for the message
	MessageId string `json:"message_id"`
	// PushName is the display name of the sender
	PushName string `json:"push_name"`
	// Timestamp is the Unix timestamp when the message was received
	Timestamp int `json:"timestamp"`
	// Text is the text content of the message (for text messages)
	Text string `json:"text,omitempty"`
	// Type is the message type (text, image, video, audio, or document)
	Type IncomingMessageType `json:"type"`
	// Media contains media information for non-text messages
	Media *IncomingMessageMediaInfo `json:"media,omitempty"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	// Error is a human-readable error message
	Error string `json:"error"`
	// Code is the HTTP status code or API error code
	Code int `json:"code"`
}
