package read

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/telemetry"
	"kg-service/internal/write"
)

var (
	ErrForbidden            = errors.New("forbidden")
	ErrNotFound             = errors.New("not found")
	ErrProjectionUnreadable = errors.New("projection unreadable")
	ErrValidation           = errors.New("validation")
	ErrTimeout              = errors.New("timeout")
)

type OntologyResolver interface {
	GetVisibleDomain(actor access.Identity, domainID string) (ontology.Domain, error)
	GetQueryTemplate(domainID, templateName string) (ontology.QueryTemplate, error)
	ListQueryTemplates(domainID string) []ontology.QueryTemplate
	GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error)
	Resolve(domainID, tenantID, appID string) (ontology.ResolvedSearchProfile, error)
}

type AccessResolver interface {
	ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error)
}

type AuditLogger interface {
	RecordReadAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any)
}

type Service struct {
	sourceStore    ProjectionStore
	sourceIndex    GraphIndex
	index          GraphIndex
	ontology       OntologyResolver
	accessResolver AccessResolver
	auditLogger    AuditLogger
	compiler       QueryTemplateCompiler
	maxRows        int
	queryTimeout   time.Duration
	now            func() time.Time
}

func NewService(store ProjectionStore, ontology OntologyResolver, accessResolver AccessResolver, auditLogger AuditLogger) *Service {
	return &Service{
		sourceStore:    store,
		sourceIndex:    NewProjectionGraphIndex(store),
		index:          NewProjectionGraphIndex(store),
		ontology:       ontology,
		accessResolver: accessResolver,
		auditLogger:    auditLogger,
		compiler:       NewQueryTemplateCompiler(),
		maxRows:        100,
		queryTimeout:   50 * time.Millisecond,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) SetGraphAdapter(adapter graphstore.GraphAdapter) {
	if adapter == nil {
		return
	}
	s.index = ProjectionGraphIndex{adapter: adapter}
}

func (s *Service) ListTemplates(actor access.Identity, domainID string) ([]TemplateListItem, error) {
	if _, err := s.ontology.GetVisibleDomain(actor, domainID); err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			s.recordAudit(actor, "read", "query_template_list", domainID, "deny", "domain_not_visible", nil)
			return nil, ErrForbidden
		}
		return nil, err
	}

	templates := s.ontology.ListQueryTemplates(domainID)
	result := make([]TemplateListItem, 0, len(templates))
	for _, template := range templates {
		if template.Status != "active" {
			continue
		}
		result = append(result, TemplateListItem{
			TemplateName: template.TemplateName,
			Status:       template.Status,
			ParamSchema:  toParamSchema(template.ParamSchema),
		})
	}
	s.recordAudit(actor, "read", "query_template_list", domainID, "allow", "", map[string]any{
		"result_count": len(result),
	})
	return result, nil
}

func (s *Service) ExecuteTemplate(actor access.Identity, domainID, templateName string, params map[string]any) (TemplateExecutionResponse, error) {
	return s.ExecuteTemplateWithOptions(actor, domainID, templateName, params, actor.AppID, ReadModeNonRealtime)
}

func (s *Service) ExecuteTemplateWithOptions(actor access.Identity, domainID, templateName string, params map[string]any, appID, mode string) (TemplateExecutionResponse, error) {
	domain, err := s.ontology.GetVisibleDomain(actor, domainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "deny", "domain_not_visible", nil)
			return TemplateExecutionResponse{}, ErrForbidden
		}
		return TemplateExecutionResponse{}, err
	}

	template, err := s.ontology.GetQueryTemplate(domainID, templateName)
	if err != nil {
		s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "deny", "template_not_found", nil)
		return TemplateExecutionResponse{}, ErrNotFound
	}
	if template.Status != "active" {
		s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "deny", "template_inactive", nil)
		return TemplateExecutionResponse{}, ErrNotFound
	}
	resolvedProfile, err := s.ontology.Resolve(domainID, actor.TenantID, actor.AppID)
	if err != nil {
		return TemplateExecutionResponse{}, err
	}
	compiled, err := s.compiler.Compile(domainID, template, resolvedProfile.QueryStrategy)
	if err != nil {
		s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "deny", err.Error(), nil)
		return TemplateExecutionResponse{}, err
	}

	bound, err := bindTemplateParams(template, params)
	if err != nil {
		s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "deny", err.Error(), nil)
		return TemplateExecutionResponse{}, err
	}

	allowedOwners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return TemplateExecutionResponse{}, err
	}
	visibility := visibleOwnerSet(allowedOwners)
	statusCfg, _ := s.ontology.GetStatusFieldConfig(domain.ID)
	results, err := s.executeGraphQueryWithMode(actor, appID, normalizeReadMode(mode), domainID, compiled, bound, visibility, statusCfg)
	if err != nil {
		if errors.Is(err, ErrTimeout) {
			s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "deny", "query_timeout", nil)
		}
		return TemplateExecutionResponse{}, err
	}
	s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "allow", "", map[string]any{
		"result_count": len(results),
		"query":        compiled.Query,
		"read_mode":    normalizeReadMode(mode),
		"app_id":       strings.TrimSpace(appID),
	})
	return TemplateExecutionResponse{
		Results:     results,
		QueryTimeMs: 1,
	}, nil
}

