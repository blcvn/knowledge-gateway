package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	registryv1 "kgs-platform/api/registry/v1"
	"kgs-platform/internal/biz"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const AppContextKey contextKey = "kgs_app_context"

type AppContext struct {
	AppID    string
	Scopes   string
	TenantID string
}

// Auth API key middleware for Kratos
func Auth(registryUC RegistryValidator, redisCli *redis.Client) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			if shouldSkipAuth(tr.Operation()) {
				return handler(ctx, req)
			}

			rawAPIKey := extractRawAPIKey(tr.RequestHeader().Get("Authorization"), tr.RequestHeader().Get("X-API-Key"))
			if rawAPIKey == "" {
				return nil, kerrors.Unauthorized("ERR_UNAUTHORIZED", "missing API key")
			}

			tenantID := extractTenantID(rawAPIKey)
			appCtx, ok := readCachedAppContext(ctx, redisCli, rawAPIKey)
			if !ok {
				apiKey, err := registryUC.ValidateAPIKey(ctx, rawAPIKey)
				if err != nil {
					return nil, kerrors.Unauthorized("ERR_UNAUTHORIZED", "invalid API key")
				}
				appCtx = AppContext{
					AppID:    apiKey.AppID,
					Scopes:   apiKey.Scopes,
					TenantID: tenantID,
				}
				cacheAppContext(ctx, redisCli, rawAPIKey, appCtx)
			} else {
				appCtx.TenantID = tenantID
			}

			ctx = context.WithValue(ctx, AppContextKey, appCtx)
			return handler(ctx, req)
		}
	}
}

type RegistryValidator interface {
	ValidateAPIKey(ctx context.Context, rawAPIKey string) (*biz.APIKey, error)
}

func AppContextFromContext(ctx context.Context) (AppContext, bool) {
	val := ctx.Value(AppContextKey)
	if val == nil {
		return AppContext{}, false
	}
	appCtx, ok := val.(AppContext)
	return appCtx, ok
}

func shouldSkipAuth(operation string) bool {
	return operation == registryv1.OperationRegistryCreateApp ||
		operation == registryv1.OperationRegistryListApps ||
		operation == registryv1.OperationRegistryIssueApiKey ||
		operation == registryv1.OperationRegistryGetApp ||
		strings.HasSuffix(operation, "/CreateApp") ||
		strings.HasSuffix(operation, "/ListApps") ||
		strings.HasSuffix(operation, "/IssueApiKey") ||
		strings.HasSuffix(operation, "/GetApp")
}

func extractRawAPIKey(authHeader, xAPIKey string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader != "" {
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			return strings.TrimSpace(authHeader[7:])
		}
		return authHeader
	}
	return strings.TrimSpace(xAPIKey)
}

func readCachedAppContext(ctx context.Context, redisCli *redis.Client, rawAPIKey string) (AppContext, bool) {
	if redisCli == nil {
		return AppContext{}, false
	}
	cacheKey := "kgs:apikey:" + biz.HashAPIKey(rawAPIKey)
	cached, err := redisCli.Get(ctx, cacheKey).Result()
	if err != nil || cached == "" {
		return AppContext{}, false
	}

	parts := strings.SplitN(cached, "|", 2)
	if len(parts) != 2 || parts[0] == "" {
		return AppContext{}, false
	}
	return AppContext{
		AppID:  parts[0],
		Scopes: parts[1],
	}, true
}

func cacheAppContext(ctx context.Context, redisCli *redis.Client, rawAPIKey string, appCtx AppContext) {
	if redisCli == nil {
		return
	}
	cacheKey := "kgs:apikey:" + biz.HashAPIKey(rawAPIKey)
	_ = redisCli.Set(ctx, cacheKey, appCtx.AppID+"|"+appCtx.Scopes, 5*time.Minute).Err()
}

func extractTenantID(rawToken string) string {
	if parts := strings.Split(rawToken, "."); len(parts) == 3 {
		claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err == nil {
			var claims map[string]any
			if json.NewDecoder(bytes.NewReader(claimBytes)).Decode(&claims) == nil {
				if tenantID, ok := claims["tenant_id"].(string); ok && tenantID != "" {
					return tenantID
				}
			}
		}
	}
	return "default"
}
