package read

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/write"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation")
	ErrTimeout    = errors.New("timeout")
)

type WriteProjectionStore interface {
	ListNodes() []write.NodeRecord
	ListRelationships() []write.RelationshipRecord
}

type OntologyResolver interface {
	GetVisibleDomain(actor access.Identity, domainID string) (ontology.Domain, error)
	GetQueryTemplate(domainID, templateName string) (ontology.QueryTemplate, error)
	ListQueryTemplates(domainID string) []ontology.QueryTemplate
	GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error)
}

type AccessResolver interface {
	ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error)
}

type AuditLogger interface {
	RecordReadAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any)
}

type Service struct {
	store          WriteProjectionStore
	ontology       OntologyResolver
	accessResolver AccessResolver
	auditLogger    AuditLogger
	compiler       QueryTemplateCompiler
	maxRows        int
	queryTimeout   time.Duration
	now            func() time.Time
}

func NewService(store WriteProjectionStore, ontology OntologyResolver, accessResolver AccessResolver, auditLogger AuditLogger) *Service {
	return &Service{
		store:          store,
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
	compiled, err := s.compiler.Compile(domainID, template)
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

	nodes := s.store.ListNodes()
	relationships := s.store.ListRelationships()
	nodeByID := map[string]write.NodeRecord{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}

	results := []map[string]any{}
	started := s.now()
	for _, node := range nodes {
		if s.queryTimeout > 0 && s.now().Sub(started) > s.queryTimeout {
			s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "deny", "query_timeout", nil)
			return TemplateExecutionResponse{}, ErrTimeout
		}
		if s.maxRows > 0 && len(results) >= s.maxRows {
			break
		}
		if node.IsDeleted || node.DomainID != domainID || node.NodeType != compiled.StartType {
			continue
		}
		if !isNodeVisible(node, visibility) {
			continue
		}
		if !matchesNode(node, compiled.StartMatch, bound) {
			continue
		}
		if !passesLifecycle(node, statusCfg, nil, template.PatternSpec) {
			continue
		}

		result, ok := s.walkTemplate(node, compiled, bound, nodeByID, relationships, visibility, statusCfg)
		if !ok {
			continue
		}
		results = append(results, result)
	}
	s.recordAudit(actor, "read", "query_template", domainID+"."+templateName, "allow", "", map[string]any{
		"result_count": len(results),
		"query":        compiled.Query,
	})
	return TemplateExecutionResponse{
		Results:     results,
		QueryTimeMs: 1,
	}, nil
}

func (s *Service) GetNode(actor access.Identity, nodeID string) (NodeResponse, error) {
	node, ok := findVisibleNode(s.store.ListNodes(), actor, nodeID, s.accessResolver)
	if !ok {
		s.recordAudit(actor, "read", "kg_node", nodeID, "deny", "node_not_found_or_invisible", nil)
		return NodeResponse{}, ErrNotFound
	}
	relationships := []string{}
	for _, rel := range s.store.ListRelationships() {
		if rel.FromNodeID == node.ID || rel.ToNodeID == node.ID {
			relationships = append(relationships, rel.ID)
		}
	}
	s.recordAudit(actor, "read", "kg_node", nodeID, "allow", "", map[string]any{
		"node_type": node.NodeType,
		"domain_id": node.DomainID,
	})
	return NodeResponse{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		Visibility:    node.Visibility,
		Properties:    node.Properties,
		Relationships: relationships,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
	}, nil
}

func (s *Service) walkTemplate(start write.NodeRecord, compiled CompiledTemplate, bound map[string]any, nodeByID map[string]write.NodeRecord, relationships []write.RelationshipRecord, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig) (map[string]any, bool) {
	payload := cloneMap(start.Properties)
	payload["id"] = start.ID
	payload["node_type"] = start.NodeType
	payload["domain_id"] = start.DomainID

	current := start
	for idx, hop := range compiled.Hops {
		nextNode, ok := nextVisibleHop(current, hop.RelType, hop.ToNodeType, hop.Direction, hop.Filter, bound, nodeByID, relationships, visibility, statusCfg, hop.FilterStatus)
		if !ok {
			return nil, false
		}
		current = nextNode
		payload[fmt.Sprintf("hop_%d", idx+1)] = current.ID
	}

	selected := map[string]any{}
	for _, field := range compiled.ReturnFields {
		selected[field] = resolveFieldValue(start, current, field)
	}
	for k, v := range payload {
		selected[k] = v
	}
	return selected, true
}

func nextVisibleHop(current write.NodeRecord, relType, toType, direction string, filter map[string]any, bound map[string]any, nodeByID map[string]write.NodeRecord, relationships []write.RelationshipRecord, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig, filterStatus string) (write.NodeRecord, bool) {
	for _, rel := range relationships {
		if rel.RelType != relType {
			continue
		}
		var candidateID string
		switch strings.ToLower(direction) {
		case "in":
			if rel.ToNodeID != current.ID {
				continue
			}
			candidateID = rel.FromNodeID
		default:
			if rel.FromNodeID != current.ID {
				continue
			}
			candidateID = rel.ToNodeID
		}
		candidate, ok := nodeByID[candidateID]
		if !ok || candidate.IsDeleted || candidate.NodeType != toType {
			continue
		}
		if _, ok := visibility[nodeKey(candidate.OwnerTenantID, candidate.OwnerAppID)]; !ok {
			continue
		}
		if !matchesNode(candidate, filter, bound) {
			continue
		}
		if filterStatus == "valid_only" && !passesLifecycle(candidate, statusCfg, map[string]any{}, map[string]any{}) {
			continue
		}
		return candidate, true
	}
	return write.NodeRecord{}, false
}

func matchesNode(node write.NodeRecord, match map[string]any, bound map[string]any) bool {
	for key, raw := range match {
		want, ok := resolveTemplateValue(raw, bound)
		if !ok {
			want = raw
		}
		got, ok := node.Properties[key]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			return false
		}
	}
	return true
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

func visibleOwnerSet(owners []access.VisibleOwner) map[string]struct{} {
	result := map[string]struct{}{}
	for _, owner := range owners {
		result[nodeKey(owner.TenantID, owner.AppID)] = struct{}{}
	}
	return result
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

func findVisibleNode(nodes []write.NodeRecord, actor access.Identity, nodeID string, resolver AccessResolver) (write.NodeRecord, bool) {
	owners, err := resolver.ResolveVisibleOwners(actor)
	if err != nil {
		return write.NodeRecord{}, false
	}
	visible := visibleOwnerSet(owners)
	for _, node := range nodes {
		if node.ID != nodeID || node.IsDeleted {
			continue
		}
		if _, ok := visible[nodeKey(node.OwnerTenantID, node.OwnerAppID)]; ok {
			return node, true
		}
	}
	return write.NodeRecord{}, false
}
