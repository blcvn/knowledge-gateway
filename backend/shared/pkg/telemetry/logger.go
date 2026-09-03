package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// InitLogger configures the global slog logger to use JSON format, suitable for production.
func InitLogger(level string) {
	programLevel := new(slog.LevelVar) // Info by default

	switch level {
	case "debug":
		programLevel.Set(slog.LevelDebug)
	case "warn":
		programLevel.Set(slog.LevelWarn)
	case "error":
		programLevel.Set(slog.LevelError)
	default:
		programLevel.Set(slog.LevelInfo)
	}

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: programLevel,
	})
	slog.SetDefault(slog.New(h))
}

// LogContext is a helper to inject OpenTelemetry trace and span IDs into structured logs.
// This allows seamless correlation between logs and distributed traces in tools like Grafana/Datadog.
func LogContext(ctx context.Context) []any {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return []any{} // Return empty if no active span
	}

	return []any{
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	}
}
