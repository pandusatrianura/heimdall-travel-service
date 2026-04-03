package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/ctxutil"
)

type captureHandler struct {
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(_ string) slog.Handler {
	return h
}

func TestContextHandlerHandleAddsRequestID(t *testing.T) {
	capture := &captureHandler{}
	handler := &ContextHandler{Handler: capture}

	ctx := ctxutil.ContextWithRequestID(context.Background(), "req-123")
	record := slog.NewRecord(testTime, slog.LevelInfo, "hello", 0)

	if err := handler.Handle(ctx, record); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if len(capture.records) != 1 {
		t.Fatalf("expected 1 captured record, got %d", len(capture.records))
	}

	gotAttrs := attrsToMap(capture.records[0])
	if gotAttrs["request_id"] != "req-123" {
		t.Fatalf("expected request_id attr to be added, got %#v", gotAttrs)
	}
}

func TestContextHandlerHandleSkipsUnknownRequestID(t *testing.T) {
	capture := &captureHandler{}
	handler := &ContextHandler{Handler: capture}

	record := slog.NewRecord(testTime, slog.LevelInfo, "hello", 0)

	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if len(capture.records) != 1 {
		t.Fatalf("expected 1 captured record, got %d", len(capture.records))
	}

	gotAttrs := attrsToMap(capture.records[0])
	if _, exists := gotAttrs["request_id"]; exists {
		t.Fatalf("did not expect request_id attr when context has no request id, got %#v", gotAttrs)
	}
}

func TestInitLoggerSetsDefaultLogger(t *testing.T) {
	originalDefault := slog.Default()
	originalStdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer r.Close()

	os.Stdout = w
	defer func() {
		os.Stdout = originalStdout
		slog.SetDefault(originalDefault)
	}()

	InitLogger()

	defaultLogger := slog.Default()
	if defaultLogger == nil {
		t.Fatal("expected default logger to be initialized")
	}

	handlerType := reflect.TypeOf(defaultLogger.Handler())
	if handlerType != reflect.TypeOf(&ContextHandler{}) {
		t.Fatalf("expected default handler type %v, got %v", reflect.TypeOf(&ContextHandler{}), handlerType)
	}

	ctx := ctxutil.ContextWithRequestID(context.Background(), "req-456")
	defaultLogger.InfoContext(ctx, "logger initialized")

	if err := w.Close(); err != nil {
		t.Fatalf("closing write pipe failed: %v", err)
	}

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading logger output failed: %v", err)
	}

	if len(output) == 0 {
		t.Fatal("expected JSON log output to be written to stdout")
	}
}

var testTime = time.Date(2026, time.April, 3, 10, 0, 0, 0, time.UTC)

func attrsToMap(record slog.Record) map[string]any {
	attrs := make(map[string]any)
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})
	return attrs
}
