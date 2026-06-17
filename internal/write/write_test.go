package write

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
)

type fakeAuditLogger struct {
	entries []writeAuditCall
}

type writeAuditCall struct {
	action       string
	resourceType string
	resourceID   string
	outcome      string
}

func (f *fakeAuditLogger) RecordWriteAudit(actor access.Identity, ownerTenantID, ownerAppID, action, resourceType, resourceID, outcome, reason string, metadata map[string]any) {
	f.entries = append(f.entries, writeAuditCall{
		action:       action,
		resourceType: resourceType,
		resourceID:   resourceID,
		outcome:      outcome,
	})
}

func TestCreateNodeValidatesOntologyAndCreatesOutbox(t *testing.T) {
	svc, store := newTestService(t)

	resp, err := svc.CreateNode(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Hop dong thu nghiem", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "contract-template-1",
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	if resp.NodeID == "" || resp.Status != "processing" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(store.ListOutboxEvents()) != 1 {
		t.Fatalf("outbox len = %d, want 1", len(store.ListOutboxEvents()))
	}
}

func TestCreateNodeRejectsMissingRequiredProperty(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CreateNode(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{},
	})
	if err == nil {
		t.Fatal("CreateNode() error = nil, want validation failure")
	}
}

func TestCreateNodeRejectsDuplicateExternalRef(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	_, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "One", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "dup-ref",
	})
	if err != nil {
		t.Fatalf("first CreateNode() error = %v", err)
	}

	_, err = svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Two", "bridge_dinh_kem_ids": []any{"appendix-2"}},
		ExternalRef: "dup-ref",
	})
	if err == nil {
		t.Fatal("second CreateNode() error = nil, want duplicate external_ref")
	}
}

