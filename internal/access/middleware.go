package access

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"kg-service/internal/httpapi/respond"
)

type Middleware struct {
	identityResolver *IdentityResolver
	limiter          Limiter
}

func NewMiddleware(identityResolver *IdentityResolver, limiter ...Limiter) Middleware {
	var rateLimiter Limiter
	if len(limiter) > 0 {
		rateLimiter = limiter[0]
	}
	return Middleware{identityResolver: identityResolver, limiter: rateLimiter}
}

func (m Middleware) RequireIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, err := m.identityResolver.Resolve(r.Header.Get("Authorization"))
		if err != nil {
			respond.Error(w, respond.StatusFor(respond.CodeUnauthorized), respond.CodeUnauthorized, "Authentication failed", nil)
			return
		}
		if m.limiter != nil && !m.limiter.Allow(identity) {
			respond.Error(w, respond.StatusFor(respond.CodeTooManyRequests), respond.CodeTooManyRequests, "Rate limit exceeded", map[string]any{"tenant_id": identity.TenantID})
			return
		}

		sanitizedBody, err := sanitizeIdentityFields(r.Body)
		if err != nil {
			respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
			return
		}
		r.Body = sanitizedBody

		next.ServeHTTP(w, r.WithContext(ContextWithIdentity(r.Context(), identity)))
	})
}

func sanitizeIdentityFields(body io.ReadCloser) (io.ReadCloser, error) {
	if body == nil {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	delete(data, "tenant_id")
	delete(data, "app_id")

	sanitized, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(sanitized)), nil
}
