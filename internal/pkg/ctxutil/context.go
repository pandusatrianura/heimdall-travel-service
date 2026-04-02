package ctxutil

import "context"

type contextKey string

const RequestIDKey contextKey = "requestID"

// ContextWithRequestID returns a new context with the assigned request ID.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context. Returns "unknown" if not set.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}
