package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"

	kratosMiddleware "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	httpTransport "github.com/go-kratos/kratos/v2/transport/http"
)

func Metrics() kratosMiddleware.Middleware {
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			started := time.Now()
			method := operationName(ctx)
			reply, err := handler(ctx, req)
			observability.ObserveRequest(method, started, err)
			return reply, err
		}
	}
}

func operationName(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "unknown"
	}
	operation := strings.TrimSpace(tr.Operation())
	if operation != "" {
		return operation
	}
	if ht, ok := tr.(httpTransport.Transporter); ok {
		return ht.Request().Method + " " + ht.Request().URL.Path
	}
	return "unknown"
}
