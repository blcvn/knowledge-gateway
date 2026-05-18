package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/model"
)

// mockPolicyRepo implements repository.PolicyRepository for testing.
type mockPolicyRepo struct {
	policies map[uuid.UUID]*model.Policy
}

func newMockPolicyRepo() *mockPolicyRepo {
	return &mockPolicyRepo{policies: make(map[uuid.UUID]*model.Policy)}
}

func (m *mockPolicyRepo) Create(_ context.Context, p *model.Policy) error {
	m.policies[p.ID] = p
	return nil
}

func (m *mockPolicyRepo) FindByID(_ context.Context, id uuid.UUID) (*model.Policy, error) {
	p, ok := m.policies[id]
	if !ok {
		return nil, model.ErrPolicyNotFound
	}
	return p, nil
}

func (m *mockPolicyRepo) Update(_ context.Context, p *model.Policy) error {
	m.policies[p.ID] = p
	return nil
}

func (m *mockPolicyRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.policies, id)
	return nil
}

func (m *mockPolicyRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*model.Policy, error) {
	var result []*model.Policy
	for _, p := range m.policies {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result, nil
}

func TestPolicyService_CreateAndGet(t *testing.T) {
	repo := newMockPolicyRepo()
	svc := NewPolicyService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	tenantID := uuid.New()

	p, err := svc.Create(context.Background(), tenantID, "block-pii", "Block PII data", "package vnp", "memory")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if p.Name != "block-pii" {
		t.Errorf("expected name=block-pii, got %s", p.Name)
	}
	if p.Status != model.PolicyStatusDraft {
		t.Errorf("expected status=draft, got %s", p.Status)
	}

	got, err := svc.Get(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("expected ID=%s, got %s", p.ID, got.ID)
	}
}

func TestPolicyService_Update(t *testing.T) {
	repo := newMockPolicyRepo()
	svc := NewPolicyService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	tenantID := uuid.New()

	p, _ := svc.Create(context.Background(), tenantID, "rule-1", "desc", "package test", "global")

	newName := "rule-1-updated"
	active := model.PolicyStatusActive
	updated, err := svc.Update(context.Background(), p.ID, &newName, nil, nil, &active)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "rule-1-updated" {
		t.Errorf("expected name=rule-1-updated, got %s", updated.Name)
	}
	if updated.Status != model.PolicyStatusActive {
		t.Errorf("expected status=active, got %s", updated.Status)
	}
}

func TestPolicyService_Delete(t *testing.T) {
	repo := newMockPolicyRepo()
	svc := NewPolicyService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	tenantID := uuid.New()

	p, _ := svc.Create(context.Background(), tenantID, "to-delete", "desc", "package del", "memory")
	err := svc.Delete(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.Get(context.Background(), p.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestPolicyService_ListByTenant(t *testing.T) {
	repo := newMockPolicyRepo()
	svc := NewPolicyService(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	tenantID := uuid.New()
	otherTenant := uuid.New()

	svc.Create(context.Background(), tenantID, "p1", "d1", "package p1", "global")
	svc.Create(context.Background(), tenantID, "p2", "d2", "package p2", "memory")
	svc.Create(context.Background(), otherTenant, "p3", "d3", "package p3", "global")

	policies, err := svc.ListByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ListByTenant failed: %v", err)
	}
	if len(policies) != 2 {
		t.Errorf("expected 2 policies for tenant, got %d", len(policies))
	}
}
