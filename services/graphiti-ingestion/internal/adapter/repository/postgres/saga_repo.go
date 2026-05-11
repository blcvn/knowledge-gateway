package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

type SagaRepo struct {
	db *pgxpool.Pool
}

func NewSagaRepo(db *pgxpool.Pool) *SagaRepo {
	return &SagaRepo{db: db}
}

func (r *SagaRepo) Create(ctx context.Context, state *domain.SagaState) error {
	stepHistory, err := json.Marshal(state.StepHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal step history: %w", err)
	}

	query := `
		INSERT INTO graphiti_saga_state 
		(id, episode_id, group_id, current_step, status, step_history, retry_count, error_message, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = r.db.Exec(ctx, query,
		state.ID, state.EpisodeID, state.GroupID, string(state.CurrentStep), string(state.Status),
		stepHistory, state.RetryCount, state.ErrorMessage, state.StartedAt, state.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert saga state: %w", err)
	}
	return nil
}

func (r *SagaRepo) Get(ctx context.Context, id string) (*domain.SagaState, error) {
	query := `
		SELECT id, episode_id, group_id, current_step, status, step_history, retry_count, error_message, started_at, completed_at
		FROM graphiti_saga_state WHERE id = $1
	`
	var state domain.SagaState
	var stepHistory []byte
	var currentStep, status string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&state.ID, &state.EpisodeID, &state.GroupID, &currentStep, &status,
		&stepHistory, &state.RetryCount, &state.ErrorMessage, &state.StartedAt, &state.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("saga state not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get saga state: %w", err)
	}

	state.CurrentStep = domain.PipelineStep(currentStep)
	state.Status = domain.SagaStatus(status)

	if len(stepHistory) > 0 {
		if err := json.Unmarshal(stepHistory, &state.StepHistory); err != nil {
			return nil, fmt.Errorf("failed to unmarshal step history: %w", err)
		}
	}

	return &state, nil
}

func (r *SagaRepo) Update(ctx context.Context, state *domain.SagaState) error {
	stepHistory, err := json.Marshal(state.StepHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal step history: %w", err)
	}

	query := `
		UPDATE graphiti_saga_state 
		SET current_step = $1, status = $2, step_history = $3, retry_count = $4, error_message = $5, completed_at = $6
		WHERE id = $7
	`
	tag, err := r.db.Exec(ctx, query,
		string(state.CurrentStep), string(state.Status), stepHistory, state.RetryCount, state.ErrorMessage, state.CompletedAt, state.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update saga state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("saga state not found for update")
	}
	return nil
}

func (r *SagaRepo) GetStuckSagas(ctx context.Context, timeoutMinutes int, limit int) ([]*domain.SagaState, error) {
	query := `
		SELECT id, episode_id, group_id, current_step, status, step_history, retry_count, error_message, started_at, completed_at
		FROM graphiti_saga_state 
		WHERE status = 'running' AND started_at < NOW() - ($1 || ' minutes')::INTERVAL
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.db.Query(ctx, query, timeoutMinutes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*domain.SagaState
	for rows.Next() {
		var state domain.SagaState
		var stepHistory []byte
		var currentStep, status string
		if err := rows.Scan(&state.ID, &state.EpisodeID, &state.GroupID, &currentStep, &status, &stepHistory, &state.RetryCount, &state.ErrorMessage, &state.StartedAt, &state.CompletedAt); err != nil {
			return nil, err
		}
		state.CurrentStep = domain.PipelineStep(currentStep)
		state.Status = domain.SagaStatus(status)
		if len(stepHistory) > 0 {
			_ = json.Unmarshal(stepHistory, &state.StepHistory)
		}
		states = append(states, &state)
	}
	return states, nil
}
