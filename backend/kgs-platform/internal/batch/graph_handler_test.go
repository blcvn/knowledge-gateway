package batch

import (
	"context"
	"errors"
	"testing"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
)

type fakeOverlayAppender struct {
	entityCalls int
	edgeCalls   int
	lastEntity  map[string]any
	lastEdge    map[string]any
}

func (f *fakeOverlayAppender) AddEntityDelta(ctx context.Context, overlayID, namespace, label string, properties map[string]any) (map[string]any, error) {
	f.entityCalls++
	f.lastEntity = cloneProperties(properties)
	return map[string]any{"overlay_id": overlayID}, nil
}

func (f *fakeOverlayAppender) AddEdgeDelta(ctx context.Context, overlayID, namespace, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error) {
	f.edgeCalls++
	f.lastEdge = cloneProperties(properties)
	return map[string]any{"overlay_id": overlayID}, nil
}

func TestGraphBatchHandlerAtomicEntitiesAndEdges(t *testing.T) {
	db := newBatchWriterTestDB(t)
	h := NewGraphBatchHandler(db, nil)

	req := GraphBatchRequest{
		Entities: []Entity{
			{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001"}},
			{Label: "UseCase", Properties: map[string]any{"id": "b2222222-2222-2222-2222-222222222222", "name": "UC-001"}},
		},
		Edges: []Edge{
			{EdgeID: "c3333333-3333-3333-3333-333333333333", FromEntityID: "a1111111-1111-1111-1111-111111111111", ToEntityID: "b2222222-2222-2222-2222-222222222222", RelationType: "DEPENDS_ON"},
		},
	}

	res, err := h.UpsertGraph(context.Background(), req, "app-1", "tenant-1")
	if err != nil {
		t.Fatalf("UpsertGraph error: %v", err)
	}
	if res.EntitiesCreated != 2 || res.EdgesCreated != 1 {
		t.Fatalf("unexpected result: %#v", res)
	}

	var entityCount int64
	if err := db.Model(&data.KGEntity{}).Count(&entityCount).Error; err != nil {
		t.Fatalf("count entities: %v", err)
	}
	if entityCount != 2 {
		t.Fatalf("expected 2 entities, got %d", entityCount)
	}

	var edgeCount int64
	if err := db.Model(&data.KGEdge{}).Count(&edgeCount).Error; err != nil {
		t.Fatalf("count edges: %v", err)
	}
	if edgeCount != 1 {
		t.Fatalf("expected 1 edge, got %d", edgeCount)
	}

	var outboxCount int64
	if err := db.Model(&data.KGSyncOutbox{}).Count(&outboxCount).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 3 {
		t.Fatalf("expected 3 outbox records, got %d", outboxCount)
	}
}

func TestGraphBatchHandlerEdgeFKFailRollsBack(t *testing.T) {
	db := newBatchWriterTestDB(t)
	h := NewGraphBatchHandler(db, nil)

	req := GraphBatchRequest{
		Entities: []Entity{
			{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001"}},
		},
		Edges: []Edge{
			{EdgeID: "c3333333-3333-3333-3333-333333333333", FromEntityID: "a1111111-1111-1111-1111-111111111111", ToEntityID: "missing-entity", RelationType: "DEPENDS_ON"},
		},
	}

	_, err := h.UpsertGraph(context.Background(), req, "app-1", "tenant-1")
	if err == nil {
		t.Fatalf("expected FK error")
	}

	var entityCount int64
	if err := db.Model(&data.KGEntity{}).Count(&entityCount).Error; err != nil {
		t.Fatalf("count entities: %v", err)
	}
	if entityCount != 0 {
		t.Fatalf("expected rollback to keep entity count=0, got %d", entityCount)
	}
}

func TestGraphBatchHandlerConflictPolicySkip(t *testing.T) {
	db := newBatchWriterTestDB(t)
	h := NewGraphBatchHandler(db, nil)

	seed := Entity{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001"}}
	if _, err := NewPGWriter(db).BulkCreate(context.Background(), "app-1", "tenant-1", []Entity{seed}); err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	req := GraphBatchRequest{
		ConflictPolicy: ConflictPolicySkip,
		Entities: []Entity{
			{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001 duplicate"}},
			{Label: "UseCase", Properties: map[string]any{"id": "b2222222-2222-2222-2222-222222222222", "name": "UC-001"}},
		},
	}

	res, err := h.UpsertGraph(context.Background(), req, "app-1", "tenant-1")
	if err != nil {
		t.Fatalf("UpsertGraph error: %v", err)
	}
	if res.EntitiesSkipped != 1 || res.EntitiesCreated != 1 || res.Conflicted != 1 {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestGraphBatchHandlerConflictPolicyFailFast(t *testing.T) {
	db := newBatchWriterTestDB(t)
	h := NewGraphBatchHandler(db, nil)

	seed := Entity{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001"}}
	if _, err := NewPGWriter(db).BulkCreate(context.Background(), "app-1", "tenant-1", []Entity{seed}); err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	req := GraphBatchRequest{
		ConflictPolicy: ConflictPolicyFailFast,
		Entities: []Entity{
			{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001 duplicate"}},
			{Label: "UseCase", Properties: map[string]any{"id": "b2222222-2222-2222-2222-222222222222", "name": "UC-001"}},
		},
	}

	_, err := h.UpsertGraph(context.Background(), req, "app-1", "tenant-1")
	if err == nil {
		t.Fatalf("expected fail-fast conflict error")
	}

	var entityCount int64
	if err := db.Model(&data.KGEntity{}).Count(&entityCount).Error; err != nil {
		t.Fatalf("count entities: %v", err)
	}
	if entityCount != 1 {
		t.Fatalf("expected rollback of new entity, got entity count=%d", entityCount)
	}
}

func TestGraphBatchHandlerOverlayPath(t *testing.T) {
	db := newBatchWriterTestDB(t)
	overlay := &fakeOverlayAppender{}
	h := NewGraphBatchHandler(db, overlay)

	overlayID := "ov-1"
	req := GraphBatchRequest{
		OverlayID: &overlayID,
		Entities: []Entity{
			{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001"}},
		},
		Edges: []Edge{
			{EdgeID: "c3333333-3333-3333-3333-333333333333", FromEntityID: "a1111111-1111-1111-1111-111111111111", ToEntityID: "b2222222-2222-2222-2222-222222222222", RelationType: "DEPENDS_ON"},
		},
	}

	res, err := h.UpsertGraph(context.Background(), req, "app-1", "tenant-1")
	if err != nil {
		t.Fatalf("UpsertGraph overlay error: %v", err)
	}
	if res.EntitiesCreated != 1 || res.EdgesCreated != 1 {
		t.Fatalf("unexpected overlay result: %#v", res)
	}
	if overlay.entityCalls != 1 || overlay.edgeCalls != 1 {
		t.Fatalf("expected overlay appender calls 1/1, got %d/%d", overlay.entityCalls, overlay.edgeCalls)
	}
	if got := toString(overlay.lastEntity["id"]); got != "a1111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected entity id propagated to overlay properties, got %q", got)
	}
	if got := toString(overlay.lastEdge["id"]); got != "c3333333-3333-3333-3333-333333333333" {
		t.Fatalf("expected edge id propagated to overlay properties, got %q", got)
	}
}

func TestGraphBatchHandlerOverlayPathRequiresEntityID(t *testing.T) {
	db := newBatchWriterTestDB(t)
	overlay := &fakeOverlayAppender{}
	h := NewGraphBatchHandler(db, overlay)

	overlayID := "ov-1"
	req := GraphBatchRequest{
		OverlayID: &overlayID,
		Entities: []Entity{
			{Label: "Requirement", Properties: map[string]any{"name": "FR-001"}},
		},
	}

	_, err := h.UpsertGraph(context.Background(), req, "app-1", "tenant-1")
	if !errors.Is(err, ErrOverlayEntityIDRequired) {
		t.Fatalf("expected ErrOverlayEntityIDRequired, got %v", err)
	}
	if overlay.entityCalls != 0 || overlay.edgeCalls != 0 {
		t.Fatalf("expected no overlay appender calls, got %d/%d", overlay.entityCalls, overlay.edgeCalls)
	}
}

func TestGraphBatchHandlerOverlayPathRequiresEdgeID(t *testing.T) {
	db := newBatchWriterTestDB(t)
	overlay := &fakeOverlayAppender{}
	h := NewGraphBatchHandler(db, overlay)

	overlayID := "ov-1"
	req := GraphBatchRequest{
		OverlayID: &overlayID,
		Entities: []Entity{
			{Label: "Requirement", Properties: map[string]any{"id": "a1111111-1111-1111-1111-111111111111", "name": "FR-001"}},
		},
		Edges: []Edge{
			{FromEntityID: "a1111111-1111-1111-1111-111111111111", ToEntityID: "b2222222-2222-2222-2222-222222222222", RelationType: "DEPENDS_ON"},
		},
	}

	_, err := h.UpsertGraph(context.Background(), req, "app-1", "tenant-1")
	if !errors.Is(err, ErrOverlayEdgeIDRequired) {
		t.Fatalf("expected ErrOverlayEdgeIDRequired, got %v", err)
	}
	if overlay.entityCalls != 1 || overlay.edgeCalls != 0 {
		t.Fatalf("expected only entity overlay call before edge validation fails, got %d/%d", overlay.entityCalls, overlay.edgeCalls)
	}
}
