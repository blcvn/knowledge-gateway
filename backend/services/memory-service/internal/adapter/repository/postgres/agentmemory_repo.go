package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vnp-memory/services/memory-service/internal/domain/agentmemory"
)

// AgentMemoryRepo is the PostgreSQL implementation of port.IMemoryRepo
type AgentMemoryRepo struct{ db *pgxpool.Pool }

// NewAgentMemoryRepo creates a new AgentMemoryRepo
func NewAgentMemoryRepo(db *pgxpool.Pool) *AgentMemoryRepo { return &AgentMemoryRepo{db: db} }

func (r *AgentMemoryRepo) Save(ctx context.Context, m agentmemory.AgentMemory) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO agent_memories
			(id, tenant_id, project, type, title, content, concepts, files, session_ids,
			 strength, version, parent_id, supersedes, is_latest, forget_after, agent_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`, m.ID, m.TenantID, m.Project, string(m.Type), m.Title, m.Content,
		m.Concepts, m.Files, m.SessionIDs,
		m.Strength, m.Version, nilIfEmpty(m.ParentID),
		m.Supersedes, m.IsLatest, m.ForgetAfter, m.AgentID, m.CreatedAt, m.UpdatedAt)
	return err
}

func (r *AgentMemoryRepo) GetByID(ctx context.Context, id string) (*agentmemory.AgentMemory, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, project, type, title, content, concepts, files,
		       session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
		FROM agent_memories WHERE id = $1
	`, id)
	return scanMemory(row)
}

func (r *AgentMemoryRepo) ListLatestByType(ctx context.Context, tenantID, project, memType string) ([]agentmemory.AgentMemory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, project, type, title, content, concepts, files,
		       session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
		FROM agent_memories
		WHERE tenant_id = $1 AND ($2 = '' OR project = $2) AND type = $3 AND is_latest = TRUE
		ORDER BY version DESC
	`, tenantID, project, memType)
	if err != nil {
		return nil, err
	}
	return scanMemories(rows)
}

func (r *AgentMemoryRepo) ListLatestByProject(ctx context.Context, tenantID, project string) ([]agentmemory.AgentMemory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, project, type, title, content, concepts, files,
		       session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
		FROM agent_memories
		WHERE tenant_id = $1 AND ($2 = '' OR project = $2) AND is_latest = TRUE
	`, tenantID, project)
	if err != nil {
		return nil, err
	}
	return scanMemories(rows)
}

func (r *AgentMemoryRepo) ListAll(ctx context.Context) ([]agentmemory.AgentMemory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, project, type, title, content, concepts, files,
		       session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
		FROM agent_memories WHERE is_latest = TRUE
	`)
	if err != nil {
		return nil, err
	}
	return scanMemories(rows)
}

func (r *AgentMemoryRepo) SetNotLatest(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE agent_memories SET is_latest = FALSE WHERE id = $1`, id)
	return err
}

func (r *AgentMemoryRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM agent_memories WHERE id = $1`, id)
	return err
}

func (r *AgentMemoryRepo) FindExpired(ctx context.Context) ([]agentmemory.AgentMemory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, project, type, title, content, concepts, files,
		       session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
		FROM agent_memories WHERE forget_after IS NOT NULL AND forget_after < NOW()
	`)
	if err != nil {
		return nil, err
	}
	return scanMemories(rows)
}

func (r *AgentMemoryRepo) UpdateStrength(ctx context.Context, id string, strength float64) error {
	_, err := r.db.Exec(ctx, `UPDATE agent_memories SET strength = $1, updated_at = NOW() WHERE id = $2`, strength, id)
	return err
}

func (r *AgentMemoryRepo) FlagForEviction(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE agent_memories SET flagged_eviction = TRUE WHERE id = $1`, id)
	return err
}

// nilIfEmpty returns nil for empty strings (to store NULL in DB)
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// scanRow interface for both pgx.Row and pgx.Rows
type scanRow interface {
	Scan(dest ...any) error
}

func scanMemory(row scanRow) (*agentmemory.AgentMemory, error) {
	var m agentmemory.AgentMemory
	var memType string
	err := row.Scan(
		&m.ID, &m.TenantID, &m.Project, &memType, &m.Title, &m.Content,
		&m.Concepts, &m.Files, &m.SessionIDs,
		&m.Strength, &m.Version, &m.IsLatest, &m.ForgetAfter, &m.AgentID, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.Type = agentmemory.MemoryType(memType)
	return &m, nil
}

type rowsScanner interface {
	Next() bool
	Scan(dest ...any) error
	Close()
}

func scanMemories(rows rowsScanner) ([]agentmemory.AgentMemory, error) {
	defer rows.Close()
	var memories []agentmemory.AgentMemory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			continue
		}
		memories = append(memories, *m)
	}
	return memories, nil
}
