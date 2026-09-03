package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vnp-memory/services/memory-service/internal/domain/agentmemory"
)

// SlotsRepo is the PostgreSQL implementation of port.ISlotsRepo
type SlotsRepo struct{ db *pgxpool.Pool }

// NewSlotsRepo creates a new SlotsRepo
func NewSlotsRepo(db *pgxpool.Pool) *SlotsRepo { return &SlotsRepo{db: db} }

func (r *SlotsRepo) GetSlot(ctx context.Context, tenantID, scope, label string) (*agentmemory.MemorySlot, error) {
	row := r.db.QueryRow(ctx, `
		SELECT tenant_id, project, scope, label, content, description, size_limit, pinned, read_only
		FROM memory_slots WHERE tenant_id = $1 AND scope = $2 AND label = $3
	`, tenantID, scope, label)
	var s agentmemory.MemorySlot
	err := row.Scan(&s.TenantID, &s.Project, &s.Scope, &s.Label, &s.Content,
		&s.Description, &s.SizeLimit, &s.Pinned, &s.ReadOnly)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SlotsRepo) CreateSlot(ctx context.Context, s agentmemory.MemorySlot) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO memory_slots (tenant_id, project, scope, label, content, description, size_limit, pinned, read_only)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, s.TenantID, s.Project, s.Scope, s.Label, s.Content, s.Description, s.SizeLimit, s.Pinned, s.ReadOnly)
	return err
}

func (r *SlotsRepo) UpdateSlot(ctx context.Context, s agentmemory.MemorySlot) error {
	_, err := r.db.Exec(ctx, `
		UPDATE memory_slots SET content = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND scope = $3 AND label = $4
	`, s.Content, s.TenantID, s.Scope, s.Label)
	return err
}

func (r *SlotsRepo) DeleteSlot(ctx context.Context, tenantID, scope, label string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM memory_slots WHERE tenant_id = $1 AND scope = $2 AND label = $3`, tenantID, scope, label)
	return err
}

func (r *SlotsRepo) ListSlots(ctx context.Context, tenantID, scope string) ([]agentmemory.MemorySlot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tenant_id, project, scope, label, description, size_limit, pinned, read_only
		FROM memory_slots WHERE tenant_id = $1 AND ($2 = '' OR scope = $2)
	`, tenantID, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var slots []agentmemory.MemorySlot
	for rows.Next() {
		var s agentmemory.MemorySlot
		if err := rows.Scan(&s.TenantID, &s.Project, &s.Scope, &s.Label, &s.Description, &s.SizeLimit, &s.Pinned, &s.ReadOnly); err != nil {
			continue
		}
		slots = append(slots, s)
	}
	return slots, nil
}
