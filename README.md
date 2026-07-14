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

### Send Options

Every `SendXxx` method takes a trailing variadic of `SendOption`s for canonical
chat addressing, quoted replies, @-mentions, and send idempotency. Existing call
sites keep working unchanged — options are purely additive.

```go
resp, err := client.SendText(ctx, recipient, "Hi team!",
    waga.WithChat("120363000000000000@g.us"),        // canonical recipient (see below)
    waga.WithReply("MSG_ID", senderJID, "quoted text"), // quote an existing message
    waga.WithMentions("6281111111111", "6282222222222"), // @-tag participants
    waga.WithIdempotencyKey("order-4711-notify"),       // safe retry (see Idempotency)
)
```

The same options work on the media sends (`SendImage`/`SendAudio`/`SendVideo`/
`SendDocument`/`SendSticker`) and `SendLocation`/`SendPoll`.

#### Chat Addressing

`WithChat` sets the **canonical** recipient — a bare number, a user JID
(`@s.whatsapp.net`), a group JID (`@g.us`), or a `@lid`. It takes precedence over
the positional recipient argument, which the gateway treats as the deprecated
`msisdn` alias. Send responses echo the resolved recipient in `resp.Chat`:

```go
resp, _ := client.SendText(ctx, "", "Hi!", waga.WithChat("120363000000000000@g.us"))
fmt.Println("delivered to:", resp.Chat)
```

#### Idempotency

`WithIdempotencyKey` sends an `Idempotency-Key` header. Reusing the same key
replays the gateway's original response (HTTP 200); an in-flight duplicate
returns `409` (`waga.ErrConflict`) and the same key with a **different** request
body returns `422` (`SDKError.Code == 422`). The header is only sent when a key
is provided.

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

### Audio Message

```go
file, err := os.Open("voice.ogg")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

resp, err := client.SendAudio(ctx, recipient, file, true, false) // isPTT=true
if err != nil {
    log.Fatal(err)
}
fmt.Println("Message ID:", resp.MessageId)
```

### Video Message

```go
file, err := os.Open("clip.mp4")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

resp, err := client.SendVideo(ctx, recipient, file, "Look at this", false, false)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Message ID:", resp.MessageId)
```

### Document Message

```go
file, err := os.Open("invoice.pdf")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

resp, err := client.SendDocument(ctx, recipient, file, "invoice.pdf", "Monthly invoice")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Message ID:", resp.MessageId)
```

### Location Message

```go
resp, err := client.SendLocation(ctx, recipient, -6.2088, 106.8456, "Jakarta", "Jakarta, Indonesia")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Message ID:", resp.MessageId)
```

### Poll Message

```go
options := []string{"Red", "Green", "Blue"}

// selectableCount limits how many options a user can select (0 = no limit)
resp, err := client.SendPoll(ctx, recipient, "What is your favorite color?", options, 1)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Message ID:", resp.MessageId)
```

### Sticker Message

```go
// Open sticker file (WebP format)
file, err := os.Open("sticker.webp")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

resp, err := client.SendSticker(ctx, recipient, file)
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

For reactions to incoming messages in groups/DMs, pass optional `sender_msisdn`:

```go
err := client.React(ctx, groupJID, messageID, "👍", "6281111111111@s.whatsapp.net")
```

### Check Contact

```go
contact, err := client.CheckContact(ctx, "6281234567890")
if err != nil {
    log.Fatal(err)
}
fmt.Println(contact.JID, contact.IsOnWhatsApp, contact.VerifiedName)
```

## Incoming Messages

Fetch the most recent incoming messages buffered by the gateway (newest first):

```go
resp, err := client.GetIncomingMessages(ctx, 10)
if err != nil {
    log.Fatal(err)
}
for _, msg := range resp.Messages {
    fmt.Printf("[%s] %s: %s\n", msg.Type, msg.From, msg.Text)
}
```

Note: media URLs are not populated by this endpoint; use webhooks for fetchable media.

## Contacts & Groups

All of these accept the canonical `chat` (a bare number, `@s.whatsapp.net`,
`@g.us`, or `@lid`).

### List Contacts

Locally-synced contacts, paginated (`limit` defaults to 100, max 500). Never
errors on an empty address book:

```go
page, err := client.ListContacts(ctx, 100, 0) // limit, offset
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%d of %d contacts\n", page.Count, page.Total)
for _, ct := range page.Contacts {
    fmt.Println(ct.JID, ct.PushName)
}
```

### Contact Info

Server-side profile lookup (status, picture id, verified name, device count, lid):

```go
info, err := client.GetContactInfo(ctx, "6281234567890")
if err != nil {
    log.Fatal(err)
}
fmt.Println(info.Status, info.DeviceCount, info.LID)
```

### Avatar (with conditional fetch)

Profile picture for a user *or* group. `preview` requests the thumbnail. The
returned `ID` doubles as an ETag — pass it back as the optional `priorID` to skip
re-downloading when unchanged:

```go
av, err := client.GetAvatar(ctx, "6281234567890", false)
if err != nil {
    if errors.Is(err, waga.ErrNotFound) {
        // no profile picture
    } else if errors.Is(err, waga.ErrForbidden) {
        // picture hidden by privacy settings
    }
    return
}
fmt.Println(av.URL, av.ID)

