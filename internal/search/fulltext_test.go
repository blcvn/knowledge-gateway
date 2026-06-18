package search

import (
	"context"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
)

type fullTextSearchOntologyStub struct {
	domain ontology.Domain
	cfg    *ontology.StatusFieldConfig
}

func (s fullTextSearchOntologyStub) GetVisibleDomain(actor access.Identity, domainID string) (ontology.Domain, error) {
	return s.domain, nil
}

func (s fullTextSearchOntologyStub) GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error) {
	return s.cfg, nil
}

type fullTextSearchAccessStub struct {
	owners []access.VisibleOwner
}

func (s fullTextSearchAccessStub) ResolveVisibleOwners(identity access.Identity) ([]access.VisibleOwner, error) {
	return s.owners, nil
}

func TestServiceFullTextSearch(t *testing.T) {
	adapter := fts.NewInMemoryFTSAdapter()
	now := time.Date(2026, time.June, 18, 7, 0, 0, 0, time.UTC)
	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	must(adapter.Index(context.Background(), fts.FTSDocument{
		NodeID:        "node-a",
		NodeType:      "Doc",
		DomainID:      "d1",
		OwnerTenantID: "tenant-a",
		OwnerAppID:    "app-a",
		ACLVisibleTo:  []string{"tenant-a:app-a"},
		StatusValue:   "active",
		Fields: map[string]string{
			"title": "payment gateway",
			"body":  "irrelevant text",
		},
		CreatedAt: now,
	}))
	must(adapter.Index(context.Background(), fts.FTSDocument{
		NodeID:        "node-b",
		NodeType:      "Doc",
		DomainID:      "d1",
		OwnerTenantID: "tenant-a",
		OwnerAppID:    "app-a",
		ACLVisibleTo:  []string{"tenant-a:app-a"},
		StatusValue:   "inactive",
		Fields: map[string]string{
			"title": "payment gateway",
			"body":  "other text",
		},
		CreatedAt: now.Add(time.Minute),
	}))
	must(adapter.Index(context.Background(), fts.FTSDocument{
		NodeID:        "node-c",
		NodeType:      "Doc",
		DomainID:      "d1",
		OwnerTenantID: "tenant-b",
		OwnerAppID:    "app-b",
		ACLVisibleTo:  []string{"tenant-b:app-b"},
		StatusValue:   "active",
		Fields: map[string]string{
			"title": "payment gateway",
			"body":  "other text",
		},
		CreatedAt: now.Add(2 * time.Minute),
	}))

	svc := &Service{
		ontology:       fullTextSearchOntologyStub{domain: ontology.Domain{ID: "d1"}, cfg: &ontology.StatusFieldConfig{DomainID: "d1", StatusFieldName: "status_value", ValidStatusValues: []string{"active"}}},
		accessResolver: fullTextSearchAccessStub{owners: []access.VisibleOwner{{TenantID: "tenant-a", AppID: "app-a"}}},
		auditLogger:    nil,
		ftsAdapter:     adapter,
	}

	resp, err := svc.FullTextSearch(access.Identity{TenantID: "tenant-a", AppID: "app-a"}, FullTextSearchRequest{
		Query:     "payment gateway",
		DomainIDs: []string{"d1"},
		TopK:      10,
		Mode:      "phrase",
		Fields:    []string{"title"},
	})
	if err != nil {
		t.Fatalf("FullTextSearch() error = %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(resp.Results))
	}
	if resp.Results[0].NodeID != "node-a" {
		t.Fatalf("result NodeID = %s, want node-a", resp.Results[0].NodeID)
	}
	if resp.Results[0].Content != "irrelevant text payment gateway" {
		t.Fatalf("content = %q, want normalized joined content", resp.Results[0].Content)
	}
}
