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
//
// The populated fields depend on the gateway's delivery mode:
//   - Direct mode (HTTP 200): MessageId holds the sent WhatsApp message ID.
//   - Queue mode (HTTP 202): Status is "queued" and JobID holds the job
//     identifier — poll it with GetJobStatus to obtain the message ID once
//     the job completes. MessageId is empty in this case.
type SendMessageResponse struct {
	// Success indicates whether the message was successfully sent or queued
	Success bool `json:"success"`
	// MessageId is the unique identifier for the sent message (direct mode)
	MessageId string `json:"message_id,omitempty"`
	// Status is the job status in queue mode (e.g. "queued")
	Status string `json:"status,omitempty"`
	// JobID identifies the queued job in queue mode; poll it with GetJobStatus
	JobID string `json:"job_id,omitempty"`
}

// SendLocationMessageRequest represents a request to send a location message.
type SendLocationMessageRequest struct {
	// Msisdn is the recipient's phone number in WhatsApp JID format
	Msisdn string `json:"msisdn"`
	// Latitude is the geographic latitude of the location
	Latitude float64 `json:"latitude"`
	// Longitude is the geographic longitude of the location
	Longitude float64 `json:"longitude"`
	// Name is the optional name of the location
	Name string `json:"name,omitempty"`
	// Address is the optional address of the location
	Address string `json:"address,omitempty"`
}

// SendPollMessageRequest represents a request to send a poll message.
type SendPollMessageRequest struct {
	// Msisdn is the recipient's phone number in WhatsApp JID format
	Msisdn string `json:"msisdn"`
	// Question is the poll question text
	Question string `json:"question"`
	// Options is the list of poll options
	Options []string `json:"options"`
	// SelectableCount is the optional maximum number of options a user can select
	SelectableCount int `json:"selectable_count,omitempty"`
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
	Timestamp int64 `json:"timestamp"`
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
	// IncomingMessageTypeSticker represents a sticker message
	IncomingMessageTypeSticker IncomingMessageType = "sticker"
	// IncomingMessageTypeLocation represents a location message.
	// Webhook payloads carry Latitude/Longitude/Name/Address; the polled
	// /message/incoming endpoint reports only the type.
	IncomingMessageTypeLocation IncomingMessageType = "location"
	// IncomingMessageTypePoll represents a poll message.
	// Webhook payloads carry Question/Options/SelectableCount.
	IncomingMessageTypePoll IncomingMessageType = "poll"
	// IncomingMessageTypeContact represents a contact message (reported only
	// by the polled /message/incoming endpoint; no detail fields)
	IncomingMessageTypeContact IncomingMessageType = "contact"
	// IncomingMessageTypeUnknown is reported for message types the gateway
	// does not specifically model
	IncomingMessageTypeUnknown IncomingMessageType = "unknown"
)

// IncomingMessageMediaInfo contains media information for incoming messages with attachments.
// This includes images, videos, audio, and documents.
type IncomingMessageMediaInfo struct {
	// Type is the media type (image, video, audio, or document)
	Type IncomingMessageType `json:"type"`
	// Url is the direct URL to download the media file. In webhooks it mirrors
	// StorageURL when the gateway stored the media, otherwise WhatsappURL.
	Url string `json:"url"`
	// StorageURL is the gateway-hosted URL when the media was downloaded and
	// stored (webhooks only)
	StorageURL string `json:"storage_url,omitempty"`
	// WhatsappURL is the raw WhatsApp media URL used when storage was skipped
	// or failed (webhooks only)
	WhatsappURL string `json:"whatsapp_url,omitempty"`
	// MimeType is the MIME type of the media file
	MimeType string `json:"mime_type"`
	// Filename is the original filename of the media file (if applicable)
	Filename string `json:"filename,omitempty"`
	// Caption is the text caption accompanying the media (if applicable)
	Caption string `json:"caption,omitempty"`
	// Size is the file size in bytes (if available)
	Size int `json:"size,omitempty"`
	// Sha256 is the hex-encoded SHA-256 hash of the media file (webhooks only)
	Sha256 string `json:"sha256,omitempty"`
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
	Timestamp int64 `json:"timestamp"`
	// Text is the text content of the message (for text messages)
	Text string `json:"text,omitempty"`
	// Type is the message type (see IncomingMessageType constants)
	Type IncomingMessageType `json:"type"`
	// Media contains media information for image/video/audio/document/sticker
	Media *IncomingMessageMediaInfo `json:"media,omitempty"`

	// Location fields (Type == IncomingMessageTypeLocation).
	// Latitude/Longitude are pointers because (0,0) is a valid location.
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Name      string   `json:"name,omitempty"`
	Address   string   `json:"address,omitempty"`

	// Poll fields (Type == IncomingMessageTypePoll).
	Question        string   `json:"question,omitempty"`
	Options         []string `json:"options,omitempty"`
	SelectableCount int      `json:"selectable_count,omitempty"`
}

// IncomingMessage represents a single received message returned by the
// GET /message/incoming endpoint. Field names mirror IncomingWebhookPayload
// (minus the Event envelope) so consumers see a consistent vocabulary across
// the webhook delivery and the polled inbox.
type IncomingMessage struct {
	// MessageId is the unique identifier for the message
	MessageId string `json:"message_id"`
	// Chat is the chat ID where the message was received
	Chat string `json:"chat"`
	// From is the sender's phone number in WhatsApp JID format
	From string `json:"from"`
	// IsGroup indicates whether the message was received in a group chat
	IsGroup bool `json:"is_group"`
	// PushName is the display name of the sender
	PushName string `json:"push_name"`
	// Timestamp is the Unix timestamp when the message was received
	Timestamp int64 `json:"timestamp"`
	// Text is the text content of the message (for text messages)
	Text string `json:"text,omitempty"`
	// Type is the message type (text, image, video, audio, or document)
	Type IncomingMessageType `json:"type"`
	// Media contains media metadata for non-text messages.
	// Note: Url is not populated by /message/incoming in v1; only Type, MimeType,
	// Size, Filename, and Caption are filled. Use webhooks for media URLs.
	Media *IncomingMessageMediaInfo `json:"media,omitempty"`
}

// IncomingMessagesResponse represents the response from the
// GET /message/incoming endpoint. Messages are sorted newest first.
type IncomingMessagesResponse struct {
	// Success indicates the request completed successfully
	Success bool `json:"success"`
	// Timestamp is the Unix milliseconds when the response was generated
	Timestamp int64 `json:"timestamp"`
	// Count is the number of messages returned (<= requested limit)
	Count int `json:"count"`
	// Messages is the list of incoming messages, newest first
	Messages []IncomingMessage `json:"messages"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	// Error is a human-readable error message
	Error string `json:"error"`
	// Code is the HTTP status code or API error code
	Code int `json:"code"`
}

// JobStatusResponse represents the response from the
// GET /message/job/:job_id endpoint, used to poll the status of an
// asynchronously queued message job.
type JobStatusResponse struct {
	// JobID is the identifier returned when the message was queued
	JobID string `json:"job_id"`
	// Status is one of "queued", "processing", "completed", or "failed"
	Status string `json:"status"`
	// MessageID is the WhatsApp message ID once the job completed
	MessageID *string `json:"message_id,omitempty"`
	// Error describes why the job failed, if it did
	Error *string `json:"error,omitempty"`
	// CreatedAt is when the job was created
	CreatedAt string `json:"created_at"`
	// CompletedAt is when the job finished, if it has
	CompletedAt *string `json:"completed_at,omitempty"`
}
