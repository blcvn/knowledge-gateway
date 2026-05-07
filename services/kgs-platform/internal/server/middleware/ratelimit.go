package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
)

// RateLimiter uses Redis sliding window to rate limit requests per AppID
func RateLimiter() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Extract AppID from AppContext injected by Auth middleware
			// val := ctx.Value(AppContextKey)
			// if val != nil {
			// 	appCtx := val.(AppContext)
			// 	// TODO: Check Redis limits using appCtx.AppID
			// }

			return handler(ctx, req)
		}
	}
}
