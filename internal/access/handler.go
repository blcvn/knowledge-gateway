package access

import (
	"encoding/json"
	"errors"
	"net/http"
	"github.com/gorilla/mux"
	"strconv"
	"strings"

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
	tenant, err := h.service.GetTenant(identity, mux.Vars(r)["tenant_id"])
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
	tenant, err := h.service.UpdateTenant(identity, mux.Vars(r)["tenant_id"], req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, tenant)
}

func (h Handler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	tenant, err := h.service.SuspendTenant(identity, mux.Vars(r)["tenant_id"])
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
	app, err := h.service.CreateApp(identity, mux.Vars(r)["tenant_id"], req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, app)
}

func (h Handler) RotateAppKey(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	rotated, err := h.service.RotateAppKey(identity, mux.Vars(r)["tenant_id"], mux.Vars(r)["app_id"])
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, rotated)
}

func (h Handler) ListApps(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	apps, err := h.service.ListApps(identity, mux.Vars(r)["tenant_id"])
	if err != nil {
		writeError(w, err)
		return
	}
	page := parsePage(r)
	apps, nextCursor, hasMore := paginateApps(apps, page)

	respond.OK(w, respond.ListEnvelope[AppResponse]{
		Data:       apps,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

func (h Handler) DeleteApp(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	app, err := h.service.RevokeApp(identity, mux.Vars(r)["tenant_id"], mux.Vars(r)["app_id"])
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

func (h Handler) CreateGrant(w http.ResponseWriter, r *http.Request) {
	var req GrantCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := IdentityFromContext(r.Context())
	grant, err := h.service.CreateGrant(identity, req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, grant)
}

func (h Handler) ListGrants(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	grantorTenantID, err := parseOptionalTenantQuery(r.URL.Query().Get("grantor_tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	granteeTenantID, err := parseOptionalTenantQuery(r.URL.Query().Get("grantee_tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	grants, err := h.service.ListGrants(identity, GrantListFilter{
		GrantorTenantID: grantorTenantID,
		GranteeTenantID: granteeTenantID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	page := parsePage(r)
	grants, nextCursor, hasMore := paginateGrants(grants, page)
	respond.OK(w, respond.ListEnvelope[GrantResponse]{Data: grants, NextCursor: nextCursor, HasMore: hasMore})
}

func (h Handler) DeleteGrant(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	grant, err := h.service.RevokeGrant(identity, mux.Vars(r)["id"])
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, map[string]any{
		"id":         grant.ID,
		"status":     grant.Status,
		"revoked_at": grant.RevokedAt,
	})
}

func (h Handler) ListAudit(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFromContext(r.Context())
	resourceOwnerTenantID, err := parseOptionalTenantQuery(r.URL.Query().Get("resource_owner_tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	entries, err := h.service.ListAuditLogs(identity, AuditListFilter{
		ResourceOwnerTenantID: resourceOwnerTenantID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	page := parsePage(r)
	entries, nextCursor, hasMore := paginateAudits(entries, page)
	respond.OK(w, respond.ListEnvelope[AuditLogEntry]{Data: entries, NextCursor: nextCursor, HasMore: hasMore})
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
	case errors.Is(err, ErrBadRequest):
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, err.Error(), nil)
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

func parsePage(r *http.Request) respond.Page {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return respond.NormalizePage(limit, r.URL.Query().Get("cursor"), 20, 100)
}

func parseOptionalTenantQuery(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !looksLikeUUID(value) {
		return "", ErrValidation
	}
	return value, nil
}

func looksLikeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			switch {
			case r >= '0' && r <= '9':
			case r >= 'a' && r <= 'f':
			case r >= 'A' && r <= 'F':
			default:
				return false
			}
		}
	}
	return true
}

func paginateApps(items []AppResponse, page respond.Page) ([]AppResponse, string, bool) {
	return paginateBy(items, page, func(item AppResponse) string { return item.ID })
}

func paginateGrants(items []GrantResponse, page respond.Page) ([]GrantResponse, string, bool) {
	return paginateBy(items, page, func(item GrantResponse) string { return item.ID })
}

func paginateAudits(items []AuditLogEntry, page respond.Page) ([]AuditLogEntry, string, bool) {
	return paginateBy(items, page, func(item AuditLogEntry) string { return item.ID })
}

func paginateBy[T any](items []T, page respond.Page, keyFn func(T) string) ([]T, string, bool) {
	if len(items) == 0 {
		return []T{}, "", false
	}
	start := 0
	if page.Cursor != "" {
		for i, item := range items {
			if keyFn(item) == page.Cursor {
				start = i + 1
				break
			}
		}
	}
	end := start + page.Limit
	if end > len(items) {
		end = len(items)
	}
	if start > len(items) {
		return []T{}, "", false
	}
	slice := append([]T(nil), items[start:end]...)
	hasMore := end < len(items)
	nextCursor := ""
	if hasMore && len(slice) > 0 {
		nextCursor = keyFn(slice[len(slice)-1])
	}
	return slice, nextCursor, hasMore
}
