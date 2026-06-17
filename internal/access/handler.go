package access

import (
	"encoding/json"
	"errors"
	"net/http"

	"kg-service/internal/httpapi/respond"
)

type Handler struct {
	accessResolver *AccessResolver
	service        *Service
}

func NewHandler(accessResolver *AccessResolver, service *Service) Handler {
	return Handler{accessResolver: accessResolver, service: service}
}

func (h Handler) GetResolve(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityFromContext(r.Context())
	if !ok {
		respond.Error(w, respond.StatusFor(respond.CodeUnauthorized), respond.CodeUnauthorized, "Authentication failed", nil)
		return
	}

	visibleOwners, err := h.accessResolver.ResolveVisibleOwners(identity)
	if err != nil {
		respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Failed to resolve visibility", nil)
		return
	}

	respond.OK(w, ResolveResponse{
		TenantID:      identity.TenantID,
		AppID:         identity.AppID,
		VisibleOwners: visibleOwners,
	})
}

func (h Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req TenantCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := IdentityFromContext(r.Context())
	tenant, err := h.service.CreateTenant(identity, req)
	if err != nil {
		writeError(w, err)
		return
	}

	respond.Created(w, tenant)
}

func (h Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	tenant, err := h.service.GetTenant(identity, r.PathValue("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, tenant)
}

func (h Handler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	var req TenantUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := IdentityFromContext(r.Context())
	tenant, err := h.service.UpdateTenant(identity, r.PathValue("tenant_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, tenant)
}

func (h Handler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	tenant, err := h.service.SuspendTenant(identity, r.PathValue("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}

	respond.OK(w, map[string]any{
		"id":     tenant.ID,
		"status": tenant.Status,
	})
}

func (h Handler) CreateApp(w http.ResponseWriter, r *http.Request) {
	var req AppCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := IdentityFromContext(r.Context())
	app, err := h.service.CreateApp(identity, r.PathValue("tenant_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, app)
}

func (h Handler) RotateAppKey(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	rotated, err := h.service.RotateAppKey(identity, r.PathValue("tenant_id"), r.PathValue("app_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, rotated)
}

func (h Handler) ListApps(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	apps, err := h.service.ListApps(identity, r.PathValue("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}

	respond.OK(w, respond.ListEnvelope[AppResponse]{
		Data:    apps,
		HasMore: false,
	})
}

func (h Handler) DeleteApp(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	app, err := h.service.RevokeApp(identity, r.PathValue("tenant_id"), r.PathValue("app_id"))
	if err != nil {
		writeError(w, err)
		return
	}

	respond.OK(w, map[string]any{
		"id":         app.ID,
		"status":     app.Status,
		"revoked_at": app.RevokedAt,
	})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return ErrRequestMalformed
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
	case errors.Is(err, ErrNotFound):
		respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
	case errors.Is(err, ErrValidation):
		respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
	case errors.Is(err, ErrRequestMalformed):
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
	default:
		respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
	}
}
