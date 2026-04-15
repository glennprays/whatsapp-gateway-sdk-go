package waga

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{"event":"message.incoming","from":"6281234567890"}`)

	// Compute correct signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	correctSig := "sha256=" + hex.EncodeToString(h.Sum(nil))

	// Test valid signature
	if !verifier.VerifySignature(payload, correctSig) {
		t.Error("expected signature to be valid")
	}

	// Test invalid signature
	if verifier.VerifySignature(payload, "sha256=invalid") {
		t.Error("expected signature to be invalid")
	}

	// Test missing prefix
	if verifier.VerifySignature(payload, "invalid") {
		t.Error("expected signature without prefix to be invalid")
	}

	// Test empty signature
	if verifier.VerifySignature(payload, "") {
		t.Error("expected empty signature to be invalid")
	}
}

func TestComputeSignature(t *testing.T) {
	secret := "my_hmac_secret"
	payload := []byte(`{"test":"data"}`)

	sig := ComputeSignature(payload, secret)

	verifier := NewWebhookVerifier(secret)
	if !verifier.VerifySignature(payload, sig) {
		t.Errorf("ComputeSignature produced invalid signature: %s", sig)
	}
}

func TestParseIncomingWebhook(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.incoming",
		"chat": "6281234567890@s.whatsapp.net",
		"from": "6289876543210@s.whatsapp.net",
		"is_group": false,
		"message_id": "msg_123",
		"push_name": "John Doe",
		"timestamp": 1625247600,
		"text": "Hello!",
		"type": "text"
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Event != "message.incoming" {
		t.Errorf("expected event message.incoming, got %s", webhook.Event)
	}
	if webhook.Text != "Hello!" {
		t.Errorf("expected text Hello!, got %s", webhook.Text)
	}
	if webhook.IsGroup {
		t.Error("expected is_group to be false")
	}
}

func TestParseIncomingWebhookInvalidSignature(t *testing.T) {
	verifier := NewWebhookVerifier("secret")
	payload := []byte(`{"event":"message.incoming"}`)

	_, err := verifier.ParseIncomingWebhook(payload, "sha256=invalid")
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestParseOutgoingWebhook(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.sent",
		"job_id": "job_123",
		"to": "6289876543210@s.whatsapp.net",
		"phone_number": "6281234567890",
		"timestamp": 1625247600,
		"message_id": "msg_456"
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseOutgoingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Event != "message.sent" {
		t.Errorf("expected event message.sent, got %s", webhook.Event)
	}
	if webhook.JobId != "job_123" {
		t.Errorf("expected job_id job_123, got %s", webhook.JobId)
	}
}

// ============================================================================
// Webhook Edge Case Tests
// ============================================================================

// TestParseIncomingWebhook_InvalidJSON tests parsing webhook with invalid JSON
func TestParseIncomingWebhook_InvalidJSON(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{invalid json}`)
	sig := ComputeSignature(payload, secret)

	_, err := verifier.ParseIncomingWebhook(payload, sig)
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
}

// TestParseIncomingWebhook_MissingRequiredFields tests parsing webhook with missing fields
func TestParseIncomingWebhook_MissingRequiredFields(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	// Missing required fields like event, from, etc.
	payload := []byte(`{"chat": "test"}`)
	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that missing fields are empty/zero values
	if webhook.Event != "" {
		t.Errorf("expected empty event, got %s", webhook.Event)
	}
	if webhook.From != "" {
		t.Errorf("expected empty from, got %s", webhook.From)
	}
}

