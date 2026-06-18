package search

import (
	"context"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
)

type vectorAdapterSearchOntologyStub struct {
	domain ontology.Domain
	cfg    *ontology.StatusFieldConfig
}

func (s vectorAdapterSearchOntologyStub) GetVisibleDomain(actor access.Identity, domainID string) (ontology.Domain, error) {
	return s.domain, nil
}

func (s vectorAdapterSearchOntologyStub) GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error) {
	return s.cfg, nil
}

type vectorAdapterSearchAccessStub struct {
	owners []access.VisibleOwner
}

func (s vectorAdapterSearchAccessStub) ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error) {
	return s.owners, nil
}

type recordingVectorAdapter struct {
	lastQuery  []float64
	lastFilter vectorstore.VectorFilter
	lastOpts   vectorstore.ANNOptions
	results    []vectorstore.VectorResult
}

type vectorAdapterSearchResolver struct{}

func (vectorAdapterSearchResolver) Resolve(domainID, tenantID, appID string) (ontology.ResolvedSearchProfile, error) {
	return ontology.ResolvedSearchProfile{
		Domain: ontology.Domain{ID: domainID},
		QueryStrategy: ontology.QueryStrategy{
			Key:      "deep_traversal",
			Version:  1,
			MaxDepth: 10,
			Params: map[string]any{
				"ef_search": 77,
			},
		},
	}, nil
}

func (a *recordingVectorAdapter) Upsert(ctx context.Context, doc vectorstore.VectorDocument) error {
	return nil
}

func (a *recordingVectorAdapter) Delete(ctx context.Context, nodeID string) error {
	return nil
}

func (a *recordingVectorAdapter) ANN(ctx context.Context, query []float64, filter vectorstore.VectorFilter, opts vectorstore.ANNOptions) ([]vectorstore.VectorResult, error) {
	a.lastQuery = append([]float64(nil), query...)
	a.lastFilter = filter
	a.lastOpts = opts
	return append([]vectorstore.VectorResult(nil), a.results...), nil
}

func (a *recordingVectorAdapter) Snapshot(ctx context.Context) ([]vectorstore.VectorDocument, error) {
	return nil, nil
}

func (a *recordingVectorAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	return 0, nil
}

func TestSemanticSearchUsesVectorAdapterANN(t *testing.T) {
	adapter := &recordingVectorAdapter{
		results: []vectorstore.VectorResult{
			{
				Document: vectorstore.VectorDocument{
					NodeID:        "node-a",
					NodeType:      "Doc",
					DomainID:      "d1",
					OwnerTenantID: "tenant-a",
					OwnerAppID:    "app-a",
					ACLVisibleTo:  []string{"tenant-a:app-a"},
					DomainProps: map[string]any{
						"title": "payment gateway",
					},
				},
				Score: 0.91,
			},
		},
	}
	svc := &Service{
		ontology: vectorAdapterSearchOntologyStub{
			domain: ontology.Domain{ID: "d1"},
			cfg:    &ontology.StatusFieldConfig{DomainID: "d1", StatusFieldName: "status_value", ValidStatusValues: []string{"active"}},
		},
		accessResolver: vectorAdapterSearchAccessStub{owners: []access.VisibleOwner{{TenantID: "tenant-a", AppID: "app-a"}}},
		vectorAdapter:  adapter,
		embedding:      vector.DirectRouter{Provider: vector.NewDeterministicProvider(8)},
		profiles:       vectorAdapterSearchResolver{},
	}

	resp, err := svc.SemanticSearch(access.Identity{TenantID: "tenant-a", AppID: "app-a"}, SemanticSearchRequest{
		Query:     "payment gateway",
		DomainIDs: []string{"d1"},
		TopK:      5,
	})
	if err != nil {
		t.Fatalf("SemanticSearch() error = %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].NodeID != "node-a" {
		t.Fatalf("results = %#v, want node-a", resp.Results)
	}
	if adapter.lastOpts.TopK != 5 {
		t.Fatalf("TopK = %d, want 5", adapter.lastOpts.TopK)
	}
	if adapter.lastOpts.EfSearch != 77 {
		t.Fatalf("EfSearch = %d, want 77", adapter.lastOpts.EfSearch)
	}
	if adapter.lastOpts.FilterMode != "post" {
		t.Fatalf("FilterMode = %q, want post", adapter.lastOpts.FilterMode)
	}
	if adapter.lastOpts.IndexHint != "deep_traversal" {
		t.Fatalf("IndexHint = %q, want deep_traversal", adapter.lastOpts.IndexHint)
	}
	if len(adapter.lastFilter.DomainIDs) != 1 || adapter.lastFilter.DomainIDs[0] != "d1" {
		t.Fatalf("filter domains = %#v, want d1", adapter.lastFilter.DomainIDs)
	}
	if len(adapter.lastFilter.ACLVisibleTo) != 1 || adapter.lastFilter.ACLVisibleTo[0] != "tenant-a:app-a" {
		t.Fatalf("filter acl = %#v, want tenant-a:app-a", adapter.lastFilter.ACLVisibleTo)
	}
	if len(adapter.lastQuery) == 0 {
		t.Fatal("embedding query was not generated")
	}
	if resp.Results[0].CreatedAt != (time.Time{}) {
		t.Fatalf("CreatedAt = %v, want zero value in vector adapter result", resp.Results[0].CreatedAt)
	}
}
