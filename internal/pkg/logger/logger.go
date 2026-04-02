package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/ctxutil"
)

// ContextHandler is a custom slog.Handler that extracts the requestID from the context.
type ContextHandler struct {
	slog.Handler
}

// Handle adds the requestID attribute to the record if it exists in the context.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if reqID := ctxutil.GetRequestID(ctx); reqID != "unknown" {
			r.AddAttrs(slog.String("request_id", reqID))
		}
	}
	return h.Handler.Handle(ctx, r)
}

// InitLogger initializes a global JSON logger with context support.
func InitLogger() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	parentHandler := slog.NewJSONHandler(os.Stdout, opts)
	handler := &ContextHandler{Handler: parentHandler}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
