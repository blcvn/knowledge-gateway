package middleware

import (
	"context"
	"strings"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

func Namespace() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			appCtx, ok := AppContextFromContext(ctx)
			if !ok || appCtx.AppID == "" {
				return handler(ctx, req)
			}

			namespaceHeader := strings.TrimSpace(tr.RequestHeader().Get("X-KG-Namespace"))
			if namespaceHeader == "" {
				return handler(ctx, req)
			}

			expected := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID, appCtx.OrgID)
			if namespaceHeader != expected {
				return nil, kerrors.Forbidden("ERR_FORBIDDEN", "namespace does not match application context")
			}

			return handler(ctx, req)
		}
	}
}
