package waga

// SendOption configures a single send call — canonical chat addressing, a
// reply-to quote, @-mentions, and (see WithIdempotencyKey) send idempotency.
// Options are applied left-to-right and are accepted by every SendXxx method as
// a trailing variadic, so existing call sites keep compiling unchanged.
type SendOption func(*sendConfig)

// sendConfig accumulates the optional, per-send fields set via SendOption.
type sendConfig struct {
	chat           string
	replyToID      string
	replyToSender  string
	replyToText    string
	mentions       []string
	idempotencyKey string
}

// WithChat sets the canonical recipient (a bare number, a user JID, a group JID
// "@g.us", or a "@lid"). It takes precedence over the positional recipient
// passed to the send method (which the gateway treats as the deprecated msisdn
// alias).
func WithChat(chat string) SendOption {
	return func(c *sendConfig) { c.chat = chat }
}

// WithReply quotes an existing message. messageID and sender identify the quoted
// message and its author; quotedText is an optional caller-supplied preview of
// the quoted content (the gateway is storeless and does not look it up).
func WithReply(messageID, sender, quotedText string) SendOption {
	return func(c *sendConfig) {
		c.replyToID = messageID
		c.replyToSender = sender
		c.replyToText = quotedText
	}
}

// WithMentions @-tags the given numbers/JIDs in the outgoing message.
func WithMentions(mentions ...string) SendOption {
	return func(c *sendConfig) { c.mentions = mentions }
}

// WithIdempotencyKey attaches an Idempotency-Key header to the send. Reusing the
// same key replays the gateway's original response (200 with Idempotent-Replay:
// true); an in-flight duplicate returns 409 (ErrConflict) and the same key with
// a different request body returns 422.
func WithIdempotencyKey(key string) SendOption {
	return func(c *sendConfig) { c.idempotencyKey = key }
}

// reqHeader is a single extra HTTP request header.
type reqHeader struct{ key, value string }

// headers returns the extra request headers implied by the send config. Only the
// Idempotency-Key header is emitted, and only when a key was provided.
func (c sendConfig) headers() []reqHeader {
	if c.idempotencyKey == "" {
		return nil
	}
	return []reqHeader{{"Idempotency-Key", c.idempotencyKey}}
}

// newSendConfig folds the given options into a sendConfig.
func newSendConfig(opts []SendOption) sendConfig {
	var c sendConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&c)
		}
	}
	return c
}
