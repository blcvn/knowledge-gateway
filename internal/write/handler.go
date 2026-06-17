package write

import (
	"encoding/json"
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

func (h Handler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req NodeCreateRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.CreateNode(identity, req)
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

func (h Handler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	var req NodeUpdateRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.UpdateNode(identity, r.PathValue("id"), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
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
	result, err := h.service.DeleteNode(identity, r.PathValue("id"))
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

func (h Handler) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	var req RelationshipCreateRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.CreateRelationship(identity, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
		case errors.Is(err, ErrValidation):
			respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
		case errors.Is(err, ErrNotFound):
			respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
		default:
			respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
		}
		return
	}

	respond.Created(w, result)
}
