package ontology

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"kg-service/internal/access"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation")
)

type AccessResolver interface {
	ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error)
}

type Service struct {
	store          DomainStore
	accessResolver AccessResolver
	now            func() time.Time
}

func NewService(store DomainStore, accessResolver AccessResolver) *Service {
	return &Service{
		store:          store,
		accessResolver: accessResolver,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) CreateDomain(actor access.Identity, tenantID string, req DomainCreateRequest) (Domain, error) {
	if !canManageTenant(actor, tenantID) {
		return Domain{}, ErrForbidden
	}
	if err := validateDomainCreateRequest(req); err != nil {
		return Domain{}, errors.Join(ErrValidation, err)
	}
	if _, exists := s.store.GetDomain(req.ID); exists {
		return Domain{}, errors.Join(ErrValidation, fmt.Errorf("domain already exists: %s", req.ID))
	}

	now := s.now()
	domain := Domain{
		ID:            strings.TrimSpace(req.ID),
		Name:          strings.TrimSpace(req.Name),
		Description:   strings.TrimSpace(req.Description),
		OwnerTenantID: tenantID,
		Status:        fallback(req.Status, "draft"),
		Version:       1,
		Visibility:    fallback(req.Visibility, "private"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	created := s.store.CreateDomain(domain)
	s.store.CreateVersion(OntologyVersion{
		DomainID:    created.ID,
		Version:     1,
		PublishedAt: now,
	})
	return created, nil
}

func (s *Service) CreateNodeType(actor access.Identity, tenantID, domainID string, req NodeTypeCreateRequest) (NodeTypeSchema, error) {
	if !canManageTenant(actor, tenantID) {
		return NodeTypeSchema{}, ErrForbidden
	}
	domain, ok := s.store.GetDomain(domainID)
	if !ok {
		return NodeTypeSchema{}, ErrNotFound
	}
	if domain.OwnerTenantID != tenantID {
		return NodeTypeSchema{}, ErrForbidden
	}
	if err := validateNodeTypeCreateRequest(req); err != nil {
		return NodeTypeSchema{}, errors.Join(ErrValidation, err)
	}
	if _, exists := s.store.GetNodeType(domainID, req.NodeTypeName); exists {
		return NodeTypeSchema{}, errors.Join(ErrValidation, fmt.Errorf("node type already exists: %s", req.NodeTypeName))
	}

	now := s.now()
	graphLabel := fallback(req.GraphLabel, req.NodeTypeName)
	schema := NodeTypeSchema{
		ID:              domainID + "." + req.NodeTypeName,
		DomainID:        domainID,
		NodeTypeName:    req.NodeTypeName,
		GraphLabel:      graphLabel,
		RequiredProps:   req.RequiredProps,
		OptionalProps:   req.OptionalProps,
		ValidationRules: req.ValidationRules,
		Version:         1,
		CreatedAt:       now,
	}
	return s.store.CreateNodeType(schema), nil
}

func (s *Service) CreateRelType(actor access.Identity, tenantID, domainID string, req RelTypeCreateRequest) (RelTypeSchema, error) {
	if !canManageTenant(actor, tenantID) {
		return RelTypeSchema{}, ErrForbidden
	}
	domain, ok := s.store.GetDomain(domainID)
	if !ok {
		return RelTypeSchema{}, ErrNotFound
	}
	if domain.OwnerTenantID != tenantID {
		return RelTypeSchema{}, ErrForbidden
	}
	if err := validateRelTypeCreateRequest(req); err != nil {
		return RelTypeSchema{}, errors.Join(ErrValidation, err)
	}
	if _, exists := s.store.GetRelType(domainID, req.RelTypeName, req.FromNodeType, req.ToNodeType); exists {
		return RelTypeSchema{}, errors.Join(ErrValidation, fmt.Errorf("relationship type already exists: %s", req.RelTypeName))
	}

	schema := RelTypeSchema{
		ID:            relKey(domainID, req.RelTypeName, req.FromNodeType, req.ToNodeType),
		DomainID:      domainID,
		RelTypeName:   req.RelTypeName,
		FromNodeType:  req.FromNodeType,
		ToNodeType:    req.ToNodeType,
		SameDomain:    req.SameDomain,
		RequiredProps: req.RequiredProps,
		OptionalProps: req.OptionalProps,
	}
	return s.store.CreateRelType(schema), nil
}

func (s *Service) GetEffectiveDomains(actor access.Identity, tenantID string) ([]DomainSummary, error) {
	if actor.TenantID != tenantID && !isPlatformAdmin(actor) {
		return nil, ErrForbidden
	}
	visibleOwners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return nil, err
	}

	grantDomains := map[string]struct{}{}
	grantAllOwners := map[string]struct{}{}
	for _, owner := range visibleOwners {
		if owner.Source != "grant" {
			continue
		}
		if owner.ScopeType == "domain" && owner.ScopeValue != "" {
			grantDomains[owner.ScopeValue] = struct{}{}
		}
		if owner.ScopeType == "all" {
			grantAllOwners[owner.TenantID] = struct{}{}
		}
	}

	allDomains := s.store.ListDomains()
	summaries := make([]DomainSummary, 0, len(allDomains))
	seen := map[string]struct{}{}
	for _, domain := range allDomains {
		if _, ok := seen[domain.ID]; ok {
			continue
		}
		if domain.OwnerTenantID == access.PlatformTenantID || domain.OwnerTenantID == tenantID {
			summaries = append(summaries, DomainSummary{ID: domain.ID, Owner: domain.OwnerTenantID})
			seen[domain.ID] = struct{}{}
			continue
		}
		if _, ok := grantDomains[domain.ID]; ok {
			summaries = append(summaries, DomainSummary{ID: domain.ID, Owner: domain.OwnerTenantID})
			seen[domain.ID] = struct{}{}
			continue
		}
		if _, ok := grantAllOwners[domain.OwnerTenantID]; ok {
			summaries = append(summaries, DomainSummary{ID: domain.ID, Owner: domain.OwnerTenantID})
			seen[domain.ID] = struct{}{}
		}
	}

	return summaries, nil
}

func (s *Service) GetVisibleDomain(actor access.Identity, domainID string) (Domain, error) {
	domain, ok := s.store.GetDomain(domainID)
	if !ok {
		return Domain{}, ErrNotFound
	}

	if domain.OwnerTenantID == actor.TenantID || domain.OwnerTenantID == access.PlatformTenantID {
		return domain, nil
	}

	effective, err := s.GetEffectiveDomains(actor, actor.TenantID)
	if err != nil {
		return Domain{}, err
	}
	for _, summary := range effective {
		if summary.ID == domainID {
			return domain, nil
		}
	}
	return Domain{}, ErrForbidden
}

func (s *Service) GetCurrentVersion(domainID string) (OntologyVersion, error) {
	version, ok := s.store.GetCurrentVersion(domainID)
	if !ok {
		return OntologyVersion{}, ErrNotFound
	}
	return version, nil
}

func (s *Service) GetQueryTemplate(domainID, templateName string) (QueryTemplate, error) {
	template, ok := s.store.GetQueryTemplate(domainID, templateName)
	if !ok {
		return QueryTemplate{}, ErrNotFound
	}
	return template, nil
}

func (s *Service) ListQueryTemplates(domainID string) []QueryTemplate {
	return s.store.ListQueryTemplates(domainID)
}

func (s *Service) GetNodeType(domainID, nodeTypeName string) (NodeTypeSchema, error) {
	schema, ok := s.store.GetNodeType(domainID, nodeTypeName)
	if !ok {
		return NodeTypeSchema{}, ErrNotFound
	}
	return schema, nil
}

func (s *Service) GetRelType(domainID, relTypeName, fromNodeType, toNodeType string) (RelTypeSchema, error) {
	schema, ok := s.store.GetRelType(domainID, relTypeName, fromNodeType, toNodeType)
	if !ok {
		return RelTypeSchema{}, ErrNotFound
	}
	return schema, nil
}

func (s *Service) ListCrossDomainRules(domainID string) []CrossDomainRelRule {
	return s.store.ListCrossDomainRules(domainID)
}

func (s *Service) ResolveCrossDomainRules(domainID, fromNodeType string) []CrossDomainRelRule {
	rules := s.store.ListCrossDomainRules(domainID)
	var result []CrossDomainRelRule
	for _, rule := range rules {
		if !slices.Contains(rule.FromNodeTypes, fromNodeType) {
			continue
		}
		if slices.Contains(rule.ExceptionTypes, fromNodeType) {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func (s *Service) ValidateCrossDomainTarget(rule CrossDomainRelRule, targetDomainID, targetNodeType string) error {
	if strings.TrimSpace(targetDomainID) == "" {
		return errors.Join(ErrValidation, errors.New("bridge target domain is required"))
	}
	if strings.TrimSpace(targetNodeType) == "" {
		return errors.Join(ErrValidation, errors.New("bridge target node type is required"))
	}
	if rule.ToDomainID != "" && rule.ToDomainID != targetDomainID {
		return errors.Join(ErrValidation, fmt.Errorf("bridge target domain %s does not match rule domain %s", targetDomainID, rule.ToDomainID))
	}
	if len(rule.ToNodeTypes) > 0 && !slices.Contains(rule.ToNodeTypes, targetNodeType) {
		return errors.Join(ErrValidation, fmt.Errorf("bridge target node type %s is not allowed for rule %s", targetNodeType, rule.RelTypeName))
	}
	return nil
}

func (s *Service) GetStatusFieldConfig(domainID string) (*StatusFieldConfig, error) {
	config, ok := s.store.GetStatusFieldConfig(domainID)
	if !ok {
		return nil, nil
	}
	return &config, nil
}

func (s *Service) CreateQueryTemplate(actor access.Identity, tenantID, domainID string, req QueryTemplateCreateRequest) (QueryTemplate, error) {
	if !canManageTenant(actor, tenantID) {
		return QueryTemplate{}, ErrForbidden
	}
	domain, ok := s.store.GetDomain(domainID)
	if !ok {
		return QueryTemplate{}, ErrNotFound
	}
	if domain.OwnerTenantID != tenantID {
		return QueryTemplate{}, ErrForbidden
	}
	if err := validateQueryTemplateCreateRequest(req); err != nil {
		return QueryTemplate{}, errors.Join(ErrValidation, err)
	}
	if _, exists := s.store.GetQueryTemplate(domainID, req.TemplateName); exists {
		return QueryTemplate{}, errors.Join(ErrValidation, fmt.Errorf("query template already exists: %s", req.TemplateName))
	}

	now := s.now()
	template := QueryTemplate{
		ID:           domainID + "." + req.TemplateName,
		DomainID:     domainID,
		TemplateName: req.TemplateName,
		PatternSpec:  req.PatternSpec,
		ParamSchema:  req.ParamSchema,
		ReturnFields: req.ReturnFields,
		Description:  strings.TrimSpace(req.Description),
		Status:       "draft",
		Version:      1,
		CreatedAt:    now,
	}
	return s.store.CreateQueryTemplate(template), nil
}

func (s *Service) ActivateQueryTemplate(actor access.Identity, tenantID, domainID, templateName string) (QueryTemplate, error) {
	if !canManageTenant(actor, tenantID) {
		return QueryTemplate{}, ErrForbidden
	}
	domain, ok := s.store.GetDomain(domainID)
	if !ok {
		return QueryTemplate{}, ErrNotFound
	}
	if domain.OwnerTenantID != tenantID {
		return QueryTemplate{}, ErrForbidden
	}
	template, ok := s.store.GetQueryTemplate(domainID, templateName)
	if !ok {
		return QueryTemplate{}, ErrNotFound
	}
	template.Status = "active"
	updated, ok := s.store.UpdateQueryTemplate(template)
	if !ok {
		return QueryTemplate{}, ErrNotFound
	}
	return updated, nil
}

func (s *Service) UpsertStatusFieldConfig(actor access.Identity, tenantID, domainID string, req StatusFieldConfigRequest) (StatusFieldConfig, error) {
	if !canManageTenant(actor, tenantID) {
		return StatusFieldConfig{}, ErrForbidden
	}
	domain, ok := s.store.GetDomain(domainID)
	if !ok {
		return StatusFieldConfig{}, ErrNotFound
	}
	if domain.OwnerTenantID != tenantID {
		return StatusFieldConfig{}, ErrForbidden
	}
	if err := validateStatusFieldConfigRequest(req); err != nil {
		return StatusFieldConfig{}, errors.Join(ErrValidation, err)
	}

	config := StatusFieldConfig{
		DomainID:            domainID,
		StatusFieldName:     req.StatusFieldName,
		ValidStatusValues:   req.ValidStatusValues,
		WarningStatusValues: req.WarningStatusValues,
		CascadeRules:        req.CascadeRules,
		AuthorityFieldName:  req.AuthorityFieldName,
		AuthorityValuesMap:  req.AuthorityValuesMap,
	}
	return s.store.UpsertStatusFieldConfig(config), nil
}

func (s *Service) GetDomainDetails(actor access.Identity, domainID string) (DomainDetailsResponse, error) {
	domain, err := s.GetVisibleDomain(actor, domainID)
	if err != nil {
		return DomainDetailsResponse{}, err
	}

	nodeTypes := s.store.ListNodeTypes(domainID)
	templates := s.store.ListQueryTemplates(domainID)
	var statusFieldConfig *StatusFieldConfig
	if cfg, ok := s.store.GetStatusFieldConfig(domainID); ok {
		statusFieldConfig = &cfg
	}

	return DomainDetailsResponse{
		Domain:            domain,
		NodeTypes:         nodeTypes,
		RelTypes:          s.store.ListRelTypes(domainID),
		QueryTemplates:    templates,
		StatusFieldConfig: statusFieldConfig,
	}, nil
}

func validateDomainCreateRequest(req DomainCreateRequest) error {
	if strings.TrimSpace(req.ID) == "" || strings.TrimSpace(req.Name) == "" {
		return errors.New("id and name are required")
	}
	if req.Status != "" && !isAllowed(req.Status, []string{"draft", "active", "deprecated"}) {
		return fmt.Errorf("invalid status: %s", req.Status)
	}
	if req.Visibility != "" && !isAllowed(req.Visibility, []string{"public", "tenant_shared", "private"}) {
		return fmt.Errorf("invalid visibility: %s", req.Visibility)
	}
	return nil
}

func validateNodeTypeCreateRequest(req NodeTypeCreateRequest) error {
	if strings.TrimSpace(req.NodeTypeName) == "" {
		return errors.New("node_type_name is required")
	}
	for _, prop := range append([]PropertySchema{}, append(req.RequiredProps, req.OptionalProps...)...) {
		if strings.TrimSpace(prop.Name) == "" || strings.TrimSpace(prop.Type) == "" {
			return errors.New("property name and type are required")
		}
		if !isAllowed(prop.Type, []string{"string", "number", "boolean", "object", "array"}) {
			return fmt.Errorf("invalid property type: %s", prop.Type)
		}
	}
	return nil
}

func validateRelTypeCreateRequest(req RelTypeCreateRequest) error {
	if strings.TrimSpace(req.RelTypeName) == "" {
		return errors.New("rel_type_name is required")
	}
	if strings.TrimSpace(req.FromNodeType) == "" || strings.TrimSpace(req.ToNodeType) == "" {
		return errors.New("from_node_type and to_node_type are required")
	}
	for _, prop := range append([]PropertySchema{}, append(req.RequiredProps, req.OptionalProps...)...) {
		if strings.TrimSpace(prop.Name) == "" || strings.TrimSpace(prop.Type) == "" {
			return errors.New("property name and type are required")
		}
		if !isAllowed(prop.Type, []string{"string", "number", "boolean", "object", "array"}) {
			return fmt.Errorf("invalid property type: %s", prop.Type)
		}
	}
	return nil
}

func validateQueryTemplateCreateRequest(req QueryTemplateCreateRequest) error {
	if strings.TrimSpace(req.TemplateName) == "" {
		return errors.New("template_name is required")
	}
	if len(req.ReturnFields) == 0 {
		return errors.New("return_fields is required")
	}
	if len(req.PatternSpec) == 0 {
		return errors.New("pattern_spec is required")
	}
	if _, hasCypher := req.PatternSpec["cypher"]; hasCypher {
		return errors.New("raw cypher is not allowed")
	}
	start, ok := req.PatternSpec["start"].(map[string]any)
	if !ok {
		return errors.New("pattern_spec.start is required")
	}
	if _, ok := start["node_type"].(string); !ok {
		return errors.New("pattern_spec.start.node_type is required")
	}
	if hopsRaw, ok := req.PatternSpec["hops"]; ok {
		hops, ok := hopsRaw.([]any)
		if !ok {
			return errors.New("pattern_spec.hops must be an array")
		}
		if len(hops) > 5 {
			return fmt.Errorf("pattern_spec has %d hops, exceeds limit 5", len(hops))
		}
		for _, hopRaw := range hops {
			hop, ok := hopRaw.(map[string]any)
			if !ok {
				return errors.New("pattern_spec.hops entries must be objects")
			}
			if _, ok := hop["rel_type"].(string); !ok {
				return errors.New("pattern_spec.hops.rel_type is required")
			}
			if _, ok := hop["to_node_type"].(string); !ok {
				return errors.New("pattern_spec.hops.to_node_type is required")
			}
		}
	}
	for _, param := range req.ParamSchema {
		if strings.TrimSpace(param.Name) == "" {
			return errors.New("param_schema.name is required")
		}
		if !isAllowed(param.Type, []string{"string", "number", "boolean", "object", "array"}) {
			return fmt.Errorf("invalid param type: %s", param.Type)
		}
	}
	return nil
}

func validateStatusFieldConfigRequest(req StatusFieldConfigRequest) error {
	if strings.TrimSpace(req.StatusFieldName) == "" {
		return errors.New("status_field_name is required")
	}
	if len(req.ValidStatusValues) == 0 {
		return errors.New("valid_status_values is required")
	}
	for _, rule := range req.CascadeRules {
		if strings.TrimSpace(rule.FromNodeType) == "" || strings.TrimSpace(rule.ViaRel) == "" || strings.TrimSpace(rule.ToNodeType) == "" {
			return errors.New("cascade_rules entries must include from_node_type, via_rel, and to_node_type")
		}
	}
	if req.AuthorityFieldName == "" && len(req.AuthorityValuesMap) > 0 {
		return errors.New("authority_field_name is required when authority_values_map is provided")
	}
	return nil
}

func canManageTenant(actor access.Identity, tenantID string) bool {
	if isPlatformAdmin(actor) {
		return true
	}
	return actor.TenantID == tenantID && (actor.AppType == "admin_tool" || actor.AppType == "hybrid")
}

func isPlatformAdmin(actor access.Identity) bool {
	return actor.TenantID == access.PlatformTenantID && (actor.AppType == "admin_tool" || actor.AppType == "hybrid")
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func isAllowed(value string, allowed []string) bool {
	return slices.Contains(allowed, value)
}
