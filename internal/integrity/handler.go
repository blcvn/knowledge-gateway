package integrity

import (
	"errors"
	"net/http"

	"kg-service/internal/access"
	"kg-service/internal/httpapi/respond"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) Handler {
	return Handler{service: service}
}

func (h Handler) TenantIntegrity(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	resp, err := h.service.TenantIntegrity(identity, r.PathValue("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, resp)
}

func (h Handler) MissingBridges(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	items, err := h.service.MissingBridges(identity, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, respond.ListEnvelope[MissingBridgeItem]{Data: items})
}

func (h Handler) OrphanScan(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	// Query parameter, same as the repair endpoints below and as openapi.yaml declares: the route
	// is /v1/kg/integrity/orphans, with no {tenant_id} segment for a path value to come from.
	resp, err := h.service.OrphanScan(identity, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, resp)
}

func (h Handler) RebuildProjection(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	// The tenant arrives as a query parameter, which is what the route declares and what
	// openapi.yaml documents (TenantIDQueryRequired). Reading it as a path value — as this did —
	// always yielded the empty string, because the route has no {tenant_id} segment, so the
	// endpoint answered "tenant_id is required" to every caller and could never run. The sibling
	// MissingBridges handler above reads the query string; these two had drifted from it.
	resp, err := h.service.RebuildProjection(identity, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, resp)
}

func (h Handler) PurgeOrphans(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	resp, err := h.service.PurgeOrphans(identity, r.URL.Query().Get("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, resp)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, "tenant_id is required", nil)
	case errors.Is(err, ErrForbidden):
		respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
	case errors.Is(err, ErrNotFound):
		respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
	default:
		respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
	}
}
