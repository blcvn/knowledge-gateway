package write

import (
	"context"
	"strings"
	"testing"

	"kg-service/internal/access"
)

// adminActor is the identity the relationship fixtures in this file write as. It mirrors the actor
// used by the pre-existing relationship tests so the seeded ontology applies unchanged.
func adminActor() access.Identity {
	return access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-1111-111111111111",
		AppType:  "admin_tool",
	}
}

// seedRelationshipEndpoints creates two nodes in the same graph scope, which is the precondition
// ensureSameGraphScope enforces on every relationship write.
func seedRelationshipEndpoints(t *testing.T, svc *Service, actor access.Identity, project string) (from, to NodeCreateResponse) {
	t.Helper()
	var err error
	from, err = svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Contract " + project, "bridge_dinh_kem_ids": []any{"appendix-1"}, "project_id": project},
	})
	if err != nil {
		t.Fatalf("CreateNode(from) error = %v", err)
	}
	to, err = svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "KhoanMau",
		Properties: map[string]any{"ten": "Clause " + project, "project_id": project},
	})
	if err != nil {
		t.Fatalf("CreateNode(to) error = %v", err)
	}
	return from, to
}

// countLiveRelationshipsBetween counts only the edges these tests create. A global count would
// also pick up the bridge relationship that CreateNode mints for a HopDongMau carrying
// bridge_dinh_kem_ids, which has nothing to do with what is under test here.
func countLiveRelationshipsBetween(store *MemoryStore, fromNodeID, toNodeID string) int {
	count := 0
	for _, rel := range store.ListRelationships() {
		if rel.IsDeleted {
			continue
		}
		if rel.FromNodeID == fromNodeID && rel.ToNodeID == toNodeID {
			count++
		}
	}
	return count
}

// TestRelationshipExternalRefUpserts is the behaviour the executor integration depends on: a client
// that re-persists the same logical graph must not accumulate duplicate edges. Before external_ref
// existed, the second write below produced a second row.
func TestRelationshipExternalRefUpserts(t *testing.T) {
	svc, store := newTestService(t)
	actor := adminActor()
	from, to := seedRelationshipEndpoints(t, svc, actor, "project-upsert")

	first, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:     "THAM_CHIEU",
		FromNodeID:  from.NodeID,
		ToNodeID:    to.NodeID,
		DomainID:    "noi_bo_hop_dong",
		Properties:  map[string]any{"ghi_chu": "first"},
		ExternalRef: "bas/doc-1/e/edge-1",
	})
	if err != nil {
		t.Fatalf("CreateRelationship(first) error = %v", err)
	}

	second, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:     "THAM_CHIEU",
		FromNodeID:  from.NodeID,
		ToNodeID:    to.NodeID,
		DomainID:    "noi_bo_hop_dong",
		Properties:  map[string]any{"ghi_chu": "second"},
		ExternalRef: "bas/doc-1/e/edge-1",
	})
	if err != nil {
		t.Fatalf("CreateRelationship(second) error = %v", err)
	}

	if first.RelationshipID != second.RelationshipID {
		t.Fatalf("relationship id changed across upsert: first=%s second=%s", first.RelationshipID, second.RelationshipID)
	}
	if got := countLiveRelationshipsBetween(store, from.NodeID, to.NodeID); got != 1 {
		t.Fatalf("live relationship count = %d, want 1", got)
	}
	stored, ok := store.GetRelationshipByExternalRef("bas/doc-1/e/edge-1")
	if !ok {
		t.Fatal("GetRelationshipByExternalRef() missing after upsert")
	}
	if stored.Properties["ghi_chu"] != "second" {
		t.Fatalf("properties = %v, want the second write to have won", stored.Properties)
	}
}

// TestRelationshipExternalRefRevivesSoftDeleted covers the delete-then-rewrite cycle a scoped
// snapshot performs: an edge removed in one persist and re-created in the next must come back on
// the same row rather than collide with the tombstone.
func TestRelationshipExternalRefRevivesSoftDeleted(t *testing.T) {
	svc, store := newTestService(t)
	actor := adminActor()
	from, to := seedRelationshipEndpoints(t, svc, actor, "project-revive")

	created, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:     "THAM_CHIEU",
		FromNodeID:  from.NodeID,
		ToNodeID:    to.NodeID,
		DomainID:    "noi_bo_hop_dong",
		Properties:  map[string]any{},
		ExternalRef: "bas/doc-1/e/edge-revive",
	})
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}

	if _, err := svc.DeleteRelationshipsBulkWithContext(context.Background(), actor, RelationshipBulkDeleteRequest{
		RelationshipIDs: []string{created.RelationshipID},
	}); err != nil {
		t.Fatalf("DeleteRelationshipsBulkWithContext() error = %v", err)
	}
	if got := countLiveRelationshipsBetween(store, from.NodeID, to.NodeID); got != 0 {
		t.Fatalf("live relationship count after delete = %d, want 0", got)
	}

	revived, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:     "THAM_CHIEU",
		FromNodeID:  from.NodeID,
		ToNodeID:    to.NodeID,
		DomainID:    "noi_bo_hop_dong",
		Properties:  map[string]any{},
		ExternalRef: "bas/doc-1/e/edge-revive",
	})
	if err != nil {
		t.Fatalf("CreateRelationship(revive) error = %v", err)
	}
	if revived.RelationshipID != created.RelationshipID {
		t.Fatalf("revive created a new row: %s != %s", revived.RelationshipID, created.RelationshipID)
	}
	if got := countLiveRelationshipsBetween(store, from.NodeID, to.NodeID); got != 1 {
		t.Fatalf("live relationship count after revive = %d, want 1", got)
	}
}

