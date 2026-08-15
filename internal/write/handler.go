package write

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

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
	nodeID := r.PathValue("id")
	result, err := h.service.UpdateNodeWithContext(r.Context(), identity, nodeID, req)
	if err != nil {
		log.Printf("write update_node failed tenant=%s app=%s node_id=%s graph_version_id=%s err=%v", identity.TenantID, identity.AppID, nodeID, req.GraphVersionID, err)
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
	nodeID := r.PathValue("id")
	result, err := h.service.DeleteNodeWithVersion(r.Context(), identity, nodeID, r.URL.Query().Get("graph_version_id"))
	if err != nil {
		log.Printf("write delete_node failed tenant=%s app=%s node_id=%s graph_version_id=%s err=%v", identity.TenantID, identity.AppID, nodeID, r.URL.Query().Get("graph_version_id"), err)
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
	sessionID := r.PathValue("id")
	if err := h.service.CommitSyncSession(r.Context(), identity, sessionID); err != nil {
		log.Printf("write commit_sync_session failed tenant=%s app=%s session_id=%s err=%v", identity.TenantID, identity.AppID, sessionID, err)
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
	sessionID := r.PathValue("id")
	if err := h.service.AbandonSyncSession(r.Context(), identity, sessionID); err != nil {
		log.Printf("write abandon_sync_session failed tenant=%s app=%s session_id=%s err=%v", identity.TenantID, identity.AppID, sessionID, err)
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
	result, err := h.service.GetIngestJob(identity, r.PathValue("job_id"))
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

// DeleteByScope serves POST /v1/kg/write/graph:delete-by-scope.
func (h Handler) DeleteByScope(w http.ResponseWriter, r *http.Request) {
	var req ScopeDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.DeleteByScopeWithVersion(r.Context(), identity, req)
	if err != nil {
		log.Printf("write delete_by_scope failed tenant=%s app=%s graph_scope=%s err=%v", identity.TenantID, identity.AppID, req.GraphScope, err)
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}

// DeleteRelationshipsByExternalRef serves DELETE /v1/kg/write/relationships:by-external-ref.
func (h Handler) DeleteRelationshipsByExternalRef(w http.ResponseWriter, r *http.Request) {
	var req RelationshipDeleteByExternalRefRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.DeleteRelationshipsByExternalRefWithVersion(r.Context(), identity, req)
	if err != nil {
		log.Printf("write delete_relationships_by_external_ref failed tenant=%s app=%s refs=%d err=%v", identity.TenantID, identity.AppID, len(req.ExternalRefs), err)
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}