// Later — only re-fetch if the picture changed:
av2, err := client.GetAvatar(ctx, "6281234567890", false, av.ID)
if errors.Is(err, waga.ErrNotModified) {
    // unchanged; keep the cached av
}
```

### List Groups

Joined-group summaries. This is a budgeted server read — a `429` maps to
`ErrRateLimited`:

```go
groups, err := client.ListGroups(ctx)
if errors.Is(err, waga.ErrRateLimited) {
    // read budget exhausted; back off
}
for _, g := range groups.Groups {
    fmt.Println(g.JID, g.Name, g.ParticipantCount)
}
```

### Group Info

Full detail plus participant roster. Requires a group JID; `ErrForbidden` if the
account is not a member, `ErrNotFound` if the group is absent:

```go
g, err := client.GetGroupInfo(ctx, "120363000000000000@g.us")
if err != nil {
    log.Fatal(err)
}
for _, p := range g.Participants {
    fmt.Println(p.JID, p.IsAdmin, p.IsSuperAdmin)
}
```

## Read Receipts & Presence

### Mark Messages as Read

Sends blue ticks. For group chats, pass the message author as `sender`:

```go
err := client.MarkRead(ctx, "120363000000000000@g.us",
    []string{"MSG_ID_1", "MSG_ID_2"},
    "6281234567890@s.whatsapp.net") // sender (author); "" for one-to-one chats
```

### Typing Indicator

```go
err := client.SendChatPresence(ctx, recipient, waga.PresenceComposing) // typing…
// ... waga.PresenceRecording (voice note) / waga.PresencePaused (cleared)
```

## Group & Community Management

Every group/community method requires an explicit group JID (`@g.us`); a bare
number or user JID is rejected with `400`. These endpoints are gated server-side
by `GROUP_MANAGEMENT_ENABLED` (the whole mutation surface returns `404` when the
gateway disables it). Batch operations return `200` with a per-participant
`Results` slice — a single bad member is reported there, not as an overall error.

```go
// Create a group (or a community with IsCommunity: true).
grp, err := client.CreateGroup(ctx, waga.CreateGroupRequest{
    Name:         "Project X",
    Participants: []string{"6281111111111", "6282222222222"},
})

// Roster mutations: add | remove | promote | demote.
res, err := client.UpdateGroupParticipants(ctx, grp.GroupJID, "add", []string{"6283333333333"})
for _, r := range res.Results {
    fmt.Println(r.JID, r.Status) // ok | invited (privacy-blocked add) | failed
}

// Settings, name, topic.
announce := true
client.SetGroupSettings(ctx, grp.GroupJID, &announce, nil) // ≥1 non-nil flag
client.SetGroupName(ctx, grp.GroupJID, "Project X — Q3")   // ≤25 chars
client.SetGroupTopic(ctx, grp.GroupJID, "")                // ≤512; empty clears

// Photo (multipart JPEG).
f, _ := os.Open("group.jpg")
defer f.Close()
client.SetGroupPhoto(ctx, grp.GroupJID, f)
client.DeleteGroupPhoto(ctx, grp.GroupJID)

// Invite links.
link, _ := client.GetGroupInviteLink(ctx, grp.GroupJID)
client.ResetGroupInviteLink(ctx, grp.GroupJID) // revoke + regenerate

// Preview a link without joining (a revoked link matches ErrGone).
info, err := client.GetGroupInviteInfo(ctx, "https://chat.whatsapp.com/XXXX")
if errors.Is(err, waga.ErrGone) {
    // link revoked
}
client.JoinGroup(ctx, "https://chat.whatsapp.com/XXXX") // gated by GROUP_JOIN_VIA_LINK_ENABLED

// Pending join requests.
reqs, _ := client.ListJoinRequests(ctx, grp.GroupJID)
client.ReviewJoinRequests(ctx, grp.GroupJID, "approve", []string{"6284444444444"}) // approve | reject

// Communities.
client.LinkSubGroup(ctx, communityJID, subGroupJID)
client.UnlinkSubGroup(ctx, communityJID, subGroupJID)
subs, _ := client.ListSubGroups(ctx, communityJID)
members, _ := client.ListCommunityParticipants(ctx, communityJID)
```

Read-only group/community methods (`ListGroups`, `GetGroupInfo`, `ListSubGroups`,
`ListCommunityParticipants`) stay available even when mutations are disabled.

## Admin Module

The gateway's operator-only admin plane is exposed as a **separate, opt-in
client** so tenant code can't accidentally reach it. It is served at the server
**root** (not under `/api/v1`) and is bearer-gated by the gateway's
`ADMIN_API_SECRET` (the plane returns `404` when that secret is unset).

Construct it with `NewAdminClient`, pointing `WithBaseURL` at the gateway origin
and passing the admin secret via `WithAdminSecret`:

```go
admin := waga.NewAdminClient(
    waga.WithBaseURL("https://gateway.example.com"), // server ROOT, no /api/v1
    waga.WithAdminSecret(os.Getenv("ADMIN_API_SECRET")),
)

