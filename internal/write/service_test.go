package write

import (
	"testing"
	"time"
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
