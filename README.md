# WhatsApp Gateway Go SDK

A Go SDK for the [WhatsApp Gateway](https://github.com/glennprays/whatsapp-gateway) API. This SDK provides an ergonomic client wrapper around generated types from the OpenAPI specification.

## Installation

```bash
go get github.com/glennprays/whatsapp-gateway-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/glennprays/whatsapp-gateway-sdk-go"
)

func main() {
    // Create client
    client := waga.NewClient(waga.WithBaseURL("http://localhost:3000/api/v1"))

    // Register phone number and get JWT token
    _, err := client.Register(context.Background(), "6281234567890", "your_secret_key")
    if err != nil {
        log.Fatal(err)
    }

    // Get QR code for WhatsApp login
    qr, err := client.GetQRCode(context.Background(), "json")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Scan this QR code:", qr.QRCode)

    // Send a message
    resp, err := client.SendText(context.Background(), "6289876543210@s.whatsapp.net", "Hello!")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Message sent:", resp.MessageId)
}
```

## Configuration

The client can be configured with various options:

```go
client := waga.NewClient(
    waga.WithBaseURL("https://your-gateway.com/api/v1"),
    waga.WithTimeout(30 * time.Second),
    waga.WithToken("existing_jwt_token"),  // If you already have a token
    waga.WithUserAgent("MyApp/1.0"),
)
```

### Available Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithBaseURL(url)` | API base URL | `http://localhost:3000/api/v1` |
| `WithTimeout(d)` | HTTP request timeout | `30s` |
| `WithToken(token)` | Pre-existing JWT token | `""` |
| `WithHTTPClient(c)` | Custom HTTP client | Default client |
| `WithUserAgent(ua)` | Custom User-Agent header | `WhatsApp-Gateway-SDK-Go/1.0` |

## Authentication

### Register New Phone Number

```go
resp, err := client.Register(ctx, "6281234567890", "your_secret_key")
if err != nil {
    log.Fatal(err)
}
// Token is automatically stored in client
fmt.Println("Token:", resp.Token)
```

### Use Existing Token

```go
// Option 1: During client creation
client := waga.NewClient(waga.WithToken("your_jwt_token"))

// Option 2: Set token later
client := waga.NewClient()
client.SetToken("your_jwt_token")
```

## WhatsApp Authentication

### Get QR Code

```go
// Get QR code as base64 (json format)
qr, err := client.GetQRCode(ctx, "json")
if err != nil {
    log.Fatal(err)
}
fmt.Println("QR Code:", qr.QRCode)        // Base64 encoded image
fmt.Println("Expires in:", qr.ExpiresIn)  // Seconds until expiry

// Get QR code as HTML img tag (html format)
qrHTML, err := client.GetQRCode(ctx, "html")
```

### Get Pair Code

```go
pair, err := client.GetPairCode(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Pair Code:", pair.PairCode)
fmt.Println("Expires in:", pair.ExpiresIn)
```

### Check Login Status

```go
status, err := client.GetLoginStatus(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Authenticated:", status.Authenticated)
```

### Reconnect Session

```go
err := client.Reconnect(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Logout

```go
err := client.Logout(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Sending Messages

### Text Message

```go
// Format phone number to WhatsApp JID format
recipient := waga.FormatMSISDN("6289876543210")  // -> "6289876543210@s.whatsapp.net"

resp, err := client.SendText(ctx, recipient, "Hello from Go SDK!")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Message ID:", resp.MessageId)
```

### Image Message

```go
// Open image file
file, err := os.Open("photo.jpg")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

resp, err := client.SendImage(ctx, recipient, file, "Check this out!", false)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Message ID:", resp.MessageId)
```

### Edit Message

```go
err := client.EditMessage(ctx, recipient, messageID, "Updated message")
if err != nil {
    log.Fatal(err)
}
```

### Delete Message

```go
err := client.DeleteMessage(ctx, recipient, messageID)
if err != nil {
    log.Fatal(err)
}
```

### React to Message

```go
err := client.React(ctx, recipient, messageID, "👍")
if err != nil {
    log.Fatal(err)
}
```

## Webhook Management

### Register Webhook

```go
err := client.RegisterWebhook(ctx, "https://example.com/webhook", "my_hmac_secret")
if err != nil {
    log.Fatal(err)
}
```

### Get Registered Webhook

```go
webhook, err := client.GetWebhook(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Webhook URL:", webhook.URL)
```

### Unregister Webhook

```go
err := client.UnregisterWebhook(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Webhook Verification

When receiving webhooks on your server, verify the signature:

```go
import (
    "io"
    "net/http"
    "errors"

    "github.com/glennprays/whatsapp-gateway-sdk-go"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Webhook-Signature")

    // Create verifier with your HMAC secret
    verifier := waga.NewWebhookVerifier("my_hmac_secret")

    // Verify and parse incoming message webhook
    payload, err := verifier.ParseIncomingWebhook(body, signature)
    if err != nil {
        if errors.Is(err, waga.ErrInvalidSignature) {
            http.Error(w, "Invalid signature", http.StatusUnauthorized)
            return
        }
        http.Error(w, "Invalid payload", http.StatusBadRequest)
        return
    }

    // Process the webhook
    fmt.Printf("Received message from %s: %s\n", payload.From, payload.Text)
}
```

### Webhook Payload Types

#### Incoming Message

```go
type IncomingWebhookPayload struct {
    Event      IncomingEventMessage      // "message.incoming"
    Chat       string                    // Chat ID
    From       string                    // Sender's ID
    IsGroup    bool                      // Is from group
    MessageId string                    // Message ID
    PushName   string                    // Sender's display name
    Timestamp  int                       // Unix timestamp
    Text       string                    // Message text (if text message)
    Type       IncomingMessageType       // "text", "image", "video", "audio", "document"
    Media      *IncomingMessageMediaInfo // Media info (if media message)
}
```

#### Outgoing Event

```go
type OutgoingWebhookPayload struct {
    Event       OutgoingEventMessage // "message.queued", "message.sent", "message.failed"
    JobId       string               // Job ID
    To          string               // Recipient's ID
    PhoneNumber string               // Sender's phone number
    Timestamp   int                  // Unix timestamp
    MessageId   string               // Message ID
    Metadata    map[string]interface{} // Additional metadata
}
```

## Error Handling

```go
_, err := client.SendText(ctx, recipient, "Hello")
if err != nil {
    var sdkErr *waga.SDKError
    if waga.IsUnauthorized(err) {
        // Token expired or invalid
        fmt.Println("Please re-register")
    } else if waga.IsBadRequest(err) {
        // Invalid request parameters
        fmt.Println("Check your input")
    } else if waga.IsRateLimited(err) {
        // Too many requests
        fmt.Println("Please wait before retrying")
    } else if errors.As(err, &sdkErr) {
        // Other API error
        fmt.Printf("Error %d: %s\n", sdkErr.Code, sdkErr.Message)
    } else {
        // Network or other error
        fmt.Printf("Unexpected error: %v\n", err)
    }
}
```

### Error Types

| Error | HTTP Code | Description |
|-------|-----------|-------------|
| `ErrUnauthorized` | 401 | Invalid or missing token |
| `ErrBadRequest` | 400 | Invalid request parameters |
| `ErrForbidden` | 403 | No permission for this action |
| `ErrNotFound` | 404 | Resource not found |
| `ErrConflict` | 409 | Resource conflict |
| `ErrRateLimited` | 429 | Too many requests |
| `ErrInternalServer` | 500 | Server error |
| `ErrInvalidSignature` | - | Webhook signature verification failed |
| `ErrNotAuthenticated` | - | Client has no token set |

## Helper Functions

### Format MSISDN

Convert phone number to WhatsApp JID format:

```go
jid := waga.FormatMSISDN("6281234567890")
// Returns: "6281234567890@s.whatsapp.net"
```

### Format Group ID

Convert group ID to WhatsApp group JID format:

```go
jid := waga.FormatGroupID("1234567890")
// Returns: "1234567890@g.us"
```

### Compute Signature

Compute webhook signature (useful for testing):

```go
payload := []byte(`{"event":"message.incoming"}`)
signature := waga.ComputeSignature(payload, "my_hmac_secret")
// Returns: "sha256=..."
```

## Health Check

```go
health, err := client.Health(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Status: %s\n", health.Status)
```

## Development

### Generate Types from OpenAPI

```bash
# Install oapi-codegen
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Generate types
make generate
```

### Run Tests

```bash
go test ./...
```

## Requirements

- Go 1.24 or later
- Running WhatsApp Gateway instance

## License

MIT License
