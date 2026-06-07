package waga

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithTraceID_PropagatedAsHeader(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(TraceIDHeader)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"authenticated":true}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("tok"))

	ctx := WithTraceID(context.Background(), "trace-abc-123")
	if _, err := client.GetLoginStatus(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHeader != "trace-abc-123" {
		t.Errorf("expected X-Trace-ID header trace-abc-123, got %q", gotHeader)
	}
}

func TestWithTraceID_AbsentWhenUnset(t *testing.T) {
	headerPresent := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, headerPresent = r.Header[TraceIDHeader]
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"authenticated":true}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("tok"))
	if _, err := client.GetLoginStatus(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headerPresent {
		t.Error("X-Trace-ID header must not be sent when no trace ID is set")
	}
}

func TestTraceIDFromContext_Empty(t *testing.T) {
	if got := TraceIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty trace ID, got %q", got)
	}
}
