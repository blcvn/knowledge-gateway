# TASK-004: Create `console_sdk.go` Handler

**Solution**: [SOL-002](../solutions/SOL-002-org-sdk-api.md)  
**CR**: CR-002  
**Priority**: 🟡 High  
**Estimate**: 1 hour  
**Status**: TODO

---

## Context

`gateway.go` line 60 calls `gwHandler.NewSDKHandler(registry, logger)` but `console_sdk.go` does not exist.

SDK endpoints handle API key and webhook management. The `POST /v1/console/sdk/keys` endpoint is security-critical: `raw_key` must be returned only once and must NOT be stored in plain text by `vnp-admin`.

---

## Exact Task

Create `gateway/internal/adapter/handler/console_sdk.go`:

```go
// Package handler — SDK Management console handlers for VNP Memory.
// Handles API key lifecycle (list/create/revoke) and webhook management.
// All routes require admin role and are tenant-scoped.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// SDKHandler handles /v1/console/sdk/* routes.
// All endpoints require admin role. Keys and webhooks are tenant-scoped.
type SDKHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewSDKHandler creates a new SDKHandler.
func NewSDKHandler(registry port.ServiceRegistry, logger *slog.Logger) *SDKHandler {
	return &SDKHandler{registry: registry, logger: logger}
}

// ListKeys handles GET /v1/console/sdk/keys — list API keys (raw_key NOT included).
func (h *SDKHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// CreateKey handles POST /v1/console/sdk/keys — create API key.
// IMPORTANT: raw_key is returned ONCE in the response. The UI must display it immediately.
// vnp-admin must store only SHA-256(raw_key) and never return it again.
func (h *SDKHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// DeleteKey handles DELETE /v1/console/sdk/keys/{id} — revoke API key permanently.
func (h *SDKHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// GetRateLimits handles GET /v1/console/sdk/rate-limits — rate limit configs per tier.
func (h *SDKHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// ListWebhooks handles GET /v1/console/sdk/webhooks — list configured webhooks.
func (h *SDKHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// CreateWebhook handles POST /v1/console/sdk/webhooks — create new webhook.
func (h *SDKHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// DeleteWebhook handles DELETE /v1/console/sdk/webhooks/{id} — delete webhook.
func (h *SDKHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}
```

---

## Acceptance Criteria

- [ ] File `gateway/internal/adapter/handler/console_sdk.go` exists and compiles
- [ ] `NewSDKHandler`, `ListKeys`, `CreateKey`, `DeleteKey`, `GetRateLimits`, `ListWebhooks`, `CreateWebhook`, `DeleteWebhook` are exported
- [ ] Each handler calls `requireAdmin` before forwarding
- [ ] All forward to `"vnp-admin"` service
- [ ] `go build ./gateway/...` passes
