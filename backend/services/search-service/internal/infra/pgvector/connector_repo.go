// Package pg implements ConnectorRepository using PostgreSQL.
package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vnp-memory/services/search-service/internal/domain/connector"
)

// ConnectorRepo implements port.ConnectorRepository.
type ConnectorRepo struct{ pool *pgxpool.Pool }

// NewConnectorRepo creates a ConnectorRepo.
func NewConnectorRepo(pool *pgxpool.Pool) *ConnectorRepo { return &ConnectorRepo{pool: pool} }

func (r *ConnectorRepo) Create(ctx context.Context, c *connector.Connector) error {
	cfgJSON, _ := json.Marshal(c.Config)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO search_connectors (id, tenant_id, name, type, config, sync_frequency, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ID, c.TenantID, c.Name, string(c.Type), cfgJSON, c.SyncFrequency, c.Status, c.CreatedAt,
	)
	return err
}

func (r *ConnectorRepo) GetByID(ctx context.Context, id string) (*connector.Connector, error) {
	c := &connector.Connector{}
	var cfgJSON []byte
	var connType string
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, type, config, sync_frequency, status, last_sync_at, created_at
		 FROM search_connectors WHERE id=$1`, id,
	).Scan(&c.ID, &c.TenantID, &c.Name, &connType, &cfgJSON, &c.SyncFrequency, &c.Status, &c.LastSyncAt, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("connector not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	c.Type = connector.ConnectorType(connType)
	_ = json.Unmarshal(cfgJSON, &c.Config)
	return c, nil
}

func (r *ConnectorRepo) ListByTenant(ctx context.Context, tenantID string) ([]*connector.Connector, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, type, config, sync_frequency, status, last_sync_at, created_at
		 FROM search_connectors WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var connectors []*connector.Connector
	for rows.Next() {
		c := &connector.Connector{}
		var cfgJSON []byte
		var connType string
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &connType, &cfgJSON,
			&c.SyncFrequency, &c.Status, &c.LastSyncAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Type = connector.ConnectorType(connType)
		_ = json.Unmarshal(cfgJSON, &c.Config)
		connectors = append(connectors, c)
	}
	return connectors, nil
}

func (r *ConnectorRepo) CreateJob(ctx context.Context, job *connector.SyncJob) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO search_sync_jobs (id, connector_id, status, items_synced, started_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		job.ID, job.ConnectorID, job.Status, 0, time.Now(),
	)
	return err
}
