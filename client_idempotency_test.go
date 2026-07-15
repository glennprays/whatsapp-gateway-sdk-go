package waga

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSendImage_IdempotentBodyIsDeterministic proves a retried media send with the
// same Idempotency-Key produces a byte-identical raw request body (same multipart
// boundary), so the gateway's body-hash matches and it replays the original
// response instead of rejecting the retry as a body mismatch (422).
func TestSendImage_IdempotentBodyIsDeterministic(t *testing.T) {
	var bodies [][]byte
	var ctypes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, b)
		ctypes = append(ctypes, r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"message_id":"m1","chat":"628@s.whatsapp.net"}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithToken("tok"))
	img := []byte("PNGDATA-deterministic")

	for i := 0; i < 2; i++ {
		if _, err := c.SendImage(context.Background(), "628", bytes.NewReader(img), "cap", false, WithIdempotencyKey("key-1")); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	if len(bodies) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(bodies))
	}
	if !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatalf("idempotent media retry produced different bodies:\n%q\n%q", bodies[0], bodies[1])
	}
	if ctypes[0] != ctypes[1] {
		t.Fatalf("Content-Type boundary differs across retries: %q vs %q", ctypes[0], ctypes[1])
	}

	// A different key must yield a different boundary → different body.
	if _, err := c.SendImage(context.Background(), "628", bytes.NewReader(img), "cap", false, WithIdempotencyKey("key-2")); err != nil {
		t.Fatalf("send key-2: %v", err)
	}
	if bytes.Equal(bodies[2], bodies[0]) {
		t.Fatalf("different idempotency keys should produce different bodies")
	}
}

// TestMultipartBoundary_DeterministicAndValid unit-checks the boundary helper.
func TestMultipartBoundary_DeterministicAndValid(t *testing.T) {
	b1, ok1 := sendConfig{idempotencyKey: "k"}.multipartBoundary()
	b2, ok2 := sendConfig{idempotencyKey: "k"}.multipartBoundary()
	if !ok1 || !ok2 || b1 != b2 {
		t.Fatalf("boundary must be deterministic per key: %q %q", b1, b2)
	}
	if len(b1) == 0 || len(b1) > 70 {
		t.Fatalf("boundary length %d invalid (RFC 2046 max 70)", len(b1))
	}
	if b3, _ := (sendConfig{idempotencyKey: "other"}).multipartBoundary(); b3 == b1 {
		t.Fatal("distinct keys must yield distinct boundaries")
	}
	if _, ok := (sendConfig{}).multipartBoundary(); ok {
		t.Fatal("no key must yield no custom boundary")
	}
}
