package waga

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// WebhookVerifier handles verification of incoming webhook signatures.
// It ensures that webhook payloads are genuine and haven't been tampered with
// by verifying HMAC-SHA256 signatures.
//
// Security considerations:
//   - Always keep your HMAC secret secure and never expose it in client-side code
//   - Use constant-time comparison to prevent timing attacks
//   - Reject webhooks with invalid or missing signatures
type WebhookVerifier struct {
	secret string
}

// NewWebhookVerifier creates a new webhook verifier with the given HMAC secret.
// The secret must match the one configured when registering the webhook.
//
// Example:
//
//	verifier := waga.NewWebhookVerifier("your-hmac-secret")
func NewWebhookVerifier(secret string) *WebhookVerifier {
	return &WebhookVerifier{secret: secret}
}

// VerifySignature validates the X-Webhook-Signature header from a webhook request.
// The signature header format is: "sha256=<hex_signature>".
//
// This method uses constant-time comparison to prevent timing attacks.
// It returns false if the signature is missing, malformed, or doesn't match.
//
// Example:
//
//	signature := r.Header.Get("X-Webhook-Signature")
//	if !verifier.VerifySignature(body, signature) {
//	    return ErrInvalidSignature
//	}
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

// ParseIncomingWebhook parses and verifies an incoming message webhook payload.
// It returns the parsed payload if the signature is valid, or an error otherwise.
//
// This method verifies the signature first, then unmarshals the JSON payload.
// If either step fails, it returns an error.
//
// Example:
//
//	body, _ := io.ReadAll(r.Body)
//	signature := r.Header.Get("X-Webhook-Signature")
//	webhook, err := verifier.ParseIncomingWebhook(body, signature)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Message from %s: %s\n", webhook.From, webhook.Text)
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

// ParseOutgoingWebhook parses and verifies an outgoing event webhook payload.
// It returns the parsed payload if the signature is valid, or an error otherwise.
//
// Outgoing webhooks track the delivery status of messages sent through the API.
// This method verifies the signature first, then unmarshals the JSON payload.
//
// Example:
//
//	body, _ := io.ReadAll(r.Body)
//	signature := r.Header.Get("X-Webhook-Signature")
//	webhook, err := verifier.ParseOutgoingWebhook(body, signature)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Message %s: %s\n", webhook.Event, webhook.MessageId)
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

// ComputeSignature computes the HMAC-SHA256 signature for a webhook payload.
// This is primarily useful for testing webhook implementations.
//
// The returned signature is in the format "sha256=<hex_signature>", which matches
// the format used in the X-Webhook-Signature header.
//
// Example:
//
//	payload := []byte(`{"event":"message.incoming"}`)
//	signature := waga.ComputeSignature(payload, "your-secret")
//	// Use signature to set X-Webhook-Signature header in test requests
func ComputeSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
