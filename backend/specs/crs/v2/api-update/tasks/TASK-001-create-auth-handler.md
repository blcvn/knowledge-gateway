# TASK-001: Create `auth.go` HTTP Handler

**Solution**: [SOL-001](../solutions/SOL-001-auth-api.md)  
**CR**: CR-001  
**Priority**: 🔴 Critical  
**Estimate**: 2 hours  
**Status**: TODO

---

## Context

`gateway.go` (line 40) already calls `gwHandler.NewAuthHandler(registry, logger)` and passes `authH` to `Router()` (line 63), but the file `gateway/internal/adapter/handler/auth.go` does not yet exist, causing a **compile error**.

The `sm-auth` gRPC service is available in the registry as `"sm-auth"`. Its `AuthResponse` proto contains:
- `token string` — the JWT access token
- `user UserProfile` — `{ id, name, email, role }`

The proto does NOT have `refresh_token` or `expires_in`. The gateway handler must mint sensible defaults and construct the full `LoginResponse` shape expected by the frontend.

---

## Exact Task

Create the file `gateway/internal/adapter/handler/auth.go` with the following implementation:

```go
// Package handler — Auth API handlers for VNP Memory Console.
// Routes /v1/auth/* — login, logout, me, refresh.
// Login and refresh are public (no JWT required).
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// AuthHandler handles /v1/auth/* routes.
// Forwards to sm-auth service with response transformation to match frontend LoginResponse shape.
type AuthHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(registry port.ServiceRegistry, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{registry: registry, logger: logger}
}

// Login handles POST /v1/auth/login — public, no JWT required.
// Transforms sm-auth AuthResponse → LoginResponse for frontend.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	resp, err := forwardAndCapture(h.registry, "sm-auth", h.logger, r)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Transform sm-auth AuthResponse → frontend LoginResponse
	var smResp struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
	}
	if jsonErr := json.Unmarshal(resp, &smResp); jsonErr != nil {
		// Pass through raw response if unmarshal fails
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
		return
	}

	loginResp := map[string]any{
		"access_token":  smResp.Token,
		"refresh_token": smResp.Token, // sm-auth does not yet issue separate refresh tokens; use same token
		"expires_in":    3600,
		"token_type":    "Bearer",
		"user": map[string]any{
			"id":        smResp.User.ID,
			"name":      smResp.User.Name,
			"email":     smResp.User.Email,
			"role":      smResp.User.Role,
			"tenant_id": "", // populated from JWT claims by client
		},
	}
	WriteJSON(w, http.StatusOK, loginResp)
}

// Logout handles POST /v1/auth/logout — JWT required.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// Me handles GET /v1/auth/me — JWT required, returns AuthUser from token claims.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// Refresh handles POST /v1/auth/refresh — public, no JWT required.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// forwardAndCapture forwards the request to the service and returns the raw response body.
func forwardAndCapture(registry port.ServiceRegistry, serviceName string, logger *slog.Logger, r *http.Request) ([]byte, error) {
	body, err := ReadBody(r)
	if err != nil {
		return nil, err
	}
	target, err := registry.Resolve(serviceName)
	if err != nil {
		logger.Error("resolve service failed", "service", serviceName, "error", err)
		return nil, err
	}
	fwdReq := buildForwardRequest(r, body)
	return registry.ForwardWithContext(r.Context(), target, fwdReq)
}
```

> **Note**: `buildForwardRequest` wraps the `domain.ForwardRequest` construction. If this helper does not exist in `handler.go`, inline it using the same pattern as `ForwardToService`.

---

## Acceptance Criteria

- [ ] File `gateway/internal/adapter/handler/auth.go` exists and compiles
- [ ] `NewAuthHandler`, `Login`, `Logout`, `Me`, `Refresh` are exported
- [ ] `Login` transforms the sm-auth `{ token, user }` response into `{ access_token, refresh_token, expires_in, token_type, user }` shape
- [ ] No import errors — uses same packages as `services.go` and `console.go`
- [ ] `go build ./gateway/...` passes
