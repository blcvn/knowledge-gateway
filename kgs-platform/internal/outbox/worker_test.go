package outbox

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeEntityEdgeSyncer struct {
	upsertEntityCalls int
	upsertEdgeCalls   int
	deleteEntityCalls int
	deleteEdgeCalls   int
	lastEntity        data.KGEntity
	lastEdge          data.KGEdge
	upsertEntityErr   error
	upsertEdgeErr     error
	deleteEntityErr   error
	deleteEdgeErr     error
}

func (f *fakeEntityEdgeSyncer) UpsertEntity(ctx context.Context, entity data.KGEntity) error {
	f.upsertEntityCalls++
	f.lastEntity = entity
	return f.upsertEntityErr
}
func (f *fakeEntityEdgeSyncer) UpsertEdge(ctx context.Context, edge data.KGEdge) error {
	f.upsertEdgeCalls++
	f.lastEdge = edge
	return f.upsertEdgeErr
}
func (f *fakeEntityEdgeSyncer) SoftDeleteEntity(ctx context.Context, entityID, tenantID string) error {
	f.deleteEntityCalls++
	return f.deleteEntityErr
}
func (f *fakeEntityEdgeSyncer) SoftDeleteEdge(ctx context.Context, edgeID, tenantID string) error {
	f.deleteEdgeCalls++
	return f.deleteEdgeErr
}

type fakeVectorSyncer struct {
	upsertCalls int
	deleteCalls int
	lastEntity  data.KGEntity
	upsertErr   error
	deleteErr   error
}

func (f *fakeVectorSyncer) UpsertVector(ctx context.Context, entity data.KGEntity) error {
	f.upsertCalls++
	f.lastEntity = entity
	return f.upsertErr
}
func (f *fakeVectorSyncer) DeleteVector(ctx context.Context, entityID, tenantID, appID string) error {
	f.deleteCalls++
	return f.deleteErr
}

func newOutboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&data.KGSyncOutbox{}); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	return db
}

func TestOutboxWorkerSyncOneUpsertEntity(t *testing.T) {
	neo := &fakeEntityEdgeSyncer{}
	vec := &fakeVectorSyncer{}
	w := &OutboxWorker{neo4jSyncer: neo, qdrantSyncer: vec, log: log.NewHelper(log.NewStdLogger(io.Discard))}

	payload := data.JSONMap{
		"entity_id":   "e-1",
		"app_id":      "app-1",
		"tenant_id":   "tenant-1",
		"entity_type": "Requirement",
		"name":        "FR-001",
		"properties": data.JSONMap{
			"meta":     data.JSONMap{"k": "v"},
			"keywords": []any{"a", "b"},
		},
	}
	rec := data.KGSyncOutbox{Op: data.OutboxOpUpsertEntity, Payload: payload}
	if err := w.syncOne(context.Background(), rec); err != nil {
		t.Fatalf("syncOne error: %v", err)
	}
	if neo.upsertEntityCalls != 1 || vec.upsertCalls != 1 {
		t.Fatalf("expected neo4j+qdrant calls 1/1, got %d/%d", neo.upsertEntityCalls, vec.upsertCalls)
	}
	if _, ok := neo.lastEntity.Properties["meta"].(string); !ok {
		t.Fatalf("expected neo4j entity meta property to be normalized as string, got %T", neo.lastEntity.Properties["meta"])
	}
	if _, ok := vec.lastEntity.Properties["meta"].(map[string]any); !ok {
		t.Fatalf("expected qdrant entity meta property to keep object payload, got %T", vec.lastEntity.Properties["meta"])
	}
}

