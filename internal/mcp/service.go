package mcp

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/integrity"
	"kg-service/internal/ontology"
	"kg-service/internal/read"
	"kg-service/internal/search"
	"kg-service/internal/write"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation")
)

type ReadService interface {
	ListTemplates(actor access.Identity, domainID string) ([]read.TemplateListItem, error)
	ExecuteTemplate(actor access.Identity, domainID, templateName string, params map[string]any) (read.TemplateExecutionResponse, error)
	GetNode(actor access.Identity, nodeID string) (read.NodeResponse, error)
}

type SearchService interface {
	SemanticSearch(actor access.Identity, req search.SemanticSearchRequest) (search.SemanticSearchResponse, error)
	RagSearch(actor access.Identity, req search.SemanticSearchRequest) (search.SemanticSearchResponse, error)
}

type WriteService interface {
	CreateNode(actor access.Identity, req write.NodeCreateRequest) (write.NodeCreateResponse, error)
	EntitySyncStatus(entityID, entityKind string) (map[string]any, error)
}

type OntologyResolver interface {
	GetEffectiveDomains(actor access.Identity, tenantID string) ([]ontology.DomainSummary, error)
	ListQueryTemplates(domainID string) []ontology.QueryTemplate
}

type AccessResolver interface {
	ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error)
}

type IntegrityService interface {
	TenantIntegrity(actor access.Identity, tenantID string) (integrity.TenantIntegrityResponse, error)
	MissingBridges(actor access.Identity, tenantID string) ([]integrity.MissingBridgeItem, error)
}

type Service struct {
	readService      ReadService
	searchService    SearchService
	writeService     WriteService
	ontologyResolver OntologyResolver
	accessResolver   AccessResolver
	integrityService IntegrityService
	mu               sync.RWMutex
	sessions         map[string]access.Identity
	now              func() time.Time
}

