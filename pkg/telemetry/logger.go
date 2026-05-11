package telemetry

import (
\t"context"
\t"log/slog"
\t"os"

\t"go.opentelemetry.io/otel/trace"
)

// InitLogger configures the global slog logger to use JSON format, suitable for production.
func InitLogger(level string) {
\tvar programLevel new(slog.LevelVar) // Info by default
\t
\tswitch level {
\tcase "debug":
\t\tprogramLevel.Set(slog.LevelDebug)
\tcase "warn":
\t\tprogramLevel.Set(slog.LevelWarn)
\tcase "error":
\t\tprogramLevel.Set(slog.LevelError)
\tdefault:
\t\tprogramLevel.Set(slog.LevelInfo)
\t}

\th := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
\t\tLevel: programLevel,
\t})
\tslog.SetDefault(slog.New(h))
}

// LogContext is a helper to inject OpenTelemetry trace and span IDs into structured logs.
// This allows seamless correlation between logs and distributed traces in tools like Grafana/Datadog.
func LogContext(ctx context.Context) []any {
\tspan := trace.SpanFromContext(ctx)
\tif !span.SpanContext().IsValid() {
\t\treturn []any{} // Return empty if no active span
\t}

\treturn []any{
\t\tslog.String("trace_id", span.SpanContext().TraceID().String()),
\t\tslog.String("span_id", span.SpanContext().SpanID().String()),
\t}
}
