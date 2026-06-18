package search

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
)

type hybridSearchOntologyStub struct {
	domain ontology.Domain
	cfg    *ontology.StatusFieldConfig
}

func (s hybridSearchOntologyStub) GetVisibleDomain(actor access.Identity, domainID string) (ontology.Domain, error) {
	return s.domain, nil
}

func (s hybridSearchOntologyStub) GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error) {
	return s.cfg, nil
}

type hybridSearchAccessStub struct {
	owners []access.VisibleOwner
}

func (s hybridSearchAccessStub) ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error) {
	return s.owners, nil
}

type hybridSearchIndexStub struct {
	results []SearchResult
}

func (s hybridSearchIndexStub) Search(actor access.Identity, req SemanticSearchRequest, visibility map[string]struct{}, domainSet map[string]struct{}, statusConfigs map[string]*ontology.StatusFieldConfig, allHaveStatus bool) ([]SearchResult, error) {
	return append([]SearchResult(nil), s.results...), nil
}

func (s hybridSearchIndexStub) RagSearch(actor access.Identity, req SemanticSearchRequest, visibility map[string]struct{}, domainSet map[string]struct{}, statusConfigs map[string]*ontology.StatusFieldConfig, allHaveStatus bool) ([]SearchResult, error) {
	return append([]SearchResult(nil), s.results...), nil
}

type hybridSearchFTSStub struct {
	results []fts.FTSResult
}

func (s hybridSearchFTSStub) Index(ctx context.Context, doc fts.FTSDocument) error {
	return nil
}

func (s hybridSearchFTSStub) Delete(ctx context.Context, nodeID string) error {
	return nil
}

func (s hybridSearchFTSStub) Search(ctx context.Context, query fts.FTSQuery, filter fts.FTSFilter, opts fts.SearchOptions) ([]fts.FTSResult, error) {
	return append([]fts.FTSResult(nil), s.results...), nil
}

func TestHybridSearchRRFUsesBothRankings(t *testing.T) {
	svc := &Service{
		ontology: hybridSearchOntologyStub{
			domain: ontology.Domain{ID: "d1"},
			cfg:    &ontology.StatusFieldConfig{DomainID: "d1", StatusFieldName: "status_value", ValidStatusValues: []string{"active"}},
		},
		accessResolver: hybridSearchAccessStub{owners: []access.VisibleOwner{{TenantID: "tenant-a", AppID: "app-a"}}},
		index: hybridSearchIndexStub{
			results: []SearchResult{
				{NodeID: "node-a", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", Score: 0.95, CreatedAt: time.Date(2026, 6, 18, 7, 0, 0, 0, time.UTC)},
				{NodeID: "node-b", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", Score: 0.90, CreatedAt: time.Date(2026, 6, 18, 7, 1, 0, 0, time.UTC)},
			},
		},
		ftsAdapter: hybridSearchFTSStub{
			results: []fts.FTSResult{
				{Document: fts.FTSDocument{NodeID: "node-b", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", CreatedAt: time.Date(2026, 6, 18, 7, 1, 0, 0, time.UTC)}, Score: 0.88},
				{Document: fts.FTSDocument{NodeID: "node-c", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", CreatedAt: time.Date(2026, 6, 18, 7, 2, 0, 0, time.UTC)}, Score: 0.80},
			},
		},
	}

	resp, err := svc.HybridSearch(access.Identity{TenantID: "tenant-a", AppID: "app-a"}, HybridSearchRequest{
		Query:          "payment gateway",
		DomainIDs:      []string{"d1"},
		TopK:           10,
		FTSOperator:    "phrase",
		SemanticWeight: 0.75,
	})
	if err != nil {
		t.Fatalf("HybridSearch() error = %v", err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("results len = %d, want 3", len(resp.Results))
	}
	if resp.Results[0].NodeID != "node-b" || resp.Results[1].NodeID != "node-a" || resp.Results[2].NodeID != "node-c" {
		t.Fatalf("result order = %#v, want node-b > node-a > node-c", []string{resp.Results[0].NodeID, resp.Results[1].NodeID, resp.Results[2].NodeID})
	}

	wantScores := map[string]float64{
		"node-a": 0.75 / 61.0,
		"node-b": 0.75/62.0 + 0.25/61.0,
		"node-c": 0.25 / 62.0,
	}
	for _, result := range resp.Results {
		if diff := result.Score - wantScores[result.NodeID]; diff < -1e-9 || diff > 1e-9 {
			t.Fatalf("score[%s] = %.12f, want %.12f", result.NodeID, result.Score, wantScores[result.NodeID])
		}
	}
}

func TestSearchHandlerServesHybridEndpoint(t *testing.T) {
	svc := &Service{
		ontology: hybridSearchOntologyStub{
			domain: ontology.Domain{ID: "d1"},
			cfg:    &ontology.StatusFieldConfig{DomainID: "d1", StatusFieldName: "status_value", ValidStatusValues: []string{"active"}},
		},
		accessResolver: hybridSearchAccessStub{owners: []access.VisibleOwner{{TenantID: "tenant-a", AppID: "app-a"}}},
		index: hybridSearchIndexStub{
			results: []SearchResult{
				{NodeID: "node-a", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", Score: 0.95, CreatedAt: time.Date(2026, 6, 18, 7, 0, 0, 0, time.UTC)},
			},
		},
		ftsAdapter: hybridSearchFTSStub{
			results: []fts.FTSResult{
				{Document: fts.FTSDocument{NodeID: "node-a", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", CreatedAt: time.Date(2026, 6, 18, 7, 0, 0, 0, time.UTC)}, Score: 0.88},
			},
		},
	}

	handler := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/v1/kg/search/hybrid", bytes.NewReader([]byte(`{"query":"payment gateway","top_k":5,"semantic_weight":0.75,"fts_operator":"phrase"}`)))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), access.Identity{TenantID: "tenant-a", AppID: "app-a"}))
	rec := httptest.NewRecorder()

	handler.HybridSearch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HybridSearch status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"results"`) {
		t.Fatalf("HybridSearch body = %s", rec.Body.String())
	}
}
