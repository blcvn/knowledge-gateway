package search

import (
	"errors"
	"fmt"
	"strings"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/vector"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation")
)

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
	index          VectorIndex
	ontology       OntologyResolver
	accessResolver AccessResolver
	auditLogger    AuditLogger
}

func NewService(store ProjectionStore, ontology OntologyResolver, accessResolver AccessResolver, auditLogger AuditLogger) *Service {
	return &Service{
		index:          NewProjectionVectorIndex(store, ontology, vector.NewDeterministicProvider(8)),
		ontology:       ontology,
		accessResolver: accessResolver,
		auditLogger:    auditLogger,
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

	results, err := s.index.Search(actor, req, visibility, domainSet, statusConfigs, allHaveStatus)
	if err != nil {
		return SemanticSearchResponse{}, err
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

	results, err := s.index.RagSearch(actor, req, visibility, domainSet, statusConfigs, allHaveStatus)
	if err != nil {
		return SemanticSearchResponse{}, err
	}
	s.recordAudit(actor, "search", "rag_search", strings.Join(req.DomainIDs, ","), "allow", "", map[string]any{
		"result_count": len(results),
	})
	return SemanticSearchResponse{
		Results:      results,
		SearchTimeMs: 1,
	}, nil
}

func (s *Service) recordAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.RecordReadAudit(actor, action, resourceType, resourceID, outcome, reason, metadata)
}
