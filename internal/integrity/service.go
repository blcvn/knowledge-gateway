package integrity

import (
	"errors"
	"slices"
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
	ListRelationships() []write.RelationshipRecord
}

type OntologyResolver interface {
	ListCrossDomainRules(domainID string) []ontology.CrossDomainRelRule
}

type Service struct {
	store    ProjectionStore
	ontology OntologyResolver
	now      func() time.Time
}

func NewService(store ProjectionStore, ontology OntologyResolver) *Service {
	return &Service{
		store:    store,
		ontology: ontology,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) TenantIntegrity(actor access.Identity, tenantID string) (TenantIntegrityResponse, error) {
	if tenantID == "" {
		return TenantIntegrityResponse{}, ErrValidation
	}
	if !canInspectTenant(actor, tenantID) {
		return TenantIntegrityResponse{}, ErrForbidden
	}

	nodes := s.tenantNodes(tenantID)
	missingDomainID := 0
	missingBridges := 0
	for _, node := range nodes {
		if node.DomainID == "" {
			missingDomainID++
		}
		if len(s.missingBridgeItemsForNode(node)) > 0 {
			missingBridges++
		}
	}

	checks := []CheckResult{
		{
			ID:     "IC-04",
			Name:   "TyLeThue thiếu bridge",
			Result: missingBridges,
			Status: statusForCount(missingBridges),
		},
		{
			ID:     "IC-12",
			Name:   "Node thiếu domain_id",
			Result: missingDomainID,
			Status: statusForCount(missingDomainID),
		},
	}

	return TenantIntegrityResponse{
		Checks:  checks,
		Overall: overallStatus(checks),
	}, nil
}

func (s *Service) MissingBridges(actor access.Identity, tenantID string) ([]MissingBridgeItem, error) {
	if tenantID == "" {
		return nil, ErrValidation
	}
	if !canInspectTenant(actor, tenantID) {
		return nil, ErrForbidden
	}
	var result []MissingBridgeItem
	for _, node := range s.tenantNodes(tenantID) {
		result = append(result, s.missingBridgeItemsForNode(node)...)
	}
	return result, nil
}

func (s *Service) tenantNodes(tenantID string) []write.NodeRecord {
	var result []write.NodeRecord
	for _, node := range s.store.ListNodes() {
		if node.OwnerTenantID != tenantID {
			continue
		}
		result = append(result, node)
	}
	return result
}

func (s *Service) missingBridgeItemsForNode(node write.NodeRecord) []MissingBridgeItem {
	if node.IsDeleted || node.DomainID == "" {
		return nil
	}

	rules := s.ontology.ListCrossDomainRules(node.DomainID)
	if len(rules) == 0 {
		return nil
	}

	relationships := s.store.ListRelationships()
	var result []MissingBridgeItem
	for _, rule := range rules {
		if !rule.Required || !slices.Contains(rule.FromNodeTypes, node.NodeType) {
			continue
		}
		if hasBridgeRelationship(node.ID, rule.RelTypeName, relationships) {
			continue
		}
		result = append(result, MissingBridgeItem{
			NodeID:   node.ID,
			NodeType: node.NodeType,
			DomainID: node.DomainID,
		})
	}
	return result
}

func hasBridgeRelationship(nodeID, relType string, relationships []write.RelationshipRecord) bool {
	for _, rel := range relationships {
		if rel.RelType == relType && rel.FromNodeID == nodeID {
			return true
		}
	}
	return false
}

func statusForCount(count int) string {
	if count == 0 {
		return "pass"
	}
	return "fail"
}

func overallStatus(checks []CheckResult) string {
	for _, check := range checks {
		if check.Status != "pass" {
			return "fail"
		}
	}
	return "pass"
}

func canInspectTenant(actor access.Identity, tenantID string) bool {
	if actor.TenantID == access.PlatformTenantID && (actor.AppType == "admin_tool" || actor.AppType == "hybrid") {
		return true
	}
	return actor.TenantID == tenantID && (actor.AppType == "admin_tool" || actor.AppType == "hybrid")
}
