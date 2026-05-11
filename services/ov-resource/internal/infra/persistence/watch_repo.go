package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"openviking.com/ov-resource/internal/domain/model"
)

type watchRepo struct {
	db *sql.DB
}

func NewWatchRepository(db *sql.DB) *watchRepo {
	return &watchRepo{db: db}
}

func (r *watchRepo) Create(ctx context.Context, task *model.WatchTask) error {
	patterns, _ := json.Marshal(task.Patterns)
	query := `
		INSERT INTO ov_watch_tasks (
			id, account_id, source_path, target_path, patterns, poll_interval_ms, 
			status, last_poll_at, files_tracked, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.AccountID, task.SourcePath, task.TargetPath, string(patterns),
		task.PollIntervalMs, task.Status, task.LastPollAt, task.FilesTracked, time.Now(), time.Now(),
	)
	return err
}

func (r *watchRepo) Update(ctx context.Context, task *model.WatchTask) error {
	patterns, _ := json.Marshal(task.Patterns)
	query := `
		UPDATE ov_watch_tasks SET
			status = $1, patterns = $2, poll_interval_ms = $3, updated_at = $4
		WHERE id = $5 AND account_id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		task.Status, string(patterns), task.PollIntervalMs, time.Now(), task.ID, task.AccountID,
	)
	return err
}

func (r *watchRepo) Delete(ctx context.Context, id, accountID string) error {
	query := `UPDATE ov_watch_tasks SET status = 'deleted', updated_at = $1 WHERE id = $2 AND account_id = $3`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id, accountID)
	return err
}

func (r *watchRepo) GetActiveTasks(ctx context.Context) ([]*model.WatchTask, error) {
	query := `
		SELECT id, account_id, source_path, target_path, patterns, poll_interval_ms, 
		status, last_poll_at, files_tracked, created_at, updated_at
		FROM ov_watch_tasks
		WHERE status = 'active'
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.WatchTask
	for rows.Next() {
		task := &model.WatchTask{}
		var patternsStr string
		var lastPollAt sql.NullTime
		err := rows.Scan(
			&task.ID, &task.AccountID, &task.SourcePath, &task.TargetPath, &patternsStr,
			&task.PollIntervalMs, &task.Status, &lastPollAt, &task.FilesTracked,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if lastPollAt.Valid {
			task.LastPollAt = lastPollAt.Time
		}
		_ = json.Unmarshal([]byte(patternsStr), &task.Patterns)
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (r *watchRepo) UpdateLastPoll(ctx context.Context, id string) error {
	query := `UPDATE ov_watch_tasks SET last_poll_at = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), id)
	return err
}
