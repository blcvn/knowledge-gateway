package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

type EpisodeRepo struct {
	db *pgxpool.Pool
}

func NewEpisodeRepo(db *pgxpool.Pool) *EpisodeRepo {
	return &EpisodeRepo{db: db}
}

func (r *EpisodeRepo) Create(ctx context.Context, episode *domain.Episode) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	entityTypes, _ := json.Marshal(episode.EntityTypes)
	edgeTypes, _ := json.Marshal(episode.EdgeTypes)

	queryEp := `
		INSERT INTO graphiti_episodes 
		(uuid, name, group_id, body, source, reference_time, content_hash, saga_id, entity_types, edge_types, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = tx.Exec(ctx, queryEp,
		episode.UUID, episode.Name, episode.GroupID, episode.Body, string(episode.Source), episode.ReferenceTime,
		episode.ContentHash, episode.SagaID, entityTypes, edgeTypes, episode.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateEpisode
		}
		return fmt.Errorf("failed to insert episode: %w", err)
	}

	queryDedup := `
		INSERT INTO graphiti_episode_dedup (content_hash, episode_id)
		VALUES ($1, $2)
	`
	_, err = tx.Exec(ctx, queryDedup, episode.ContentHash, episode.UUID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateEpisode
		}
		return fmt.Errorf("failed to insert dedup record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *EpisodeRepo) GetByHash(ctx context.Context, contentHash string) (*domain.Episode, error) {
	query := `
		SELECT e.uuid, e.name, e.group_id, e.body, e.source, e.reference_time, e.content_hash, e.saga_id, e.entity_types, e.edge_types, e.created_at
		FROM graphiti_episodes e
		JOIN graphiti_episode_dedup d ON e.uuid = d.episode_id
		WHERE d.content_hash = $1
	`
	var ep domain.Episode
	var source string
	var entityTypes, edgeTypes []byte

	err := r.db.QueryRow(ctx, query, contentHash).Scan(
		&ep.UUID, &ep.Name, &ep.GroupID, &ep.Body, &source, &ep.ReferenceTime, &ep.ContentHash, &ep.SagaID, &entityTypes, &edgeTypes, &ep.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEpisodeNotFound
		}
		return nil, fmt.Errorf("failed to get episode by hash: %w", err)
	}

	ep.Source = domain.EpisodeType(source)
	if len(entityTypes) > 0 {
		_ = json.Unmarshal(entityTypes, &ep.EntityTypes)
	}
	if len(edgeTypes) > 0 {
		_ = json.Unmarshal(edgeTypes, &ep.EdgeTypes)
	}

	return &ep, nil
}
