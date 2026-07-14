# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Chat addressing, replies, and mentions on all sends.** Every `SendXxx`
  method now accepts a trailing variadic of `SendOption`s (additive; existing
  calls are unchanged):
  - `WithChat(chat)` sets the canonical recipient — a bare number, a user JID,
    a group JID (`@g.us`), or a `@lid` — taking precedence over the positional
    `msisdn` alias. Send request structs gained `Chat`, and `SendMessageResponse`
    now echoes the resolved recipient in `Chat`.
  - `WithReply(id, sender, quotedText)` quotes an existing message.
  - `WithMentions(numbers...)` @-tags participants (sent as repeated multipart
    fields for media sends).
- **`WithIdempotencyKey(key)`** attaches an `Idempotency-Key` header to a send
  (JSON and multipart). Reusing a key replays the original response; an in-flight
  duplicate is `409` (`ErrConflict`) and a key reused with a different body is
  `422`. The header is only sent when a key is provided.
- **`ParseWebhook(payload, signature)`** on `WebhookVerifier`: a unified,
  signature-verifying dispatcher returning a discriminated `WebhookEvent`
  (`Incoming` / `Outgoing` / `Session`). The existing `ParseIncomingWebhook` /
  `ParseOutgoingWebhook` are unchanged.
- **Session lifecycle webhooks.** New `SessionEvent` type and the six
  `session.*` event constants (`logged_out`, `banned`, `connect_failure`,
  `connected`, `disconnected`, `replaced`), plus a unified `WebhookEventType`
  catalog covering the four message events too.
- New `ErrUnknownWebhookEvent` sentinel returned by `ParseWebhook` for
  unrecognized events.
- **Contact and group read methods** with matching response types:
  - `ListContacts(ctx, limit, offset)` → `GET /contact/` (paginated locally-synced
    contacts; `count`/`total`; never 404s on empty).
  - `GetContactInfo(ctx, chat)` → `GET /contact/info` (status, picture id,
    verified name, device count, lid).
  - `GetAvatar(ctx, chat, preview, priorID...)` → `GET /contact/avatar` for a user
    or group. Pass the previous `AvatarResponse.ID` as the optional `priorID` for
    conditional (`If-None-Match`) fetches: an unchanged picture returns the new
    `ErrNotModified` sentinel (304); no picture → `ErrNotFound` (404); hidden →
    `ErrForbidden` (403).
  - `ListGroups(ctx)` → `GET /group/` (joined-group summaries; may return
    `ErrRateLimited` (429) when the per-account read budget is exhausted).
  - `GetGroupInfo(ctx, chat)` → `GET /group/info` (full detail + participant
    roster; `ErrForbidden` if not a member, `ErrNotFound` if absent).
- **Two-way primitives:**
  - `MarkRead(ctx, chat, messageIDs, sender)` → `POST /message/read` (blue ticks;
    `sender` required for group chats).
  - `SendChatPresence(ctx, chat, state)` → `POST /chat/presence` with the
    `PresenceComposing` / `PresenceRecording` / `PresencePaused` states.
- New `ErrNotModified` sentinel (returned by `GetAvatar` on a 304).

## [0.6.0] - 2026-07-05

### Added
- New media send methods for v0.11 gateway parity:
  - `SendAudio(ctx, msisdn, audio, isPTT, isViewOnce)`
  - `SendVideo(ctx, msisdn, video, caption, isGif, isViewOnce)`
  - `SendDocument(ctx, msisdn, document, fileName, caption)`
- `CheckContact(ctx, msisdn)` for `GET /contact/check`.
- New `ContactCheckResponse` type.

### Changed
- `MessageReactRequest` now supports optional `sender_msisdn`.
- `React` now accepts an optional variadic sender argument:
  `React(ctx, msisdn, messageID, emoji, senderMsisdn...)`.

## [0.5.0] - 2026-06-08

### Added
- Send methods now surface queued-mode results: `SendMessageResponse` gained
  `Status` and `JobID`. In queue mode the gateway returns `202` with a job ID
  (previously silently dropped); poll it with `GetJobStatus`. Direct mode still
  returns `MessageID`.
- Incoming webhook payloads now model sticker, location, and poll messages:
  new `IncomingMessageType` constants (`Sticker`, `Location`, `Poll`, plus
  `Contact`/`Unknown` reported by the polled inbox), top-level location fields
  (`Latitude`, `Longitude`, `Name`, `Address`) and poll fields (`Question`,
  `Options`, `SelectableCount`) on `IncomingWebhookPayload`.
- `IncomingMessageMediaInfo` gained `Sha256`, `StorageURL`, and `WhatsappURL`.
- Failed requests now carry the gateway's trace ID: `SDKError.TraceID` is
  populated from the `X-Trace-ID` response header and shown in `Error()`, so
  failures can be correlated with gateway logs.

### Changed
- **Breaking (minor, pre-1.0):** webhook `Timestamp` fields are now `int64`
  (were `int`) on `OutgoingWebhookPayload`, `IncomingWebhookPayload`, and
  `IncomingMessage`, matching the gateway's int64 Unix-second timestamps and
  avoiding overflow on 32-bit platforms.

## [0.4.0] - 2026-06-08

### Added
- `GetIncomingMessages(ctx, limit)` to fetch the most recent inbound messages
  buffered by the gateway, newest first. Paired with new types `IncomingMessage`
  and `IncomingMessagesResponse` mirroring the webhook payload vocabulary.
- `GetJobStatus(ctx, jobID)` to poll the status of asynchronously queued
  message jobs (gateway queue mode returns `202` with a `job_id`). Paired
  with the new `JobStatusResponse` type.
- `WithTraceID(ctx, id)` context helper: every request made with the
  returned context sends the ID in the `X-Trace-ID` header for end-to-end
  trace correlation with gateway logs. Closes the trace propagation
  limitation noted in 0.1.0.

### Fixed
- Data race on the client token: `SetToken`/`Register` could write the
  token while concurrent requests read it. Token access is now guarded by
  a `sync.RWMutex`, matching the documented "safe for concurrent use"
  guarantee.

## [0.1.0] - 2026-05-28

### Added
- `SendLocation` method for sending location messages with coordinates
- `SendPoll` method for sending poll messages with question and options
- `SendSticker` method for sending sticker messages

### Known limitations
- `IncomingMessage.Media.Url` is not populated by the `/message/incoming`
  endpoint; only metadata (type, mime type, size, filename, caption) is
  returned. Use webhooks for fetchable URLs.
- `doRequest` does not yet propagate an `X-Trace-ID` header; the gateway
  generates one on receipt. End-to-end trace correlation will land in a
  follow-up release.

## [0.0.1] - 2026-04-16

### Added
- Initial WhatsApp Gateway Go SDK implementation with waga package
- Support for text and image messaging
- Webhook registration and verification with HMAC-SHA256
- QR code and pair code login methods
- Message editing, deletion, and reaction support
- Session management (login status, logout, reconnect)
- Comprehensive unit testing with 93.3% coverage
- Automated CI/CD pipeline with release workflows
- Complete Go documentation for pkg.go.dev
