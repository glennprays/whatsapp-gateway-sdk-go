package waga

import "time"

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
	// Chat is the canonical recipient: a bare number, a user JID
	// ("@s.whatsapp.net"), a group JID ("@g.us"), or a "@lid". When both Chat
	// and Msisdn are set, the gateway resolves Chat.
	Chat string `json:"chat,omitempty"`
	// Msisdn is the recipient in WhatsApp JID format.
	//
	// Deprecated: use Chat. Msisdn remains a permanent back-compat alias.
	Msisdn string `json:"msisdn,omitempty"`
	// Message is the text content to send
	Message string `json:"message"`
	// ReplyToID quotes an existing message by ID (optional).
	ReplyToID string `json:"reply_to_id,omitempty"`
	// ReplyToSender is the number/JID of the quoted message's author (optional).
	ReplyToSender string `json:"reply_to_sender,omitempty"`
	// ReplyToText is an optional caller-supplied preview of the quoted message.
	ReplyToText string `json:"reply_to_text,omitempty"`
	// Mentions are the numbers/JIDs to @-tag (optional).
	Mentions []string `json:"mentions,omitempty"`
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
	// Chat is the resolved canonical recipient JID the gateway addressed.
	Chat string `json:"chat"`
}

// SendLocationMessageRequest represents a request to send a location message.
type SendLocationMessageRequest struct {
	// Chat is the canonical recipient (see SendMessageTextRequest.Chat).
	Chat string `json:"chat,omitempty"`
	// Msisdn is the recipient in WhatsApp JID format.
	//
	// Deprecated: use Chat. Msisdn remains a permanent back-compat alias.
	Msisdn string `json:"msisdn,omitempty"`
	// Latitude is the geographic latitude of the location
	Latitude float64 `json:"latitude"`
	// Longitude is the geographic longitude of the location
	Longitude float64 `json:"longitude"`
	// Name is the optional name of the location
	Name string `json:"name,omitempty"`
	// Address is the optional address of the location
	Address string `json:"address,omitempty"`
	// ReplyToID quotes an existing message by ID (optional).
	ReplyToID string `json:"reply_to_id,omitempty"`
	// ReplyToSender is the number/JID of the quoted message's author (optional).
	ReplyToSender string `json:"reply_to_sender,omitempty"`
	// ReplyToText is an optional caller-supplied preview of the quoted message.
	ReplyToText string `json:"reply_to_text,omitempty"`
	// Mentions are the numbers/JIDs to @-tag (optional).
	Mentions []string `json:"mentions,omitempty"`
}

// SendPollMessageRequest represents a request to send a poll message.
type SendPollMessageRequest struct {
	// Chat is the canonical recipient (see SendMessageTextRequest.Chat).
	Chat string `json:"chat,omitempty"`
	// Msisdn is the recipient in WhatsApp JID format.
	//
	// Deprecated: use Chat. Msisdn remains a permanent back-compat alias.
	Msisdn string `json:"msisdn,omitempty"`
	// Question is the poll question text
	Question string `json:"question"`
	// Options is the list of poll options
	Options []string `json:"options"`
	// SelectableCount is the optional maximum number of options a user can select
	SelectableCount int `json:"selectable_count,omitempty"`
	// ReplyToID quotes an existing message by ID (optional).
	ReplyToID string `json:"reply_to_id,omitempty"`
	// ReplyToSender is the number/JID of the quoted message's author (optional).
	ReplyToSender string `json:"reply_to_sender,omitempty"`
	// ReplyToText is an optional caller-supplied preview of the quoted message.
	ReplyToText string `json:"reply_to_text,omitempty"`
	// Mentions are the numbers/JIDs to @-tag (optional).
	Mentions []string `json:"mentions,omitempty"`
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
	// SenderMsisdn is optional and should be set when reacting to an incoming
	// message from another sender (e.g. group chats).
	SenderMsisdn string `json:"sender_msisdn,omitempty"`
}

