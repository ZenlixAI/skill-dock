// Package common provides the transport-level helpers shared across the catalog
// API modules: JSON encoding/decoding, query-parameter parsing, error
// translation, and request-scoped metadata propagation.
package common

import "context"

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"
	traceIDContextKey   contextKey = "trace_id"
)

// ContextWithRequestMetadata attaches the request and trace identifiers so that
// downstream handlers, audit records, and error responses can echo them back.
func ContextWithRequestMetadata(ctx context.Context, requestID string, traceID string) context.Context {
	ctx = context.WithValue(ctx, requestIDContextKey, requestID)
	return context.WithValue(ctx, traceIDContextKey, traceID)
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func TraceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDContextKey).(string)
	return traceID
}
