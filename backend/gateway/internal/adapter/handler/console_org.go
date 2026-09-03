// Package handler — Org Settings console handlers for VNP Memory.
// All routes require admin role and are scoped to the current tenant (X-Tenant-ID from JWT).
package handler

import (
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// OrgHandler handles /v1/console/org/* routes.
// All endpoints require admin role. Data is tenant-scoped via X-Tenant-ID header.
type OrgHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewOrgHandler creates a new OrgHandler.
func NewOrgHandler(registry port.ServiceRegistry, logger *slog.Logger) *OrgHandler {
	return &OrgHandler{registry: registry, logger: logger}
}

// GetSettings handles GET /v1/console/org/settings — org configuration.
func (h *OrgHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// UpdateSettings handles PUT /v1/console/org/settings — update org config.
func (h *OrgHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// ListMembers handles GET /v1/console/org/members — list org members.
func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// ListRoles handles GET /v1/console/org/roles — list org roles and permissions.
func (h *OrgHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}