// ContactCheckResponse is the result of validating a recipient number on
// WhatsApp.
type ContactCheckResponse struct {
	// Query is the original queried number.
	Query string `json:"query"`
	// JID is the canonical WhatsApp JID.
	JID string `json:"jid"`
	// IsOnWhatsApp indicates whether the number is registered on WhatsApp.
	IsOnWhatsApp bool `json:"is_on_whatsapp"`
	// VerifiedName is the business verified name, if available.
	VerifiedName *string `json:"verified_name,omitempty"`
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
	// Error is the failure reason carried by message.failed (and populated on
	// message.sent/queued when a reason exists); empty on success.
	Error string `json:"error,omitempty"`
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
	// AddressingMode is the JID addressing mode of the chat ("pn" or "lid").
	AddressingMode string `json:"addressing_mode,omitempty"`
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
	// AddressingMode is the JID addressing mode of the chat ("pn" or "lid").
	AddressingMode string `json:"addressing_mode,omitempty"`
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

// WebhookEventType is the type discriminator carried by every webhook envelope's
// "event" field. It is the single catalog ParseWebhook dispatches on and mirrors
// the gateway's domain/queue.StatusWebhookEvent.
type WebhookEventType string

const (
	// WebhookEventMessageIncoming is emitted for a received message.
	WebhookEventMessageIncoming WebhookEventType = "message.incoming"
	// WebhookEventMessageQueued is emitted when a message is queued.
	WebhookEventMessageQueued WebhookEventType = "message.queued"
	// WebhookEventMessageSent is emitted when a message is sent.
	WebhookEventMessageSent WebhookEventType = "message.sent"
	// WebhookEventMessageFailed is emitted when a message fails to send.
	WebhookEventMessageFailed WebhookEventType = "message.failed"

	// WebhookEventSessionLoggedOut is emitted when the account is logged out.
	WebhookEventSessionLoggedOut WebhookEventType = "session.logged_out"
	// WebhookEventSessionBanned is emitted when the account is temporarily banned.
	WebhookEventSessionBanned WebhookEventType = "session.banned"
	// WebhookEventSessionConnectFailure is emitted on a connection failure.
	WebhookEventSessionConnectFailure WebhookEventType = "session.connect_failure"
	// WebhookEventSessionConnected is emitted when the session connects.
	WebhookEventSessionConnected WebhookEventType = "session.connected"
	// WebhookEventSessionDisconnected is emitted when the session disconnects.
	WebhookEventSessionDisconnected WebhookEventType = "session.disconnected"
	// WebhookEventSessionReplaced is emitted when another process takes the socket.
	WebhookEventSessionReplaced WebhookEventType = "session.replaced"
)

// SessionEvent is the flat envelope the gateway emits for session.* lifecycle
// events. The envelope fields (Event, PhoneNumber, JID, Timestamp) are always
// present; the remaining fields are event-specific extras and are zero when the
// event does not carry them:
//   - session.logged_out:      OnConnect, Reason, ReasonText
//   - session.banned:          Code, ReasonText, ExpiresIn
//   - session.connect_failure: Reason, ReasonText, Message
//   - session.connected / disconnected / replaced: envelope only
type SessionEvent struct {
	// Event is the session.* event type.
	Event WebhookEventType `json:"event"`
	// PhoneNumber is the account the event concerns.
	PhoneNumber string `json:"phone_number"`
	// JID is the account's device JID.
	JID string `json:"jid"`
	// Timestamp is the Unix timestamp of the event.
	Timestamp int64 `json:"timestamp"`

	// OnConnect (session.logged_out) is true when the logout happened while
	// (re)connecting rather than via an explicit remote logout.
	OnConnect bool `json:"on_connect,omitempty"`
	// Reason (session.logged_out, session.connect_failure) is the numeric reason code.
	Reason int `json:"reason,omitempty"`
	// ReasonText (session.logged_out, session.banned, session.connect_failure)
	// is the human-readable reason.
	ReasonText string `json:"reason_text,omitempty"`
	// Code (session.banned) is the numeric ban code.
	Code int `json:"code,omitempty"`
	// ExpiresIn (session.banned) is the ban duration in seconds.
	ExpiresIn int `json:"expires_in,omitempty"`
	// Message (session.connect_failure) is the failure message.
	Message string `json:"message,omitempty"`
}

// WebhookEvent is the discriminated result of ParseWebhook. Exactly one of the
// payload pointers is non-nil, selected by Event:
//   - Incoming: message.incoming
//   - Outgoing: message.queued / message.sent / message.failed
//   - Session:  session.*
type WebhookEvent struct {
	// Event is the parsed event type.
	Event WebhookEventType
	// Incoming is set for message.incoming.
	Incoming *IncomingWebhookPayload
	// Outgoing is set for message.queued / message.sent / message.failed.
	Outgoing *OutgoingWebhookPayload
	// Session is set for session.* events.
	Session *SessionEvent
}

// ContactListItem is one entry in the account's locally-synced contact list.
type ContactListItem struct {
	JID          string `json:"jid"`
	PushName     string `json:"push_name,omitempty"`
	FullName     string `json:"full_name,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	BusinessName string `json:"business_name,omitempty"`
}

// ContactListResponse is a page of the account's locally-synced contacts. The
// list reflects the synced address book, so an empty or partial result is not an
// error (GET /contact/ never 404s on empty).
type ContactListResponse struct {
	// Contacts is this page of synced contacts.
	Contacts []ContactListItem `json:"contacts"`
	// Count is the number of items in this page.
	Count int `json:"count"`
	// Total is the total number of synced contacts.
	Total int `json:"total"`
	// Note is an optional gateway-supplied advisory (e.g. sync status).
	Note string `json:"note,omitempty"`
}

// ContactInfoResponse is a server-side profile lookup for one user: status text,
// current picture id, verified business name, linked-device count, and lid.
type ContactInfoResponse struct {
	JID          string `json:"jid"`
	Status       string `json:"status,omitempty"`
	PictureID    string `json:"picture_id,omitempty"`
	VerifiedName string `json:"verified_name,omitempty"`
	DeviceCount  int    `json:"device_count"`
	LID          string `json:"lid,omitempty"`
}

// AvatarResponse is a chat's (user or group) profile picture. URL is a
// time-limited WhatsApp CDN link the caller downloads directly. ID doubles as
// the ETag: pass it back to GetAvatar as priorID to get ErrNotModified (304)
// when the picture is unchanged.
type AvatarResponse struct {
	JID        string `json:"jid"`
	URL        string `json:"url"`
	ID         string `json:"id"`
	Type       string `json:"type"` // "image" (full res) or "preview" (thumbnail)
	DirectPath string `json:"direct_path,omitempty"`
}

// GroupListItem is one entry in the account's joined-groups list (a lightweight
// summary with no participant roster).
type GroupListItem struct {
	JID              string `json:"jid"` // the group's @g.us JID
	Name             string `json:"name,omitempty"`
	Topic            string `json:"topic,omitempty"`
	OwnerJID         string `json:"owner_jid,omitempty"`
	ParticipantCount int    `json:"participant_count"`
	IsAnnounce       bool   `json:"is_announce"`  // only admins can send
	IsLocked         bool   `json:"is_locked"`    // only admins can edit group info
	IsCommunity      bool   `json:"is_community"` // this group is a community parent
}

// GroupListResponse is the account's joined groups. Not paginated; Count always
// equals len(Groups).
type GroupListResponse struct {
	Groups []GroupListItem `json:"groups"`
	Count  int             `json:"count"`
}

// GroupParticipantItem is one member in a group's roster.
type GroupParticipantItem struct {
	JID          string `json:"jid"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	LID          string `json:"lid,omitempty"`
	IsAdmin      bool   `json:"is_admin"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

// GroupInfoResponse is the full detail of a single group, including its member
// roster. Requires the account to be a participant (403 otherwise, 404 if absent).
type GroupInfoResponse struct {
	JID              string                 `json:"jid"`
	Name             string                 `json:"name,omitempty"`
	Topic            string                 `json:"topic,omitempty"`
	OwnerJID         string                 `json:"owner_jid,omitempty"`
	ParticipantCount int                    `json:"participant_count"`
	IsAnnounce       bool                   `json:"is_announce"`
	IsLocked         bool                   `json:"is_locked"`
	IsCommunity      bool                   `json:"is_community"`
	IsEphemeral      bool                   `json:"is_ephemeral"`
	Participants     []GroupParticipantItem `json:"participants"`
}

// MarkReadRequest marks one or more messages in a chat as read (blue ticks).
type MarkReadRequest struct {
	// Chat is the canonical recipient (see SendMessageTextRequest.Chat).
	Chat string `json:"chat"`
	// MessageIDs are the message IDs to mark read.
	MessageIDs []string `json:"message_ids"`
	// Sender is the message author's JID/number; required for group chats.
	Sender string `json:"sender,omitempty"`
}

// PresenceState is a chat typing-indicator state.
type PresenceState = string

const (
	// PresenceComposing shows the "typing…" indicator.
	PresenceComposing PresenceState = "composing"
	// PresenceRecording shows the "recording audio…" indicator.
	PresenceRecording PresenceState = "recording"
	// PresencePaused clears the typing indicator.
	PresencePaused PresenceState = "paused"
)

// ChatPresenceRequest sets the typing indicator in a chat.
type ChatPresenceRequest struct {
	// Chat is the canonical recipient (see SendMessageTextRequest.Chat).
	Chat string `json:"chat"`
	// State is one of composing | recording | paused.
	State string `json:"state"`
}

// ---- Group & community management ----
//
// Every group/community op requires an explicit group JID ("@g.us"); a bare
// number or user JID is a 400. Batch ops (participant add/remove/promote/demote,
// join-request approve/reject) return 200 with a per-participant Results slice —
// a single bad member is reported there, not as an overall error.

// CreateGroupRequest creates a group, or a community when IsCommunity is true.
// An empty Participants list creates a group of just the account; adding
// participants at creation is gated server-side by GROUP_ADD_PARTICIPANTS_ENABLED.
type CreateGroupRequest struct {
	Name                   string   `json:"name"`
	Participants           []string `json:"participants,omitempty"`
	IsCommunity            bool     `json:"is_community,omitempty"`
	LinkedParentJID        string   `json:"linked_parent_jid,omitempty"`
	IsAnnounce             bool     `json:"is_announce,omitempty"`
	IsLocked               bool     `json:"is_locked,omitempty"`
	IsJoinApprovalRequired bool     `json:"is_join_approval_required,omitempty"`
}

// ParticipantInvite is present on a ParticipantResult when an add was converted
// to an invite because the target's privacy blocks a direct add.
type ParticipantInvite struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// ParticipantResult is the per-participant outcome shared by every batch
// mutation. Status is one of "ok", "invited" (add converted to an invite), or
// "failed"; Code carries the underlying error code when failed.
type ParticipantResult struct {
	JID    string             `json:"jid"`
	LID    string             `json:"lid,omitempty"`
	Status string             `json:"status"`
	Code   int                `json:"code,omitempty"`
	Invite *ParticipantInvite `json:"invite,omitempty"`
}

// CreateGroupResponse echoes the new group's JID, full info, and per-participant
// add results.
type CreateGroupResponse struct {
	GroupJID  string              `json:"group_jid"`
	GroupInfo *GroupInfoResponse  `json:"group_info"`
	Results   []ParticipantResult `json:"results"`
}

// GroupParticipantsResponse is the per-participant outcome of a roster mutation
// (partial success is a 200, never an overall error).
type GroupParticipantsResponse struct {
	GroupJID string              `json:"group_jid"`
	Action   string              `json:"action"`
	Results  []ParticipantResult `json:"results"`
}

// GroupSettingsResponse reports which settings were applied (e.g. ["announce"]).
type GroupSettingsResponse struct {
	GroupJID string   `json:"group_jid"`
	Applied  []string `json:"applied"`
}

// GroupPhotoResponse reports the new picture id (on set) or removal.
type GroupPhotoResponse struct {
	PictureID string `json:"picture_id,omitempty"`
	Removed   bool   `json:"removed,omitempty"`
}

// GroupInviteLinkResponse is a group's invite link.
type GroupInviteLinkResponse struct {
	Chat       string `json:"chat"`        // resolved @g.us JID
	InviteLink string `json:"invite_link"` // https://chat.whatsapp.com/<code>
}

// JoinGroupResponse is the JID whatsmeow returned for a join-by-link (the group,
// or a pending membership-approval request — whatsmeow does not distinguish).
type JoinGroupResponse struct {
	GroupJID string `json:"group_jid"`
}

// GroupJoinRequestItem is one pending join request.
type GroupJoinRequestItem struct {
	JID         string `json:"jid"`
	RequestedAt string `json:"requested_at,omitempty"` // RFC3339
}

// GroupJoinRequestsResponse lists a group's pending join requests.
type GroupJoinRequestsResponse struct {
	Chat     string                 `json:"chat"`
	Requests []GroupJoinRequestItem `json:"requests"`
	Count    int                    `json:"count"`
}

// GroupJoinRequestsActionResponse is the per-participant outcome of an
// approve/reject (partial success is a 200).
type GroupJoinRequestsActionResponse struct {
	Chat    string              `json:"chat"`
	Action  string              `json:"action"`
	Results []ParticipantResult `json:"results"`
}

// SubGroupItem is one linked group under a community. The default sub-group is
// the community's announcement group.
type SubGroupItem struct {
	JID               string `json:"jid"`
	Name              string `json:"name,omitempty"`
	IsDefaultSubGroup bool   `json:"is_default_sub_group"`
}

// SubGroupListResponse is a community's linked sub-groups (Count == len(SubGroups)).
type SubGroupListResponse struct {
	SubGroups []SubGroupItem `json:"sub_groups"`
	Count     int            `json:"count"`
}

// CommunityParticipantItem is one member across a community's linked groups.
type CommunityParticipantItem struct {
	JID string `json:"jid"`
}

// CommunityParticipantsResponse is every participant across a community's linked
// groups (Count == len(Participants)).
type CommunityParticipantsResponse struct {
	Participants []CommunityParticipantItem `json:"participants"`
	Count        int                        `json:"count"`
}