// TestRelationshipWithoutExternalRefStillInserts pins the backward-compatible half of the change:
// callers that never send an external_ref keep getting insert-only behaviour, so two writes of the
// same edge remain two rows exactly as before.
func TestRelationshipWithoutExternalRefStillInserts(t *testing.T) {
	svc, store := newTestService(t)
	actor := adminActor()
	from, to := seedRelationshipEndpoints(t, svc, actor, "project-legacy")

	first, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:    "THAM_CHIEU",
		FromNodeID: from.NodeID,
		ToNodeID:   to.NodeID,
		DomainID:   "noi_bo_hop_dong",
		Properties: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateRelationship(first) error = %v", err)
	}
	second, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:    "THAM_CHIEU",
		FromNodeID: from.NodeID,
		ToNodeID:   to.NodeID,
		DomainID:   "noi_bo_hop_dong",
		Properties: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateRelationship(second) error = %v", err)
	}
	if first.RelationshipID == second.RelationshipID {
		t.Fatal("relationships without external_ref must not be deduplicated")
	}
	if got := countLiveRelationshipsBetween(store, from.NodeID, to.NodeID); got != 2 {
		t.Fatalf("live relationship count = %d, want 2", got)
	}
}

// TestRelationshipResolvesEndpointsByExternalRef covers the addressing half: a client that owns its
// identifiers can name both endpoints by their node external_ref and never learn the service UUIDs.
func TestRelationshipResolvesEndpointsByExternalRef(t *testing.T) {
	svc, store := newTestService(t)
	actor := adminActor()

	from, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Ref Contract", "bridge_dinh_kem_ids": []any{"appendix-1"}, "project_id": "project-ref"},
		ExternalRef: "bas/doc-1/n/node-from",
	})
	if err != nil {
		t.Fatalf("CreateNode(from) error = %v", err)
	}
	to, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "KhoanMau",
		Properties:  map[string]any{"ten": "Ref Clause", "project_id": "project-ref"},
		ExternalRef: "bas/doc-1/n/node-to",
	})
	if err != nil {
		t.Fatalf("CreateNode(to) error = %v", err)
	}

	created, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:         "THAM_CHIEU",
		FromExternalRef: "bas/doc-1/n/node-from",
		ToExternalRef:   "bas/doc-1/n/node-to",
		DomainID:        "noi_bo_hop_dong",
		Properties:      map[string]any{},
		ExternalRef:     "bas/doc-1/e/edge-by-ref",
	})
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}

	stored, ok := store.GetRelationshipByID(created.RelationshipID)
	if !ok {
		t.Fatal("relationship not stored")
	}
	if stored.FromNodeID != from.NodeID {
		t.Fatalf("from_node_id = %s, want %s", stored.FromNodeID, from.NodeID)
	}
	if stored.ToNodeID != to.NodeID {
		t.Fatalf("to_node_id = %s, want %s", stored.ToNodeID, to.NodeID)
	}
}

// TestRelationshipExplicitNodeIDWinsOverExternalRef keeps the resolution rule unambiguous: an
// explicit id is never silently overridden by a reference that points elsewhere.
func TestRelationshipExplicitNodeIDWinsOverExternalRef(t *testing.T) {
	svc, store := newTestService(t)
	actor := adminActor()

	from, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Explicit Contract", "bridge_dinh_kem_ids": []any{"appendix-1"}, "project_id": "project-explicit"},
		ExternalRef: "bas/doc-2/n/node-from",
	})
	if err != nil {
		t.Fatalf("CreateNode(from) error = %v", err)
	}
	to, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "KhoanMau",
		Properties:  map[string]any{"ten": "Explicit Clause", "project_id": "project-explicit"},
		ExternalRef: "bas/doc-2/n/node-to",
	})
	if err != nil {
		t.Fatalf("CreateNode(to) error = %v", err)
	}
	decoy, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "KhoanMau",
		Properties:  map[string]any{"ten": "Decoy Clause", "project_id": "project-explicit"},
		ExternalRef: "bas/doc-2/n/node-decoy",
	})
	if err != nil {
		t.Fatalf("CreateNode(decoy) error = %v", err)
	}

	created, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:         "THAM_CHIEU",
		FromNodeID:      from.NodeID,
		ToNodeID:        to.NodeID,
		FromExternalRef: "bas/doc-2/n/node-decoy",
		ToExternalRef:   "bas/doc-2/n/node-decoy",
		DomainID:        "noi_bo_hop_dong",
		Properties:      map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}
	stored, ok := store.GetRelationshipByID(created.RelationshipID)
	if !ok {
		t.Fatal("relationship not stored")
	}
	if stored.FromNodeID != from.NodeID || stored.ToNodeID != to.NodeID {
		t.Fatalf("explicit ids were overridden: from=%s to=%s decoy=%s", stored.FromNodeID, stored.ToNodeID, decoy.NodeID)
	}
}

