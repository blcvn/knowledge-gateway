package overlay

import (
	"context"
	"errors"
	"testing"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"
)

func TestCommitV2WritesPostgresAndOutbox(t *testing.T) {
	ctx := context.Background()
	db := newOverlayTestDB(t)
	store := newMemoryStore()
	vm := &fakeVersionManager{
		versions: []version.GraphVersion{{ID: "v1", Namespace: "graph/app/tenant"}},
	}
	manager := &Manager{
		store:      store,
		db:         db,
		versionMgr: vm,
	}

	item, err := manager.Create(ctx, "graph/app/tenant", "s-1", "")
	if err != nil {
		t.Fatalf("create overlay: %v", err)
	}
	if _, err := manager.AddEntityDelta(ctx, item.OverlayID, "graph/app/tenant", "Requirement", map[string]any{"id": "e1", "name": "N1"}); err != nil {
		t.Fatalf("add entity1: %v", err)
	}
	if _, err := manager.AddEntityDelta(ctx, item.OverlayID, "graph/app/tenant", "Requirement", map[string]any{"id": "e2", "name": "N2"}); err != nil {
		t.Fatalf("add entity2: %v", err)
	}
	if _, err := manager.AddEdgeDelta(ctx, item.OverlayID, "graph/app/tenant", "RELATES_TO", "e1", "e2", map[string]any{"id": "r1"}); err != nil {
		t.Fatalf("add edge: %v", err)
	}

	result, err := manager.Commit(ctx, item.OverlayID, PolicyKeepOverlay)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if result.EntitiesCommitted != 2 || result.EdgesCommitted != 1 {
		t.Fatalf("unexpected commit result: %#v", result)
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

	if _, err := store.Get(ctx, item.OverlayID); err == nil {
		t.Fatalf("expected overlay removed after commit")
	}
}

func TestCommitV2DeleteWritesSoftDeleteAndOutbox(t *testing.T) {
	ctx := context.Background()
	db := newOverlayTestDB(t)
	store := newMemoryStore()
	vm := &fakeVersionManager{
		versions: []version.GraphVersion{{ID: "v1", Namespace: "graph/app/tenant"}},
	}
	manager := &Manager{
		store:      store,
		db:         db,
		versionMgr: vm,
	}

	seed := []data.KGEntity{
		{EntityID: "d1", AppID: "app", TenantID: "tenant", EntityType: "Requirement", Name: "D1", Properties: data.JSONMap{"id": "d1"}, Version: 1},
		{EntityID: "d2", AppID: "app", TenantID: "tenant", EntityType: "Requirement", Name: "D2", Properties: data.JSONMap{"id": "d2"}, Version: 1},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed entity %d: %v", i, err)
		}
	}
	edge := data.KGEdge{
		EdgeID:       "de1",
		AppID:        "app",
		TenantID:     "tenant",
		FromEntityID: "d1",
		ToEntityID:   "d2",
		RelationType: "RELATES_TO",
		Properties:   data.JSONMap{"id": "de1"},
	}
	if err := db.Create(&edge).Error; err != nil {
		t.Fatalf("seed edge: %v", err)
	}

	item, err := manager.Create(ctx, "graph/app/tenant", "s-2", "")
	if err != nil {
		t.Fatalf("create overlay: %v", err)
	}
	if err := manager.DeleteEntityDelta(ctx, item.OverlayID, "d1"); err != nil {
		t.Fatalf("delete node delta: %v", err)
	}
	if err := manager.DeleteEdgeDelta(ctx, item.OverlayID, "de1"); err != nil {
		t.Fatalf("delete edge delta: %v", err)
	}

	if _, err := manager.Commit(ctx, item.OverlayID, PolicyKeepOverlay); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var deletedEntity data.KGEntity
	if err := db.Where("entity_id = ?", "d1").Take(&deletedEntity).Error; err != nil {
		t.Fatalf("load deleted entity: %v", err)
	}
	if !deletedEntity.IsDeleted {
		t.Fatalf("expected entity soft deleted")
	}

	var deletedEdge data.KGEdge
	if err := db.Where("edge_id = ?", "de1").Take(&deletedEdge).Error; err != nil {
		t.Fatalf("load deleted edge: %v", err)
	}
	if !deletedEdge.IsDeleted {
		t.Fatalf("expected edge soft deleted")
	}

	var outbox []data.KGSyncOutbox
	if err := db.Order("id asc").Find(&outbox).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(outbox) != 2 {
		t.Fatalf("expected 2 outbox records, got %d", len(outbox))
	}
	if outbox[0].Op != data.OutboxOpDeleteEntity || outbox[1].Op != data.OutboxOpDeleteEdge {
		t.Fatalf("unexpected outbox ops: %#v", outbox)
	}
}

