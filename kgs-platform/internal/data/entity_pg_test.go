package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func makeKGEntity(appID, tenantID, entityType, name string, version int) KGEntity {
	return KGEntity{
		EntityID:   uuid.NewString(),
		AppID:      appID,
		TenantID:   tenantID,
		EntityType: entityType,
		Name:       name,
		Properties: JSONMap{"name": name},
		Confidence: 0.95,
		Domains:    StringArr{"core"},
		Aliases:    StringArr{"alias"},
		Version:    version,
	}
}

func TestInsertEntityTxCreated(t *testing.T) {
	db := newKGTestDB(t)
	e := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)

	op, err := insertEntityTx(db, e)
	if err != nil {
		t.Fatalf("insertEntityTx error: %v", err)
	}
	if op != opCreated {
		t.Fatalf("expected opCreated, got %v", op)
	}
}

func TestInsertEntityTxAlreadyExists(t *testing.T) {
	db := newKGTestDB(t)
	e := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)
	if _, err := insertEntityTx(db, e); err != nil {
		t.Fatalf("seed insertEntityTx error: %v", err)
	}

	_, err := insertEntityTx(db, e)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestInsertEntityTxAllowsDuplicateNameCaseInsensitive(t *testing.T) {
	db := newKGTestDB(t)
	first := makeKGEntity("app-1", "tenant-1", "Requirement", "Payment Gateway", 0)
	if _, err := insertEntityTx(db, first); err != nil {
		t.Fatalf("seed insertEntityTx error: %v", err)
	}

	second := makeKGEntity("app-1", "tenant-1", "Requirement", "payment gateway", 0)
	op, err := insertEntityTx(db, second)
	if err != nil {
		t.Fatalf("expected duplicate name to be allowed, got err=%v", err)
	}
	if op != opCreated {
		t.Fatalf("expected opCreated for second duplicate-name entity, got %v", op)
	}
}

func TestUpdateEntityTxSuccessIncrementsVersion(t *testing.T) {
	db := newKGTestDB(t)
	e := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)
	if _, err := insertEntityTx(db, e); err != nil {
		t.Fatalf("seed insertEntityTx error: %v", err)
	}

	e.Name = "FR-001 Updated"
	e.Version = 1
	op, err := updateEntityTx(db, e)
	if err != nil {
		t.Fatalf("updateEntityTx error: %v", err)
	}
	if op != opUpdated {
		t.Fatalf("expected opUpdated, got %v", op)
	}

	got, err := getEntityPG(context.Background(), db, e.EntityID)
	if err != nil {
		t.Fatalf("getEntityPG error: %v", err)
	}
	if got.Version != 2 {
		t.Fatalf("expected version=2, got %d", got.Version)
	}
}

func TestUpdateEntityTxVersionConflict(t *testing.T) {
	db := newKGTestDB(t)
	e := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)
	if _, err := insertEntityTx(db, e); err != nil {
		t.Fatalf("seed insertEntityTx error: %v", err)
	}

	e.Version = 99
	_, err := updateEntityTx(db, e)
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestUpsertEntityTxOverlayCreated(t *testing.T) {
	db := newKGTestDB(t)
	e := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)

	op, err := upsertEntityTxOverlay(db, e)
	if err != nil {
		t.Fatalf("upsertEntityTxOverlay create error: %v", err)
	}
	if op != opCreated {
		t.Fatalf("expected opCreated, got %v", op)
	}
}

func TestUpsertEntityTxOverlayUpdatesExisting(t *testing.T) {
	db := newKGTestDB(t)
	e := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)
	if _, err := insertEntityTx(db, e); err != nil {
		t.Fatalf("seed insertEntityTx error: %v", err)
	}

	e.Name = "FR-001 Updated By Overlay"
	e.Properties = JSONMap{"id": e.EntityID, "name": e.Name}

	op, err := upsertEntityTxOverlay(db, e)
	if err != nil {
		t.Fatalf("upsertEntityTxOverlay update error: %v", err)
	}
	if op != opUpdated {
		t.Fatalf("expected opUpdated, got %v", op)
	}

	got, err := getEntityPG(context.Background(), db, e.EntityID)
	if err != nil {
		t.Fatalf("getEntityPG error: %v", err)
	}
	if got.Name != e.Name {
		t.Fatalf("expected name=%q, got %q", e.Name, got.Name)
	}
	if got.Version != 2 {
		t.Fatalf("expected version=2, got %d", got.Version)
	}
}

func TestSoftDeleteEntityPG(t *testing.T) {
	db := newKGTestDB(t)
	e := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)
	if _, err := insertEntityTx(db, e); err != nil {
		t.Fatalf("seed insertEntityTx error: %v", err)
	}

	if err := softDeleteEntityPG(context.Background(), db, e.EntityID, e.TenantID); err != nil {
		t.Fatalf("softDeleteEntityPG error: %v", err)
	}

	var row KGEntity
	if err := db.Where("entity_id = ?", e.EntityID).Take(&row).Error; err != nil {
		t.Fatalf("read entity after delete: %v", err)
	}
	if !row.IsDeleted {
		t.Fatalf("expected entity to be soft deleted")
	}
}

func TestGetEntityVersionsBatchPG(t *testing.T) {
	db := newKGTestDB(t)
	e1 := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-001", 0)
	e2 := makeKGEntity("app-1", "tenant-1", "Requirement", "FR-002", 0)
	if _, err := insertEntityTx(db, e1); err != nil {
		t.Fatalf("seed insert e1: %v", err)
	}
	if _, err := insertEntityTx(db, e2); err != nil {
		t.Fatalf("seed insert e2: %v", err)
	}

	e2.Version = 1
	e2.Name = "FR-002 Updated"
	if _, err := updateEntityTx(db, e2); err != nil {
		t.Fatalf("update e2: %v", err)
	}

	versions, err := getEntityVersionsBatchPG(context.Background(), db, []string{e1.EntityID, e2.EntityID, "missing"})
	if err != nil {
		t.Fatalf("getEntityVersionsBatchPG error: %v", err)
	}
	if versions[e1.EntityID] != 1 {
		t.Fatalf("expected e1 version=1, got %d", versions[e1.EntityID])
	}
	if versions[e2.EntityID] != 2 {
		t.Fatalf("expected e2 version=2, got %d", versions[e2.EntityID])
	}
	if _, ok := versions["missing"]; ok {
		t.Fatalf("missing entity should not be returned")
	}
}