// Per-instance session inventory (masked phones, honest states, hostname).
inv, err := admin.Sessions(ctx)
for _, s := range inv.Sessions {
    fmt.Println(s.PhoneMasked, s.State)
}

// One account (ErrNotFound if unknown).
one, err := admin.Session(ctx, "6281234567890")

// Root health probes (no admin secret required).
live, _ := admin.Live(ctx)   // always 200 for a running process
ready, err := admin.Ready(ctx)
if err == nil && ready.Status != "ready" {
    // not_ready (HTTP 503): DB or queue down — the body is still returned
}
```

## Job Status

When the gateway runs in queue mode, send methods return a job ID instead of a
message ID. The send response tells you which mode you're in:

```go
resp, err := client.SendText(ctx, recipient, "Hello!")
if err != nil {
    log.Fatal(err)
}
if resp.JobID != "" {
    // Queue mode: poll the job until it completes
    status, _ := client.GetJobStatus(ctx, resp.JobID)
    fmt.Println("Status:", status.Status) // queued | processing | completed | failed
    if status.MessageID != nil {
        fmt.Println("Message ID:", *status.MessageID)
    }
} else {
    // Direct mode: the message ID is available immediately
    fmt.Println("Message ID:", resp.MessageId)
}
```

## Trace IDs

Attach a trace ID to correlate your application logs with gateway logs. Every
request made with the returned context sends it as the `X-Trace-ID` header:

```go
ctx := waga.WithTraceID(context.Background(), "order-12345")
resp, err := client.SendText(ctx, recipient, "Your order shipped!")
```

If a request fails, the gateway's trace ID is attached to the error so you can
find the failing request in the gateway logs:

```go
if err != nil {
    var apiErr *waga.SDKError
    if errors.As(err, &apiErr) {
        log.Printf("gateway error (trace %s): %s", apiErr.TraceID, apiErr.Message)
    }
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

### Unified Dispatch with ParseWebhook

When a single endpoint receives every webhook type, `ParseWebhook` verifies the
signature and dispatches on the `event` field into a discriminated
`WebhookEvent`. Exactly one of `Incoming`, `Outgoing`, or `Session` is non-nil.
The narrower `ParseIncomingWebhook` / `ParseOutgoingWebhook` remain available.

```go
ev, err := verifier.ParseWebhook(body, signature)
if err != nil {
    if errors.Is(err, waga.ErrUnknownWebhookEvent) {
        http.Error(w, "unknown event", http.StatusBadRequest)
        return
    }
    // errors.Is(err, waga.ErrInvalidSignature) for a bad signature
    http.Error(w, "invalid webhook", http.StatusBadRequest)
    return
}

switch ev.Event {
case waga.WebhookEventMessageIncoming:
    fmt.Printf("from %s: %s\n", ev.Incoming.From, ev.Incoming.Text)
case waga.WebhookEventMessageSent, waga.WebhookEventMessageQueued, waga.WebhookEventMessageFailed:
    fmt.Printf("%s: %s\n", ev.Outgoing.Event, ev.Outgoing.MessageId)
case waga.WebhookEventSessionBanned:
    fmt.Printf("account %s banned for %ds\n", ev.Session.PhoneNumber, ev.Session.ExpiresIn)
default: // other session.* lifecycle events
    fmt.Printf("session event %s for %s\n", ev.Event, ev.Session.JID)
}
```

#### Session Events

Besides the four message events, the gateway emits six `session.*` lifecycle
events, decoded into `SessionEvent` (flat envelope `Event`/`PhoneNumber`/`JID`/
`Timestamp` plus event-specific extras):

| Event | Extras |
|-------|--------|
| `WebhookEventSessionLoggedOut` | `OnConnect`, `Reason`, `ReasonText` |
| `WebhookEventSessionBanned` | `Code`, `ReasonText`, `ExpiresIn` |
| `WebhookEventSessionConnectFailure` | `Reason`, `ReasonText`, `Message` |
| `WebhookEventSessionConnected` | envelope only |
| `WebhookEventSessionDisconnected` | envelope only |
| `WebhookEventSessionReplaced` | envelope only |

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
| `ErrGone` | 410 | Resource gone (e.g. revoked group invite link) |
| `ErrNotModified` | 304 | `GetAvatar` — picture unchanged (conditional fetch) |
| `ErrRateLimited` | 429 | Too many requests |
| `ErrInternalServer` | 500 | Server error |
| `ErrInvalidSignature` | - | Webhook signature verification failed |
| `ErrUnknownWebhookEvent` | - | `ParseWebhook` got an unrecognized event |
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
