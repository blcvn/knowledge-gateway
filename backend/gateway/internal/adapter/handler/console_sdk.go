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

// DeleteKey handles DELETE /v1/console/sdk/keys/{id} — revoke an API key.
func (h *SDKHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// GetRateLimits handles GET /v1/console/sdk/rate-limits — rate limit configuration.
func (h *SDKHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// ListWebhooks handles GET /v1/console/sdk/webhooks — list registered webhooks.
func (h *SDKHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// CreateWebhook handles POST /v1/console/sdk/webhooks — register a new webhook.
func (h *SDKHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// DeleteWebhook handles DELETE /v1/console/sdk/webhooks/{id} — remove a webhook.
func (h *SDKHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}
