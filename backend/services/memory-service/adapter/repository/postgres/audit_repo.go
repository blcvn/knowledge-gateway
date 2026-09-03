package postgres

import (
	"encoding/json"
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/lib/pq"
    "vnp-memory/services/memory-service/internal/domain/agentmemory"
)

type AuditRepo struct{ db *pgxpool.Pool }
func NewAuditRepo(db *pgxpool.Pool) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) Save(ctx context.Context, entry agentmemory.AuditEntry) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO audit_entries
            (id, tenant_id, timestamp, operation, target_ids, performed_by, project, details, reason)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, entry.ID, entry.TenantID, entry.Timestamp, entry.Operation,
        pq.Array(entry.TargetIDs), entry.PerformedBy, entry.Project,
        marshalJSON(entry.Details), entry.Reason)
    return err
}

type AuditFilter struct {
    TenantID  string
    Operation string
    Project   string
    FromTime  *time.Time
    ToTime    *time.Time
    Limit     int
    Offset    int
}

func (r *AuditRepo) List(ctx context.Context, filter AuditFilter) ([]agentmemory.AuditEntry, error) {
    if filter.Limit == 0 { filter.Limit = 50 }
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
    `, filter.TenantID, filter.Operation, filter.Project, filter.FromTime, filter.ToTime,
        filter.Limit, filter.Offset)
    if err != nil { return nil, err }
    defer rows.Close()

    var entries []agentmemory.AuditEntry
    for rows.Next() {
        var e agentmemory.AuditEntry
        rows.Scan(&e.ID, &e.TenantID, &e.Timestamp, &e.Operation,
            pq.Array(&e.TargetIDs), &e.PerformedBy, &e.Project, &e.Reason)
        entries = append(entries, e)
    }
    return entries, nil
}

func marshalJSON(v any) []byte {
    if v == nil { return []byte("{}") }
    b, _ := json.Marshal(v)
    return b
}
