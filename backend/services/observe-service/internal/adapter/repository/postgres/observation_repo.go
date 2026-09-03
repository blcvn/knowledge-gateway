package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-memory/services/observe-service/internal/domain"
)

// ObservationRepo is the PostgreSQL implementation of port.IObservationRepo
type ObservationRepo struct{ db *pgxpool.Pool }

// NewObservationRepo creates a new ObservationRepo
func NewObservationRepo(db *pgxpool.Pool) *ObservationRepo { return &ObservationRepo{db: db} }

func (r *ObservationRepo) SaveRaw(ctx context.Context, raw domain.RawObservation) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO raw_observations
			(id, session_id, tenant_id, hook_type, tool_name, tool_input, tool_output,
			 user_prompt, assistant_response, modality, agent_id, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, raw.ID, raw.SessionID, raw.TenantID, raw.HookType, raw.ToolName,
		raw.ToolInput, raw.ToolOutput, raw.UserPrompt, raw.AssistantResponse,
		raw.Modality, raw.AgentID, raw.Timestamp)
	return err
}

func (r *ObservationRepo) SaveCompressed(ctx context.Context, comp domain.CompressedObservation) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO compressed_observations
			(id, session_id, tenant_id, obs_type, title, subtitle, facts, narrative,
			 concepts, files, importance, confidence, agent_id, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, comp.ID, comp.SessionID, comp.TenantID, comp.ObsType, comp.Title, comp.Subtitle,
		comp.Facts, comp.Narrative, comp.Concepts, comp.Files,
		comp.Importance, comp.Confidence, comp.AgentID, comp.Timestamp)
	return err
}

func (r *ObservationRepo) ListCompressed(ctx context.Context, sessionID string, limit, offset int) ([]domain.CompressedObservation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, tenant_id, obs_type, title, subtitle, facts, narrative,
		       concepts, files, importance, confidence, agent_id, timestamp
		FROM compressed_observations
		WHERE session_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []domain.CompressedObservation
	for rows.Next() {
		var obs domain.CompressedObservation
		if err := rows.Scan(
			&obs.ID, &obs.SessionID, &obs.TenantID, &obs.ObsType, &obs.Title, &obs.Subtitle,
			&obs.Facts, &obs.Narrative, &obs.Concepts, &obs.Files,
			&obs.Importance, &obs.Confidence, &obs.AgentID, &obs.Timestamp,
		); err != nil {
			continue
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

func (r *ObservationRepo) DeleteBySessionID(ctx context.Context, sessionID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM raw_observations WHERE session_id = $1`, sessionID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM compressed_observations WHERE session_id = $1`, sessionID)
	return err
}
