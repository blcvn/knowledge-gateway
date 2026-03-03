package middleware

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

type contextKey string

const AppContextKey contextKey = "kgs_app_context"

type AppContext struct {
	AppID  string
	Scopes string
}

// Auth API key middleware for Kratos
func Auth() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				// Get API Key from Authorization header or X-API-Key
				apiKey := tr.RequestHeader().Get("Authorization")
				if apiKey == "" {
					apiKey = tr.RequestHeader().Get("X-API-Key")
				}

				if apiKey == "" {
					return nil, errors.New("missing API key")
				}

				// TODO: validate API key hash with Redis/Postgres
				// For now, inject a mock App Context
				appCtx := AppContext{
					AppID:  "mock-app-id",
					Scopes: "read,write",
				}

				ctx = context.WithValue(ctx, AppContextKey, appCtx)
			}
			return handler(ctx, req)
		}
	}
}
