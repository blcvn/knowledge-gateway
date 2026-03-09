package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	stdlog "log"
	"strings"
	"time"

	registryv1 "kgs-platform/api/registry/v1"
	"kgs-platform/internal/biz"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	httpTransport "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const AppContextKey contextKey = "kgs_app_context"

type AppContext struct {
	AppID    string
	Scopes   string
	TenantID string
	OrgID    string
}

// Auth API key middleware for Kratos
func Auth(registryUC RegistryValidator, redisCli *redis.Client) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			if shouldSkipAuth(tr.Operation()) || isObservabilityPath(tr) {
				return handler(ctx, req)
			}

			rawAPIKey := extractRawAPIKey(tr.RequestHeader().Get("Authorization"), tr.RequestHeader().Get("X-API-Key"))
			if rawAPIKey == "" {
				return nil, kerrors.Unauthorized("ERR_UNAUTHORIZED", "missing API key")
			}

			tenantID := resolveTenantID(tr.RequestHeader(), rawAPIKey)
			orgID := strings.TrimSpace(tr.RequestHeader().Get("X-Org-ID"))
			cacheHit := false
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
					OrgID:    orgID,
				}
			} else {
				cacheHit = true
				appCtx.TenantID = tenantID
				appCtx.OrgID = orgID
			}
			cacheAppContext(ctx, redisCli, rawAPIKey, appCtx)
			stdlog.Printf("[KGS][Auth] operation=%s app_id=%s tenant_id=%s org_id=%s cache_hit=%t",
				tr.Operation(), appCtx.AppID, appCtx.TenantID, appCtx.OrgID, cacheHit)

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
	op := strings.ToLower(strings.TrimSpace(operation))
	if strings.Contains(op, "healthz") || strings.Contains(op, "readyz") || strings.Contains(op, "metrics") {
		return true
	}
	return operation == registryv1.OperationRegistryCreateApp ||
		operation == registryv1.OperationRegistryListApps ||
		operation == registryv1.OperationRegistryIssueApiKey ||
		operation == registryv1.OperationRegistryGetApp ||
		strings.HasSuffix(operation, "/CreateApp") ||
		strings.HasSuffix(operation, "/ListApps") ||
		strings.HasSuffix(operation, "/IssueApiKey") ||
		strings.HasSuffix(operation, "/GetApp")
}

func isObservabilityPath(tr transport.Transporter) bool {
	ht, ok := tr.(httpTransport.Transporter)
	if !ok {
		return false
	}
	path := strings.TrimSpace(ht.Request().URL.Path)
	return path == "/healthz" || path == "/readyz" || path == "/metrics"
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

	var appCtx AppContext
	if err := json.Unmarshal([]byte(cached), &appCtx); err != nil {
		return AppContext{}, false
	}
	if strings.TrimSpace(appCtx.AppID) == "" {
		return AppContext{}, false
	}
	return appCtx, true
}

func cacheAppContext(ctx context.Context, redisCli *redis.Client, rawAPIKey string, appCtx AppContext) {
	if redisCli == nil {
		return
	}
	payload, err := json.Marshal(appCtx)
	if err != nil {
		return
	}
	cacheKey := "kgs:apikey:" + biz.HashAPIKey(rawAPIKey)
	_ = redisCli.Set(ctx, cacheKey, payload, 5*time.Minute).Err()
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

func resolveTenantID(header transport.Header, rawToken string) string {
	if header != nil {
		if tenantID := strings.TrimSpace(header.Get("X-Tenant-ID")); tenantID != "" {
			return tenantID
		}
		if tenantID := strings.TrimSpace(header.Get("x-tenant-id")); tenantID != "" {
			return tenantID
		}
	}
	return extractTenantID(rawToken)
}
