package middleware

import (
	"context"
	"errors"
	stdlog "log"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	kratosMiddleware "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/trace"
)

func AccessLog() kratosMiddleware.Middleware {
	return func(handler kratosMiddleware.Handler) kratosMiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			started := time.Now()
			reply, err := handler(ctx, req)

			tr, _ := transport.FromServerContext(ctx)
			operation := operationName(ctx)
			system := "unknown"
			if tr != nil {
				system = strings.TrimSpace(tr.Kind().String())
			}
			appCtx, _ := AppContextFromContext(ctx)
			traceID := traceIDFromContext(ctx)
			status := statusCodeFromErr(err)

			if err != nil {
				stdlog.Printf(
					"[KGS][Access] system=%s operation=%s status=%d app_id=%s tenant_id=%s org_id=%s trace_id=%s duration=%s err=%v",
					system,
					operation,
					status,
					appCtx.AppID,
					appCtx.TenantID,
					appCtx.OrgID,
					traceID,
					time.Since(started),
					err,
				)
				return reply, err
			}

			stdlog.Printf(
				"[KGS][Access] system=%s operation=%s status=%d app_id=%s tenant_id=%s org_id=%s trace_id=%s duration=%s",
				system,
				operation,
				status,
				appCtx.AppID,
				appCtx.TenantID,
				appCtx.OrgID,
				traceID,
				time.Since(started),
			)
			return reply, nil
		}
	}
}

func statusCodeFromErr(err error) int {
	if err == nil {
		return 200
	}
	var kerr *kerrors.Error
	if errors.As(err, &kerr) && kerr != nil && kerr.Code > 0 {
		return int(kerr.Code)
	}
	return 500
}

func traceIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.TraceID().String()
}
