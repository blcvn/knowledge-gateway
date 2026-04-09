package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

func enqueueOutboxTx(tx *gorm.DB, rec KGSyncOutbox) error {
	if tx == nil {
		return fmt.Errorf("enqueueOutboxTx: nil transaction")
	}
	if strings.TrimSpace(rec.Status) == "" {
		rec.Status = OutboxStatusPending
	}
	normalizedPayload, err := normalizeOutboxPayload(rec.Payload)
	if err != nil {
		return err
	}
	rec.Payload = normalizedPayload
	return tx.Create(&rec).Error
}

func enqueueOutboxBatchTx(tx *gorm.DB, recs []KGSyncOutbox) error {
	if tx == nil {
		return fmt.Errorf("enqueueOutboxBatchTx: nil transaction")
	}
	if len(recs) == 0 {
		return nil
	}
	for i := range recs {
		if strings.TrimSpace(recs[i].Status) == "" {
			recs[i].Status = OutboxStatusPending
		}
		normalizedPayload, err := normalizeOutboxPayload(recs[i].Payload)
		if err != nil {
			return err
		}
		recs[i].Payload = normalizedPayload
	}
	return tx.CreateInBatches(recs, 200).Error
}

func normalizeOutboxPayload(payload JSONMap) (JSONMap, error) {
	if payload == nil {
		return JSONMap{}, nil
	}
	raw, err := json.Marshal(map[string]any(payload))
	if err != nil {
		return nil, fmt.Errorf("normalizeOutboxPayload: marshal failed: %w", err)
	}
	var normalized map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, fmt.Errorf("normalizeOutboxPayload: unmarshal failed: %w", err)
	}
	return JSONMap(normalized), nil
}

func pollPendingOutbox(ctx context.Context, db *gorm.DB, limit int) ([]KGSyncOutbox, error) {
	if db == nil {
		return nil, fmt.Errorf("pollPendingOutbox: nil db")
	}
	if limit <= 0 {
		limit = 100
	}

	rows := make([]KGSyncOutbox, 0, limit)
	if db.Dialector.Name() == "postgres" {
		err := db.WithContext(ctx).Raw(`
			SELECT id, op, entity_id, edge_id, tenant_id, app_id,
			       payload, status, attempts, last_error, created_at, synced_at
			FROM kg_sync_outbox
			WHERE status IN ('PENDING','FAILED')
			  AND (attempts = 0 OR synced_at < CURRENT_TIMESTAMP - INTERVAL '1 minute' * POWER(2, GREATEST(LEAST(attempts, 5) - 1, 0)))
			ORDER BY id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		`, limit).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		return rows, nil
	}

	// SQLite fallback for unit tests.
	err := db.WithContext(ctx).
		Where("status IN ?", []string{OutboxStatusPending, OutboxStatusFailed}).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	filtered := make([]KGSyncOutbox, 0, len(rows))
	for _, row := range rows {
		if row.Attempts <= 0 {
			filtered = append(filtered, row)
			continue
		}
		if row.SyncedAt == nil {
			continue
		}
		backoff := retryBackoff(row.Attempts)
		if row.SyncedAt.Add(backoff).Before(now) || row.SyncedAt.Add(backoff).Equal(now) {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

func markOutboxSyncing(ctx context.Context, db *gorm.DB, id int64) error {
	if db == nil {
		return fmt.Errorf("markOutboxSyncing: nil db")
	}
	return db.WithContext(ctx).
		Model(&KGSyncOutbox{}).
		Where("id = ?", id).
		Update("status", OutboxStatusSyncing).Error
}

func markOutboxDone(ctx context.Context, db *gorm.DB, id int64) error {
	if db == nil {
		return fmt.Errorf("markOutboxDone: nil db")
	}
	return db.WithContext(ctx).Exec(`
		UPDATE kg_sync_outbox
		SET status = $1,
		    synced_at = CURRENT_TIMESTAMP,
		    last_error = ''
		WHERE id = $2
	`, OutboxStatusDone, id).Error
}

func markOutboxFailed(ctx context.Context, db *gorm.DB, id int64, errMsg string) error {
	if db == nil {
		return fmt.Errorf("markOutboxFailed: nil db")
	}
	return db.WithContext(ctx).Exec(`
		UPDATE kg_sync_outbox
		SET status = $1,
		    attempts = attempts + 1,
		    last_error = $2,
		    synced_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, OutboxStatusFailed, errMsg, id).Error
}

func cleanupOutboxDone(ctx context.Context, db *gorm.DB, olderThan time.Duration) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("cleanupOutboxDone: nil db")
	}
	if olderThan <= 0 {
		olderThan = 7 * 24 * time.Hour
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	res := db.WithContext(ctx).Exec(`
		DELETE FROM kg_sync_outbox
		WHERE status = $1
		  AND synced_at IS NOT NULL
		  AND synced_at < $2
	`, OutboxStatusDone, cutoff)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func countPendingOutbox(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("countPendingOutbox: nil db")
	}
	var total int64
	err := db.WithContext(ctx).
		Model(&KGSyncOutbox{}).
		Where("status IN ?", []string{OutboxStatusPending, OutboxStatusFailed}).
		Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func EnqueueOutboxTx(tx *gorm.DB, rec KGSyncOutbox) error {
	return enqueueOutboxTx(tx, rec)
}

func EnqueueOutboxBatchTx(tx *gorm.DB, recs []KGSyncOutbox) error {
	return enqueueOutboxBatchTx(tx, recs)
}

func PollPendingOutbox(ctx context.Context, db *gorm.DB, limit int) ([]KGSyncOutbox, error) {
	return pollPendingOutbox(ctx, db, limit)
}

func MarkOutboxSyncing(ctx context.Context, db *gorm.DB, id int64) error {
	return markOutboxSyncing(ctx, db, id)
}

func MarkOutboxDone(ctx context.Context, db *gorm.DB, id int64) error {
	return markOutboxDone(ctx, db, id)
}

func MarkOutboxFailed(ctx context.Context, db *gorm.DB, id int64, errMsg string) error {
	return markOutboxFailed(ctx, db, id, errMsg)
}

func CleanupOutboxDone(ctx context.Context, db *gorm.DB, olderThan time.Duration) (int64, error) {
	return cleanupOutboxDone(ctx, db, olderThan)
}

func CountPendingOutbox(ctx context.Context, db *gorm.DB) (int64, error) {
	return countPendingOutbox(ctx, db)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func retryBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return 0
	}
	exp := minInt(attempts-1, 4)
	return time.Minute * time.Duration(1<<exp)
}
