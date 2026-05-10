// Package persistence implements PostgreSQL repositories for vnp-platform.
package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/analytics"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/event"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/project"
)

// TenantRepo implements port.TenantRepository.
type TenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

func (r *TenantRepo) Create(ctx context.Context, t *admin.Tenant) error {
	configJSON, _ := json.Marshal(t.Config)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug, tier, config, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Name, t.Slug, t.Tier, configJSON, t.Active, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *TenantRepo) FindByID(ctx context.Context, id uuid.UUID) (*admin.Tenant, error) {
	t := &admin.Tenant{}
	var configJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, tier, config, active, created_at, updated_at
		 FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Tier, &configJSON, &t.Active, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("tenant not found: %s", id)
	}
	_ = json.Unmarshal(configJSON, &t.Config)
	return t, err
}

func (r *TenantRepo) Update(ctx context.Context, t *admin.Tenant) error {
	configJSON, _ := json.Marshal(t.Config)
	_, err := r.pool.Exec(ctx,
		`UPDATE tenants SET name=$1, slug=$2, tier=$3, config=$4, active=$5, updated_at=$6 WHERE id=$7`,
		t.Name, t.Slug, t.Tier, configJSON, t.Active, t.UpdatedAt, t.ID,
	)
	return err
}

func (r *TenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	return err
}

func (r *TenantRepo) List(ctx context.Context, offset, limit int) ([]*admin.Tenant, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&total)

	rows, err := r.pool.Query(ctx,
		`SELECT id, name, slug, tier, config, active, created_at, updated_at
		 FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []*admin.Tenant
	for rows.Next() {
		t := &admin.Tenant{}
		var configJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Tier, &configJSON, &t.Active, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(configJSON, &t.Config)
		tenants = append(tenants, t)
	}
	return tenants, total, nil
}

// EventRepo implements port.EventRepository.
type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) Create(ctx context.Context, evt *event.UserEvent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_events (id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		evt.ID, evt.UserID, evt.TenantID, evt.Source, evt.Content, evt.Tags,
		evt.CreatedAt, evt.ValidAt, evt.InvalidAt,
	)
	return err
}

func (r *EventRepo) FindByUser(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]event.UserEvent, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_events WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&total)

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at
		 FROM user_events WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY valid_at DESC LIMIT $3`,
		tenantID, userID, limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []event.UserEvent
	for rows.Next() {
		var e event.UserEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.TenantID, &e.Source, &e.Content, &e.Tags,
			&e.CreatedAt, &e.ValidAt, &e.InvalidAt); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (r *EventRepo) SearchByText(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]event.UserEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at
		 FROM user_events
		 WHERE tenant_id = $1 AND content ILIKE '%' || $2 || '%'
		 ORDER BY valid_at DESC LIMIT $3`,
		tenantID, query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.UserEvent
	for rows.Next() {
		var e event.UserEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.TenantID, &e.Source, &e.Content, &e.Tags,
			&e.CreatedAt, &e.ValidAt, &e.InvalidAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// UsageRepo implements port.UsageRepository.
type UsageRepo struct {
	pool *pgxpool.Pool
}

func NewUsageRepo(pool *pgxpool.Pool) *UsageRepo {
	return &UsageRepo{pool: pool}
}

func (r *UsageRepo) Upsert(ctx context.Context, rec *analytics.UsageRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO usage_records (id, tenant_id, period, api_calls, storage_mb, llm_tokens, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (tenant_id, period) DO UPDATE SET
		   api_calls = usage_records.api_calls + EXCLUDED.api_calls,
		   storage_mb = EXCLUDED.storage_mb,
		   llm_tokens = usage_records.llm_tokens + EXCLUDED.llm_tokens,
		   updated_at = NOW()`,
		rec.ID, rec.TenantID, rec.Period, rec.APICalls, rec.StorageMB, rec.LLMTokens,
	)
	return err
}

func (r *UsageRepo) FindByTenant(ctx context.Context, tenantID uuid.UUID, period string) ([]*analytics.UsageRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, period, api_calls, storage_mb, llm_tokens, created_at, updated_at
		 FROM usage_records WHERE tenant_id = $1 AND period = $2`,
		tenantID, period,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*analytics.UsageRecord
	for rows.Next() {
		rec := &analytics.UsageRecord{}
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Period, &rec.APICalls,
			&rec.StorageMB, &rec.LLMTokens, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// SpaceRepo implements port.SpaceRepository.
type SpaceRepo struct {
	pool *pgxpool.Pool
}

func NewSpaceRepo(pool *pgxpool.Pool) *SpaceRepo {
	return &SpaceRepo{pool: pool}
}

func (r *SpaceRepo) Create(ctx context.Context, s *project.Space) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO spaces (id, tenant_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		s.ID, s.TenantID, s.Name, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *SpaceRepo) FindByID(ctx context.Context, id uuid.UUID) (*project.Space, error) {
	s := &project.Space{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, created_at, updated_at FROM spaces WHERE id = $1`, id,
	).Scan(&s.ID, &s.TenantID, &s.Name, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("space not found: %s", id)
	}
	return s, err
}

func (r *SpaceRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*project.Space, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, created_at, updated_at FROM spaces WHERE tenant_id = $1 ORDER BY name`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spaces []*project.Space
	for rows.Next() {
		s := &project.Space{}
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		spaces = append(spaces, s)
	}
	return spaces, nil
}

func (r *SpaceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM spaces WHERE id = $1`, id)
	return err
}