func TestCreateNodeHandlerReturnsAccepted(t *testing.T) {
	svc, _ := newTestService(t)
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{"domain_id":"noi_bo_hop_dong","node_type":"HopDongMau","properties":{"ten":"Hop dong A","bridge_dinh_kem_ids":["appendix-1"]}}`))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}))
	rec := httptest.NewRecorder()

	handler.CreateNode(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"node_id"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestUpdateNodeRevalidatesAndEmitsOutbox(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	updated, err := svc.UpdateNode(actor, created.NodeID, NodeUpdateRequest{
		Properties: map[string]any{"ghi_chu": "ban cap nhat"},
	})
	if err != nil {
		t.Fatalf("UpdateNode() error = %v", err)
	}
	if updated.NodeID != created.NodeID {
		t.Fatalf("updated node id = %q", updated.NodeID)
	}
	if len(store.ListOutboxEvents()) != 2 {
		t.Fatalf("outbox len = %d, want 2", len(store.ListOutboxEvents()))
	}
}

func TestDeleteNodeMarksDeletedAndEmitsOutbox(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	deleted, err := svc.DeleteNode(actor, created.NodeID)
	if err != nil {
		t.Fatalf("DeleteNode() error = %v", err)
	}
	if !deleted.IsDeleted {
		t.Fatal("DeleteNode() did not mark node deleted")
	}
	if len(store.ListOutboxEvents()) != 2 {
		t.Fatalf("outbox len = %d, want 2", len(store.ListOutboxEvents()))
	}
}

func TestUpdateNodeHandlerReturnsOK(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	handler := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPut, "/v1/kg/write/nodes/"+created.NodeID, strings.NewReader(`{"properties":{"ghi_chu":"cap nhat"}}`))
	req.SetPathValue("id", created.NodeID)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()

	handler.UpdateNode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteNodeHandlerReturnsOK(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	handler := NewHandler(svc)
	req := httptest.NewRequest(http.MethodDelete, "/v1/kg/write/nodes/"+created.NodeID, nil)
	req.SetPathValue("id", created.NodeID)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()

	handler.DeleteNode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func newTestService(t *testing.T) (*Service, *MemoryStore) {
	t.Helper()

	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}

	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologyService := ontology.NewService(ontologyStore, accessResolver)
	_, err = ontologyService.CreateDomain(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", ontology.DomainCreateRequest{
		ID:   "noi_bo_hop_dong",
		Name: "Noi Bo Hop Dong",
	})
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	_, err = ontologyService.CreateNodeType(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName: "HopDongMau",
		RequiredProps: []ontology.PropertySchema{
			{Name: "ten", Type: "string"},
		},
	})
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("CreateNodeType() error = %v", err)
		}
	}
	_, err = ontologyService.CreateNodeType(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName: "PhuLucHopDong",
		RequiredProps: []ontology.PropertySchema{
			{Name: "ten", Type: "string"},
		},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType() error = %v", err)
	}

	store := NewMemoryStore()
	seedBridgeTarget(store)
	return NewService(store, ontologyService, accessResolver, &postgres.SessionManager{}, nil), store
}

func TestCreateNodeRequiresBridgePropertyWhenRuleIsConfigured(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CreateNode(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A"},
	})
	if err == nil {
		t.Fatal("CreateNode() error = nil, want bridge validation failure")
	}
}

func TestCreateNodeBuildsBridgeRelationshipsAndTracksSessionScope(t *testing.T) {
	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}

	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)
	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologyService := ontology.NewService(ontologyStore, accessResolver)
	_, err = ontologyService.CreateDomain(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", ontology.DomainCreateRequest{
		ID:   "noi_bo_hop_dong",
		Name: "Noi Bo Hop Dong",
	})
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	sessionManager := &postgres.SessionManager{}
	store := NewMemoryStore()
	seedBridgeTarget(store)
	svc := NewService(store, ontologyService, accessResolver, sessionManager, nil)

	resp, err := svc.CreateNode(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, NodeCreateRequest{
		DomainID: "noi_bo_hop_dong",
		NodeType: "HopDongMau",
		Properties: map[string]any{
			"ten":                 "Hop dong A",
			"bridge_dinh_kem_ids": []any{"appendix-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	if resp.NodeID == "" {
		t.Fatal("node id is empty")
	}
	if len(store.ListOutboxEvents()) != 1 {
		t.Fatalf("outbox len = %d, want 1", len(store.ListOutboxEvents()))
	}
	if len(sessionManager.LastScope.Statements) != 2 {
		t.Fatalf("session statements len = %d, want 2", len(sessionManager.LastScope.Statements))
	}
}

func TestCreateNodeRejectsBridgeTargetWithWrongType(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	wrongTarget, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Khoan 1"},
	})
	if err != nil {
		t.Fatalf("CreateNode(target) error = %v", err)
	}

	_, err = svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{wrongTarget.NodeID}},
	})
	if err == nil {
		t.Fatal("CreateNode() error = nil, want bridge target validation failure")
	}
}

func TestCreateRelationshipValidatesSchemaAndEmitsOutbox(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	fromNode, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode(from) error = %v", err)
	}
	toNode, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Khoan 1"},
	})
	if err != nil {
		t.Fatalf("CreateNode(to) error = %v", err)
	}

	resp, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:    "THAM_CHIEU",
		FromNodeID: fromNode.NodeID,
		ToNodeID:   toNode.NodeID,
		DomainID:   "noi_bo_hop_dong",
		Properties: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}
	if resp.RelationshipID == "" {
		t.Fatal("relationship id is empty")
	}
	if len(store.ListOutboxEvents()) != 3 {
		t.Fatalf("outbox len = %d, want 3", len(store.ListOutboxEvents()))
	}
}

func TestCreateRelationshipRejectsInvalidEndpoints(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	fromNode, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Khoan 1"},
	})
	if err != nil {
		t.Fatalf("CreateNode(from) error = %v", err)
	}
	toNode, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode(to) error = %v", err)
	}

	_, err = svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:    "THAM_CHIEU",
		FromNodeID: fromNode.NodeID,
		ToNodeID:   toNode.NodeID,
		DomainID:   "noi_bo_hop_dong",
		Properties: map[string]any{},
	})
	if err == nil {
		t.Fatal("CreateRelationship() error = nil, want validation failure")
	}
}

func TestCreateRelationshipHandlerReturnsCreated(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	fromNode, _ := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	toNode, _ := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Khoan 1"},
	})

	handler := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/relationships", strings.NewReader(`{"rel_type":"THAM_CHIEU","from_node_id":"`+fromNode.NodeID+`","to_node_id":"`+toNode.NodeID+`","domain_id":"noi_bo_hop_dong","properties":{}}`))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()

	handler.CreateRelationship(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWriteServiceEmitsAuditEntriesForSuccessfulMutations(t *testing.T) {
	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}

	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)
	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologyService := ontology.NewService(ontologyStore, accessResolver)
	_, err = ontologyService.CreateDomain(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", ontology.DomainCreateRequest{
		ID:   "noi_bo_hop_dong",
		Name: "Noi Bo Hop Dong",
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	_, err = ontologyService.CreateNodeType(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName: "HopDongMau",
		RequiredProps: []ontology.PropertySchema{
			{Name: "ten", Type: "string"},
		},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType() error = %v", err)
	}
	store := NewMemoryStore()
	seedBridgeTarget(store)
	auditLogger := &fakeAuditLogger{}
	svc := NewService(store, ontologyService, accessResolver, &postgres.SessionManager{}, auditLogger)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	_, err = svc.UpdateNode(actor, created.NodeID, NodeUpdateRequest{
		Properties: map[string]any{"ghi_chu": "cap nhat"},
	})
	if err != nil {
		t.Fatalf("UpdateNode() error = %v", err)
	}
	_, err = svc.DeleteNode(actor, created.NodeID)
	if err != nil {
		t.Fatalf("DeleteNode() error = %v", err)
	}

	if len(auditLogger.entries) != 3 {
		t.Fatalf("audit len = %d, want 3", len(auditLogger.entries))
	}
	if auditLogger.entries[0].action != "kg.node.create" {
		t.Fatalf("first action = %q", auditLogger.entries[0].action)
	}
	if auditLogger.entries[1].action != "kg.node.update" {
		t.Fatalf("second action = %q", auditLogger.entries[1].action)
	}
	if auditLogger.entries[2].action != "kg.node.delete" {
		t.Fatalf("third action = %q", auditLogger.entries[2].action)
	}
}

func seedBridgeTarget(store *MemoryStore) {
	now := time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC)
	store.nodes["appendix-1"] = NodeRecord{
		ID:            "appendix-1",
		NodeType:      "PhuLucHopDong",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: "11111111-1111-1111-1111-111111111111",
		OwnerAppID:    "11111111-admin-1111-admin-111111111111",
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Phu luc 1"},
		DomainVersion: 1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
