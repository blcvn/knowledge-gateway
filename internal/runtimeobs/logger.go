package runtimeobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"

	"kg-service/internal/config"
)

type Logger struct {
	base      *slog.Logger
	component string
}

type contextKey string

const (
	requestMetaKey contextKey = "request-meta"
)

type RequestMeta struct {
	RequestID string
	TraceID   string
	SpanID    string
}

func NewLogger(cfg config.Config, component string) *Logger {
	return newLogger(cfg, component, os.Stdout)
}

func NewLoggerWithWriter(cfg config.Config, component string, w io.Writer) *Logger {
	return newLogger(cfg, component, w)
}

func newLogger(cfg config.Config, component string, w io.Writer) *Logger {
	var handler slog.Handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: parseLevel(cfg.Observability.LogLevel),
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			switch attr.Key {
			case slog.TimeKey:
				attr.Key = "ts"
				if attr.Value.Kind() == slog.KindTime {
					attr.Value = slog.StringValue(attr.Value.Time().UTC().Format(time.RFC3339Nano))
				}
			case slog.MessageKey:
				attr.Key = "msg"
			}
			return attr
		},
	})
	if strings.EqualFold(cfg.Observability.LogFormat, "text") {
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{Level: parseLevel(cfg.Observability.LogLevel)})
	}

	base := slog.New(handler).With(
		slog.String("service", fallback(cfg.Observability.ServiceName, "kg-service")),
		slog.String("version", fallback(cfg.Observability.ServiceVersion, "dev")),
		slog.String("component", fallback(component, "runtime")),
	)
	return &Logger{base: base, component: component}
}

func (l *Logger) Log(level kratoslog.Level, keyvals ...any) error {
	if l == nil || l.base == nil {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(keyvals)/2+4)
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		if i+1 >= len(keyvals) {
			attrs = append(attrs, slog.String(key, ""))
			break
		}
		value := kratoslog.Value(context.Background(), keyvals[i+1])
		attrs = append(attrs, slog.Any(key, value))
	}
	l.base.LogAttrs(context.Background(), toSlogLevel(level), "", attrs...)
	return nil
}

func parseLevel(raw string) slog.Leveler {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func fallback(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func (l *Logger) With(args ...any) *Logger {
	if l == nil || l.base == nil {
		return l
	}
	return &Logger{base: l.base.With(args...), component: l.component}
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logAttrs(ctx, slog.LevelInfo, msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logAttrs(ctx, slog.LevelWarn, msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logAttrs(ctx, slog.LevelError, msg, args...)
}

func (l *Logger) Printf(format string, args ...any) {
	l.logAttrs(context.Background(), slog.LevelInfo, fmt.Sprintf(format, args...))
}

func toSlogLevel(level kratoslog.Level) slog.Level {
	switch level {
	case kratoslog.LevelDebug:
		return slog.LevelDebug
	case kratoslog.LevelWarn:
		return slog.LevelWarn
	case kratoslog.LevelError:
		return slog.LevelError
	case kratoslog.LevelFatal:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *Logger) logAttrs(ctx context.Context, level slog.Level, msg string, args ...any) {
	if l == nil || l.base == nil {
		return
	}
	attrs := []slog.Attr{}
	if meta := RequestMetaFromContext(ctx); meta != (RequestMeta{}) {
		if meta.RequestID != "" {
			attrs = append(attrs, slog.String("request_id", meta.RequestID))
		}
		if meta.TraceID != "" {
			attrs = append(attrs, slog.String("trace_id", meta.TraceID))
		}
		if meta.SpanID != "" {
			attrs = append(attrs, slog.String("span_id", meta.SpanID))
		}
	}
	attrs = append(attrs, attrsFromAny(args)...)
	l.base.LogAttrs(ctx, level, msg, attrs...)
}

func attrsFromAny(args []any) []slog.Attr {
	if len(args) == 0 {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(args)/2+1)
	for i := 0; i < len(args); i += 2 {
		key := fmt.Sprint(args[i])
		if i+1 >= len(args) {
			attrs = append(attrs, slog.String(key, ""))
			break
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}
	return attrs
}

func WithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaKey, meta)
}

func RequestMetaFromContext(ctx context.Context) RequestMeta {
	if ctx == nil {
		return RequestMeta{}
	}
	meta, _ := ctx.Value(requestMetaKey).(RequestMeta)
	return meta
}

func NewRequestMeta(requestID, traceID, spanID string) RequestMeta {
	return RequestMeta{
		RequestID: requestID,
		TraceID:   traceID,
		SpanID:    spanID,
	}
}

func GenerateRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}
