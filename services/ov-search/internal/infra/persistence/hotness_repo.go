package persistence

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"vnp-memory/ov-search/internal/domain/model"
	"vnp-memory/ov-search/internal/domain/repository"
)

type hotnessRepo struct {
	pool *pgxpool.Pool
}

func NewHotnessRepo(ctx context.Context, dsn string) (repository.HotnessRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &hotnessRepo{pool: pool}, nil
}

func (r *hotnessRepo) Get(ctx context.Context, accountID, path string) (*model.HotnessScore, error) {
	query := `SELECT base_score, access_count, session_ref_count, computed_hotness, last_accessed_at, last_modified_at, updated_at
	          FROM ov_hotness_scores WHERE account_id = $1 AND path = $2`
	
	s := &model.HotnessScore{AccountID: accountID, Path: path}
	err := r.pool.QueryRow(ctx, query, accountID, path).Scan(
		&s.BaseScore, &s.AccessCount, &s.SessionRefCount, &s.ComputedHotness,
		&s.LastAccessedAt, &s.LastModifiedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *hotnessRepo) GetMulti(ctx context.Context, accountID string, paths []string) (map[string]*model.HotnessScore, error) {
	query := `SELECT path, base_score, access_count, session_ref_count, computed_hotness, last_accessed_at, last_modified_at, updated_at
	          FROM ov_hotness_scores WHERE account_id = $1 AND path = ANY($2)`
	
	rows, err := r.pool.Query(ctx, query, accountID, paths)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]*model.HotnessScore)
	for rows.Next() {
		var s model.HotnessScore
		s.AccountID = accountID
		err := rows.Scan(&s.Path, &s.BaseScore, &s.AccessCount, &s.SessionRefCount, &s.ComputedHotness, &s.LastAccessedAt, &s.LastModifiedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		res[s.Path] = &s
	}
	return res, nil
}

func (r *hotnessRepo) Save(ctx context.Context, score *model.HotnessScore) error {
	query := `
		INSERT INTO ov_hotness_scores (path, account_id, base_score, access_count, session_ref_count, computed_hotness, last_accessed_at, last_modified_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (path, account_id) DO UPDATE SET
			base_score = EXCLUDED.base_score,
			access_count = EXCLUDED.access_count,
			session_ref_count = EXCLUDED.session_ref_count,
			computed_hotness = EXCLUDED.computed_hotness,
			last_accessed_at = EXCLUDED.last_accessed_at,
			last_modified_at = EXCLUDED.last_modified_at,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, query,
		score.Path, score.AccountID, score.BaseScore, score.AccessCount, score.SessionRefCount,
		score.ComputedHotness, score.LastAccessedAt, score.LastModifiedAt, score.UpdatedAt,
	)
	return err
}

func (r *hotnessRepo) GetAll(ctx context.Context) ([]*model.HotnessScore, error) {
	query := `SELECT path, account_id, base_score, access_count, session_ref_count, computed_hotness, last_accessed_at, last_modified_at, updated_at
	          FROM ov_hotness_scores`
	
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*model.HotnessScore
	for rows.Next() {
		var s model.HotnessScore
		err := rows.Scan(&s.Path, &s.AccountID, &s.BaseScore, &s.AccessCount, &s.SessionRefCount, &s.ComputedHotness, &s.LastAccessedAt, &s.LastModifiedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		res = append(res, &s)
	}
	return res, nil
}
