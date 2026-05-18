package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/model"
)

// mockAuditRepo implements repository.AuditLogRepository for testing.
type mockAuditRepo struct {
	logs []*model.AuditLog
}

func (m *mockAuditRepo) Create(_ context.Context, log *model.AuditLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditRepo) Search(_ context.Context, filter model.AuditLogFilter) ([]*model.AuditLog, int, error) {
	var result []*model.AuditLog
	for _, l := range m.logs {
		if filter.UserID != "" && l.UserID != filter.UserID {
			continue
		}
		if filter.Action != "" && l.Action != filter.Action {
			continue
		}
		if filter.From != nil && l.CreatedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && l.CreatedAt.After(*filter.To) {
			continue
		}
		result = append(result, l)
	}
	// Apply pagination
	total := len(result)
	if filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, total, nil
}

func TestAuditService_Record(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := NewAuditService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	tenantID := uuid.New()
	log, err := svc.Record(context.Background(), tenantID, "user-1", model.AuditActionCreate, "memory", "mem_001")
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
	if log.UserID != "user-1" {
		t.Errorf("expected user_id=user-1, got %s", log.UserID)
	}
	if log.Action != model.AuditActionCreate {
		t.Errorf("expected action=create, got %s", log.Action)
	}
	if len(repo.logs) != 1 {
		t.Errorf("expected 1 log in repo, got %d", len(repo.logs))
	}
}

func TestAuditService_Search_WithFilters(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := NewAuditService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	tenantID := uuid.New()

	// Record multiple events
	svc.Record(context.Background(), tenantID, "user-1", model.AuditActionCreate, "memory", "mem_001")
	svc.Record(context.Background(), tenantID, "user-2", model.AuditActionDelete, "memory", "mem_002")
	svc.Record(context.Background(), tenantID, "user-1", model.AuditActionUpdate, "policy", "pol_001")

	// Search by user
	logs, total, err := svc.Search(context.Background(), model.AuditLogFilter{UserID: "user-1", Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 logs for user-1, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 results, got %d", len(logs))
	}
}

func TestAuditService_Search_DefaultLimit(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := NewAuditService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	// Search with no limit → should default to 50
	_, _, err := svc.Search(context.Background(), model.AuditLogFilter{Limit: 0})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
}

func TestAuditService_Search_TimeRange(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := NewAuditService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	tenantID := uuid.New()

	svc.Record(context.Background(), tenantID, "user-1", model.AuditActionCreate, "memory", "mem_001")
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	now := time.Now()
	future := now.Add(1 * time.Hour)
	logs, _, err := svc.Search(context.Background(), model.AuditLogFilter{To: &future, Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}
