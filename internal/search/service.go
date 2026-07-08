package search

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
	"kg-service/internal/write"
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
	store          ProjectionStore
	index          VectorIndex
	ontology       OntologyResolver
	accessResolver AccessResolver
	auditLogger    AuditLogger
	embedding      vector.EmbeddingRouter
	vectorAdapter  vectorstore.VectorAdapter
	ftsAdapter     fts.FTSAdapter
	profiles       ontology.SearchProfileResolver
}

func NewService(store ProjectionStore, ontologyResolver OntologyResolver, accessResolver AccessResolver, auditLogger AuditLogger, profileResolver ...ontology.SearchProfileResolver) *Service {
	provider := vector.NewDeterministicProvider(8)
	router := vector.DirectRouter{Provider: provider}
	var resolver ontology.SearchProfileResolver
	if len(profileResolver) > 0 {
		resolver = profileResolver[0]
	}
	if resolver == nil {
		if candidate, ok := ontologyResolver.(ontology.SearchProfileResolver); ok {
			resolver = candidate
		}
	}
	return &Service{
		store:          store,
		index:          NewProjectionVectorIndex(store, ontologyResolver, router),
		ontology:       ontologyResolver,
		accessResolver: accessResolver,
		auditLogger:    auditLogger,
		embedding:      router,
		vectorAdapter:  nil,
		ftsAdapter:     fts.NewInMemoryFTSAdapter(),
		profiles:       resolver,
	}
}

func (s *Service) SetEmbeddingRouter(router vector.EmbeddingRouter) {
	if router == nil {
		return
	}
	s.embedding = router
	s.index = NewProjectionVectorIndex(s.store, s.ontology, router)
}

func (s *Service) SetFTSAdapter(adapter fts.FTSAdapter) {
	if adapter == nil {
		return
	}
	s.ftsAdapter = adapter
}

func (s *Service) SetVectorAdapter(adapter vectorstore.VectorAdapter) {
	if adapter == nil {
		return
	}
	s.vectorAdapter = adapter
}

func (s *Service) SetSearchProfileResolver(resolver ontology.SearchProfileResolver) {
	if resolver == nil {
		return
	}
	s.profiles = resolver
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
	scope, err := s.prepareSearchScope(actor, req.DomainIDs)
	if err != nil {
		return SemanticSearchResponse{}, err
	}

	results, err := s.searchWithScope(actor, req, scope, false)
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
	scope, err := s.prepareSearchScope(actor, req.DomainIDs)
	if err != nil {
		return SemanticSearchResponse{}, err
	}

	results, err := s.searchWithScope(actor, req, scope, true)
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

func (s *Service) FullTextSearch(actor access.Identity, req FullTextSearchRequest) (FullTextSearchResponse, error) {
	if strings.TrimSpace(req.Query) == "" {
		return FullTextSearchResponse{}, errors.Join(ErrValidation, errors.New("query is required"))
	}
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if len(req.DomainIDs) == 0 {
		req.DomainIDs = nil
	}
	scope, err := s.prepareSearchScope(actor, req.DomainIDs)
	if err != nil {
		return FullTextSearchResponse{}, err
	}
	results, err := s.fullTextSearchWithScope(req, scope)
	if err != nil {
		return FullTextSearchResponse{}, err
	}
	searchResults := materializeFTSResults(results, scope, s.sourceNodesByIDs(ftsResultNodeIDs(results)))
	if len(searchResults) > req.TopK {
		searchResults = searchResults[:req.TopK]
	}
	s.recordAudit(actor, "search", "full_text_search", strings.Join(req.DomainIDs, ","), "allow", "", map[string]any{
		"result_count": len(searchResults),
	})
	return FullTextSearchResponse{
		Results:      searchResults,
		SearchTimeMs: 1,
	}, nil
}

func (s *Service) HybridSearch(actor access.Identity, req HybridSearchRequest) (HybridSearchResponse, error) {
	if strings.TrimSpace(req.Query) == "" {
		return HybridSearchResponse{}, errors.Join(ErrValidation, errors.New("query is required"))
	}
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if len(req.DomainIDs) == 0 {
		req.DomainIDs = nil
	}
	if req.SemanticWeight < 0 || req.SemanticWeight > 1 {
		return HybridSearchResponse{}, errors.Join(ErrValidation, errors.New("semantic_weight must be between 0 and 1"))
	}
	if req.FTSOperator == "" {
		req.FTSOperator = "all_tokens"
	}
	scope, err := s.prepareSearchScope(actor, req.DomainIDs)
	if err != nil {
		return HybridSearchResponse{}, err
	}

	semanticReq := SemanticSearchRequest{
		Query:     req.Query,
		DomainIDs: req.DomainIDs,
		TopK:      req.TopK,
	}
	ftsReq := FullTextSearchRequest{
		Query:     req.Query,
		DomainIDs: req.DomainIDs,
		TopK:      req.TopK,
		Mode:      req.FTSOperator,
	}

	var (
		semanticResults []SearchResult
		ftsResults      []fts.FTSResult
		semanticErr     error
		ftsErr          error
	)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		semanticResults, semanticErr = s.searchWithScope(actor, semanticReq, scope, false)
	}()
	go func() {
		defer wg.Done()
		ftsResults, ftsErr = s.fullTextSearchWithScope(ftsReq, scope)
	}()
	wg.Wait()
	if semanticErr != nil {
		return HybridSearchResponse{}, semanticErr
	}
	if ftsErr != nil {
		return HybridSearchResponse{}, ftsErr
	}

	results := fuseHybridResults(semanticResults, materializeFTSResults(ftsResults, scope, s.sourceNodesByIDs(ftsResultNodeIDs(ftsResults))), req.SemanticWeight, req.TopK)
	s.recordAudit(actor, "search", "hybrid_search", strings.Join(req.DomainIDs, ","), "allow", "", map[string]any{
		"result_count": len(results),
	})
	return HybridSearchResponse{
		Results:      results,
		SearchTimeMs: 1,
	}, nil
}

