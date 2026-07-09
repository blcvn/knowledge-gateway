package write

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"github.com/gorilla/mux"

	"kg-service/internal/access"
	"kg-service/internal/httpapi/respond"
)

var ErrMalformedRequest = errors.New("malformed json body")

type Handler struct {
	service *Service
}

func NewHandler(service *Service) Handler {
	return Handler{service: service}
}

func (h Handler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req NodeCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.CreateNodeWithContext(r.Context(), identity, req)
	if err != nil {
		log.Printf("write create_node failed tenant=%s app=%s domain_id=%s external_ref=%s err=%v", identity.TenantID, identity.AppID, req.DomainID, req.ExternalRef, err)
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrControlPlaneNotReady):
			respond.Error(w, respond.StatusFor(respond.CodeServiceUnavailable), respond.CodeServiceUnavailable, "Control plane is not ready", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}

	respond.Accepted(w, result)
}

func (h Handler) CreateNodesBulk(w http.ResponseWriter, r *http.Request) {
	var req NodeBulkCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.CreateNodesBulkWithContext(r.Context(), identity, req)
	if err != nil {
		writeError(w, err)
		return
	}

	respond.Accepted(w, result)
}

func (h Handler) OpenSyncSession(w http.ResponseWriter, r *http.Request) {
	var req OpenSyncSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.OpenSyncSession(r.Context(), identity, req)
	if err != nil {
		log.Printf("write open_sync_session failed tenant=%s app=%s domain_id=%s graph_scope=%s err=%v", identity.TenantID, identity.AppID, req.DomainID, req.GraphScope, err)
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrControlPlaneNotReady):
			respond.Error(w, respond.StatusFor(respond.CodeServiceUnavailable), respond.CodeServiceUnavailable, "Control plane is not ready", nil)
		case errors.Is(err, ErrScopeLocked):
			respond.Error(w, respond.StatusFor(respond.CodeSyncScopeLocked), respond.CodeSyncScopeLocked, "Sync scope is locked", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}
	respond.Accepted(w, result)
}

func (h Handler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	var req NodeUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.UpdateNodeWithContext(r.Context(), identity, mux.Vars(r)["id"], req)
	if err != nil {
		log.Printf("write update_node failed tenant=%s app=%s node_id=%s graph_version_id=%s err=%v", identity.TenantID, identity.AppID, mux.Vars(r)["id"], req.GraphVersionID, err)
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrControlPlaneNotReady):
			respond.Error(w, respond.StatusFor(respond.CodeServiceUnavailable), respond.CodeServiceUnavailable, "Control plane is not ready", nil)
		case errors.Is(err, ErrNotFound):
			respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}

	respond.OK(w, result)
}

func (h Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.DeleteNodeWithVersion(r.Context(), identity, mux.Vars(r)["id"], r.URL.Query().Get("graph_version_id"))
	if err != nil {
		log.Printf("write delete_node failed tenant=%s app=%s node_id=%s graph_version_id=%s err=%v", identity.TenantID, identity.AppID, mux.Vars(r)["id"], r.URL.Query().Get("graph_version_id"), err)
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrNotFound):
			respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}

	respond.OK(w, result)
}

func (h Handler) DeleteNodesByExternalRefPrefix(w http.ResponseWriter, r *http.Request) {
	var req NodeDeleteByExternalRefPrefixRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.DeleteNodesByExternalRefPrefixWithVersion(r.Context(), identity, req, r.URL.Query().Get("graph_version_id"))
	if err != nil {
		writeError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h Handler) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	var req RelationshipCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.CreateRelationshipWithContext(r.Context(), identity, req)
	if err != nil {
		log.Printf("write create_relationship failed tenant=%s app=%s rel_type=%s from=%s to=%s err=%v", identity.TenantID, identity.AppID, req.RelType, req.FromNodeID, req.ToNodeID, err)
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrControlPlaneNotReady):
			respond.Error(w, respond.StatusFor(respond.CodeServiceUnavailable), respond.CodeServiceUnavailable, "Control plane is not ready", nil)
		case errors.Is(err, ErrNotFound):
			respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}

	respond.Created(w, result)
}

func (h Handler) CreateRelationshipsBulk(w http.ResponseWriter, r *http.Request) {
	var req RelationshipBulkCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.CreateRelationshipsBulkWithContext(r.Context(), identity, req)
	if err != nil {
		writeError(w, err)
		return
	}

	respond.Created(w, result)
}

func (h Handler) CommitSyncSession(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	if err := h.service.CommitSyncSession(r.Context(), identity, mux.Vars(r)["id"]); err != nil {
		log.Printf("write commit_sync_session failed tenant=%s app=%s session_id=%s err=%v", identity.TenantID, identity.AppID, mux.Vars(r)["id"], err)
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrNotFound):
			respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
		case errors.Is(err, ErrSessionAbandoned):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrScopeLocked):
			respond.Error(w, respond.StatusFor(respond.CodeSyncScopeLocked), respond.CodeSyncScopeLocked, "Sync scope is locked", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}
	respond.OK(w, map[string]any{"status": "ok"})
}

func (h Handler) AbandonSyncSession(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	if err := h.service.AbandonSyncSession(r.Context(), identity, mux.Vars(r)["id"]); err != nil {
		log.Printf("write abandon_sync_session failed tenant=%s app=%s session_id=%s err=%v", identity.TenantID, identity.AppID, mux.Vars(r)["id"], err)
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrNotFound):
			respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
		case errors.Is(err, ErrSessionAbandoned):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}
	respond.NoContent(w)
}

func (h Handler) DeleteRelationshipsBulk(w http.ResponseWriter, r *http.Request) {
	var req RelationshipBulkDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.DeleteRelationshipsBulkWithContext(r.Context(), identity, req)
	if err != nil {
		writeError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h Handler) IngestDocument(w http.ResponseWriter, r *http.Request) {
	var req IngestDocumentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.IngestDocument(identity, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}

	respond.Accepted(w, result)
}

func (h Handler) GetIngestJob(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.GetIngestJob(identity, mux.Vars(r)["job_id"])
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrNotFound):
			respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}

	respond.OK(w, result)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return ErrMalformedRequest
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	log.Printf("write request failed err=%v", err)
	switch {
	case errors.Is(err, ErrMalformedRequest):
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
	case errors.Is(err, ErrForbidden):
		respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
	case errors.Is(err, ErrNotFound):
		respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
	case errors.Is(err, ErrValidation):
		respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
	case errors.Is(err, ErrControlPlaneNotReady):
		respond.Error(w, respond.StatusFor(respond.CodeServiceUnavailable), respond.CodeServiceUnavailable, "Control plane is not ready", nil)
	case errors.Is(err, ErrScopeLocked):
		respond.Error(w, respond.StatusFor(respond.CodeSyncScopeLocked), respond.CodeSyncScopeLocked, "Sync scope is locked", nil)
	default:
		respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
	}
}
