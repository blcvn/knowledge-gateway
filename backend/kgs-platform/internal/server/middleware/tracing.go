package middleware

import (
	"context"
	"strings"

	kratosMiddleware "github.com/go-kratos/kratos/v2/middleware"
	kratosTracing "github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func Tracing() kratosMiddleware.Middleware {
	base := kratosTracing.Server(kratosTracing.WithTracerName("kgs-platform"))
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return base(func(ctx context.Context, req any) (any, error) {
			if span := trace.SpanFromContext(ctx); span != nil {
				if tr, ok := transport.FromServerContext(ctx); ok {
					span.SetAttributes(
						attribute.String("rpc.system", tr.Kind().String()),
						attribute.String("rpc.operation", strings.TrimSpace(tr.Operation())),
					)
				}
			}
			return handler(ctx, req)
		})
	}
}
