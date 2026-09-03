// Package usecase — Memory Visibility API for admin governance.
// SOL-ENT-003 / TASK-ENT-007
package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// MemoryVisibilityFilter defines query filters for cross-engine memory listing.
type MemoryVisibilityFilter struct {
	TenantID  uuid.UUID
	UserID    string // optional: scope to a specific user
	Engine    string // optional: "graphiti" | "cognee" | "memobase" | "zep" | "sm" | ""
	After     *time.Time
	Before    *time.Time
	Tags      []string
	Limit     int
	Offset    int
}

// MemoryRecord is a normalized memory record from any engine.
type MemoryRecord struct {
	ID        string    `json:"id"`
	Engine    string    `json:"engine"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id,omitempty"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MemoryVisibilityRepo is the port for cross-engine memory access.
type MemoryVisibilityRepo interface {
	ListMemories(ctx context.Context, filter MemoryVisibilityFilter) ([]*MemoryRecord, int, error)
	DeleteMemory(ctx context.Context, tenantID uuid.UUID, engine, memoryID string) error
	GetMemoryDetails(ctx context.Context, tenantID uuid.UUID, engine, memoryID string) (*MemoryRecord, error)
}

// MemoryVisibilityService provides admin governance APIs for memory access.
type MemoryVisibilityService struct {
	repo   MemoryVisibilityRepo
	audit  *AuditService
	logger *slog.Logger
}

// NewMemoryVisibilityService creates a new MemoryVisibilityService.
func NewMemoryVisibilityService(repo MemoryVisibilityRepo, audit *AuditService, logger *slog.Logger) *MemoryVisibilityService {
	return &MemoryVisibilityService{repo: repo, audit: audit, logger: logger}
}

// ListMemories lists memories across all engines for a tenant (governance view).
func (s *MemoryVisibilityService) ListMemories(ctx context.Context, filter MemoryVisibilityFilter) ([]*MemoryRecord, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	records, total, err := s.repo.ListMemories(ctx, filter)
	if err != nil {
		s.logger.Error("memory_visibility: list failed", "tenant_id", filter.TenantID, "error", err)
		return nil, 0, err
	}
	s.logger.Info("memory_visibility: listed", "tenant_id", filter.TenantID, "count", len(records))
	return records, total, nil
}

// DeleteMemory removes a specific memory from an engine (admin override / GDPR).
func (s *MemoryVisibilityService) DeleteMemory(ctx context.Context, tenantID uuid.UUID, engine, memoryID, adminUserID string) error {
	if err := s.repo.DeleteMemory(ctx, tenantID, engine, memoryID); err != nil {
		return err
	}
	// Audit the admin deletion
	if s.audit != nil {
		_, _ = s.audit.Record(ctx, tenantID, adminUserID, "memory.deleted", engine, memoryID)
	}
	return nil
}

// GetMemoryDetails returns full details of a single memory record.
func (s *MemoryVisibilityService) GetMemoryDetails(ctx context.Context, tenantID uuid.UUID, engine, memoryID string) (*MemoryRecord, error) {
	return s.repo.GetMemoryDetails(ctx, tenantID, engine, memoryID)
}
