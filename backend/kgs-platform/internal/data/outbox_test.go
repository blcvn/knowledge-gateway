package data

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestOutboxEnqueueAndPoll(t *testing.T) {
	db := newKGTestDB(t)
	entityID := "00000000-0000-0000-0000-000000000001"
	rec := KGSyncOutbox{
		Op:       OutboxOpUpsertEntity,
		EntityID: &entityID,
		TenantID: "tenant-1",
		AppID:    "app-1",
		Payload:  JSONMap{"id": entityID},
	}
	if err := enqueueOutboxTx(db, rec); err != nil {
		t.Fatalf("enqueueOutboxTx error: %v", err)
	}

	rows, err := pollPendingOutbox(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("pollPendingOutbox error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 outbox row, got %d", len(rows))
	}
	if rows[0].Op != OutboxOpUpsertEntity {
		t.Fatalf("unexpected op %s", rows[0].Op)
	}
}

func TestOutboxMarkDoneRemovesFromPoll(t *testing.T) {
	db := newKGTestDB(t)
	entityID := "00000000-0000-0000-0000-000000000002"
	rec := KGSyncOutbox{
		Op:       OutboxOpUpsertEntity,
		EntityID: &entityID,
		TenantID: "tenant-1",
		AppID:    "app-1",
		Payload:  JSONMap{"id": entityID},
	}
	if err := enqueueOutboxTx(db, rec); err != nil {
		t.Fatalf("enqueueOutboxTx error: %v", err)
	}

	rows, err := pollPendingOutbox(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("pollPendingOutbox error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row before done, got %d", len(rows))
	}
	if err := markOutboxDone(context.Background(), db, rows[0].ID); err != nil {
		t.Fatalf("markOutboxDone error: %v", err)
	}

	rows, err = pollPendingOutbox(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("pollPendingOutbox after done error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows after mark done, got %d", len(rows))
	}
}

func TestOutboxFailedBackoff(t *testing.T) {
	db := newKGTestDB(t)
	entityID := "00000000-0000-0000-0000-000000000003"
	rec := KGSyncOutbox{
		Op:       OutboxOpUpsertEntity,
		EntityID: &entityID,
		TenantID: "tenant-1",
		AppID:    "app-1",
		Payload:  JSONMap{"id": entityID},
	}
	if err := enqueueOutboxTx(db, rec); err != nil {
		t.Fatalf("enqueueOutboxTx error: %v", err)
	}

	rows, err := pollPendingOutbox(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("pollPendingOutbox error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row before failure, got %d", len(rows))
	}
	if err := markOutboxFailed(context.Background(), db, rows[0].ID, "sync failed"); err != nil {
		t.Fatalf("markOutboxFailed error: %v", err)
	}

	rows, err = pollPendingOutbox(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("pollPendingOutbox immediate retry error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no retry before 1 minute, got %d rows", len(rows))
	}

	cutoff := time.Now().UTC().Add(-2 * time.Minute)
	if err := db.Model(&KGSyncOutbox{}).
		Where("entity_id = ?", entityID).
		Update("synced_at", cutoff).Error; err != nil {
		t.Fatalf("update synced_at: %v", err)
	}

	rows, err = pollPendingOutbox(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("pollPendingOutbox after backoff error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected retry row after backoff, got %d", len(rows))
	}
}

func TestOutboxPollSkipLockedParallel(t *testing.T) {
	db := newKGTestDB(t)
	if db.Dialector.Name() != "postgres" {
		t.Skip("FOR UPDATE SKIP LOCKED semantics require postgres")
	}

	entityID := "00000000-0000-0000-0000-000000000004"
	rec := KGSyncOutbox{
		Op:       OutboxOpUpsertEntity,
		EntityID: &entityID,
		TenantID: "tenant-1",
		AppID:    "app-1",
		Payload:  JSONMap{"id": entityID},
	}
	if err := enqueueOutboxTx(db, rec); err != nil {
		t.Fatalf("enqueueOutboxTx error: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	results := make([][]KGSyncOutbox, 2)
	errs := make([]error, 2)

	for i := 0; i < 2; i++ {
		idx := i
		go func() {
			defer wg.Done()
			tx := db.Begin()
			defer tx.Rollback()
			results[idx], errs[idx] = pollPendingOutbox(context.Background(), tx, 1)
		}()
	}
	wg.Wait()

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("poll errors: %v %v", errs[0], errs[1])
	}
	if len(results[0]) == 1 && len(results[1]) == 1 {
		t.Fatalf("expected skip-locked to avoid duplicate rows across concurrent polls")
	}
}
