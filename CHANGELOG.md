# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `GetIncomingMessages(ctx, limit)` to fetch the most recent inbound messages
  buffered by the gateway, newest first. Paired with new types `IncomingMessage`
  and `IncomingMessagesResponse` mirroring the webhook payload vocabulary.

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
