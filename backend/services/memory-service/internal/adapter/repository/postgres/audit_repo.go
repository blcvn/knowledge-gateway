package postgres

import (
	"vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vnp-memory/services/memory-service/internal/domain/agentmemory"
)

// AuditRepo implements IAuditRepo using PostgreSQL
type AuditRepo struct {
	db *pgxpool.Pool
}

// NewAuditRepo creates a new audit repository
func NewAuditRepo(db *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{db: db}
}

// Save persists a single audit entry
func (r *AuditRepo) Save(ctx context.Context, entry agentmemory.AuditEntry) error {
	details := marshalDetails(entry.Details)
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_entries
			(id, tenant_id, timestamp, operation, target_ids, performed_by, project, details, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, entry.ID, entry.TenantID, entry.Timestamp, entry.Operation,
		entry.TargetIDs, entry.PerformedBy, entry.Project,
		details, entry.Reason)
	return err
}

// List returns audit entries matching the provided filter
func (r *AuditRepo) List(ctx context.Context, filter port.AuditFilter) ([]agentmemory.AuditEntry, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, timestamp, operation, target_ids, performed_by, project, reason
		FROM audit_entries
		WHERE tenant_id = $1
		  AND ($2 = '' OR operation = $2)
		  AND ($3 = '' OR project = $3)
		  AND ($4::TIMESTAMPTZ IS NULL OR timestamp >= $4)
		  AND ($5::TIMESTAMPTZ IS NULL OR timestamp <= $5)
		ORDER BY timestamp DESC
		LIMIT $6 OFFSET $7
	`, filter.TenantID, filter.Operation, filter.Project,
		nullTime(filter.FromTime), nullTime(filter.ToTime),
		filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []agentmemory.AuditEntry
	for rows.Next() {
		var e agentmemory.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.Timestamp, &e.Operation,
			&e.TargetIDs, &e.PerformedBy, &e.Project, &e.Reason,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func marshalDetails(v any) []byte {
	if v == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(v)
	return b
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
