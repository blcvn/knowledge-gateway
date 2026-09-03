package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func makeKGEdge(appID, tenantID, fromID, toID string) KGEdge {
	return KGEdge{
		EdgeID:       uuid.NewString(),
		AppID:        appID,
		TenantID:     tenantID,
		FromEntityID: fromID,
		ToEntityID:   toID,
		RelationType: "DEPENDS_ON",
		Properties:   JSONMap{"type": "DEPENDS_ON"},
		Confidence:   0.9,
	}
}

func TestUpsertEdgeTxCreated(t *testing.T) {
	db := newKGTestDB(t)
	from := makeKGEntity("app-1", "tenant-1", "Requirement", "R1", 0)
	to := makeKGEntity("app-1", "tenant-1", "UseCase", "U1", 0)
	if _, err := insertEntityTx(db, from); err != nil {
		t.Fatalf("insert from entity: %v", err)
	}
	if _, err := insertEntityTx(db, to); err != nil {
		t.Fatalf("insert to entity: %v", err)
	}

	edge := makeKGEdge("app-1", "tenant-1", from.EntityID, to.EntityID)
	op, err := upsertEdgeTx(db, edge)
	if err != nil {
		t.Fatalf("upsertEdgeTx error: %v", err)
	}
	if op != opCreated {
		t.Fatalf("expected opCreated, got %v", op)
	}
}

func TestUpsertEdgeTxEntityNotFound(t *testing.T) {
	db := newKGTestDB(t)
	to := makeKGEntity("app-1", "tenant-1", "UseCase", "U1", 0)
	if _, err := insertEntityTx(db, to); err != nil {
		t.Fatalf("insert to entity: %v", err)
	}

	edge := makeKGEdge("app-1", "tenant-1", uuid.NewString(), to.EntityID)
	_, err := upsertEdgeTx(db, edge)
	if !errors.Is(err, ErrEntityNotFound) {
		t.Fatalf("expected ErrEntityNotFound, got %v", err)
	}
}

func TestUpsertEdgeTxUpdatedWhenEdgeIDExists(t *testing.T) {
	db := newKGTestDB(t)
	from := makeKGEntity("app-1", "tenant-1", "Requirement", "R1", 0)
	to := makeKGEntity("app-1", "tenant-1", "UseCase", "U1", 0)
	if _, err := insertEntityTx(db, from); err != nil {
		t.Fatalf("insert from entity: %v", err)
	}
	if _, err := insertEntityTx(db, to); err != nil {
		t.Fatalf("insert to entity: %v", err)
	}

	edge := makeKGEdge("app-1", "tenant-1", from.EntityID, to.EntityID)
	if _, err := upsertEdgeTx(db, edge); err != nil {
		t.Fatalf("seed upsertEdgeTx error: %v", err)
	}

	edge.Properties = JSONMap{"type": "DEPENDS_ON", "weight": 0.7}
	op, err := upsertEdgeTx(db, edge)
	if err != nil {
		t.Fatalf("upsertEdgeTx update error: %v", err)
	}
	if op != opUpdated {
		t.Fatalf("expected opUpdated, got %v", op)
	}
}

func TestSoftDeleteEdgePG(t *testing.T) {
	db := newKGTestDB(t)
	from := makeKGEntity("app-1", "tenant-1", "Requirement", "R1", 0)
	to := makeKGEntity("app-1", "tenant-1", "UseCase", "U1", 0)
	if _, err := insertEntityTx(db, from); err != nil {
		t.Fatalf("insert from entity: %v", err)
	}
	if _, err := insertEntityTx(db, to); err != nil {
		t.Fatalf("insert to entity: %v", err)
	}

	edge := makeKGEdge("app-1", "tenant-1", from.EntityID, to.EntityID)
	if _, err := upsertEdgeTx(db, edge); err != nil {
		t.Fatalf("seed upsertEdgeTx error: %v", err)
	}

	if err := softDeleteEdgePG(context.Background(), db, edge.EdgeID, edge.TenantID); err != nil {
		t.Fatalf("softDeleteEdgePG error: %v", err)
	}

	var row KGEdge
	if err := db.Where("edge_id = ?", edge.EdgeID).Take(&row).Error; err != nil {
		t.Fatalf("read edge after delete: %v", err)
	}
	if !row.IsDeleted {
		t.Fatalf("expected edge to be soft deleted")
	}
}