// TestParseIncomingWebhook_MediaMessage tests parsing webhook with media message
func TestParseIncomingWebhook_MediaMessage(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.incoming",
		"chat": "6281234567890@s.whatsapp.net",
		"from": "6289876543210@s.whatsapp.net",
		"is_group": false,
		"message_id": "msg_media_123",
		"push_name": "Jane Doe",
		"timestamp": 1625247600,
		"type": "image",
		"media": {
			"type": "image",
			"url": "https://example.com/image.jpg",
			"mime_type": "image/jpeg",
			"filename": "photo.jpg",
			"caption": "Check this out!",
			"size": 102400
		}
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Type != IncomingMessageTypeImage {
		t.Errorf("expected type 'image', got %s", webhook.Type)
	}
	if webhook.Media == nil {
		t.Fatal("expected media to be present")
	}
	if webhook.Media.Type != IncomingMessageTypeImage {
		t.Errorf("expected media type 'image', got %s", webhook.Media.Type)
	}
	if webhook.Media.Url != "https://example.com/image.jpg" {
		t.Errorf("expected URL 'https://example.com/image.jpg', got %s", webhook.Media.Url)
	}
	if webhook.Media.Caption != "Check this out!" {
		t.Errorf("expected caption 'Check this out!', got %s", webhook.Media.Caption)
	}
	if webhook.Media.Size != 102400 {
		t.Errorf("expected size 102400, got %d", webhook.Media.Size)
	}
}

// TestParseIncomingWebhook_VideoMessage tests parsing webhook with video message
func TestParseIncomingWebhook_VideoMessage(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.incoming",
		"chat": "6281234567890@s.whatsapp.net",
		"from": "6289876543210@s.whatsapp.net",
		"is_group": false,
		"message_id": "msg_video_123",
		"push_name": "John Doe",
		"timestamp": 1625247600,
		"type": "video",
		"media": {
			"type": "video",
			"url": "https://example.com/video.mp4",
			"mime_type": "video/mp4"
		}
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Type != IncomingMessageTypeVideo {
		t.Errorf("expected type 'video', got %s", webhook.Type)
	}
	if webhook.Media.MimeType != "video/mp4" {
		t.Errorf("expected mime type 'video/mp4', got %s", webhook.Media.MimeType)
	}
}

// TestParseIncomingWebhook_AudioMessage tests parsing webhook with audio message
func TestParseIncomingWebhook_AudioMessage(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.incoming",
		"chat": "6281234567890@s.whatsapp.net",
		"from": "6289876543210@s.whatsapp.net",
		"is_group": false,
		"message_id": "msg_audio_123",
		"push_name": "Voice Sender",
		"timestamp": 1625247600,
		"type": "audio",
		"media": {
			"type": "audio",
			"url": "https://example.com/audio.ogg",
			"mime_type": "audio/ogg"
		}
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Type != IncomingMessageTypeAudio {
		t.Errorf("expected type 'audio', got %s", webhook.Type)
	}
}

// TestParseIncomingWebhook_DocumentMessage tests parsing webhook with document message
func TestParseIncomingWebhook_DocumentMessage(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.incoming",
		"chat": "6281234567890@s.whatsapp.net",
		"from": "6289876543210@s.whatsapp.net",
		"is_group": false,
		"message_id": "msg_doc_123",
		"push_name": "Doc Sender",
		"timestamp": 1625247600,
		"type": "document",
		"media": {
			"type": "document",
			"url": "https://example.com/document.pdf",
			"mime_type": "application/pdf",
			"filename": "report.pdf",
			"size": 2048000
		}
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Type != IncomingMessageTypeDocument {
		t.Errorf("expected type 'document', got %s", webhook.Type)
	}
	if webhook.Media.Filename != "report.pdf" {
		t.Errorf("expected filename 'report.pdf', got %s", webhook.Media.Filename)
	}
}

// TestParseIncomingWebhook_GroupMessage tests parsing webhook from group
func TestParseIncomingWebhook_GroupMessage(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.incoming",
		"chat": "1234567890@g.us",
		"from": "6289876543210@s.whatsapp.net",
		"is_group": true,
		"message_id": "msg_group_123",
		"push_name": "Group Member",
		"timestamp": 1625247600,
		"text": "Group message",
		"type": "text"
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !webhook.IsGroup {
		t.Error("expected is_group to be true")
	}
	if webhook.Chat != "1234567890@g.us" {
		t.Errorf("expected group chat ID, got %s", webhook.Chat)
	}
}

