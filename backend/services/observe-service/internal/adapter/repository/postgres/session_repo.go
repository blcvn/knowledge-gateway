package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/lib/pq"
    "github.com/vnp-memory/services/observe-service/internal/domain"
)

type SessionRepo struct{ db *pgxpool.Pool }

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Save(ctx context.Context, s domain.Session) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO agent_sessions
            (id, tenant_id, project, cwd, model, agent_id, status, first_prompt, tags, started_at, last_active_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `, s.ID, s.TenantID, s.Project, s.CWD, s.Model, s.AgentID, s.Status,
        s.FirstPrompt, pq.Array(s.Tags), s.StartedAt, s.LastActiveAt)
    return err
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
    row := r.db.QueryRow(ctx, `
        SELECT id, tenant_id, project, cwd, model, agent_id, status, summary,
               observation_count, tags, started_at, ended_at, last_active_at
        FROM agent_sessions WHERE id = $1
    `, id)
    var s domain.Session
    err := row.Scan(&s.ID, &s.TenantID, &s.Project, &s.CWD, &s.Model, &s.AgentID,
        &s.Status, &s.Summary, &s.ObservationCount, pq.Array(&s.Tags),
        &s.StartedAt, &s.EndedAt, &s.LastActiveAt)
    if err != nil { return nil, err }
    return &s, nil
}

func (r *SessionRepo) List(ctx context.Context, tenantID, status, project string, limit, offset int) ([]domain.Session, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, tenant_id, project, status, observation_count, started_at
        FROM agent_sessions
        WHERE tenant_id = $1
          AND ($2 = '' OR status = $2)
          AND ($3 = '' OR project = $3)
        ORDER BY started_at DESC LIMIT $4 OFFSET $5
    `, tenantID, status, project, limit, offset)
    if err != nil { return nil, err }
    defer rows.Close()

    var sessions []domain.Session
    for rows.Next() {
        var s domain.Session
        rows.Scan(&s.ID, &s.TenantID, &s.Project, &s.Status, &s.ObservationCount, &s.StartedAt)
        sessions = append(sessions, s)
    }
    return sessions, nil
}

func (r *SessionRepo) UpdateStatus(ctx context.Context, id, status string) error {
    now := time.Now()
    _, err := r.db.Exec(ctx, `
        UPDATE agent_sessions SET status = $1, ended_at = $2 WHERE id = $3
    `, status, now, id)
    return err
}

func (r *SessionRepo) IncrementObsCount(ctx context.Context, id string) error {
    _, err := r.db.Exec(ctx, `
        UPDATE agent_sessions
        SET observation_count = observation_count + 1, last_active_at = NOW()
        WHERE id = $1
    `, id)
    return err
}

func (r *SessionRepo) GetObsCount(ctx context.Context, id string) (int, error) {
    var count int
    err := r.db.QueryRow(ctx, `SELECT observation_count FROM agent_sessions WHERE id = $1`, id).Scan(&count)
    return count, err
}
