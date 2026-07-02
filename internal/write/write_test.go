package write

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/session"
	"kg-service/internal/telemetry"
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

type recordingSessionManager struct {
	LastScope session.SessionScope
}

const (
	bridgeTargetID        = "11111111-1111-4111-8111-111111111111"
	missingBridgeTargetID = "22222222-2222-4222-8222-222222222222"
)

func (m *recordingSessionManager) Within(ctx context.Context, identity session.WriteIdentity, fn func(session.SessionScope) error) (session.SessionScope, error) {
	scope := session.SessionScope{
		Identity: identity,
		Statements: []string{
			"BEGIN",
			"SET LOCAL app.tenant_id = '" + identity.TenantID + "'",
			"SET LOCAL app.app_id = '" + identity.AppID + "'",
			"COMMIT",
		},
		Transactional: true,
	}
	m.LastScope = scope
	return scope, fn(scope)
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
	if !looksLikeUUID(resp.NodeID) || resp.Status != "processing" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.GraphIdentifierID == "" || resp.GraphVersionID == "" || resp.ReferenceID == "" || resp.GraphVersionNumber == 0 {
		t.Fatalf("graph lineage missing from response: %+v", resp)
	}
	if len(store.ListOutboxEvents()) != 1 {
		t.Fatalf("outbox len = %d, want 1", len(store.ListOutboxEvents()))
	}
	payload := store.ListOutboxEvents()[0].Payload
	if payload["graph_identifier_id"] == "" || payload["graph_version_id"] == "" || payload["reference_id"] == "" {
		t.Fatalf("outbox payload missing graph lineage: %#v", payload)
	}
}

func TestCreateNodeAdvancesPostgresFTSHead(t *testing.T) {
	svc, store := newTestService(t)
	svc.SetFTSBackendKind("postgres")
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	resp, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "FTS Head", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "fts-head-node",
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	head, ok := store.GetGraphProjectionHead(resp.GraphIdentifierID, "fts", "postgres")
	if !ok {
		t.Fatalf("GetGraphProjectionHead() missing fts head for identifier %s", resp.GraphIdentifierID)
	}
	if head.AppliedVersionID != resp.GraphVersionID || head.AppliedVersionNumber != resp.GraphVersionNumber {
		t.Fatalf("fts head = %+v, want version %s/%d", head, resp.GraphVersionID, resp.GraphVersionNumber)
	}
}

