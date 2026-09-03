package batch

import (
	"context"
	"testing"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBatchWriterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable fk: %v", err)
	}
	if err := db.AutoMigrate(&data.KGEntity{}, &data.KGEdge{}, &data.KGSyncOutbox{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestPGWriterBulkCreatePersistsEntitiesAndOutbox(t *testing.T) {
	db := newBatchWriterTestDB(t)
	writer := NewPGWriter(db)

	entities := []Entity{
		{Label: "Requirement", Properties: map[string]any{"id": "11111111-1111-1111-1111-111111111111", "name": "FR-001"}},
		{Label: "UseCase", Properties: map[string]any{"id": "22222222-2222-2222-2222-222222222222", "name": "UC-001"}},
	}

	created, err := writer.BulkCreate(context.Background(), "app-1", "tenant-1", entities)
	if err != nil {
		t.Fatalf("BulkCreate error: %v", err)
	}
	if created != 2 {
		t.Fatalf("expected created=2, got %d", created)
	}

	var entityCount int64
	if err := db.Model(&data.KGEntity{}).Count(&entityCount).Error; err != nil {
		t.Fatalf("count entities: %v", err)
	}
	if entityCount != 2 {
		t.Fatalf("expected 2 entities, got %d", entityCount)
	}

	var outboxCount int64
	if err := db.Model(&data.KGSyncOutbox{}).Count(&outboxCount).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 2 {
		t.Fatalf("expected 2 outbox records, got %d", outboxCount)
	}
}

func TestPGWriterBulkCreateHandlesEntityIDConflict(t *testing.T) {
	db := newBatchWriterTestDB(t)
	writer := NewPGWriter(db)

	entities := []Entity{
		{Label: "Requirement", Properties: map[string]any{"id": "33333333-3333-3333-3333-333333333333", "name": "FR-001"}},
		{Label: "Requirement", Properties: map[string]any{"id": "33333333-3333-3333-3333-333333333333", "name": "FR-001 duplicate"}},
	}

	created, err := writer.BulkCreate(context.Background(), "app-1", "tenant-1", entities)
	if err != nil {
		t.Fatalf("BulkCreate error: %v", err)
	}
	if created != 1 {
		t.Fatalf("expected created=1 with duplicate entity_id skipped, got %d", created)
	}

	var entityCount int64
	if err := db.Model(&data.KGEntity{}).Count(&entityCount).Error; err != nil {
		t.Fatalf("count entities: %v", err)
	}
	if entityCount != 1 {
		t.Fatalf("expected 1 entity, got %d", entityCount)
	}

	var outboxCount int64
	if err := db.Model(&data.KGSyncOutbox{}).Count(&outboxCount).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected 1 outbox record, got %d", outboxCount)
	}
}
