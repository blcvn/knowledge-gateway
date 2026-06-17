package integrity

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/write"
)

func TestTenantIntegrityReportsBridgeAndDomainChecks(t *testing.T) {
	svc, actor := newIntegrityFixture(t)

	resp, err := svc.TenantIntegrity(actor, actor.TenantID)
	if err != nil {
		t.Fatalf("TenantIntegrity() error = %v", err)
	}
	if resp.Overall != "fail" {
		t.Fatalf("Overall = %q, want fail", resp.Overall)
	}
	if len(resp.Checks) != 2 {
		t.Fatalf("Checks len = %d, want 2", len(resp.Checks))
	}
}

func TestMissingBridgesReturnsListEnvelopeData(t *testing.T) {
	svc, actor := newIntegrityFixture(t)
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/kg/integrity/missing-bridges?tenant_id="+actor.TenantID, nil)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()

	handler.MissingBridges(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"data"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestTenantIntegrityHandlerReturnsSummary(t *testing.T) {
	svc, actor := newIntegrityFixture(t)
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/kg/integrity/tenant/"+actor.TenantID, nil)
	req.SetPathValue("tenant_id", actor.TenantID)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()

	handler.TenantIntegrity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"checks"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func newIntegrityFixture(t *testing.T) (*Service, access.Identity) {
	t.Helper()

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	store := write.NewMemoryStore()
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	seedIntegrityNode(t, store, write.NodeRecord{
		ID:            "missing-bridge-node",
		NodeType:      "HopDongMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Hop dong missing bridge"},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
	})
	seedIntegrityNode(t, store, write.NodeRecord{
		ID:            "missing-domain-node",
		NodeType:      "HopDongMau",
		DomainID:      "",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Node missing domain"},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 12, 1, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 12, 1, 0, 0, time.UTC),
	})
	return NewService(store, ontologyStore), actor
}

func seedIntegrityNode(t *testing.T, store *write.MemoryStore, node write.NodeRecord) {
	t.Helper()
	if err := store.CreateNodeWithOutbox(node, write.OutboxEvent{
		ID:            "evt-" + node.ID,
		AggregateType: "kg_node",
		AggregateID:   node.ID,
		EventType:     "NODE_UPSERTED",
		Payload:       map[string]any{"node_id": node.ID, "domain_id": node.DomainID},
		Status:        "PENDING",
		CreatedAt:     node.CreatedAt,
	}); err != nil {
		t.Fatalf("CreateNodeWithOutbox(%s) error = %v", node.ID, err)
	}
}