func (s *Service) searchWithScope(actor access.Identity, req SemanticSearchRequest, scope searchScope, rag bool) ([]SearchResult, error) {
	if s.vectorAdapter != nil {
		return s.searchWithVectorAdapter(actor, req, scope, rag)
	}
	return s.searchWithProjectionIndex(actor, req, scope, rag)
}

func (s *Service) searchWithProjectionIndex(actor access.Identity, req SemanticSearchRequest, scope searchScope, rag bool) ([]SearchResult, error) {
	if rag {
		return s.index.RagSearch(actor, req, scope.visibility, scope.domainSet, scope.statusConfigs, scope.allHaveStatus)
	}
	return s.index.Search(actor, req, scope.visibility, scope.domainSet, scope.statusConfigs, scope.allHaveStatus)
}

func (s *Service) searchWithVectorAdapter(actor access.Identity, req SemanticSearchRequest, scope searchScope, rag bool) ([]SearchResult, error) {
	resolved, err := s.resolveProfileForSearch(actor, req.DomainIDs)
	if err != nil {
		return nil, err
	}
	queryEmbedding, err := s.embeddingForSearch(actor, req.DomainIDs, req.Query)
	if err != nil {
		return nil, err
	}
	results, err := s.vectorAdapter.ANN(context.Background(), queryEmbedding, vectorstore.VectorFilter{
		DomainIDs:      req.DomainIDs,
		ACLVisibleTo:   scope.aclTokens,
		OwnerTenantIDs: scope.ownerTenantIDs,
		OwnerAppIDs:    scope.ownerAppIDs,
	}, vectorstore.ANNOptions{
		TopK:       req.TopK,
		FilterMode: vectorFilterMode(resolved),
		IndexHint:  vectorIndexHint(resolved),
		EfSearch:   vectorEfSearch(resolved),
	})
	if err != nil {
		return nil, err
	}
	sourceIDs := make([]string, 0, len(results))
	seenIDs := map[string]struct{}{}
	for _, result := range results {
		if result.Document.NodeID == "" {
			continue
		}
		if _, ok := seenIDs[result.Document.NodeID]; ok {
			continue
		}
		seenIDs[result.Document.NodeID] = struct{}{}
		sourceIDs = append(sourceIDs, result.Document.NodeID)
	}
	sourceNodes := s.sourceNodesByIDs(sourceIDs)
	searchResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		doc := result.Document
		if node, ok := sourceNodes[doc.NodeID]; ok {
			doc = hydrateVectorDocumentFromSource(doc, node)
		}
		cfg := scope.statusConfigs[doc.DomainID]
		if len(scope.domainSet) > 0 {
			if _, ok := scope.domainSet[doc.DomainID]; !ok {
				continue
			}
		}
		// Check visibility: accept if actor has exact tenant:app access OR tenant-wide access (app_id="")
		_, exactVisible := scope.visibility[nodeKey(doc.OwnerTenantID, doc.OwnerAppID)]
		_, tenantWide := scope.visibility[nodeKey(doc.OwnerTenantID, "")]
		if !exactVisible && !tenantWide {
			continue
		}
		if scope.allHaveStatus && cfg != nil && doc.StatusValue != "" && !slices.Contains(cfg.ValidStatusValues, doc.StatusValue) {
			continue
		}
		content := ""
		if rag {
			content = ftsDocumentContent(fts.FTSDocument{
				NodeID:         doc.NodeID,
				NodeType:       doc.NodeType,
				DomainID:       doc.DomainID,
				OwnerTenantID:  doc.OwnerTenantID,
				OwnerAppID:     doc.OwnerAppID,
				ACLVisibleTo:   doc.ACLVisibleTo,
				IsDeleted:      doc.IsDeleted,
				StatusValue:    doc.StatusValue,
				AuthorityScore: doc.AuthorityScore,
				DomainProps:    doc.DomainProps,
				CreatedAt:      time.Time{},
			})
		} else {
			if node, ok := sourceNodes[doc.NodeID]; ok {
				content = nodeContent(node)
			} else {
				content = vectorDocumentContent(doc)
			}
		}
		searchResults = append(searchResults, SearchResult{
			NodeID:         doc.NodeID,
			NodeType:       doc.NodeType,
			DomainID:       doc.DomainID,
			OwnerTenantID:  doc.OwnerTenantID,
			OwnerAppID:     doc.OwnerAppID,
			ACLVisibleTo:   append([]string(nil), doc.ACLVisibleTo...),
			IsDeleted:      doc.IsDeleted,
			StatusValue:    doc.StatusValue,
			AuthorityScore: doc.AuthorityScore,
			DomainProps:    doc.DomainProps,
			Content:        content,
			Score:          result.Score,
			CreatedAt:      time.Time{},
		})
	}
	return sortSearchResults(searchResults, req.TopK), nil
}

