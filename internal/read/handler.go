package read

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

func (h Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	templates, err := h.service.ListTemplates(identity, r.URL.Query().Get("domain_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, respond.ListEnvelope[TemplateListItem]{Data: templates, HasMore: false})
}

func (h Handler) ExecuteTemplate(w http.ResponseWriter, r *http.Request) {
	var req TemplateExecutionRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
		return
	}

	identity, _ := access.IdentityFromContext(r.Context())
	result, err := h.service.ExecuteTemplate(identity, r.PathValue("domain_id"), r.PathValue("template_name"), req.Params)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, result)
}

func (h Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	node, err := h.service.GetNode(identity, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, node)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
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
