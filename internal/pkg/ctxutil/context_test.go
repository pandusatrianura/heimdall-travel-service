package ctxutil

import (
	"context"
	"testing"
)

func TestContextWithRequestIDAndGetRequestID(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "req-123")

	got := GetRequestID(ctx)
	if got != "req-123" {
		t.Fatalf("GetRequestID() = %q, want %q", got, "req-123")
	}
}

func TestGetRequestIDReturnsUnknownWhenMissing(t *testing.T) {
	got := GetRequestID(context.Background())
	if got != "unknown" {
		t.Fatalf("GetRequestID() = %q, want %q", got, "unknown")
	}
}

func TestGetRequestIDReturnsUnknownForNonStringValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, 123)

	got := GetRequestID(ctx)
	if got != "unknown" {
		t.Fatalf("GetRequestID() = %q, want %q", got, "unknown")
	}
}