func (s *Service) fullTextSearchWithScope(req FullTextSearchRequest, scope searchScope) ([]fts.FTSResult, error) {
	adapter := s.ftsAdapter
	if adapter == nil {
		adapter = fts.NewInMemoryFTSAdapter()
	}
	return adapter.Search(context.Background(), fts.FTSQuery{
		Text:   req.Query,
		Mode:   req.Mode,
		Fields: req.Fields,
	}, fts.FTSFilter{
		DomainIDs:      req.DomainIDs,
		ACLVisibleTo:   scope.aclTokens,
		NodeTypes:      nil,
		OwnerTenantIDs: scope.ownerTenantIDs,
		OwnerAppIDs:    scope.ownerAppIDs,
	}, fts.SearchOptions{TopK: req.TopK})
}

func materializeFTSResults(results []fts.FTSResult, scope searchScope, sourceNodes map[string]write.NodeRecord) []SearchResult {
	searchResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		doc := result.Document
		cfg := scope.statusConfigs[doc.DomainID]
		if len(scope.domainSet) > 0 {
			if _, ok := scope.domainSet[doc.DomainID]; !ok {
				continue
			}
		}
		if _, ok := scope.visibility[nodeKey(doc.OwnerTenantID, doc.OwnerAppID)]; !ok {
			continue
		}
		if scope.allHaveStatus && cfg != nil && doc.StatusValue != "" && !slices.Contains(cfg.ValidStatusValues, doc.StatusValue) {
			continue
		}
		if node, ok := sourceNodes[doc.NodeID]; ok {
			doc = hydrateFTSDocumentFromSource(doc, node)
		}
		searchResults = append(searchResults, SearchResult{
			NodeID:         doc.NodeID,
			NodeType:       doc.NodeType,
			DomainID:       doc.DomainID,
			OwnerTenantID:  doc.OwnerTenantID,
			OwnerAppID:     doc.OwnerAppID,
			ACLVisibleTo:   append([]string(nil), doc.ACLVisibleTo...),
			IsDeleted:      doc.IsDeleted,
			StatusValue:    doc.StatusValue,
			AuthorityScore: authorityScoreForFTS(doc, cfg),
			DomainProps:    doc.DomainProps,
			Content:        ftsDocumentContent(doc),
			Score:          result.Score,
			CreatedAt:      doc.CreatedAt,
		})
	}
	slices.SortFunc(searchResults, func(a, b SearchResult) int {
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
	return searchResults
}

func fuseHybridResults(semanticResults, ftsResults []SearchResult, semanticWeight float64, topK int) []SearchResult {
	if topK <= 0 {
		topK = 10
	}
	if semanticWeight < 0 {
		semanticWeight = 0
	}
	if semanticWeight > 1 {
		semanticWeight = 1
	}
	ftsWeight := 1 - semanticWeight
	type fusedResult struct {
		result SearchResult
		score  float64
	}
	acc := map[string]*fusedResult{}
	const rrfK = 60.0
	merge := func(results []SearchResult, weight float64) {
		for i, result := range results {
			entry, ok := acc[result.NodeID]
			if !ok {
				copyResult := result
				entry = &fusedResult{result: copyResult}
				acc[result.NodeID] = entry
			}
			entry.score += weight / (rrfK + float64(i+1))
			if entry.result.NodeID == "" || result.Score > entry.result.Score {
				entry.result = result
			}
		}
	}
	merge(semanticResults, semanticWeight)
	merge(ftsResults, ftsWeight)

	results := make([]SearchResult, 0, len(acc))
	for _, entry := range acc {
		entry.result.Score = entry.score
		results = append(results, entry.result)
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
	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

func (s *Service) recordAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.RecordReadAudit(actor, action, resourceType, resourceID, outcome, reason, metadata)
}

type searchScope struct {
	visibility     map[string]struct{}
	aclTokens      []string
	ownerTenantIDs []string
	ownerAppIDs    []string
	domainSet      map[string]struct{}
	statusConfigs  map[string]*ontology.StatusFieldConfig
	allHaveStatus  bool
}

func (s *Service) prepareSearchScope(actor access.Identity, domainIDs []string) (searchScope, error) {
	for _, domainID := range domainIDs {
		if strings.TrimSpace(domainID) == "" {
			return searchScope{}, errors.Join(ErrValidation, errors.New("domain_ids must be non-empty strings"))
		}
	}

	visibleOwners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return searchScope{}, err
	}
	scope := searchScope{
		visibility:     visibleOwnerSet(visibleOwners),
		aclTokens:      make([]string, 0, len(visibleOwners)),
		ownerTenantIDs: make([]string, 0, len(visibleOwners)),
		ownerAppIDs:    make([]string, 0, len(visibleOwners)),
		domainSet:      map[string]struct{}{},
		statusConfigs:  map[string]*ontology.StatusFieldConfig{},
		allHaveStatus:  true,
	}
	// Identify tenants with wildcard app access (app_id == "" means all apps in that tenant)
	wildcardTenants := map[string]bool{}
	for _, owner := range visibleOwners {
		if strings.TrimSpace(owner.AppID) == "" {
			wildcardTenants[owner.TenantID] = true
		}
	}

	for _, owner := range visibleOwners {
		// For wildcard tenants (app_id == ""), the owner_tenant_id SQL filter is sufficient.
		// Adding acl tokens with empty app_id would never match node-level ACL entries (which
		// always contain a specific app_id), causing the acl_visible_to overlap check to fail.
		if !wildcardTenants[owner.TenantID] {
			scope.aclTokens = append(scope.aclTokens, nodeKey(owner.TenantID, owner.AppID))
		}
		scope.ownerTenantIDs = append(scope.ownerTenantIDs, owner.TenantID)
		// Only restrict by app_id when the tenant does NOT have wildcard access
		if strings.TrimSpace(owner.AppID) != "" && !wildcardTenants[owner.TenantID] {
			scope.ownerAppIDs = append(scope.ownerAppIDs, owner.AppID)
		}
	}
	for _, domainID := range domainIDs {
		if _, err := s.ontology.GetVisibleDomain(actor, domainID); err != nil {
			return searchScope{}, errors.Join(ErrValidation, fmt.Errorf("invalid domain_id: %s", domainID))
		}
		scope.domainSet[domainID] = struct{}{}
		if s.profiles != nil {
			if _, err := s.profiles.Resolve(domainID, actor.TenantID, actor.AppID); err != nil && !errors.Is(err, ontology.ErrNotFound) {
				return searchScope{}, err
			}
		}
	}
	for _, domainID := range domainIDs {
		cfg, err := s.ontology.GetStatusFieldConfig(domainID)
		if err != nil {
			return searchScope{}, err
		}
		if cfg == nil || cfg.StatusFieldName == "" {
			scope.allHaveStatus = false
		}
		scope.statusConfigs[domainID] = cfg
	}
	return scope, nil
}

func ftsDocumentContent(doc fts.FTSDocument) string {
	parts := make([]string, 0, len(doc.Fields)+len(doc.DomainProps))
	fieldKeys := make([]string, 0, len(doc.Fields))
	for key := range doc.Fields {
		fieldKeys = append(fieldKeys, key)
	}
	slices.Sort(fieldKeys)
	for _, key := range fieldKeys {
		if value := strings.TrimSpace(doc.Fields[key]); value != "" {
			parts = append(parts, value)
		}
	}
	propKeys := make([]string, 0, len(doc.DomainProps))
	for key := range doc.DomainProps {
		propKeys = append(propKeys, key)
	}
	slices.Sort(propKeys)
	for _, key := range propKeys {
		if value := strings.TrimSpace(fmt.Sprint(doc.DomainProps[key])); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func vectorDocumentContent(doc vectorstore.VectorDocument) string {
	parts := make([]string, 0, len(doc.DomainProps)+5)
	parts = append(parts, doc.NodeID, doc.NodeType, doc.DomainID, doc.OwnerTenantID, doc.OwnerAppID)
	propKeys := make([]string, 0, len(doc.DomainProps))
	for key := range doc.DomainProps {
		propKeys = append(propKeys, key)
	}
	slices.Sort(propKeys)
	for _, key := range propKeys {
		if value := strings.TrimSpace(fmt.Sprint(doc.DomainProps[key])); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func sortSearchResults(results []SearchResult, topK int) []SearchResult {
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
	if topK > 0 && len(results) > topK {
		return results[:topK]
	}
	return results
}

func (s *Service) resolveProfileForSearch(actor access.Identity, domainIDs []string) (ontology.ResolvedSearchProfile, error) {
	if s.profiles == nil {
		return ontology.ResolvedSearchProfile{}, nil
	}
	for _, domainID := range domainIDs {
		if strings.TrimSpace(domainID) == "" {
			continue
		}
		resolved, err := s.profiles.Resolve(domainID, actor.TenantID, actor.AppID)
		if err != nil {
			if errors.Is(err, ontology.ErrNotFound) {
				// No search profile configured — use system defaults
				return ontology.ResolvedSearchProfile{}, nil
			}
			return ontology.ResolvedSearchProfile{}, err
		}
		return resolved, nil
	}
	return ontology.ResolvedSearchProfile{}, nil
}

type sourceNodeBatchReader interface {
	GetNodesByIDs(ids []string) map[string]write.NodeRecord
}

func (s *Service) sourceNodesByIDs(ids []string) map[string]write.NodeRecord {
	if s == nil || s.store == nil {
		return map[string]write.NodeRecord{}
	}
	reader, ok := s.store.(sourceNodeBatchReader)
	if !ok || reader == nil {
		return map[string]write.NodeRecord{}
	}
	if len(ids) == 0 {
		return map[string]write.NodeRecord{}
	}
	return reader.GetNodesByIDs(ids)
}

func ftsResultNodeIDs(results []fts.FTSResult) []string {
	ids := make([]string, 0, len(results))
	seen := map[string]struct{}{}
	for _, result := range results {
		if result.Document.NodeID == "" {
			continue
		}
		if _, ok := seen[result.Document.NodeID]; ok {
			continue
		}
		seen[result.Document.NodeID] = struct{}{}
		ids = append(ids, result.Document.NodeID)
	}
	return ids
}

func hydrateVectorDocumentFromSource(doc vectorstore.VectorDocument, node write.NodeRecord) vectorstore.VectorDocument {
	doc.NodeType = node.NodeType
	doc.DomainID = node.DomainID
	doc.OwnerTenantID = node.OwnerTenantID
	doc.OwnerAppID = node.OwnerAppID
	doc.ACLVisibleTo = nodeACLVisibleTo(node)
	doc.IsDeleted = node.IsDeleted
	doc.StatusValue = node.StatusValue
	doc.DomainProps = cloneMap(node.Properties)
	return doc
}

func hydrateFTSDocumentFromSource(doc fts.FTSDocument, node write.NodeRecord) fts.FTSDocument {
	doc.NodeType = node.NodeType
	doc.DomainID = node.DomainID
	doc.OwnerTenantID = node.OwnerTenantID
	doc.OwnerAppID = node.OwnerAppID
	doc.ACLVisibleTo = nodeACLVisibleTo(node)
	doc.IsDeleted = node.IsDeleted
	doc.StatusValue = node.StatusValue
	doc.DomainProps = cloneMap(node.Properties)
	return doc
}

func (s *Service) embeddingForSearch(actor access.Identity, domainIDs []string, query string) ([]float64, error) {
	router := s.embedding
	if router == nil {
		router = vector.DirectRouter{Provider: vector.NewDeterministicProvider(8)}
	}
	domainID := ""
	if len(domainIDs) > 0 {
		domainID = domainIDs[0]
	}
	return router.RouteContext(actor.TenantID, domainID).Embed(context.Background(), query)
}

func vectorFilterMode(resolved ontology.ResolvedSearchProfile) string {
	if resolved.QueryStrategy.MaxDepth > 5 {
		return "post"
	}
	return "pre"
}

func vectorIndexHint(resolved ontology.ResolvedSearchProfile) string {
	key := resolved.QueryStrategy.Key
	// "default" strategy uses sequential scan — no special index hint needed.
	// Only force index hint for custom strategies (e.g. hnsw_custom, deep_traversal).
	if key != "" && key != "default" {
		return key
	}
	return ""
}

func vectorEfSearch(resolved ontology.ResolvedSearchProfile) int {
	if value, ok := resolved.QueryStrategy.Params["ef_search"].(int); ok {
		return value
	}
	return 0
}

func authorityScoreForFTS(doc fts.FTSDocument, cfg *ontology.StatusFieldConfig) *int {
	if doc.AuthorityScore != nil {
		return doc.AuthorityScore
	}
	if cfg == nil || cfg.AuthorityFieldName == "" || len(cfg.AuthorityValuesMap) == 0 {
		return nil
	}
	if doc.DomainProps == nil {
		return nil
	}
	value, ok := doc.DomainProps[cfg.AuthorityFieldName]
	if !ok {
		return nil
	}
	score, ok := cfg.AuthorityValuesMap[fmt.Sprintf("%v", value)]
	if !ok {
		return nil
	}
	return &score
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
