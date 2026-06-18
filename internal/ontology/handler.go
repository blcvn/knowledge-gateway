package ontology

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

func (h Handler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	var req DomainCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	domain, err := h.service.CreateDomain(identity, r.PathValue("tenant_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, domain)
}

func (h Handler) CreateNodeType(w http.ResponseWriter, r *http.Request) {
	var req NodeTypeCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	schema, err := h.service.CreateNodeType(identity, r.PathValue("tenant_id"), r.PathValue("domain_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, schema)
}

func (h Handler) CreateRelType(w http.ResponseWriter, r *http.Request) {
	var req RelTypeCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	schema, err := h.service.CreateRelType(identity, r.PathValue("tenant_id"), r.PathValue("domain_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, schema)
}

func (h Handler) GetEffective(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	domains, err := h.service.GetEffectiveDomains(identity, r.PathValue("tenant_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, EffectiveOntologyResponse{Domains: domains})
}

func (h Handler) GetDomain(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	domain, err := h.service.GetDomainDetails(identity, r.PathValue("domain_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, domain)
}

func (h Handler) GetSearchProfile(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	profile, err := h.service.Resolve(r.PathValue("domain_id"), identity.TenantID, identity.AppID)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, profile)
}

func (h Handler) UpsertSearchProfile(w http.ResponseWriter, r *http.Request) {
	var req SearchProfile
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	profile, err := h.service.UpsertSearchProfile(identity, r.PathValue("tenant_id"), r.PathValue("domain_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, profile)
}

func (h Handler) ListQueryStrategies(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, h.service.ListQueryStrategies())
}

func (h Handler) CreateQueryStrategy(w http.ResponseWriter, r *http.Request) {
	var req QueryStrategy
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	strategy, err := h.service.UpsertQueryStrategy(identity, r.PathValue("tenant_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, strategy)
}

func (h Handler) UpdateQueryStrategy(w http.ResponseWriter, r *http.Request) {
	var req QueryStrategy
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.Key = r.PathValue("key")
	identity, _ := access.IdentityFromContext(r.Context())
	strategy, err := h.service.UpsertQueryStrategy(identity, r.PathValue("tenant_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, strategy)
}

func (h Handler) CreateQueryTemplate(w http.ResponseWriter, r *http.Request) {
	var req QueryTemplateCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	template, err := h.service.CreateQueryTemplate(identity, r.PathValue("tenant_id"), r.PathValue("domain_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, template)
}

func (h Handler) ActivateQueryTemplate(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	template, err := h.service.ActivateQueryTemplate(identity, r.PathValue("tenant_id"), r.PathValue("domain_id"), r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	respond.OK(w, template)
}

func (h Handler) UpsertStatusFieldConfig(w http.ResponseWriter, r *http.Request) {
	var req StatusFieldConfigRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	identity, _ := access.IdentityFromContext(r.Context())
	config, err := h.service.UpsertStatusFieldConfig(identity, r.PathValue("tenant_id"), r.PathValue("domain_id"), req)
	if err != nil {
		writeError(w, err)
		return
	}
	respond.Created(w, config)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return access.ErrRequestMalformed
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
	case errors.Is(err, access.ErrRequestMalformed):
		respond.Error(w, respond.StatusFor(respond.CodeBadRequest), respond.CodeBadRequest, "Malformed JSON body", nil)
	default:
		respond.Error(w, respond.StatusFor(respond.CodeInternal), respond.CodeInternal, "Internal server error", nil)
	}
}