func TestCommitV2IdempotentRetryAfterCleanupFailure(t *testing.T) {
	ctx := context.Background()
	db := newOverlayTestDB(t)
	base := newMemoryStore()
	store := &failDeleteStore{memoryStore: base}
	vm := &fakeVersionManager{
		versions: []version.GraphVersion{{ID: "v1", Namespace: "graph/app/tenant"}},
	}
	manager := &Manager{
		store:      store,
		db:         db,
		versionMgr: vm,
	}

	item, err := manager.Create(ctx, "graph/app/tenant", "s-3", "")
	if err != nil {
		t.Fatalf("create overlay: %v", err)
	}
	if _, err := manager.AddEntityDelta(ctx, item.OverlayID, "graph/app/tenant", "Requirement", map[string]any{"id": "x1", "name": "X1"}); err != nil {
		t.Fatalf("add entity delta: %v", err)
	}

	store.failNextDelete = true
	if _, err := manager.Commit(ctx, item.OverlayID, PolicyKeepOverlay); err == nil {
		t.Fatalf("expected first commit to fail on delete cleanup")
	}

	var outboxCountAfterFirst int64
	if err := db.Model(&data.KGSyncOutbox{}).Count(&outboxCountAfterFirst).Error; err != nil {
		t.Fatalf("count outbox first: %v", err)
	}

	second, err := manager.Commit(ctx, item.OverlayID, PolicyKeepOverlay)
	if err != nil {
		t.Fatalf("second commit should be idempotent: %v", err)
	}
	if second.NewVersionID == "" {
		t.Fatalf("expected committed version id on idempotent retry")
	}

	var outboxCountAfterSecond int64
	if err := db.Model(&data.KGSyncOutbox{}).Count(&outboxCountAfterSecond).Error; err != nil {
		t.Fatalf("count outbox second: %v", err)
	}
	if outboxCountAfterSecond != outboxCountAfterFirst {
		t.Fatalf("expected no duplicate writes on retry, first=%d second=%d", outboxCountAfterFirst, outboxCountAfterSecond)
	}
}

func TestCommitV2OverlayNotActive(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	item := &OverlayGraph{
		OverlayID:     "ov-not-active",
		Namespace:     "graph/app/tenant",
		SessionID:     "s-4",
		BaseVersionID: "v1",
		Status:        StatusDiscarded,
	}
	if err := store.Save(ctx, item, 0); err != nil {
		t.Fatalf("save overlay: %v", err)
	}

	manager := &Manager{
		store: store,
		db:    newOverlayTestDB(t),
	}
	if _, err := manager.Commit(ctx, item.OverlayID, PolicyKeepOverlay); err == nil {
		t.Fatalf("expected overlay not active error")
	}
}

type failDeleteStore struct {
	*memoryStore
	failNextDelete bool
}

func (s *failDeleteStore) Delete(ctx context.Context, overlayID string) error {
	if s.failNextDelete {
		s.failNextDelete = false
		return errors.New("forced delete error")
	}
	return s.memoryStore.Delete(ctx, overlayID)
}
