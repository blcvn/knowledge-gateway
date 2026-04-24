package data

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var ErrEntityNotFound = errors.New("edge references non-existing entity")

func upsertEdgeTx(tx *gorm.DB, e KGEdge) (upsertOp, error) {
	if tx == nil {
		return opConflict, fmt.Errorf("upsertEdgeTx: nil transaction")
	}
	if err := ensureEdgeEndpointsExistTx(tx, e); err != nil {
		return opConflict, err
	}

	if tx.Dialector.Name() != "postgres" {
		return upsertEdgeTxPortable(tx, e)
	}

	var inserted bool
	row := tx.Raw(`
		INSERT INTO kg_edges
			(edge_id, app_id, tenant_id, from_entity_id, to_entity_id,
			 relation_type, properties, confidence, version_id, is_deleted, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,FALSE,CURRENT_TIMESTAMP)
		ON CONFLICT (edge_id) DO UPDATE
		   SET app_id         = EXCLUDED.app_id,
		       tenant_id      = EXCLUDED.tenant_id,
		       from_entity_id = EXCLUDED.from_entity_id,
		       to_entity_id   = EXCLUDED.to_entity_id,
		       relation_type  = EXCLUDED.relation_type,
		       properties     = EXCLUDED.properties,
		       confidence     = EXCLUDED.confidence,
		       version_id     = EXCLUDED.version_id,
		       is_deleted     = FALSE
		RETURNING (xmax = 0) AS inserted
	`,
		e.EdgeID,
		e.AppID,
		e.TenantID,
		e.FromEntityID,
		e.ToEntityID,
		e.RelationType,
		e.Properties,
		e.Confidence,
		nullableUUID(e.VersionID),
	).Row()
	if err := row.Scan(&inserted); err != nil {
		if isPGCode(err, "23503") || strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			return opConflict, ErrEntityNotFound
		}
		return opConflict, err
	}
	if inserted {
		return opCreated, nil
	}
	return opUpdated, nil
}

func upsertEdgeTxPortable(tx *gorm.DB, e KGEdge) (upsertOp, error) {
	var exists int64
	if err := tx.Model(&KGEdge{}).Where("edge_id = ?", e.EdgeID).Count(&exists).Error; err != nil {
		return opConflict, err
	}
	if err := ensureEdgeEndpointsExistTx(tx, e); err != nil {
		if errors.Is(err, ErrEntityNotFound) {
			return opConflict, ErrEntityNotFound
		}
		return opConflict, err
	}

	res := tx.Exec(`
		INSERT INTO kg_edges
			(edge_id, app_id, tenant_id, from_entity_id, to_entity_id,
			 relation_type, properties, confidence, version_id, is_deleted, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,FALSE,CURRENT_TIMESTAMP)
		ON CONFLICT (edge_id) DO UPDATE
		   SET app_id         = EXCLUDED.app_id,
		       tenant_id      = EXCLUDED.tenant_id,
		       from_entity_id = EXCLUDED.from_entity_id,
		       to_entity_id   = EXCLUDED.to_entity_id,
		       relation_type  = EXCLUDED.relation_type,
		       properties     = EXCLUDED.properties,
		       confidence     = EXCLUDED.confidence,
		       version_id     = EXCLUDED.version_id,
		       is_deleted     = FALSE
	`,
		e.EdgeID,
		e.AppID,
		e.TenantID,
		e.FromEntityID,
		e.ToEntityID,
		e.RelationType,
		e.Properties,
		e.Confidence,
		nullableUUID(e.VersionID),
	)
	if res.Error != nil {
		if isPGCode(res.Error, "23503") || strings.Contains(strings.ToLower(res.Error.Error()), "foreign key") {
			return opConflict, ErrEntityNotFound
		}
		return opConflict, res.Error
	}
	if exists > 0 {
		return opUpdated, nil
	}
	return opCreated, nil
}

func softDeleteEdgePG(ctx context.Context, db *gorm.DB, edgeID, tenantID string) error {
	if db == nil {
		return fmt.Errorf("softDeleteEdgePG: nil db")
	}
	res := db.WithContext(ctx).Exec(`
		UPDATE kg_edges
		SET is_deleted = TRUE
		WHERE edge_id = $1
		  AND tenant_id = $2
		  AND is_deleted = FALSE
	`, edgeID, tenantID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func UpsertEdgeTx(tx *gorm.DB, e KGEdge) (UpsertOp, error) {
	return upsertEdgeTx(tx, e)
}

func SoftDeleteEdgePG(ctx context.Context, db *gorm.DB, edgeID, tenantID string) error {
	return softDeleteEdgePG(ctx, db, edgeID, tenantID)
}

func ensureEdgeEndpointsExistTx(tx *gorm.DB, e KGEdge) error {
	fromID := strings.TrimSpace(e.FromEntityID)
	toID := strings.TrimSpace(e.ToEntityID)
	if fromID == "" || toID == "" {
		return ErrEntityNotFound
	}
	needed := int64(2)
	entityIDs := []string{fromID, toID}
	if fromID == toID {
		needed = 1
		entityIDs = []string{fromID}
	}
	var entityCount int64
	if err := tx.Model(&KGEntity{}).
		Where("app_id = ? AND tenant_id = ? AND entity_id IN ? AND is_deleted = FALSE", e.AppID, e.TenantID, entityIDs).
		Count(&entityCount).Error; err != nil {
		return err
	}
	if entityCount < needed {
		return ErrEntityNotFound
	}
	return nil
}
