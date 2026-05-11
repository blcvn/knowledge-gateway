package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// Setup creates a new slog JSON handler and sets it as the default logger.
// It includes logic to extract trace_id from the context.
func Setup(level string) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: l,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

// WithContext returns a logger enriched with OTEL trace context.
func WithContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		logger = logger.With(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	}

	// Example: Extract tenant_id from context
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		logger = logger.With(slog.Any("tenant_id", tenantID))
	}

	return logger
}