func (s *Service) GraphSearch(actor access.Identity, req GraphSearchRequest) (TemplateExecutionResponse, error) {
	if strings.TrimSpace(req.DomainID) == "" {
		return TemplateExecutionResponse{}, errors.Join(ErrValidation, errors.New("domain_id is required"))
	}
	if strings.TrimSpace(req.StartNodeType) == "" {
		return TemplateExecutionResponse{}, errors.Join(ErrValidation, errors.New("start_node_type is required"))
	}

	domain, err := s.ontology.GetVisibleDomain(actor, req.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			s.recordAudit(actor, "read", "graph_search", req.DomainID, "deny", "domain_not_visible", nil)
			return TemplateExecutionResponse{}, ErrForbidden
		}
		return TemplateExecutionResponse{}, err
	}
	resolvedProfile, err := s.ontology.Resolve(req.DomainID, actor.TenantID, actor.AppID)
	if err != nil {
		return TemplateExecutionResponse{}, err
	}

	query := req.ToGraphQuery()
	query.ACLTokensParam = "acl_tokens"
	allowedDepth := resolvedProfile.QueryStrategy.MaxDepth
	if allowedDepth <= 0 {
		allowedDepth = 5
	}
	if query.MaxDepth <= 0 || query.MaxDepth > allowedDepth {
		query.MaxDepth = allowedDepth
	}
	if query.Strategy == "" {
		query.Strategy = resolvedProfile.QueryStrategy.Key
	}
	if query.Strategy == "" {
		query.Strategy = "default"
	}

	allowedOwners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return TemplateExecutionResponse{}, err
	}
	visibility := visibleOwnerSet(allowedOwners)
	statusCfg, _ := s.ontology.GetStatusFieldConfig(domain.ID)
	compiled := CompiledTemplate{
		DomainID:     req.DomainID,
		TemplateName: "graph_search",
		StartType:    req.StartNodeType,
		ReturnFields: query.ReturnFields,
		GraphQuery:   query,
	}
	results, err := s.executeGraphQueryWithMode(actor, req.AppID, normalizeReadMode(req.Mode), req.DomainID, compiled, req.Params, visibility, statusCfg)
	if err != nil {
		if errors.Is(err, ErrTimeout) {
			s.recordAudit(actor, "read", "graph_search", req.DomainID, "deny", "query_timeout", nil)
		}
		return TemplateExecutionResponse{}, err
	}
	s.recordAudit(actor, "read", "graph_search", req.DomainID, "allow", "", map[string]any{
		"result_count": len(results),
		"read_mode":    normalizeReadMode(req.Mode),
		"app_id":       strings.TrimSpace(req.AppID),
	})
	return TemplateExecutionResponse{
		Results:     results,
		QueryTimeMs: 1,
	}, nil
}

func (s *Service) executeGraphQueryWithMode(actor access.Identity, appID, mode, domainID string, compiled CompiledTemplate, bound map[string]any, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig) ([]map[string]any, error) {
	targetAppID := strings.TrimSpace(appID)
	if targetAppID == "" {
		targetAppID = actor.AppID
	}
	if mode != ReadModeRealtime {
		results, err := s.index.ExecuteTemplate(actor, domainID, compiled, bound, visibility, statusCfg, s.maxRows, s.queryTimeout, s.now)
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			if startNodeID, ok := resolveStartNodeID(compiled.GraphQuery.StartMatch, bound); ok && s.projectionUnreadable(targetAppID, startNodeID, visibility) {
				return nil, ErrProjectionUnreadable
			}
		}
		return results, nil
	}
	startNodeID, ok := resolveStartNodeID(compiled.GraphQuery.StartMatch, bound)
	if !ok {
		return s.index.ExecuteTemplate(actor, domainID, compiled, bound, visibility, statusCfg, s.maxRows, s.queryTimeout, s.now)
	}
	sourceNode, ok := s.sourceStore.GetNodeByID(startNodeID)
	if !ok || sourceNode.IsDeleted || !isNodeVisible(sourceNode, visibility) || !matchesAppScope(sourceNode.OwnerAppID, targetAppID) {
		return nil, ErrNotFound
	}
	graphVersion, err := s.index.ReadSyncVersion(startNodeID)
	if err == nil && graphVersion == int64(sourceNode.DomainVersion) {
		return s.index.ExecuteTemplate(actor, domainID, compiled, bound, visibility, statusCfg, s.maxRows, s.queryTimeout, s.now)
	}
	telemetry.RecordRealtimeReadFallback(sourceNode.DomainID, targetAppID)
	return s.sourceIndex.ExecuteTemplate(actor, domainID, compiled, bound, visibility, statusCfg, s.maxRows, s.queryTimeout, s.now)
}