func TestCreateNodesBulkAdvancesPostgresFTSHead(t *testing.T) {
	svc, store := newTestService(t)
	svc.SetFTSBackendKind("postgres")
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	resp, err := svc.CreateNodesBulkWithContext(context.Background(), actor, NodeBulkCreateRequest{
		Nodes: []NodeCreateRequest{
			{
				DomainID:    "noi_bo_hop_dong",
				NodeType:    "HopDongMau",
				Properties:  map[string]any{"ten": "Bulk FTS", "bridge_dinh_kem_ids": []any{"appendix-1"}},
				ExternalRef: "bulk-fts-node",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNodesBulkWithContext() error = %v", err)
	}
	if len(resp.Succeeded) != 1 {
		t.Fatalf("succeeded len = %d, want 1", len(resp.Succeeded))
	}
	head, ok := store.GetGraphProjectionHead(resp.Succeeded[0].GraphIdentifierID, "fts", "postgres")
	if !ok {
		t.Fatalf("GetGraphProjectionHead() missing fts head for identifier %s", resp.Succeeded[0].GraphIdentifierID)
	}
	if head.AppliedVersionID != resp.Succeeded[0].GraphVersionID || head.AppliedVersionNumber != resp.Succeeded[0].GraphVersionNumber {
		t.Fatalf("fts head = %+v, want version %s/%d", head, resp.Succeeded[0].GraphVersionID, resp.Succeeded[0].GraphVersionNumber)
	}
}

func TestSyncSessionLifecycleOnMemoryStore(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	sessionResp, err := svc.OpenSyncSession(context.Background(), actor, OpenSyncSessionRequest{
		DomainID:   "noi_bo_hop_dong",
		GraphScope: "project:session-a",
	})
	if err != nil {
		t.Fatalf("OpenSyncSession() error = %v", err)
	}
	if sessionResp.SessionID == "" || sessionResp.GraphVersionID == "" {
		t.Fatalf("session response missing ids: %+v", sessionResp)
	}
	if _, err := svc.OpenSyncSession(context.Background(), actor, OpenSyncSessionRequest{
		DomainID:   "noi_bo_hop_dong",
		GraphScope: "project:session-a",
	}); !errors.Is(err, ErrScopeLocked) {
		t.Fatalf("second OpenSyncSession() error = %v, want ErrScopeLocked", err)
	}

	bulkResp, err := svc.CreateNodesBulkWithContext(context.Background(), actor, NodeBulkCreateRequest{
		GraphVersionID: sessionResp.GraphVersionID,
		Nodes: []NodeCreateRequest{
			{
				DomainID:    "noi_bo_hop_dong",
				NodeType:    "HopDongMau",
				Properties:  map[string]any{"ten": "Session node", "project_id": "session-a", "bridge_dinh_kem_ids": []any{"appendix-1"}},
				ExternalRef: "session-node-a",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNodesBulkWithContext() error = %v", err)
	}
	if len(bulkResp.Succeeded) != 1 {
		t.Fatalf("bulk succeeded len = %d, want 1", len(bulkResp.Succeeded))
	}

	if err := svc.CommitSyncSession(context.Background(), actor, sessionResp.SessionID); err != nil {
		t.Fatalf("CommitSyncSession() error = %v", err)
	}
	events := store.ListOutboxEvents()
	if len(events) != 1 {
		t.Fatalf("outbox len = %d, want 1", len(events))
	}
	if events[0].EventType != "GRAPH_VERSION_SEALED" {
		t.Fatalf("event type = %s, want GRAPH_VERSION_SEALED", events[0].EventType)
	}
	if got := events[0].Payload["graph_version_id"]; got != sessionResp.GraphVersionID {
		t.Fatalf("graph_version_id payload = %v, want %s", got, sessionResp.GraphVersionID)
	}
}

func TestMemoryStoreCleanupExpiredSyncSessionDoesNotReleaseNewerLease(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now().UTC()
	identity := GraphIdentityRecord{
		IdentifierID:  "graph-1",
		OwnerTenantID: "tenant-a",
		OwnerAppID:    "app-a",
		GraphScope:    "project:test",
	}
	store.graphIdentities[graphIdentityKey(identity.OwnerTenantID, identity.OwnerAppID, identity.GraphScope)] = identity
	store.graphVersions[identity.IdentifierID] = []GraphVersionRecord{
		{
			VersionID:     "version-old",
			IdentifierID:  identity.IdentifierID,
			VersionStatus: "PENDING_ENTITIES",
			CreatedAt:     now.Add(-3 * time.Hour),
		},
	}
	store.scopeLeases[scopeLeaseKey(identity.OwnerTenantID, identity.OwnerAppID, identity.GraphScope)] = ScopeLeaseRecord{
		OwnerTenantID: identity.OwnerTenantID,
		OwnerAppID:    identity.OwnerAppID,
		GraphScope:    identity.GraphScope,
		VersionID:     "version-new",
		ExpiresAt:     now.Add(time.Hour),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := store.CleanupExpiredSyncSession(context.Background(), "version-old"); err != nil {
		t.Fatalf("CleanupExpiredSyncSession() error = %v", err)
	}

	lease, ok := store.GetScopeLease(context.Background(), identity.OwnerTenantID, identity.OwnerAppID, identity.GraphScope)
	if !ok {
		t.Fatal("GetScopeLease() = missing, want newer lease to remain")
	}
	if lease.VersionID != "version-new" {
		t.Fatalf("lease version_id = %q, want version-new", lease.VersionID)
	}
	version, ok := store.GetGraphVersionByID(context.Background(), "version-old")
	if !ok {
		t.Fatal("GetGraphVersionByID() = missing, want expired version to remain for audit")
	}
	if version.VersionStatus != "ABANDONED" {
		t.Fatalf("version status = %q, want ABANDONED", version.VersionStatus)
	}
}

func TestAbandonSyncSessionLeavesNoEvent(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	sessionResp, err := svc.OpenSyncSession(context.Background(), actor, OpenSyncSessionRequest{
		DomainID:   "noi_bo_hop_dong",
		GraphScope: "project:session-b",
	})
	if err != nil {
		t.Fatalf("OpenSyncSession() error = %v", err)
	}
	if err := svc.AbandonSyncSession(context.Background(), actor, sessionResp.SessionID); err != nil {
		t.Fatalf("AbandonSyncSession() error = %v", err)
	}
	if len(store.ListOutboxEvents()) != 0 {
		t.Fatalf("outbox len = %d, want 0", len(store.ListOutboxEvents()))
	}
}

func TestSealGraphVersionCanFinalizePendingVersion(t *testing.T) {
	store := NewMemoryStore()
	identity, version, err := store.SealGraphVersion(context.Background(), GraphVersionSealRequest{
		OwnerTenantID: "tenant-a",
		OwnerAppID:    "app-a",
		GraphScope:    "scope-a",
		ReferenceID:   "ref-a",
		StorageClass:  "ONLINE",
		VersionStatus: "PENDING_ENTITIES",
		Entities: []GraphVersionEntityRecord{
			{EntityKind: "node", EntityID: "node-a", ChangeKind: "UPSERT"},
		},
	})
	if err != nil {
		t.Fatalf("SealGraphVersion() error = %v", err)
	}
	if version.VersionStatus != "PENDING_ENTITIES" {
		t.Fatalf("version status = %s, want PENDING_ENTITIES", version.VersionStatus)
	}
	if affected, err := store.FinalizeGraphVersion(context.Background(), version.VersionID); err != nil {
		t.Fatalf("FinalizeGraphVersion() error = %v", err)
	} else if affected != 1 {
		t.Fatalf("FinalizeGraphVersion() rows = %d, want 1", affected)
	}
	found := false
	for _, stored := range store.graphVersions[identity.IdentifierID] {
		if stored.VersionID != version.VersionID {
			continue
		}
		found = true
		if stored.VersionStatus != "SEALED" {
			t.Fatalf("stored version status = %s, want SEALED", stored.VersionStatus)
		}
	}
	if !found {
		t.Fatalf("version %s missing from store", version.VersionID)
	}
}

func TestGraphProjectionHeadDoesNotRegress(t *testing.T) {
	store := NewMemoryStore()
	head := GraphProjectionHeadRecord{
		IdentifierID:         "graph-a",
		BackendKind:          "graph",
		BackendName:          "",
		AppliedVersionID:     "v5",
		AppliedVersionNumber: 5,
	}
	if err := store.UpsertGraphProjectionHead(context.Background(), head); err != nil {
		t.Fatalf("UpsertGraphProjectionHead() error = %v", err)
	}
	if err := store.UpsertGraphProjectionHead(context.Background(), GraphProjectionHeadRecord{
		IdentifierID:         "graph-a",
		BackendKind:          "graph",
		BackendName:          "",
		AppliedVersionID:     "v4",
		AppliedVersionNumber: 4,
	}); err != nil {
		t.Fatalf("second UpsertGraphProjectionHead() error = %v", err)
	}
	got, ok := store.GetGraphProjectionHead("graph-a", "graph", "")
	if !ok {
		t.Fatal("head missing")
	}
	if got.AppliedVersionNumber != 5 || got.AppliedVersionID != "v5" {
		t.Fatalf("head = %+v, want v5", got)
	}
}

func TestCreateNodesBulkCreatesEachNode(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	resp, err := svc.CreateNodesBulkWithContext(context.Background(), actor, NodeBulkCreateRequest{
		Nodes: []NodeCreateRequest{
			{
				DomainID:    "noi_bo_hop_dong",
				NodeType:    "HopDongMau",
				Properties:  map[string]any{"ten": "Bulk A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
				ExternalRef: "bulk-node-a",
			},
			{
				DomainID:    "noi_bo_hop_dong",
				NodeType:    "HopDongMau",
				Properties:  map[string]any{"ten": "Bulk B", "bridge_dinh_kem_ids": []any{"appendix-1"}},
				ExternalRef: "bulk-node-b",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNodesBulkWithContext() error = %v", err)
	}
	if len(resp.Succeeded) != 2 {
		t.Fatalf("succeeded len = %d, want 2", len(resp.Succeeded))
	}
	if _, ok := store.GetNodeByExternalRef("bulk-node-a"); !ok {
		t.Fatal("bulk-node-a missing from store")
	}
	if _, ok := store.GetNodeByExternalRef("bulk-node-b"); !ok {
		t.Fatal("bulk-node-b missing from store")
	}
}

func TestCreateNodesBulkReturnsPartialSuccess(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	resp, err := svc.CreateNodesBulkWithContext(context.Background(), actor, NodeBulkCreateRequest{
		Nodes: []NodeCreateRequest{
			{
				DomainID:    "noi_bo_hop_dong",
				NodeType:    "HopDongMau",
				Properties:  map[string]any{"ten": "Bulk A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
				ExternalRef: "bulk-rollback-a",
			},
			{
				DomainID:   "noi_bo_hop_dong",
				NodeType:   "HopDongMau",
				Properties: map[string]any{},
			},
			{
				DomainID:    "noi_bo_hop_dong",
				NodeType:    "HopDongMau",
				Properties:  map[string]any{"ten": "Bulk C", "bridge_dinh_kem_ids": []any{"appendix-2"}},
				ExternalRef: "bulk-rollback-c",
			},
			{
				DomainID:    "noi_bo_hop_dong",
				NodeType:    "HopDongMau",
				Properties:  map[string]any{"ten": "Bulk D", "bridge_dinh_kem_ids": []any{"appendix-1"}},
				ExternalRef: "bulk-rollback-d",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNodesBulkWithContext() error = %v", err)
	}
	if len(resp.Succeeded) != 2 {
		t.Fatalf("succeeded len = %d, want 2", len(resp.Succeeded))
	}
	if _, ok := store.GetNodeByExternalRef("bulk-rollback-a"); !ok {
		t.Fatal("first node missing from store")
	}
	if _, ok := store.GetNodeByExternalRef("bulk-rollback-d"); !ok {
		t.Fatal("third node missing from store")
	}
	if len(resp.Failed) != 2 {
		t.Fatalf("failed len = %d, want 2", len(resp.Failed))
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

func TestCreateNodeReusesExternalRefCanonicalUUID(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	first, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "One", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "dup-ref",
	})
	if err != nil {
		t.Fatalf("first CreateNode() error = %v", err)
	}

	second, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Two", "ghi_chu": "updated", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "dup-ref",
	})
	if err != nil {
		t.Fatalf("second CreateNode() error = %v", err)
	}
	if second.NodeID != first.NodeID {
		t.Fatalf("second node id = %q, want %q", second.NodeID, first.NodeID)
	}
	if !looksLikeUUID(second.NodeID) {
		t.Fatalf("node id = %q, want UUID", second.NodeID)
	}
	node, ok := store.GetNodeByID(first.NodeID)
	if !ok {
		t.Fatalf("GetNodeByID(%s) missing", first.NodeID)
	}
	if got := node.Properties["ghi_chu"]; got != "updated" {
		t.Fatalf("updated property = %v, want updated", got)
	}
}

func TestCreateRelationshipsBulkCreatesEachRelationship(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	fromOne, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Rel Bulk A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode(fromOne) error = %v", err)
	}
	toOne, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Rel Bulk B"},
	})
	if err != nil {
		t.Fatalf("CreateNode(toOne) error = %v", err)
	}
	toTwo, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Rel Bulk C"},
	})
	if err != nil {
		t.Fatalf("CreateNode(toTwo) error = %v", err)
	}

	resp, err := svc.CreateRelationshipsBulkWithContext(context.Background(), actor, RelationshipBulkCreateRequest{
		Relationships: []RelationshipCreateRequest{
			{
				RelType:    "THAM_CHIEU",
				FromNodeID: fromOne.NodeID,
				ToNodeID:   toOne.NodeID,
				DomainID:   "noi_bo_hop_dong",
				Properties: map[string]any{},
			},
			{
				RelType:    "THAM_CHIEU",
				FromNodeID: fromOne.NodeID,
				ToNodeID:   toTwo.NodeID,
				DomainID:   "noi_bo_hop_dong",
				Properties: map[string]any{},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateRelationshipsBulkWithContext() error = %v", err)
	}
	if len(resp.Succeeded) != 2 {
		t.Fatalf("succeeded len = %d, want 2", len(resp.Succeeded))
	}
	if len(store.ListOutboxEvents()) != 5 {
		t.Fatalf("outbox len = %d, want 5", len(store.ListOutboxEvents()))
	}
}

func TestCreateRelationshipRejectsCrossGraphScopeEndpoints(t *testing.T) {
	svc, _ := newTestService(t)
	before := telemetry.Default().Snapshot().GraphScopeConflictCount
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-1111-111111111111",
		AppType:  "admin_tool",
	}

	projectA, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Graph Scope A", "bridge_dinh_kem_ids": []any{"appendix-1"}, "project_id": "project-a"},
	})
	if err != nil {
		t.Fatalf("CreateNode(projectA) error = %v", err)
	}
	projectB, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Graph Scope B", "project_id": "project-b"},
	})
	if err != nil {
		t.Fatalf("CreateNode(projectB) error = %v", err)
	}

	_, err = svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:    "THAM_CHIEU",
		FromNodeID: projectA.NodeID,
		ToNodeID:   projectB.NodeID,
		DomainID:   "noi_bo_hop_dong",
		Properties: map[string]any{},
	})
	if err == nil {
		t.Fatal("CreateRelationship() error = nil, want graph scope validation failure")
	}
	if !strings.Contains(err.Error(), "same graph scope") {
		t.Fatalf("CreateRelationship() error = %v, want graph scope failure", err)
	}
	after := telemetry.Default().Snapshot().GraphScopeConflictCount
	if after != before+1 {
		t.Fatalf("GraphScopeConflictCount = %d, want %d", after, before+1)
	}
}

func TestCreateRelationshipAllowsSameGraphScopeEndpoints(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-1111-111111111111",
		AppType:  "admin_tool",
	}

	projectA, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Graph Scope A", "bridge_dinh_kem_ids": []any{"appendix-1"}, "project_id": "project-a"},
	})
	if err != nil {
		t.Fatalf("CreateNode(projectA) error = %v", err)
	}
	projectAClause, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Graph Scope A Clause", "project_id": "project-a"},
	})
	if err != nil {
		t.Fatalf("CreateNode(projectAClause) error = %v", err)
	}

	created, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:    "THAM_CHIEU",
		FromNodeID: projectA.NodeID,
		ToNodeID:   projectAClause.NodeID,
		DomainID:   "noi_bo_hop_dong",
		Properties: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}
	if created.RelationshipID == "" {
		t.Fatal("relationship id is empty")
	}
	if created.GraphIdentifierID == "" || created.GraphVersionID == "" || created.ReferenceID == "" || created.GraphVersionNumber == 0 {
		t.Fatalf("graph lineage missing from relationship response: %+v", created)
	}
}

