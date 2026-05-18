package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/vnp-community/vnp-memory/gateway/usecase"
)

// Auth returns HTTP middleware that authenticates requests via JWT or API Key.
// It extracts credentials from Authorization header (Bearer token) or X-API-Key header.
func Auth(authUC *usecase.AuthUseCase, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health/metrics endpoints
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for OPTIONS (CORS preflight)
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Try API Key first (X-API-Key header)
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				authCtx, err := authUC.AuthenticateAPIKey(ctx, apiKey)
				if err != nil {
					logger.Debug("api key auth failed", "error", err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":{"code":"UNAUTHENTICATED","message":"invalid API key"}}`))
					return
				}
				ctx = WithAuthContext(ctx, authCtx)
				// Set tenant header for downstream propagation
				w.Header().Set("X-Tenant-ID", authCtx.TenantID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try JWT (Authorization: Bearer <token>)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" || authUC.IsDevMode() {
				token := ""
				if authHeader != "" {
					token = strings.TrimPrefix(authHeader, "Bearer ")
				}
				authCtx, err := authUC.AuthenticateJWT(ctx, token)
				if err != nil {
					logger.Debug("jwt auth failed", "error", err)
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("WWW-Authenticate", "Bearer")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":{"code":"UNAUTHENTICATED","message":"invalid or missing JWT"}}`))
					return
				}
				ctx = WithAuthContext(ctx, authCtx)
				w.Header().Set("X-Tenant-ID", authCtx.TenantID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No credentials provided
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"code":"UNAUTHENTICATED","message":"missing authentication credentials"}}`))
		})
	}
}

// RateLimit returns HTTP middleware that enforces per-tenant rate limits.
func RateLimit(rateLimitUC *usecase.RateLimitUseCase, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health/metrics
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			auth, ok := AuthFromContext(r.Context())
			if !ok {
				// No auth context → skip rate limiting (will be caught by auth middleware)
				next.ServeHTTP(w, r)
				return
			}

			allowed, remaining, err := rateLimitUC.CheckWithTier(
				r.Context(),
				auth.TenantID,
				normalizePath(r.URL.Path),
				auth.RateTier,
			)
			if err != nil {
				// Fail-open
				logger.Error("rate limit check error, fail-open", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if !allowed {
				RateLimitRejected.WithLabelValues(auth.TenantID).Inc()
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"code":"RESOURCE_EXHAUSTED","message":"rate limit exceeded"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isPublicPath returns true for paths that skip auth/rate-limiting.
func isPublicPath(path string) bool {
	publicPaths := []string{"/healthz", "/readyz", "/metrics", "/healthz/deep"}
	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}
	return false
}