func resolveStartNodeID(match map[string]any, bound map[string]any) (string, bool) {
	if match == nil {
		return "", false
	}
	raw, ok := match["id"]
	if !ok {
		return "", false
	}
	value, resolved := resolveTemplateValue(raw, bound)
	if !resolved {
		value = raw
	}
	id := strings.TrimSpace(fmt.Sprint(value))
	if id == "" {
		return "", false
	}
	return id, true
}

func (s *Service) GetNode(actor access.Identity, nodeID string) (NodeResponse, error) {
	return s.GetNodeForAppWithMode(actor, actor.AppID, nodeID, ReadModeNonRealtime)
}

func (s *Service) GetNodeWithMode(actor access.Identity, nodeID, mode string) (NodeResponse, error) {
	return s.GetNodeForAppWithMode(actor, actor.AppID, nodeID, mode)
}

func (s *Service) GetNodeForAppWithMode(actor access.Identity, appID, nodeID, mode string) (NodeResponse, error) {
	owners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return NodeResponse{}, err
	}
	visibility := visibleOwnerSet(owners)
	targetAppID := strings.TrimSpace(appID)
	if targetAppID == "" {
		targetAppID = actor.AppID
	}
	node, ok := s.nodeForMode(actor, targetAppID, nodeID, normalizeReadMode(mode), visibility)
	if !ok {
		if normalizeReadMode(mode) == ReadModeNonRealtime && s.projectionUnreadable(targetAppID, nodeID, visibility) {
			s.recordAudit(actor, "read", "kg_node", nodeID, "deny", "projection_unreadable", nil)
			return NodeResponse{}, ErrProjectionUnreadable
		}
		s.recordAudit(actor, "read", "kg_node", nodeID, "deny", "node_not_found_or_invisible", nil)
		return NodeResponse{}, ErrNotFound
	}
	s.recordAudit(actor, "read", "kg_node", nodeID, "allow", "", map[string]any{
		"node_type": node.NodeType,
		"domain_id": node.DomainID,
		"read_mode": normalizeReadMode(mode),
		"app_id":    targetAppID,
	})
	return node, nil
}

func (s *Service) projectionUnreadable(appID, nodeID string, visibility map[string]struct{}) bool {
	sourceNode, ok := s.sourceStore.GetNodeByID(nodeID)
	if !ok || sourceNode.IsDeleted || !isNodeVisible(sourceNode, visibility) || !matchesAppScope(sourceNode.OwnerAppID, appID) {
		return false
	}
	graphVersion, err := s.index.ReadSyncVersion(nodeID)
	if err != nil || graphVersion <= 0 {
		return false
	}
	return graphVersion >= int64(sourceNode.DomainVersion)
}

func (s *Service) nodeForMode(actor access.Identity, appID, nodeID, mode string, visibility map[string]struct{}) (NodeResponse, bool) {
	if mode != ReadModeRealtime {
		node, ok := s.index.GetNode(actor, nodeID, visibility)
		if !ok || !matchesAppScope(node.OwnerAppID, appID) {
			return NodeResponse{}, false
		}
		return node, true
	}

	sourceNode, ok := s.sourceStore.GetNodeByID(nodeID)
	if !ok || sourceNode.IsDeleted || !isNodeVisible(sourceNode, visibility) || !matchesAppScope(sourceNode.OwnerAppID, appID) {
		return NodeResponse{}, false
	}
	graphVersion, err := s.index.ReadSyncVersion(nodeID)
	if err == nil && graphVersion == int64(sourceNode.DomainVersion) {
		if node, ok := s.index.GetNode(actor, nodeID, visibility); ok {
			if matchesAppScope(node.OwnerAppID, appID) {
				return node, true
			}
		}
	}
	telemetry.RecordRealtimeReadFallback(sourceNode.DomainID, appID)
	return sourceNodeResponse(sourceNode, s.sourceStore.ListRelationships()), true
}

func normalizeReadMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ReadModeNonRealtime:
		return ReadModeNonRealtime
	case ReadModeRealtime:
		return ReadModeRealtime
	default:
		return ReadModeNonRealtime
	}
}

func sourceNodeResponse(node write.NodeRecord, rels []write.RelationshipRecord) NodeResponse {
	relationships := make([]string, 0)
	for _, rel := range rels {
		if rel.IsDeleted {
			continue
		}
		if rel.FromNodeID == node.ID || rel.ToNodeID == node.ID {
			relationships = append(relationships, rel.ID)
		}
	}
	return NodeResponse{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		Visibility:    node.Visibility,
		SyncVersion:   int64(node.DomainVersion),
		Properties:    cloneMap(node.Properties),
		Relationships: relationships,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
	}
}

func matchesAppScope(ownerAppID, requestedAppID string) bool {
	requestedAppID = strings.TrimSpace(requestedAppID)
	if requestedAppID == "" {
		return true
	}
	return ownerAppID == requestedAppID
}

func matchesNode(node write.NodeRecord, match map[string]any, bound map[string]any) bool {
	for key, raw := range match {
		want, ok := resolveTemplateValue(raw, bound)
		if !ok {
			want = raw
		}
		got, ok := builtInNodeField(node, key)
		if !ok {
			got, ok = node.Properties[key]
			if !ok {
				return false
			}
		}
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			return false
		}
	}
	return true
}

func builtInNodeField(node write.NodeRecord, key string) (any, bool) {
	switch key {
	case "id":
		return node.ID, true
	case "node_type":
		return node.NodeType, true
	case "domain_id":
		return node.DomainID, true
	case "owner_tenant_id":
		return node.OwnerTenantID, true
	case "owner_app_id":
		return node.OwnerAppID, true
	case "external_ref":
		return node.ExternalRef, true
	default:
		return nil, false
	}
}

func bindTemplateParams(template ontology.QueryTemplate, params map[string]any) (map[string]any, error) {
	bound := map[string]any{}
	for _, schema := range template.ParamSchema {
		value, ok := params[schema.Name]
		if !ok {
			if schema.Required {
				return nil, errors.Join(ErrValidation, fmt.Errorf("%s is required", schema.Name))
			}
			continue
		}
		if err := validateParamType(schema.Type, value); err != nil {
			return nil, errors.Join(ErrValidation, fmt.Errorf("%s: %w", schema.Name, err))
		}
		bound[schema.Name] = value
	}
	return bound, nil
}

func validateParamType(expected string, value any) error {
	switch expected {
	case "string":
		if _, ok := value.(string); !ok {
			return errors.New("must be a string")
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
		default:
			return errors.New("must be a number")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("must be a boolean")
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return errors.New("must be an object")
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return errors.New("must be an array")
		}
	}
	return nil
}

func toParamSchema(in []ontology.ParameterSchema) []ParamSchemaItem {
	out := make([]ParamSchemaItem, 0, len(in))
	for _, item := range in {
		out = append(out, ParamSchemaItem{Name: item.Name, Type: item.Type, Required: item.Required})
	}
	return out
}

func isNodeVisible(node write.NodeRecord, visibility map[string]struct{}) bool {
	_, ok := visibility[nodeKey(node.OwnerTenantID, node.OwnerAppID)]
	return ok
}

func nodeKey(tenantID, appID string) string {
	return tenantID + ":" + appID
}

func resolveFieldValue(start, current write.NodeRecord, field string) any {
	if strings.Contains(field, ".") {
		parts := strings.SplitN(field, ".", 2)
		switch parts[0] {
		case start.NodeType:
			return start.Properties[parts[1]]
		case current.NodeType:
			return current.Properties[parts[1]]
		}
	}
	return current.Properties[field]
}

func resolveTemplateValue(raw any, bound map[string]any) (any, bool) {
	str, ok := raw.(string)
	if !ok || !strings.HasPrefix(str, "$") {
		return raw, false
	}
	return bound[strings.TrimPrefix(str, "$")], true
}

func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func passesLifecycle(node write.NodeRecord, statusCfg *ontology.StatusFieldConfig, _ map[string]any, _ map[string]any) bool {
	if statusCfg == nil || statusCfg.StatusFieldName == "" {
		return true
	}
	if statusCfg.ValidStatusValues == nil {
		return true
	}
	if node.StatusValue == "" {
		return true
	}
	return slices.Contains(statusCfg.ValidStatusValues, node.StatusValue)
}

func (s *Service) recordAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.RecordReadAudit(actor, action, resourceType, resourceID, outcome, reason, metadata)
}
