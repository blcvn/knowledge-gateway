package search

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/write"
)

type recordingAuditLogger struct {
	entries []auditEntry
}

type auditEntry struct {
	action       string
	resourceType string
	resourceID   string
	outcome      string
	reason       string
}

func (r *recordingAuditLogger) RecordReadAudit(actor access.Identity, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	r.entries = append(r.entries, auditEntry{
		action:       action,
		resourceType: resourceType,
		resourceID:   resourceID,
		outcome:      outcome,
		reason:       reason,
	})
}

func TestSemanticSearchFiltersByACLAndReturnsMetadata(t *testing.T) {
	svc, auditLogger, actor := newSearchFixture(t)

	resp, err := svc.SemanticSearch(actor, SemanticSearchRequest{
		Query:     "Returns workflow",
		DomainIDs: []string{"sample-policy"},
		TopK:      10,
	})
	if err != nil {
		t.Fatalf("SemanticSearch() error = %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(resp.Results))
	}
	got := resp.Results[0]
	if got.NodeID != "visible-node" {
		t.Fatalf("NodeID = %q, want visible-node", got.NodeID)
	}
	if got.AuthorityScore == nil || *got.AuthorityScore != 4 {
		t.Fatalf("AuthorityScore = %v, want 4", got.AuthorityScore)
	}
	if len(got.ACLVisibleTo) == 0 || got.ACLVisibleTo[0] != actor.TenantID+":"+actor.AppID {
		t.Fatalf("ACLVisibleTo = %#v, want owner token", got.ACLVisibleTo)
	}
	if len(auditLogger.entries) != 1 || auditLogger.entries[0].outcome != "allow" {
		t.Fatalf("audit entries = %#v, want allow audit", auditLogger.entries)
	}
}

func TestSemanticSearchRejectsInvisibleDomainFilter(t *testing.T) {
	svc, _, actor := newSearchFixture(t)

	_, err := svc.SemanticSearch(actor, SemanticSearchRequest{
		Query:     "Hộ kinh doanh",
		DomainIDs: []string{"secret-domain"},
	})
	if err == nil {
		t.Fatal("SemanticSearch() error = nil, want validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("SemanticSearch() error = %v, want validation", err)
	}
}

func TestSemanticSearchWithoutDomainFilterReturnsAllVisibleDomains(t *testing.T) {
	svc, _, actor := newSearchFixture(t)

	resp, err := svc.SemanticSearch(actor, SemanticSearchRequest{
		Query: "Returns workflow",
		TopK:  10,
	})
	if err != nil {
		t.Fatalf("SemanticSearch() error = %v", err)
	}
	gotDomains := map[string]bool{}
	for _, result := range resp.Results {
		gotDomains[result.DomainID] = true
		if result.NodeID == "deleted-visible-node" {
			t.Fatalf("deleted node unexpectedly returned: %#v", result)
		}
	}
	if !gotDomains["sample-policy"] || !gotDomains["shared-domain"] {
		t.Fatalf("domains = %#v, want visible domains", gotDomains)
	}
}

func TestSemanticSearchIgnoresLifecycleWhenTargetedDomainsAreMixed(t *testing.T) {
	svc, _, actor := newSearchFixture(t)

	resp, err := svc.SemanticSearch(actor, SemanticSearchRequest{
		Query:     "Mixed lifecycle node",
		DomainIDs: []string{"sample-policy", "shared-domain"},
	})
	if err != nil {
		t.Fatalf("SemanticSearch() error = %v", err)
	}
	found := false
	for _, result := range resp.Results {
		if result.NodeID == "mixed-lifecycle-node" {
			found = true
		}
	}
	if !found {
		t.Fatalf("results = %#v, want mixed-lifecycle-node", resp.Results)
	}
}

func TestSearchHandlerServesSemanticAndRagEndpoints(t *testing.T) {
	svc, _, actor := newSearchFixture(t)
	handler := NewHandler(svc)

	body := []byte(`{"query":"Returns workflow","top_k":5}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/search/semantic", bytes.NewReader(body))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()
	handler.SemanticSearch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("SemanticSearch status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"results"`) {
		t.Fatalf("SemanticSearch body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/kg/search/rag", bytes.NewReader(body))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec = httptest.NewRecorder()
	handler.RagSearch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("RagSearch status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func newSearchFixture(t *testing.T) (*Service, *recordingAuditLogger, access.Identity) {
	t.Helper()

	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}

	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(
		append(ontology.SeedDomains(), ontology.Domain{
			ID:            "secret-domain",
			Name:          "Secret Domain",
			OwnerTenantID: "22222222-2222-2222-2222-222222222222",
			Status:        "active",
			Version:       1,
			Visibility:    "private",
			CreatedAt:     time.Date(2026, 6, 17, 11, 0, 0, 0, time.UTC),
			UpdatedAt:     time.Date(2026, 6, 17, 11, 0, 0, 0, time.UTC),
		}),
		append(ontology.SeedVersions(), ontology.OntologyVersion{
			DomainID:    "secret-domain",
			Version:     1,
			PublishedAt: time.Date(2026, 6, 17, 11, 0, 0, 0, time.UTC),
		}),
		ontology.SeedNodeTypes(),
		ontology.SeedRelTypes(),
		ontology.SeedCrossDomainRules(),
		ontology.SeedQueryTemplates(),
		ontology.SeedStatusFieldConfigs(),
	)
	ontologyService := ontology.NewService(ontologyStore, accessResolver)

	store := write.NewMemoryStore()
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
		AppType:  "agent_consumer",
	}
	now := time.Date(2026, 6, 17, 11, 5, 0, 0, time.UTC)
	mustInsertNode(t, store, write.NodeRecord{
		ID:            "visible-node",
		NodeType:      "Doc",
		DomainID:      "sample-policy",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Properties: map[string]any{
			"summary":       "Returns workflow for marketplace sellers",
			"document_class": "policy",
			"record_status":  "active",
			"domain_marker": "visible",
		},
		StatusValue: "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}, now)
	mustInsertNode(t, store, write.NodeRecord{
		ID:            "hidden-acl-node",
		NodeType:      "Doc",
		DomainID:      "sample-policy",
		OwnerTenantID: "22222222-2222-2222-2222-222222222222",
		OwnerAppID:    "22222222-aaaa-2222-aaaa-222222222222",
		ACLVisibleTo:  []string{"22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222"},
		Properties: map[string]any{
			"summary":        "Hidden workflow document",
			"document_class": "policy",
			"record_status":  "active",
		},
		StatusValue: "active",
		CreatedAt:   now.Add(time.Minute),
		UpdatedAt:   now.Add(time.Minute),
	}, now.Add(time.Minute))
	mustInsertNode(t, store, write.NodeRecord{
		ID:            "secret-node",
		NodeType:      "SecretDoc",
		DomainID:      "secret-domain",
		OwnerTenantID: "22222222-2222-2222-2222-222222222222",
		OwnerAppID:    "22222222-aaaa-2222-aaaa-222222222222",
		ACLVisibleTo:  []string{"22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222"},
		Properties: map[string]any{
			"summary": "Hidden domain content",
		},
		CreatedAt: now.Add(2 * time.Minute),
		UpdatedAt: now.Add(2 * time.Minute),
	}, now.Add(2*time.Minute))
	mustInsertNode(t, store, write.NodeRecord{
		ID:            "shared-domain-visible-node",
		NodeType:      "Doc",
		DomainID:      "shared-domain",
		OwnerTenantID: "22222222-2222-2222-2222-222222222222",
		OwnerAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Properties: map[string]any{
			"summary": "Returns workflow knowledge article",
		},
		CreatedAt: now.Add(3 * time.Minute),
		UpdatedAt: now.Add(3 * time.Minute),
	}, now.Add(3*time.Minute))
	mustInsertNode(t, store, write.NodeRecord{
		ID:            "deleted-visible-node",
		NodeType:      "Doc",
		DomainID:      "sample-policy",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Properties: map[string]any{
			"summary": "Archived workflow document",
		},
		IsDeleted: true,
		CreatedAt: now.Add(4 * time.Minute),
		UpdatedAt: now.Add(4 * time.Minute),
	}, now.Add(4*time.Minute))
	mustInsertNode(t, store, write.NodeRecord{
		ID:            "mixed-lifecycle-node",
		NodeType:      "Doc",
		DomainID:      "sample-policy",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Properties: map[string]any{
			"summary":       "Mixed lifecycle node",
			"document_class": "guide",
			"domain_marker": "mixed",
		},
		StatusValue: "inactive",
		CreatedAt:   now.Add(5 * time.Minute),
		UpdatedAt:   now.Add(5 * time.Minute),
	}, now.Add(5*time.Minute))

	auditLogger := &recordingAuditLogger{}
	return NewService(store, ontologyService, accessResolver, auditLogger), auditLogger, actor
}

func mustInsertNode(t *testing.T, store *write.MemoryStore, node write.NodeRecord, createdAt time.Time) {
	t.Helper()
	if err := store.CreateNodeWithOutbox(node, write.OutboxEvent{
		ID:            "evt-" + node.ID,
		AggregateType: "kg_node",
		AggregateID:   node.ID,
		EventType:     "NODE_UPSERTED",
		Payload: map[string]any{
			"node_id":   node.ID,
			"domain_id": node.DomainID,
		},
		Status:    "PENDING",
		CreatedAt: createdAt,
	}); err != nil {
		t.Fatalf("CreateNodeWithOutbox(%s) error = %v", node.ID, err)
	}
}
