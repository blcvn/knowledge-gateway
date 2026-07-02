package write

import (
	"context"
	"testing"
	"time"

	"kg-service/internal/access"
)

func TestEstimateSyncETA_WithHistory(t *testing.T) {
	now := time.Now().UTC()
	store := NewMemoryStore()
	for i, lag := range []time.Duration{200 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond, 800 * time.Millisecond, 1200 * time.Millisecond} {
		nodeID := "node-" + string(rune('a'+i))
		node := NodeRecord{
			ID:            nodeID,
			DomainID:      "domain-1",
			DomainVersion: i + 1,
			CreatedAt:     now.Add(-time.Minute),
			UpdatedAt:     now.Add(-time.Minute),
		}
		_ = store.CreateNodeWithOutbox(nil, node, OutboxEvent{ID: "evt-" + nodeID, AggregateType: "kg_node", AggregateID: nodeID, CreatedAt: now.Add(-time.Minute)})
		_ = store.UpsertProjectionVersion(nil, ProjectionVersionRecord{
			EntityID:          nodeID,
			EntityKind:        "kg_node",
			SourceVersion:     int64(i + 1),
			SourceEventID:     "evt-" + nodeID,
			SourceUpdatedAt:   now.Add(-time.Minute),
			GraphBackend:      "graph",
			GraphVersion:      int64(i + 1),
			LastGraphSyncedAt: now.Add(-time.Minute).Add(lag),
		})
	}
	if got := estimateSyncETA(store, "domain-1", 5000); got != 1200 {
		t.Fatalf("estimateSyncETA() = %d, want 1200", got)
	}
}

func TestEstimateSyncETA_FallbackDefault(t *testing.T) {
	now := time.Now().UTC()
	store := NewMemoryStore()
	for i := 0; i < 3; i++ {
		nodeID := "node-" + string(rune('a'+i))
		node := NodeRecord{
			ID:            nodeID,
			DomainID:      "domain-1",
			DomainVersion: i + 1,
			CreatedAt:     now.Add(-time.Minute),
			UpdatedAt:     now.Add(-time.Minute),
		}
		_ = store.CreateNodeWithOutbox(nil, node, OutboxEvent{ID: "evt-" + nodeID, AggregateType: "kg_node", AggregateID: nodeID, CreatedAt: now.Add(-time.Minute)})
		_ = store.UpsertProjectionVersion(nil, ProjectionVersionRecord{
			EntityID:          nodeID,
			EntityKind:        "kg_node",
			SourceVersion:     int64(i + 1),
			SourceEventID:     "evt-" + nodeID,
			SourceUpdatedAt:   now.Add(-time.Minute),
			GraphBackend:      "graph",
			GraphVersion:      int64(i + 1),
			LastGraphSyncedAt: now.Add(-time.Minute).Add(800 * time.Millisecond),
		})
	}
	if got := estimateSyncETA(store, "domain-1", 5000); got != 5000 {
		t.Fatalf("estimateSyncETA() = %d, want 5000", got)
	}
}

func TestEntitySyncStatusRequiresGraphProjectionHead(t *testing.T) {
	svc, store := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	created, err := svc.CreateNode(actor, NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Graph head status", "bridge_dinh_kem_ids": []any{bridgeTargetID}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	event := store.ListOutboxEvents()[0]
	now := time.Now().UTC()
	if err := store.UpsertProjectionVersion(context.Background(), ProjectionVersionRecord{
		EntityID:          created.NodeID,
		EntityKind:        "kg_node",
		SourceVersion:     int64(created.DomainVersion),
		SourceEventID:     event.ID,
		SourceUpdatedAt:   now,
		GraphBackend:      "graph",
		GraphVersion:      int64(created.DomainVersion),
		LastGraphSyncedAt: now,
	}); err != nil {
		t.Fatalf("UpsertProjectionVersion() error = %v", err)
	}

	status, err := svc.EntitySyncStatus(created.NodeID, "kg_node")
	if err != nil {
		t.Fatalf("EntitySyncStatus() error = %v", err)
	}
	if ready, _ := status["graph_projection_ready"].(bool); ready {
		t.Fatalf("status = %#v, want graph projection to remain unreadable without a graph head", status)
	}
	if status["graph_lag_class"] == "SYNCED" {
		t.Fatalf("status = %#v, want graph lag to remain non-synced without a graph head", status)
	}

	if err := store.UpsertGraphProjectionHead(context.Background(), GraphProjectionHeadRecord{
		IdentifierID:         created.GraphIdentifierID,
		BackendKind:          "graph",
		BackendName:          "",
		AppliedVersionID:     created.GraphVersionID,
		AppliedVersionNumber: created.GraphVersionNumber,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("UpsertGraphProjectionHead() error = %v", err)
	}

	status, err = svc.EntitySyncStatus(created.NodeID, "kg_node")
	if err != nil {
		t.Fatalf("EntitySyncStatus() error = %v", err)
	}
	if ready, _ := status["graph_projection_ready"].(bool); !ready {
		t.Fatalf("status = %#v, want graph projection ready after head advancement", status)
	}
	if status["graph_lag_class"] != "SYNCED" {
		t.Fatalf("status = %#v, want synced graph lag after head advancement", status)
	}
}
