package search

import (
	"encoding/json"
	"errors"
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

func (h Handler) SemanticSearch(w http.ResponseWriter, r *http.Request) {
	var req SemanticSearchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.SemanticSearch(identity, req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}

func (h Handler) RagSearch(w http.ResponseWriter, r *http.Request) {
	var req SemanticSearchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.RagSearch(identity, req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}

func (h Handler) FullTextSearch(w http.ResponseWriter, r *http.Request) {
	var req FullTextSearchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.FullTextSearch(identity, req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}

func (h Handler) HybridSearch(w http.ResponseWriter, r *http.Request) {
	var req HybridSearchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.HybridSearch(identity, req)
	if err != nil {
		writeError(w, err)
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
	switch {
	case errors.Is(err, ErrMalformedRequest):
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
	case errors.Is(err, ErrForbidden):
		respond.Error(w, respond.StatusFor(respond.CodeForbidden), respond.CodeForbidden, "Forbidden", nil)
	case errors.Is(err, ErrNotFound):
		respond.Error(w, respond.StatusFor(respond.CodeNotFound), respond.CodeNotFound, "Resource not found", nil)
	case errors.Is(err, ErrValidation):
		respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
	default:
		respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
	}
}
