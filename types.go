// Package waga provides types for the WhatsApp Gateway SDK.
package waga

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	PhoneNumber string `json:"phone_number"`
	SecretKey   string `json:"secret_key"`
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	Token string `json:"token"`
}

// LoginQrResponse represents the QR code login response
type LoginQrResponse struct {
	QrCode    string `json:"qr_code"`
	ExpiresIn int    `json:"expires_in"`
}

// LoginPairResponse represents the pair code login response
type LoginPairResponse struct {
	PairCode  string `json:"pair_code"`
	ExpiresIn int    `json:"expires_in"`
}

// WebhookRegisterRequest represents the webhook registration request
type WebhookRegisterRequest struct {
	Url        string  `json:"url"`
	HmacSecret *string `json:"hmac_secret,omitempty"`
}

// SendMessageTextRequest represents the send text message request
type SendMessageTextRequest struct {
	Msisdn  string `json:"msisdn"`
	Message string `json:"message"`
}

// SendMessageResponse represents the send message response
type SendMessageResponse struct {
	Success   bool   `json:"success"`
	MessageId string `json:"message_id"`
}

// MessageDeleteRequest represents the delete message request
type MessageDeleteRequest struct {
	MessageId string `json:"message_id"`
	Msisdn    string `json:"msisdn"`
}

// MessageEditRequest represents the edit message request
type MessageEditRequest struct {
	MessageId  string `json:"message_id"`
	Msisdn     string `json:"msisdn"`
	NewMessage string `json:"new_message"`
}

// MessageReactRequest represents the react to message request
type MessageReactRequest struct {
	MessageId string `json:"message_id"`
	Msisdn    string `json:"msisdn"`
	Emoji     string `json:"emoji"`
}

// OutgoingEventMessage represents outgoing message event types
type OutgoingEventMessage string

const (
	OutgoingEventMessageMessageQueued OutgoingEventMessage = "message.queued"
	OutgoingEventMessageMessageSent   OutgoingEventMessage = "message.sent"
	OutgoingEventMessageMessageFailed OutgoingEventMessage = "message.failed"
)

// OutgoingWebhookPayload represents the outgoing webhook payload
type OutgoingWebhookPayload struct {
	Event       OutgoingEventMessage    `json:"event"`
	JobId       string                  `json:"job_id"`
	To          string                  `json:"to"`
	PhoneNumber string                  `json:"phone_number"`
	Timestamp   int                     `json:"timestamp"`
	MessageId   string                  `json:"message_id"`
	Metadata    *map[string]interface{} `json:"metadata,omitempty"`
}

// IncomingEventMessage represents incoming message event types
type IncomingEventMessage string

const (
	IncomingEventMessageMessageIncoming IncomingEventMessage = "message.incoming"
)

// IncomingMessageType represents the type of incoming message
type IncomingMessageType string

const (
	IncomingMessageTypeText     IncomingMessageType = "text"
	IncomingMessageTypeImage    IncomingMessageType = "image"
	IncomingMessageTypeVideo    IncomingMessageType = "video"
	IncomingMessageTypeAudio    IncomingMessageType = "audio"
	IncomingMessageTypeDocument IncomingMessageType = "document"
)

// IncomingMessageMediaInfo contains media information for incoming messages
type IncomingMessageMediaInfo struct {
	Type     IncomingMessageType `json:"type"`
	Url      string              `json:"url"`
	MimeType string              `json:"mime_type"`
	Filename string              `json:"filename,omitempty"`
	Caption  string              `json:"caption,omitempty"`
	Size     int                 `json:"size,omitempty"`
}

// IncomingWebhookPayload represents the incoming webhook payload
type IncomingWebhookPayload struct {
	Event     IncomingEventMessage     `json:"event"`
	Chat      string                   `json:"chat"`
	From      string                   `json:"from"`
	IsGroup   bool                     `json:"is_group"`
	MessageId string                   `json:"message_id"`
	PushName  string                   `json:"push_name"`
	Timestamp int                      `json:"timestamp"`
	Text      string                   `json:"text,omitempty"`
	Type      IncomingMessageType      `json:"type"`
	Media     *IncomingMessageMediaInfo `json:"media,omitempty"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}
