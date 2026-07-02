package read

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"kg-service/internal/access"
	"kg-service/internal/httpapi/respond"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) Handler {
	return Handler{service: service}
}

func (h Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	templates, err := h.service.ListTemplates(identity, r.URL.Query().Get("domain_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	page := parsePage(r)
	templates, nextCursor, hasMore := paginateTemplates(templates, page)
	respond.OK(w, respond.ListEnvelope[TemplateListItem]{Data: templates, NextCursor: nextCursor, HasMore: hasMore})
}

func (h Handler) ExecuteTemplate(w http.ResponseWriter, r *http.Request) {
	var req TemplateExecutionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.ExecuteTemplateWithOptions(identity, r.PathValue("domain_id"), r.PathValue("template_name"), req.Params, req.AppID, req.Mode)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}

func (h Handler) GraphSearch(w http.ResponseWriter, r *http.Request) {
	var req GraphSearchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.GraphSearch(identity, req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}

func (h Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	node, err := h.service.GetNodeForAppWithMode(identity, r.URL.Query().Get("app_id"), r.PathValue("id"), r.URL.Query().Get("mode"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, node)
}

var ErrMalformedRequest = errors.New("malformed json body")

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
	case errors.Is(err, ErrProjectionUnreadable):
		respond.Error(w, respond.StatusFor(respond.CodeProjectionInconsistent), respond.CodeProjectionInconsistent, "Projection inconsistency", nil)
	case errors.Is(err, ErrValidation):
		respond.Error(w, respond.StatusFor(respond.CodeValidationFailed), respond.CodeValidationFailed, err.Error(), nil)
	case errors.Is(err, ErrTimeout):
		respond.Error(w, respond.StatusFor(respond.CodeRequestTimedOut), respond.CodeRequestTimedOut, "Read timed out", nil)
	default:
		respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
	}
}

func parsePage(r *http.Request) respond.Page {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return respond.NormalizePage(limit, r.URL.Query().Get("cursor"), 20, 100)
}

func paginateTemplates(items []TemplateListItem, page respond.Page) ([]TemplateListItem, string, bool) {
	return paginateBy(items, page, func(item TemplateListItem) string { return item.TemplateName })
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