func TestCreateNodeRejectsCrossTenantWriteWithoutGrant(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
		AppType:  "agent_consumer",
	}

	_, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "shared-domain",
		NodeType:   "SharedDocument",
		Properties: map[string]any{"title": "Blocked shared document"},
	})
	if err == nil {
		t.Fatal("CreateNode() error = nil, want forbidden")
	}
	if err != ErrForbidden {
		t.Fatalf("CreateNode() error = %v, want forbidden", err)
	}
}

func TestCrossTenantWriteGrantAllowsAndRevokeDeniesMutation(t *testing.T) {
	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}
	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)
	accessSvc := access.NewService(accessStore, &cache)
	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologyService := ontology.NewService(ontologyStore, accessResolver)
	if _, err := ontologyService.CreateDomain(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", ontology.DomainCreateRequest{
		ID:   "noi_bo_hop_dong",
		Name: "Noi Bo Hop Dong",
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologyService.CreateNodeType(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "HopDongMau",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType() error = %v", err)
	}
	store := NewMemoryStore()
	seedBridgeTarget(store)
	svc := NewService(store, ontologyService, accessResolver, &recordingSessionManager{}, nil)
	grantActor := access.Identity{
		TenantID: "22222222-2222-2222-2222-222222222222",
		AppID:    "22222222-bbbb-2222-bbbb-222222222222",
		AppType:  "admin_tool",
	}
	writeActor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
		AppType:  "agent_consumer",
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	createdGrant, err := accessSvc.CreateGrant(grantActor, access.GrantCreateRequest{
		GranteeTenantID: writeActor.TenantID,
		GranteeAppID:    writeActor.AppID,
		ScopeType:       "domain",
		ScopeValue:      "shared-domain",
		Permission:      "write",
		ExpiresAt:       &expiresAt,
	})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}
	if createdGrant.ID == "" {
		t.Fatal("created grant id is empty")
	}

	created, err := svc.CreateNode(writeActor, NodeCreateRequest{
		DomainID:   "shared-domain",
		NodeType:   "SharedDocument",
		Properties: map[string]any{"title": "Granted shared document"},
	})
	if err != nil {
		t.Fatalf("CreateNode() with grant error = %v", err)
	}
	if created.NodeID == "" {
		t.Fatal("created node id is empty")
	}

	if _, err := accessSvc.RevokeGrant(grantActor, createdGrant.ID); err != nil {
		t.Fatalf("RevokeGrant() error = %v", err)
	}
	_, err = svc.CreateNode(writeActor, NodeCreateRequest{
		DomainID:   "shared-domain",
		NodeType:   "SharedDocument",
		Properties: map[string]any{"title": "Blocked again"},
	})
	if err == nil {
		t.Fatal("CreateNode() after revoke error = nil, want forbidden")
	}
	if err != ErrForbidden {
		t.Fatalf("CreateNode() after revoke error = %v, want forbidden", err)
	}
}

