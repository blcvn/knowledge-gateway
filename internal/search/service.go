package search

import (
	"errors"
	"fmt"
	"math"
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
)

type ProjectionStore interface {
	ListNodes() []write.NodeRecord
}

type OntologyResolver interface {
	GetVisibleDomain(actor access.Identity, domainID string) (ontology.Domain, error)
	GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error)
}

type AccessResolver interface {
	ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error)
}

type AuditLogger interface {
	RecordReadAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any)
}

type Service struct {
	store          ProjectionStore
	ontology       OntologyResolver
	accessResolver AccessResolver
	auditLogger    AuditLogger
	now            func() time.Time
}

func NewService(store ProjectionStore, ontology OntologyResolver, accessResolver AccessResolver, auditLogger AuditLogger) *Service {
	return &Service{
		store:          store,
		ontology:       ontology,
		accessResolver: accessResolver,
		auditLogger:    auditLogger,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) SemanticSearch(actor access.Identity, req SemanticSearchRequest) (SemanticSearchResponse, error) {
	if strings.TrimSpace(req.Query) == "" {
		return SemanticSearchResponse{}, errors.Join(ErrValidation, errors.New("query is required"))
	}
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if len(req.DomainIDs) == 0 {
		req.DomainIDs = nil
	}
	for _, domainID := range req.DomainIDs {
		if strings.TrimSpace(domainID) == "" {
			return SemanticSearchResponse{}, errors.Join(ErrValidation, errors.New("domain_ids must be non-empty strings"))
		}
	}

	visibleOwners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return SemanticSearchResponse{}, err
	}
	visibility := visibleOwnerSet(visibleOwners)
	domainSet := map[string]struct{}{}
	for _, domainID := range req.DomainIDs {
		if _, err := s.ontology.GetVisibleDomain(actor, domainID); err != nil {
			return SemanticSearchResponse{}, errors.Join(ErrValidation, fmt.Errorf("invalid domain_id: %s", domainID))
		}
		domainSet[domainID] = struct{}{}
	}

	statusConfigs := map[string]*ontology.StatusFieldConfig{}
	allHaveStatus := true
	for _, domainID := range req.DomainIDs {
		cfg, err := s.ontology.GetStatusFieldConfig(domainID)
		if err != nil {
			return SemanticSearchResponse{}, err
		}
		if cfg == nil || cfg.StatusFieldName == "" {
			allHaveStatus = false
		}
		statusConfigs[domainID] = cfg
	}

	results := make([]SearchResult, 0)
	for _, node := range s.store.ListNodes() {
		if node.IsDeleted {
			continue
		}
		if len(domainSet) > 0 {
			if _, ok := domainSet[node.DomainID]; !ok {
				continue
			}
		} else {
			if _, err := s.ontology.GetVisibleDomain(actor, node.DomainID); err != nil {
				continue
			}
		}
		if _, ok := visibility[nodeKey(node.OwnerTenantID, node.OwnerAppID)]; !ok {
			continue
		}
		if !matchesQuery(node, req.Query) {
			continue
		}
		if len(domainSet) > 0 && allHaveStatus {
			cfg := statusConfigs[node.DomainID]
			if cfg != nil && node.StatusValue != "" && !slices.Contains(cfg.ValidStatusValues, node.StatusValue) {
				continue
			}
		}

		results = append(results, SearchResult{
			NodeID:         node.ID,
			NodeType:       node.NodeType,
			DomainID:       node.DomainID,
			OwnerTenantID:  node.OwnerTenantID,
			OwnerAppID:     node.OwnerAppID,
			ACLVisibleTo:   nodeACLVisibleTo(node),
			IsDeleted:      node.IsDeleted,
			StatusValue:    node.StatusValue,
			AuthorityScore: authorityScoreFor(node, statusConfigs[node.DomainID]),
			DomainProps:    node.Properties,
			Content:        nodeContent(node),
			Score:          scoreNode(node, req.Query),
			CreatedAt:      node.CreatedAt,
		})
	}

	slices.SortFunc(results, func(a, b SearchResult) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		if a.NodeID < b.NodeID {
			return -1
		}
		if a.NodeID > b.NodeID {
			return 1
		}
		return 0
	})

	if len(results) > req.TopK {
		results = results[:req.TopK]
	}
	s.recordAudit(actor, "search", "semantic_search", strings.Join(req.DomainIDs, ","), "allow", "", map[string]any{
		"result_count": len(results),
	})
	return SemanticSearchResponse{
		Results:      results,
		SearchTimeMs: 1,
	}, nil
}

func (s *Service) RagSearch(actor access.Identity, req SemanticSearchRequest) (SemanticSearchResponse, error) {
	return s.SemanticSearch(actor, req)
}

func scoreNode(node write.NodeRecord, query string) float64 {
	content := strings.ToLower(nodeContent(node))
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0
	}
	score := 0.1
	if strings.Contains(content, q) {
		score += 0.8
	}
	for _, token := range strings.Fields(q) {
		if strings.Contains(content, token) {
			score += 0.05
		}
	}
	return math.Min(score, 1.0)
}

func matchesQuery(node write.NodeRecord, query string) bool {
	return strings.Contains(strings.ToLower(nodeContent(node)), strings.ToLower(query))
}

func nodeContent(node write.NodeRecord) string {
	parts := []string{node.ID, node.NodeType, node.DomainID, node.ExternalRef, node.StatusValue}
	for k, v := range node.Properties {
		parts = append(parts, k, fmt.Sprintf("%v", v))
	}
	return strings.Join(parts, " ")
}

func authorityScoreFor(node write.NodeRecord, cfg *ontology.StatusFieldConfig) *int {
	if cfg == nil || cfg.AuthorityFieldName == "" || len(cfg.AuthorityValuesMap) == 0 {
		return nil
	}
	if node.Properties == nil {
		return nil
	}
	value, ok := node.Properties[cfg.AuthorityFieldName]
	if !ok {
		return nil
	}
	score, ok := cfg.AuthorityValuesMap[fmt.Sprintf("%v", value)]
	if !ok {
		return nil
	}
	return &score
}

func visibleOwnerSet(owners []access.VisibleOwner) map[string]struct{} {
	result := map[string]struct{}{}
	for _, owner := range owners {
		result[nodeKey(owner.TenantID, owner.AppID)] = struct{}{}
	}
	return result
}

func nodeACLVisibleTo(node write.NodeRecord) []string {
	if len(node.ACLVisibleTo) > 0 {
		return append([]string(nil), node.ACLVisibleTo...)
	}
	return []string{nodeKey(node.OwnerTenantID, node.OwnerAppID)}
}

func nodeKey(tenantID, appID string) string {
	return tenantID + ":" + appID
}

func (s *Service) recordAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.RecordReadAudit(actor, action, resourceType, resourceID, outcome, reason, metadata)
}
