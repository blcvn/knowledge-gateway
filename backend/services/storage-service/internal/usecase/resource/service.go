// Package resource implements resource ingestion usecase for storage-service.
//
// Provides: Ingest, GetStatus, List
// Absorbed from: ov-resource (MERGE-P1-T4)
package resource

import (
	"context"
	"fmt"
	"time"

	"vnp-memory/services/storage-service/internal/domain/resource"
)

// Repository is the output port for resource persistence.
type Repository interface {
	Create(ctx context.Context, res *resource.Resource) error
	FindByID(ctx context.Context, id string) (*resource.Resource, error)
	UpdateStatus(ctx context.Context, id, status string) error
	List(ctx context.Context, tenantID string) ([]*resource.Resource, error)
}

// Service implements resource ingestion use cases.
type Service struct {
	repo Repository
}

// NewService creates a resource Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Ingest creates a new resource and queues it for processing.
func (s *Service) Ingest(ctx context.Context, tenantID string, job *resource.IngestJob) (*resource.Resource, error) {
	res := &resource.Resource{
		ID:        generateID("res"),
		TenantID:  tenantID,
		URI:       job.URI,
		Type:      guessType(job.URI),
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	job.ResourceID = res.ID
	job.TenantID = tenantID
	job.CreatedAt = time.Now()

	if err := s.repo.Create(ctx, res); err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// TODO: Publish to NATS for async embedding processing
	// publisher.Publish(ctx, "storage.resource.ingest", job)

	return res, nil
}

// GetStatus returns current ingestion status for a resource.
func (s *Service) GetStatus(ctx context.Context, resourceID string) (*resource.Resource, error) {
	res, err := s.repo.FindByID(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("find resource: %w", err)
	}
	return res, nil
}

// List returns all resources for a tenant.
func (s *Service) List(ctx context.Context, tenantID string) ([]*resource.Resource, error) {
	return s.repo.List(ctx, tenantID)
}

func generateID(prefix string) string {
	return prefix + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
}

func guessType(uri string) string {
	switch {
	case len(uri) > 5 && uri[:7] == "http://" || uri[:8] == "https://":
		return "web"
	case len(uri) > 3 && uri[len(uri)-4:] == ".pdf":
		return "document"
	case len(uri) > 3 && uri[len(uri)-3:] == ".go" || uri[len(uri)-3:] == ".py" || uri[len(uri)-3:] == ".ts":
		return "code"
	default:
		return "document"
	}
}