func TestCreateNodeIgnoresCallerSuppliedTenantAndAppFields(t *testing.T) {
	svc, store := newTestService(t)
	handler := NewHandler(svc)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{"tenant_id":"22222222-2222-2222-2222-222222222222","app_id":"22222222-bbbb-2222-bbbb-222222222222","domain_id":"noi_bo_hop_dong","node_type":"HopDongMau","properties":{"ten":"Hop dong bi gia mao","bridge_dinh_kem_ids":["appendix-1"]}}`))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()

	handler.CreateNode(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload NodeCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	node, ok := store.GetNodeByID(payload.NodeID)
	if !ok {
		t.Fatalf("GetNodeByID(%s) missing", payload.NodeID)
	}
	if node.OwnerTenantID != actor.TenantID || node.OwnerAppID != actor.AppID {
		t.Fatalf("node owner = %s:%s, want %s:%s", node.OwnerTenantID, node.OwnerAppID, actor.TenantID, actor.AppID)
	}
}

func TestCreateNodePersistsExternalRefAndUpdateKeepsUniqueness(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "ext-1",
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	node, ok := store.GetNodeByExternalRef("ext-1")
	if !ok || node.ID != created.NodeID {
		t.Fatalf("GetNodeByExternalRef() = %#v ok=%v", node, ok)
	}

	updated, err := svc.UpdateNode(actor, created.NodeID, NodeUpdateRequest{
		ExternalRef: "ext-2",
		Properties:  map[string]any{"ghi_chu": "cap nhat"},
	})
	if err != nil {
		t.Fatalf("UpdateNode() error = %v", err)
	}
	if updated.NodeID != created.NodeID {
		t.Fatalf("updated node id = %q, want %q", updated.NodeID, created.NodeID)
	}
	if _, ok := store.GetNodeByExternalRef("ext-1"); ok {
		t.Fatal("old external_ref still resolvable, want removed")
	}
	node, ok = store.GetNodeByExternalRef("ext-2")
	if !ok || node.ID != created.NodeID {
		t.Fatalf("updated external_ref lookup = %#v ok=%v", node, ok)
	}
}

func TestDeleteNodePreventsFurtherMutationAndKeepsSoftDelete(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "ext-delete",
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
	node, ok := store.GetNodeByID(created.NodeID)
	if !ok || !node.IsDeleted {
		t.Fatalf("deleted node = %#v ok=%v", node, ok)
	}
	if _, err := svc.UpdateNode(actor, created.NodeID, NodeUpdateRequest{Properties: map[string]any{"ghi_chu": "late update"}}); err == nil {
		t.Fatal("UpdateNode() error = nil after delete, want not found")
	}
}

func TestDeleteNodesByExternalRefPrefixDeletesMatchingNodes(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	first, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Prefix A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "codegraph/a",
	})
	if err != nil {
		t.Fatalf("CreateNode(first) error = %v", err)
	}
	second, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Prefix B", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "codegraph/b",
	})
	if err != nil {
		t.Fatalf("CreateNode(second) error = %v", err)
	}
	_, err = svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Other", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		ExternalRef: "other/c",
	})
	if err != nil {
		t.Fatalf("CreateNode(other) error = %v", err)
	}

	resp, err := svc.DeleteNodesByExternalRefPrefixWithContext(context.Background(), actor, NodeDeleteByExternalRefPrefixRequest{
		ExternalRefPrefix: "codegraph/",
	})
	if err != nil {
		t.Fatalf("DeleteNodesByExternalRefPrefixWithContext() error = %v", err)
	}
	if resp.Count != 2 {
		t.Fatalf("count = %d, want 2", resp.Count)
	}
	if len(resp.NodeIDs) != 2 || resp.NodeIDs[0] != first.NodeID || resp.NodeIDs[1] != second.NodeID {
		t.Fatalf("node_ids = %#v, want deleted ids", resp.NodeIDs)
	}
	if node, ok := store.GetNodeByID(first.NodeID); !ok || !node.IsDeleted {
		t.Fatalf("first node not soft-deleted: %#v ok=%v", node, ok)
	}
	if node, ok := store.GetNodeByID(second.NodeID); !ok || !node.IsDeleted {
		t.Fatalf("second node not soft-deleted: %#v ok=%v", node, ok)
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

func TestBulkHandlersAndPrefixDeleteReturnExpectedShapes(t *testing.T) {
	svc, _ := newTestService(t)
	handler := NewHandler(svc)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	nodesReq := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes/bulk", strings.NewReader(`{"nodes":[{"domain_id":"noi_bo_hop_dong","node_type":"HopDongMau","properties":{"ten":"Bulk H","bridge_dinh_kem_ids":["appendix-1"]}},{"domain_id":"noi_bo_hop_dong","node_type":"HopDongMau","properties":{"ten":"Bulk I","bridge_dinh_kem_ids":["appendix-1"]}}]}`))
	nodesReq = nodesReq.WithContext(access.ContextWithIdentity(nodesReq.Context(), actor))
	nodesRec := httptest.NewRecorder()
	handler.CreateNodesBulk(nodesRec, nodesReq)
	if nodesRec.Code != http.StatusAccepted {
		t.Fatalf("CreateNodesBulk status = %d body=%s", nodesRec.Code, nodesRec.Body.String())
	}
	if !strings.Contains(nodesRec.Body.String(), `"succeeded"`) {
		t.Fatalf("CreateNodesBulk body = %s", nodesRec.Body.String())
	}

	fromNode, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Bulk Rel From", "bridge_dinh_kem_ids": []any{"appendix-1"}},
	})
	if err != nil {
		t.Fatalf("CreateNode(fromNode) error = %v", err)
	}
	toNode, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Bulk Rel To"},
	})
	if err != nil {
		t.Fatalf("CreateNode(toNode) error = %v", err)
	}

	relReq := httptest.NewRequest(http.MethodPost, "/v1/kg/write/relationships/bulk", strings.NewReader(`{"relationships":[{"rel_type":"THAM_CHIEU","from_node_id":"`+fromNode.NodeID+`","to_node_id":"`+toNode.NodeID+`","domain_id":"noi_bo_hop_dong","properties":{}},{"rel_type":"THAM_CHIEU","from_node_id":"`+fromNode.NodeID+`","to_node_id":"`+toNode.NodeID+`","domain_id":"noi_bo_hop_dong","properties":{}}]}`))
	relReq = relReq.WithContext(access.ContextWithIdentity(relReq.Context(), actor))
	relRec := httptest.NewRecorder()
	handler.CreateRelationshipsBulk(relRec, relReq)
	if relRec.Code != http.StatusCreated {
		t.Fatalf("CreateRelationshipsBulk status = %d body=%s", relRec.Code, relRec.Body.String())
	}
	if !strings.Contains(relRec.Body.String(), `"succeeded"`) {
		t.Fatalf("CreateRelationshipsBulk body = %s", relRec.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/v1/kg/write/nodes:by-external-ref-prefix", strings.NewReader(`{"external_ref_prefix":"codegraph/"}`))
	delReq = delReq.WithContext(access.ContextWithIdentity(delReq.Context(), actor))
	delRec := httptest.NewRecorder()
	handler.DeleteNodesByExternalRefPrefix(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("DeleteNodesByExternalRefPrefix status = %d body=%s", delRec.Code, delRec.Body.String())
	}
}

func TestWriteHandlerReturnsStandardErrorEnvelope(t *testing.T) {
	svc, _ := newTestService(t)
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{`))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}))
	rec := httptest.NewRecorder()
	handler.CreateNode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("payload = %#v", payload)
	}
	if errObj["code"] == "" || errObj["message"] == "" {
		t.Fatalf("error payload = %#v", errObj)
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

func TestIngestDocumentCreatesLookupJob(t *testing.T) {
	svc, _ := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "ingestion_producer",
	}

	job, err := svc.IngestDocument(actor, IngestDocumentRequest{
		FileURL:    "s3://bucket/doc.pdf",
		LoaiVanBan: "NghiDinh",
		DomainID:   "sample-policy",
	})
	if err != nil {
		t.Fatalf("IngestDocument() error = %v", err)
	}
	if job.Status != "queued" || job.JobID == "" {
		t.Fatalf("IngestDocument() job = %+v", job)
	}

	lookup, err := svc.GetIngestJob(actor, job.JobID)
	if err != nil {
		t.Fatalf("GetIngestJob() error = %v", err)
	}
	if lookup.Status != "completed" {
		t.Fatalf("GetIngestJob() status = %q, want completed", lookup.Status)
	}
}