func NewService(readService ReadService, searchService SearchService, writeService WriteService, ontologyResolver OntologyResolver, accessResolver AccessResolver, integrityService IntegrityService) *Service {
	return &Service{
		readService:      readService,
		searchService:    searchService,
		writeService:     writeService,
		ontologyResolver: ontologyResolver,
		accessResolver:   accessResolver,
		integrityService: integrityService,
		sessions:         map[string]access.Identity{},
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) CreateSession(actor access.Identity) SessionInfo {
	sessionID := fmt.Sprintf("mcp_%d", s.now().UnixNano())
	s.mu.Lock()
	s.sessions[sessionID] = actor
	s.mu.Unlock()
	return SessionInfo{SessionID: sessionID, Actor: actor.AppID}
}

func (s *Service) IdentityForSession(sessionID string) (access.Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	actor, ok := s.sessions[sessionID]
	return actor, ok
}

func (s *Service) ListTools() []ToolSpec {
	return []ToolSpec{
		{Name: "kg_search", Description: "Semantic search over visible KG projections."},
		{Name: "kg_search_rag", Description: "RAG retrieval over visible KG projections."},
		{Name: "kg_read_pattern", Description: "Execute a registered query template."},
		{Name: "kg_list_domains", Description: "List effective visible domains."},
		{Name: "kg_list_templates", Description: "List visible active query templates for a domain."},
		{Name: "kg_get_node", Description: "Fetch a visible node and its relationships."},
		{Name: "kg_write_node", Description: "Create a node through the write path."},
		{Name: "kg_entity_sync_status", Description: "Inspect sync status for a node or relationship."},
		{Name: "kg_check_access", Description: "Inspect effective visible owners and domains."},
		{Name: "kg_integrity", Description: "Inspect tenant integrity and bridge gaps."},
	}
}

func (s *Service) CallTool(actor access.Identity, name string, args map[string]any) (any, *JSONRPCError) {
	switch name {
	case "kg_list_domains":
		return s.callListDomains(actor)
	case "kg_list_templates":
		return s.callListTemplates(actor, args)
	case "kg_read_pattern":
		return s.callReadPattern(actor, args)
	case "kg_get_node":
		return s.callGetNode(actor, args)
	case "kg_write_node":
		return s.callWriteNode(actor, args)
	case "kg_entity_sync_status":
		return s.callEntitySyncStatus(actor, args)
	case "kg_search":
		return s.callSearch(actor, args, false)
	case "kg_search_rag":
		return s.callSearch(actor, args, true)
	case "kg_check_access":
		return s.callCheckAccess(actor)
	case "kg_integrity":
		return s.callIntegrity(actor, args)
	default:
		return nil, &JSONRPCError{Code: -32601, Message: "unknown tool"}
	}
}

func (s *Service) callListDomains(actor access.Identity) (any, *JSONRPCError) {
	domains, err := s.ontologyResolver.GetEffectiveDomains(actor, actor.TenantID)
	if err != nil {
		return nil, toToolError(err)
	}
	items := make([]map[string]any, 0, len(domains))
	for _, domain := range domains {
		items = append(items, map[string]any{
			"domain_id":  domain.ID,
			"owner":      domain.Owner,
			"permission": "read",
		})
	}
	return map[string]any{"domains": items}, nil
}

func (s *Service) callListTemplates(actor access.Identity, args map[string]any) (any, *JSONRPCError) {
	domainID, paramErr := requireString(args, "domain_id")
	if paramErr != nil {
		return nil, paramErr
	}
	templates, callErr := s.readService.ListTemplates(actor, domainID)
	if callErr != nil {
		return nil, toToolError(callErr)
	}
	return map[string]any{"templates": templates}, nil
}

func (s *Service) callReadPattern(actor access.Identity, args map[string]any) (any, *JSONRPCError) {
	domainID, paramErr := requireString(args, "domain_id")
	if paramErr != nil {
		return nil, paramErr
	}
	templateName, paramErr := requireString(args, "template_name")
	if paramErr != nil {
		return nil, paramErr
	}
	params, _ := args["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	result, callErr := s.readService.ExecuteTemplate(actor, domainID, templateName, params)
	if callErr != nil {
		return nil, toToolError(callErr)
	}
	return result, nil
}

func (s *Service) callGetNode(actor access.Identity, args map[string]any) (any, *JSONRPCError) {
	nodeID, err := requireString(args, "id")
	if err != nil {
		return nil, err
	}
	node, callErr := s.readService.GetNode(actor, nodeID)
	if callErr != nil {
		return nil, toToolError(callErr)
	}
	return map[string]any{
		"node":          node,
		"relationships": node.Relationships,
	}, nil
}

func (s *Service) callWriteNode(actor access.Identity, args map[string]any) (any, *JSONRPCError) {
	req := write.NodeCreateRequest{}
	if domainID, ok := args["domain_id"].(string); ok {
		req.DomainID = domainID
	}
	if nodeType, ok := args["node_type"].(string); ok {
		req.NodeType = nodeType
	}
	if visibility, ok := args["visibility"].(string); ok {
		req.Visibility = visibility
	}
	if externalRef, ok := args["external_ref"].(string); ok {
		req.ExternalRef = externalRef
	}
	if props, ok := args["properties"].(map[string]any); ok {
		req.Properties = props
	}
	result, callErr := s.writeService.CreateNode(actor, req)
	if callErr != nil {
		return nil, toToolError(callErr)
	}
	return result, nil
}

func (s *Service) callEntitySyncStatus(actor access.Identity, args map[string]any) (any, *JSONRPCError) {
	entityID, err := requireString(args, "entity_id")
	if err != nil {
		return nil, err
	}
	entityKind, err := requireString(args, "entity_kind")
	if err != nil {
		return nil, err
	}
	status, callErr := s.writeService.EntitySyncStatus(entityID, entityKind)
	if callErr != nil {
		return nil, toToolError(callErr)
	}
	_ = actor
	return status, nil
}

func (s *Service) callSearch(actor access.Identity, args map[string]any, rag bool) (any, *JSONRPCError) {
	query, err := requireString(args, "query")
	if err != nil {
		return nil, err
	}
	topK := 10
	if raw, ok := args["top_k"].(float64); ok {
		topK = int(raw)
	}
	req := search.SemanticSearchRequest{Query: query, TopK: topK}
	if raw, ok := args["domain_ids"].([]any); ok {
		req.DomainIDs = make([]string, 0, len(raw))
		for _, item := range raw {
			if str, ok := item.(string); ok {
				req.DomainIDs = append(req.DomainIDs, str)
			}
		}
	}
	if rag {
		result, callErr := s.searchService.RagSearch(actor, req)
		if callErr != nil {
			return nil, toToolError(callErr)
		}
		return result, nil
	}
	result, callErr := s.searchService.SemanticSearch(actor, req)
	if callErr != nil {
		return nil, toToolError(callErr)
	}
	return result, nil
}

func (s *Service) callCheckAccess(actor access.Identity) (any, *JSONRPCError) {
	visibleOwners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return nil, toToolError(err)
	}
	domains, err := s.ontologyResolver.GetEffectiveDomains(actor, actor.TenantID)
	if err != nil {
		return nil, toToolError(err)
	}
	return map[string]any{
		"visible_owners":  visibleOwners,
		"visible_domains": domains,
	}, nil
}

func (s *Service) callIntegrity(actor access.Identity, args map[string]any) (any, *JSONRPCError) {
	tenantID := actor.TenantID
	if raw, ok := args["tenant_id"].(string); ok && raw != "" {
		tenantID = raw
	}
	summary, err := s.integrityService.TenantIntegrity(actor, tenantID)
	if err != nil {
		return nil, toToolError(err)
	}
	missing, err := s.integrityService.MissingBridges(actor, tenantID)
	if err != nil {
		return nil, toToolError(err)
	}
	return map[string]any{
		"summary":         summary,
		"missing_bridges": missing,
	}, nil
}

func requireString(args map[string]any, key string) (string, *JSONRPCError) {
	raw, ok := args[key]
	if !ok {
		return "", &JSONRPCError{Code: -32602, Message: "missing parameter", Data: map[string]any{"param": key}}
	}
	value, ok := raw.(string)
	if !ok || value == "" {
		return "", &JSONRPCError{Code: -32602, Message: "invalid parameter", Data: map[string]any{"param": key}}
	}
	return value, nil
}

func toToolError(err error) *JSONRPCError {
	switch {
	case errors.Is(err, ErrValidation), errors.Is(err, read.ErrValidation), errors.Is(err, search.ErrValidation), errors.Is(err, integrity.ErrValidation), errors.Is(err, write.ErrValidation):
		return &JSONRPCError{Code: -32602, Message: "validation failed"}
	case errors.Is(err, ErrForbidden), errors.Is(err, read.ErrForbidden), errors.Is(err, search.ErrForbidden), errors.Is(err, integrity.ErrForbidden), errors.Is(err, write.ErrForbidden):
		return &JSONRPCError{Code: -32603, Message: "forbidden"}
	case errors.Is(err, ErrNotFound), errors.Is(err, read.ErrNotFound), errors.Is(err, search.ErrNotFound), errors.Is(err, integrity.ErrNotFound), errors.Is(err, write.ErrNotFound):
		return &JSONRPCError{Code: -32603, Message: "not found"}
	default:
		return &JSONRPCError{Code: -32603, Message: "internal error"}
	}
}
