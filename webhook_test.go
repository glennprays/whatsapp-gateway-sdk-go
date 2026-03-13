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