func TestIngestDocumentHandlerReturnsAccepted(t *testing.T) {
	svc, _ := newTestService(t)
	handler := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/ingest/document", strings.NewReader(`{"file_url":"s3://bucket/doc.pdf","loai_van_ban":"NghiDinh","domain_id":"sample-policy"}`))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "ingestion_producer",
	}))
	rec := httptest.NewRecorder()

	handler.IngestDocument(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"job_id"`) {
		t.Fatalf("body = %s", rec.Body.String())
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
	return NewService(store, ontologyService, accessResolver, &recordingSessionManager{}, nil), store
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
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	sessionManager := &recordingSessionManager{}
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
	if len(sessionManager.LastScope.Statements) != 4 {
		t.Fatalf("session statements len = %d, want 4", len(sessionManager.LastScope.Statements))
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
	if !looksLikeUUID(resp.RelationshipID) {
		t.Fatalf("relationship id = %q, want UUID", resp.RelationshipID)
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
	svc := NewService(store, ontologyService, accessResolver, &recordingSessionManager{}, auditLogger)
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
	store.nodes[bridgeTargetID] = NodeRecord{
		ID:            bridgeTargetID,
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
	store.externalRefs["appendix-1"] = bridgeTargetID
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func looksLikeUUID(value string) bool {
	return uuidPattern.MatchString(strings.TrimSpace(value))
}