func TestOutboxWorkerSyncOneUpsertEdge(t *testing.T) {
	neo := &fakeEntityEdgeSyncer{}
	vec := &fakeVectorSyncer{}
	w := &OutboxWorker{neo4jSyncer: neo, qdrantSyncer: vec, log: log.NewHelper(log.NewStdLogger(io.Discard))}

	payload := data.JSONMap{
		"edge_id":        "edge-1",
		"app_id":         "app-1",
		"tenant_id":      "tenant-1",
		"from_entity_id": "e1",
		"to_entity_id":   "e2",
		"relation_type":  "DEPENDS_ON",
		"properties": data.JSONMap{
			"exception_flows": []any{
				data.JSONMap{"id": "x1", "reason": "bad input"},
			},
		},
	}
	rec := data.KGSyncOutbox{Op: data.OutboxOpUpsertEdge, Payload: payload}
	if err := w.syncOne(context.Background(), rec); err != nil {
		t.Fatalf("syncOne error: %v", err)
	}
	if neo.upsertEdgeCalls != 1 {
		t.Fatalf("expected neo4j edge call, got %d", neo.upsertEdgeCalls)
	}
	if _, ok := neo.lastEdge.Properties["exception_flows"].(string); !ok {
		t.Fatalf("expected neo4j edge exception_flows to be normalized as string, got %T", neo.lastEdge.Properties["exception_flows"])
	}
	if vec.upsertCalls != 0 {
		t.Fatalf("expected no qdrant upsert for edge")
	}
}

func TestOutboxWorkerProcessBatchMarksFailed(t *testing.T) {
	db := newOutboxTestDB(t)
	entityID := "e-1"
	rec := data.KGSyncOutbox{
		Op:       data.OutboxOpUpsertEntity,
		EntityID: &entityID,
		TenantID: "tenant-1",
		AppID:    "app-1",
		Payload:  data.JSONMap{"entity_id": "e-1", "app_id": "app-1", "tenant_id": "tenant-1", "entity_type": "Requirement", "name": "FR-001"},
		Status:   data.OutboxStatusPending,
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("seed outbox: %v", err)
	}

	neo := &fakeEntityEdgeSyncer{upsertEntityErr: errors.New("neo4j down")}
	w := &OutboxWorker{
		db:           db,
		neo4jSyncer:  neo,
		qdrantSyncer: &fakeVectorSyncer{},
		batchSize:    10,
		maxAttempts:  10,
		log:          log.NewHelper(log.NewStdLogger(io.Discard)),
	}

	w.processBatch(context.Background())

	var row data.KGSyncOutbox
	if err := db.First(&row, rec.ID).Error; err != nil {
		t.Fatalf("read outbox row: %v", err)
	}
	if row.Status != data.OutboxStatusFailed {
		t.Fatalf("expected status FAILED, got %s", row.Status)
	}
	if row.Attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", row.Attempts)
	}
}

func TestOutboxWorkerMaxAttemptsSkipsRecord(t *testing.T) {
	db := newOutboxTestDB(t)
	entityID := "e-2"
	now := time.Now().UTC().Add(-2 * time.Hour)
	rec := data.KGSyncOutbox{
		Op:       data.OutboxOpUpsertEntity,
		EntityID: &entityID,
		TenantID: "tenant-1",
		AppID:    "app-1",
		Payload:  data.JSONMap{"entity_id": "e-2", "app_id": "app-1", "tenant_id": "tenant-1", "entity_type": "Requirement", "name": "FR-002"},
		Status:   data.OutboxStatusFailed,
		Attempts: 10,
		SyncedAt: &now,
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("seed outbox: %v", err)
	}

	neo := &fakeEntityEdgeSyncer{}
	w := &OutboxWorker{
		db:           db,
		neo4jSyncer:  neo,
		qdrantSyncer: &fakeVectorSyncer{},
		batchSize:    10,
		maxAttempts:  10,
		log:          log.NewHelper(log.NewStdLogger(io.Discard)),
	}

	w.processBatch(context.Background())

	if neo.upsertEntityCalls != 0 {
		t.Fatalf("expected no sync attempts after max attempts, got %d", neo.upsertEntityCalls)
	}

	var row data.KGSyncOutbox
	if err := db.First(&row, rec.ID).Error; err != nil {
		t.Fatalf("read outbox row: %v", err)
	}
	if row.Status != data.OutboxStatusFailed || row.Attempts != 10 {
		t.Fatalf("record should remain FAILED attempts=10, got status=%s attempts=%d", row.Status, row.Attempts)
	}
}