// TestRelationshipUnknownExternalRefIsRejected proves the resolver fails loudly. Falling through
// with an empty endpoint would produce a relationship pointing at nothing.
func TestRelationshipUnknownExternalRefIsRejected(t *testing.T) {
	svc, _ := newTestService(t)
	actor := adminActor()
	_, to := seedRelationshipEndpoints(t, svc, actor, "project-unknown")

	_, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:         "THAM_CHIEU",
		FromExternalRef: "bas/doc-1/n/does-not-exist",
		ToNodeID:        to.NodeID,
		DomainID:        "noi_bo_hop_dong",
		Properties:      map[string]any{},
	})
	if err == nil {
		t.Fatal("CreateRelationship() error = nil, want unknown reference failure")
	}
	if !strings.Contains(err.Error(), "unknown from_external_ref") {
		t.Fatalf("CreateRelationship() error = %v, want unknown from_external_ref", err)
	}
}

// TestRelationshipBulkReportsUnknownExternalRefPerItem keeps a single bad item from failing a whole
// batch: the executor writes edges in large bulks and needs the index of the offender.
func TestRelationshipBulkReportsUnknownExternalRefPerItem(t *testing.T) {
	svc, _ := newTestService(t)
	actor := adminActor()
	from, to := seedRelationshipEndpoints(t, svc, actor, "project-bulk")

	resp, err := svc.CreateRelationshipsBulkWithContext(context.Background(), actor, RelationshipBulkCreateRequest{
		Relationships: []RelationshipCreateRequest{
			{
				RelType:     "THAM_CHIEU",
				FromNodeID:  from.NodeID,
				ToNodeID:    to.NodeID,
				DomainID:    "noi_bo_hop_dong",
				Properties:  map[string]any{},
				ExternalRef: "bas/doc-1/e/bulk-ok",
			},
			{
				RelType:         "THAM_CHIEU",
				FromExternalRef: "bas/doc-1/n/missing",
				ToNodeID:        to.NodeID,
				DomainID:        "noi_bo_hop_dong",
				Properties:      map[string]any{},
				ExternalRef:     "bas/doc-1/e/bulk-bad",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateRelationshipsBulkWithContext() error = %v", err)
	}
	if len(resp.Succeeded) != 1 {
		t.Fatalf("succeeded = %d, want 1", len(resp.Succeeded))
	}
	if len(resp.Failed) != 1 {
		t.Fatalf("failed = %d, want 1", len(resp.Failed))
	}
	if resp.Failed[0].Index != 1 {
		t.Fatalf("failed index = %d, want 1", resp.Failed[0].Index)
	}
	if !strings.Contains(resp.Failed[0].Error, "unknown from_external_ref") {
		t.Fatalf("failed error = %s, want unknown from_external_ref", resp.Failed[0].Error)
	}
}

// TestRelationshipExternalRefStillEnforcesGraphScope guards against the new addressing path
// becoming a way around the cross-scope rule. The check must run on the resolved endpoints.
func TestRelationshipExternalRefStillEnforcesGraphScope(t *testing.T) {
	svc, _ := newTestService(t)
	actor := adminActor()

	if _, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Scope A", "bridge_dinh_kem_ids": []any{"appendix-1"}, "project_id": "scope-a"},
		ExternalRef: "bas/doc-3/n/scope-a",
	}); err != nil {
		t.Fatalf("CreateNode(scope-a) error = %v", err)
	}
	if _, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "KhoanMau",
		Properties:  map[string]any{"ten": "Scope B", "project_id": "scope-b"},
		ExternalRef: "bas/doc-3/n/scope-b",
	}); err != nil {
		t.Fatalf("CreateNode(scope-b) error = %v", err)
	}

	_, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
		RelType:         "THAM_CHIEU",
		FromExternalRef: "bas/doc-3/n/scope-a",
		ToExternalRef:   "bas/doc-3/n/scope-b",
		DomainID:        "noi_bo_hop_dong",
		Properties:      map[string]any{},
	})
	if err == nil {
		t.Fatal("CreateRelationship() error = nil, want graph scope validation failure")
	}
	if !strings.Contains(err.Error(), "same graph scope") {
		t.Fatalf("CreateRelationship() error = %v, want graph scope failure", err)
	}
}