// TestParseIncomingWebhook_EmptyPayload tests parsing webhook with empty payload
func TestParseIncomingWebhook_EmptyPayload(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{}`)
	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseIncomingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All fields should be zero values
	if webhook.Event != "" {
		t.Errorf("expected empty event, got %s", webhook.Event)
	}
}

// TestParseOutgoingWebhook_InvalidJSON tests parsing outgoing webhook with invalid JSON
func TestParseOutgoingWebhook_InvalidJSON(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{invalid json}`)
	sig := ComputeSignature(payload, secret)

	_, err := verifier.ParseOutgoingWebhook(payload, sig)
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
}

// TestParseOutgoingWebhook_WithMetadata tests parsing outgoing webhook with metadata
func TestParseOutgoingWebhook_WithMetadata(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.sent",
		"job_id": "job_123",
		"to": "6289876543210@s.whatsapp.net",
		"phone_number": "6281234567890",
		"timestamp": 1625247600,
		"message_id": "msg_456",
		"metadata": {
			"custom_field": "custom_value",
			"priority": "high",
			"retry_count": 3
		}
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseOutgoingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Metadata == nil {
		t.Fatal("expected metadata to be present")
	}

	metadata := *webhook.Metadata
	if metadata["custom_field"] != "custom_value" {
		t.Errorf("expected custom_field 'custom_value', got %v", metadata["custom_field"])
	}
	if metadata["priority"] != "high" {
		t.Errorf("expected priority 'high', got %v", metadata["priority"])
	}
	if metadata["retry_count"] != float64(3) {
		t.Errorf("expected retry_count 3, got %v", metadata["retry_count"])
	}
}

// TestParseOutgoingWebhook_MessagQueuedEvent tests parsing message.queued event
func TestParseOutgoingWebhook_MessagQueuedEvent(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.queued",
		"job_id": "job_queue_123",
		"to": "6289876543210@s.whatsapp.net",
		"phone_number": "6281234567890",
		"timestamp": 1625247600,
		"message_id": "msg_queue_456"
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseOutgoingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Event != OutgoingEventMessageMessageQueued {
		t.Errorf("expected event 'message.queued', got %s", webhook.Event)
	}
}

// TestParseOutgoingWebhook_MessageFailedEvent tests parsing message.failed event
func TestParseOutgoingWebhook_MessageFailedEvent(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte(`{
		"event": "message.failed",
		"job_id": "job_failed_123",
		"to": "6289876543210@s.whatsapp.net",
		"phone_number": "6281234567890",
		"timestamp": 1625247600,
		"message_id": "msg_failed_456"
	}`)

	sig := ComputeSignature(payload, secret)

	webhook, err := verifier.ParseOutgoingWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if webhook.Event != OutgoingEventMessageMessageFailed {
		t.Errorf("expected event 'message.failed', got %s", webhook.Event)
	}
}

// TestVerifySignature_WithDifferentSecrets tests signature verification with different secrets
func TestVerifySignature_WithDifferentSecrets(t *testing.T) {
	secret1 := "secret_one"
	secret2 := "secret_two"

	payload := []byte(`{"test":"data"}`)
	sig1 := ComputeSignature(payload, secret1)

	verifier2 := NewWebhookVerifier(secret2)
	if verifier2.VerifySignature(payload, sig1) {
		t.Error("expected signature to be invalid with different secret")
	}
}

// TestVerifySignature_WithEmptyPayload tests signature verification with empty payload
func TestVerifySignature_WithEmptyPayload(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload := []byte{}
	sig := ComputeSignature(payload, secret)

	if !verifier.VerifySignature(payload, sig) {
		t.Error("expected signature to be valid for empty payload")
	}
}

// TestVerifySignature_ModifiedPayload tests signature verification with modified payload
func TestVerifySignature_WithModifiedPayload(t *testing.T) {
	secret := "my_hmac_secret"
	verifier := NewWebhookVerifier(secret)

	payload1 := []byte(`{"original":"data"}`)
	payload2 := []byte(`{"modified":"data"}`)
	sig1 := ComputeSignature(payload1, secret)

	if verifier.VerifySignature(payload2, sig1) {
		t.Error("expected signature to be invalid for modified payload")
	}
}
