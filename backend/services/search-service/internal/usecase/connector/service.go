// Package connector implements connector management usecases.
//
// Absorbed from: sm-connector (MERGE-P2-T4)
package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"vnp-memory/services/search-service/internal/domain/connector"
	"vnp-memory/services/search-service/internal/usecase/port"
)

// Service manages external data connectors.
type Service struct {
	repo port.ConnectorRepository
	pub  port.EventPublisher
}

// NewService creates a connector Service.
func NewService(repo port.ConnectorRepository, pub port.EventPublisher) *Service {
	return &Service{repo: repo, pub: pub}
}

// CreateConnector persists a new connector configuration.
func (s *Service) CreateConnector(ctx context.Context, tenantID, name, connType, freq string, config map[string]any) (*connector.Connector, error) {
	conn := &connector.Connector{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Name:          name,
		Type:          connector.ConnectorType(connType),
		Config:        config,
		SyncFrequency: freq,
		Status:        "active",
		CreatedAt:     time.Now(),
	}
	if err := s.repo.Create(ctx, conn); err != nil {
		return nil, fmt.Errorf("create connector: %w", err)
	}
	if s.pub != nil {
		_ = s.pub.Publish(ctx, "search.connector.created", conn)
	}
	return conn, nil
}

// ListConnectors returns all connectors for a tenant.
func (s *Service) ListConnectors(ctx context.Context, tenantID string) ([]*connector.Connector, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// SyncConnector triggers an async sync job via NATS.
func (s *Service) SyncConnector(ctx context.Context, connectorID string) (*connector.SyncJob, error) {
	conn, err := s.repo.GetByID(ctx, connectorID)
	if err != nil {
		return nil, fmt.Errorf("connector not found: %w", err)
	}

	job := &connector.SyncJob{
		ID:          uuid.New().String(),
		ConnectorID: conn.ID,
		Status:      "pending",
		StartedAt:   time.Now(),
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("create sync job: %w", err)
	}
	if s.pub != nil {
		_ = s.pub.Publish(ctx, "search.connector.sync.requested", job)
	}
	return job, nil
}
