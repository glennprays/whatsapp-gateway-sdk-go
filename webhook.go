package waga

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// WebhookVerifier handles verification of incoming webhook signatures
type WebhookVerifier struct {
	secret string
}

// NewWebhookVerifier creates a new webhook verifier with the given HMAC secret
func NewWebhookVerifier(secret string) *WebhookVerifier {
	return &WebhookVerifier{secret: secret}
}

// VerifySignature validates the X-Webhook-Signature header
// Header format: sha256=<hex_signature>
func (v *WebhookVerifier) VerifySignature(payload []byte, signatureHeader string) bool {
	if signatureHeader == "" {
		return false
	}

	// Extract signature from header (format: "sha256=<hex>")
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}
	expectedSig := strings.TrimPrefix(signatureHeader, "sha256=")

	// Compute HMAC-SHA256 of payload
	h := hmac.New(sha256.New, []byte(v.secret))
	h.Write(payload)
	computedSig := hex.EncodeToString(h.Sum(nil))

	// Use constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(expectedSig), []byte(computedSig))
}

// ParseIncomingWebhook parses and verifies an incoming message webhook
// Returns the parsed payload if signature is valid, error otherwise
func (v *WebhookVerifier) ParseIncomingWebhook(payload []byte, signature string) (*IncomingWebhookPayload, error) {
	if !v.VerifySignature(payload, signature) {
		return nil, ErrInvalidSignature
	}

	var webhook IncomingWebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// ParseOutgoingWebhook parses and verifies an outgoing event webhook
// Returns the parsed payload if signature is valid, error otherwise
func (v *WebhookVerifier) ParseOutgoingWebhook(payload []byte, signature string) (*OutgoingWebhookPayload, error) {
	if !v.VerifySignature(payload, signature) {
		return nil, ErrInvalidSignature
	}

	var webhook OutgoingWebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// ComputeSignature computes the HMAC-SHA256 signature for a payload
// This is useful for testing webhook implementations
func ComputeSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
