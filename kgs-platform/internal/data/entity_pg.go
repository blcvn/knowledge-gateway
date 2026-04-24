package data

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrAlreadyExists   = errors.New("entity already exists")
	ErrVersionConflict = errors.New("version conflict")
	ErrNameConflict    = errors.New("name conflict")
)

type UpsertOp int

const (
	UpsertOpCreated UpsertOp = iota
	UpsertOpUpdated
	UpsertOpConflict
)

type upsertOp = UpsertOp

const (
	opCreated  upsertOp = UpsertOpCreated
	opUpdated  upsertOp = UpsertOpUpdated
	opConflict upsertOp = UpsertOpConflict
)

func insertEntityTx(tx *gorm.DB, e KGEntity) (upsertOp, error) {
	if tx == nil {
		return opConflict, fmt.Errorf("insertEntityTx: nil transaction")
	}

	res := tx.Exec(`
		INSERT INTO kg_entities
			(entity_id, app_id, tenant_id, entity_type, name, properties,
			 confidence, source_file, chunk_id, skill_id, version_id,
			 provenance_type, domains, aliases, version, is_deleted, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,1,FALSE,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		ON CONFLICT (entity_id) DO NOTHING
	`,
		e.EntityID,
		e.AppID,
		e.TenantID,
		e.EntityType,
		e.Name,
		e.Properties,
		e.Confidence,
		e.SourceFile,
		e.ChunkID,
		e.SkillID,
		nullableUUID(e.VersionID),
		e.ProvenanceType,
		e.Domains,
		e.Aliases,
	)
	if res.Error != nil {
		if isPGCode(res.Error, "23505") || isUniqueConstraintViolation(res.Error) {
			return opConflict, ErrNameConflict
		}
		return opConflict, res.Error
	}
	if res.RowsAffected == 0 {
		return opConflict, ErrAlreadyExists
	}
	return opCreated, nil
}

func updateEntityTx(tx *gorm.DB, e KGEntity) (upsertOp, error) {
	if tx == nil {
		return opConflict, fmt.Errorf("updateEntityTx: nil transaction")
	}

	res := tx.Exec(`
		UPDATE kg_entities
		SET    app_id          = $1,
		       entity_type     = $2,
		       name            = $3,
		       properties      = $4,
		       confidence      = $5,
		       source_file     = $6,
		       chunk_id        = $7,
		       skill_id        = $8,
		       version_id      = $9,
		       provenance_type = $10,
		       domains         = $11,
		       aliases         = $12,
		       version         = version + 1,
		       is_deleted      = FALSE,
		       updated_at      = CURRENT_TIMESTAMP
		WHERE  entity_id = $13
		  AND  tenant_id = $14
		  AND  version   = $15
		  AND  is_deleted = FALSE
	`,
		e.AppID,
		e.EntityType,
		e.Name,
		e.Properties,
		e.Confidence,
		e.SourceFile,
		e.ChunkID,
		e.SkillID,
		nullableUUID(e.VersionID),
		e.ProvenanceType,
		e.Domains,
		e.Aliases,
		e.EntityID,
		e.TenantID,
		e.Version,
	)
	if res.Error != nil {
		return opConflict, res.Error
	}
	if res.RowsAffected == 0 {
		return opConflict, ErrVersionConflict
	}
	return opUpdated, nil
}

func upsertEntityTx(tx *gorm.DB, e KGEntity) (upsertOp, error) {
	if e.Version == 0 {
		return insertEntityTx(tx, e)
	}
	return updateEntityTx(tx, e)
}

func getEntityPG(ctx context.Context, db *gorm.DB, entityID string) (*KGEntity, error) {
	if db == nil {
		return nil, fmt.Errorf("getEntityPG: nil db")
	}
	var row KGEntity
	err := db.WithContext(ctx).
		Where("entity_id = ? AND is_deleted = FALSE", entityID).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func getEntityVersionPG(ctx context.Context, db *gorm.DB, entityID string) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("getEntityVersionPG: nil db")
	}
	var version int
	res := db.WithContext(ctx).Raw(`
		SELECT version
		FROM kg_entities
		WHERE entity_id = ? AND is_deleted = FALSE
		LIMIT 1
	`, entityID).Scan(&version)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return version, nil
}

func getEntityVersionsBatchPG(ctx context.Context, db *gorm.DB, entityIDs []string) (map[string]int, error) {
	if db == nil {
		return nil, fmt.Errorf("getEntityVersionsBatchPG: nil db")
	}
	out := make(map[string]int, len(entityIDs))
	if len(entityIDs) == 0 {
		return out, nil
	}

	type row struct {
		EntityID string `gorm:"column:entity_id"`
		Version  int    `gorm:"column:version"`
	}

	rows := make([]row, 0, len(entityIDs))
	if err := db.WithContext(ctx).
		Model(&KGEntity{}).
		Select("entity_id, version").
		Where("entity_id IN ? AND is_deleted = FALSE", entityIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.EntityID] = r.Version
	}
	return out, nil
}

func getEntitiesBatchPG(ctx context.Context, db *gorm.DB, entityIDs []string) ([]KGEntity, error) {
	if db == nil {
		return nil, fmt.Errorf("getEntitiesBatchPG: nil db")
	}
	if len(entityIDs) == 0 {
		return []KGEntity{}, nil
	}

	rows := make([]KGEntity, 0, len(entityIDs))
	err := db.WithContext(ctx).
		Where("entity_id IN ? AND is_deleted = FALSE", entityIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func softDeleteEntityPG(ctx context.Context, db *gorm.DB, entityID, tenantID string) error {
	if db == nil {
		return fmt.Errorf("softDeleteEntityPG: nil db")
	}
	res := db.WithContext(ctx).Exec(`
		UPDATE kg_entities
		SET is_deleted = TRUE,
		    version = version + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE entity_id = $1
		  AND tenant_id = $2
		  AND is_deleted = FALSE
	`, entityID, tenantID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func isPGCode(err error, code string) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}

func isUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicate key") || strings.Contains(msg, "sqlstate 23505")
}

func UpsertEntityTx(tx *gorm.DB, e KGEntity) (UpsertOp, error) {
	return upsertEntityTx(tx, e)
}

func GetEntityPG(ctx context.Context, db *gorm.DB, entityID string) (*KGEntity, error) {
	return getEntityPG(ctx, db, entityID)
}

func GetEntityVersionPG(ctx context.Context, db *gorm.DB, entityID string) (int, error) {
	return getEntityVersionPG(ctx, db, entityID)
}

func GetEntityVersionsBatchPG(ctx context.Context, db *gorm.DB, entityIDs []string) (map[string]int, error) {
	return getEntityVersionsBatchPG(ctx, db, entityIDs)
}

func GetEntitiesBatchPG(ctx context.Context, db *gorm.DB, entityIDs []string) ([]KGEntity, error) {
	return getEntitiesBatchPG(ctx, db, entityIDs)
}

func SoftDeleteEntityPG(ctx context.Context, db *gorm.DB, entityID, tenantID string) error {
	return softDeleteEntityPG(ctx, db, entityID, tenantID)
}
