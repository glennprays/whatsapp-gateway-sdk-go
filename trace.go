package waga

import "context"

type contextKey string

const traceIDContextKey contextKey = "waga-trace-id"

// TraceIDHeader is the HTTP header used to propagate a trace ID to the
// gateway. The gateway logs this ID with every operation, so requests can
// be correlated across your application and the gateway logs.
const TraceIDHeader = "X-Trace-ID"

// WithTraceID returns a context that carries a trace ID. Every SDK request
// made with the returned context sends the ID in the X-Trace-ID header:
//
//	ctx := waga.WithTraceID(context.Background(), "order-12345")
//	resp, err := client.SendText(ctx, msisdn, "hello")
//
// If no trace ID is set, the gateway generates one on receipt.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDContextKey, traceID)
}

// TraceIDFromContext returns the trace ID set by WithTraceID, or an empty
// string if none was set.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDContextKey).(string); ok {
		return v
	}
	return ""
}
