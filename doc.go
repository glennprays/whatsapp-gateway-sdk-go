// Package waga provides a Go client for interacting with the WhatsApp Gateway API.
//
// The waga package enables Go applications to send WhatsApp messages, manage webhooks,
// and handle authentication with a WhatsApp Gateway service. It supports text and image
// messaging, message editing/deletion, webhook verification, and session management.
//
// # Quick Start
//
// To get started, create a new client and register your phone number:
//
//	client := waga.NewClient()
//	resp, err := client.Register(ctx, "6281234567890", "your-secret-key")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Once registered, you can send messages:
//
//	msisdn := waga.FormatMSISDN("6281234567890")
//	msgResp, err := client.SendText(ctx, msisdn, "Hello, World!")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Authentication
//
// The SDK supports two authentication methods:
//
// 1. Register a new phone number to obtain a JWT token
// 2. Use an existing JWT token with SetToken() or WithToken()
//
// The client automatically stores the token returned by Register() for subsequent requests.
//
// # Message Formatting
//
// Phone numbers should be formatted as MSISDN (WhatsApp JID format). Use the helper
// functions FormatMSISDN() for individuals and FormatGroupID() for groups:
//
//	individual := waga.FormatMSISDN("6281234567890")  // "6281234567890@s.whatsapp.net"
//	group := waga.FormatGroupID("1234567890")          // "1234567890@g.us"
//
// # Webhooks
//
// Webhooks allow you to receive real-time notifications for incoming and outgoing messages.
// To handle webhooks securely, use the WebhookVerifier:
//
//	verifier := waga.NewWebhookVerifier("your-hmac-secret")
//	payload, err := verifier.ParseIncomingWebhook(body, signatureHeader)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Configuration
//
// The client can be configured with options:
//
//	client := waga.NewClient(
//	    waga.WithBaseURL("https://api.example.com"),
//	    waga.WithTimeout(60*time.Second),
//	    waga.WithToken("existing-jwt-token"),
//	)
//
// # Error Handling
//
// The SDK returns typed errors that can be checked using helper functions:
//
//	resp, err := client.SendText(ctx, msisdn, message)
//	if waga.IsUnauthorized(err) {
//	    // Handle authentication error
//	} else if waga.IsRateLimited(err) {
//	    // Handle rate limiting
//	}
//
// For more examples and detailed usage, see the examples directory and README.md.
package waga
