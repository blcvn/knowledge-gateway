package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-context/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-context/domain/repository"
)

type eventGistRepo struct {
	pool *pgxpool.Pool
}

func NewEventGistSearchRepository(pool *pgxpool.Pool) repository.EventGistSearchRepository {
	return &eventGistRepo{pool: pool}
}

func (r *eventGistRepo) SearchBySimilarity(ctx context.Context, userID, projectID string, threshold float32, windowDays int, limit int) ([]*model.EventGist, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, project_id, gist_data::text, created_at
		FROM user_event_gists
		WHERE user_id = $1 AND project_id = $2
		  AND created_at > now() - interval '%d days'
		ORDER BY created_at DESC
		LIMIT $3
	`, windowDays)

	rows, err := r.pool.Query(ctx, query, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gists []*model.EventGist
	for rows.Next() {
		e := &model.EventGist{}
		err := rows.Scan(&e.ID, &e.UserID, &e.ProjectID, &e.GistData, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		gists = append(gists, e)
	}
	return gists, nil
}
