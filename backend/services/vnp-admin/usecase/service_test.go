package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/usecase"
)

// --- Mock repositories ---

type mockTenantRepo struct {
	tenants map[uuid.UUID]*model.Tenant
	byName  map[string]*model.Tenant
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{
		tenants: make(map[uuid.UUID]*model.Tenant),
		byName:  make(map[string]*model.Tenant),
	}
}

func (m *mockTenantRepo) Create(_ context.Context, t *model.Tenant) error {
	m.tenants[t.ID] = t
	m.byName[t.Name] = t
	return nil
}

func (m *mockTenantRepo) FindByID(_ context.Context, id uuid.UUID) (*model.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, model.ErrTenantNotFound
}

func (m *mockTenantRepo) FindByName(_ context.Context, name string) (*model.Tenant, error) {
	if t, ok := m.byName[name]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *mockTenantRepo) Update(_ context.Context, t *model.Tenant) error {
	m.tenants[t.ID] = t
	m.byName[t.Name] = t
	return nil
}

func (m *mockTenantRepo) Delete(_ context.Context, id uuid.UUID) error {
	if t, ok := m.tenants[id]; ok {
		delete(m.byName, t.Name)
		delete(m.tenants, id)
		return nil
	}
	return model.ErrTenantNotFound
}

func (m *mockTenantRepo) List(_ context.Context, offset, limit int) ([]*model.Tenant, int, error) {
	all := make([]*model.Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		all = append(all, t)
	}
	total := len(all)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset >= total {
		return nil, total, nil
	}
	return all[offset:end], total, nil
}

type mockKeyRepo struct {
	keys     map[uuid.UUID]*model.APIKey
	byHash   map[string]*model.APIKey
	byTenant map[uuid.UUID][]*model.APIKey
}

func newMockKeyRepo() *mockKeyRepo {
	return &mockKeyRepo{
		keys:     make(map[uuid.UUID]*model.APIKey),
		byHash:   make(map[string]*model.APIKey),
		byTenant: make(map[uuid.UUID][]*model.APIKey),
	}
}

func (m *mockKeyRepo) Create(_ context.Context, k *model.APIKey) error {
	m.keys[k.ID] = k
	m.byHash[k.KeyHash] = k
	m.byTenant[k.TenantID] = append(m.byTenant[k.TenantID], k)
	return nil
}

func (m *mockKeyRepo) FindByID(_ context.Context, id uuid.UUID) (*model.APIKey, error) {
	if k, ok := m.keys[id]; ok {
		return k, nil
	}
	return nil, model.ErrAPIKeyNotFound
}

func (m *mockKeyRepo) FindByHash(_ context.Context, hash string) (*model.APIKey, error) {
	if k, ok := m.byHash[hash]; ok {
		return k, nil
	}
	return nil, model.ErrAPIKeyNotFound
}

func (m *mockKeyRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*model.APIKey, error) {
	return m.byTenant[tenantID], nil
}

func (m *mockKeyRepo) Revoke(_ context.Context, id uuid.UUID) error {
	if k, ok := m.keys[id]; ok {
		k.Active = false
		return nil
	}
	return model.ErrAPIKeyNotFound
}

type mockPublisher struct {
	events []string
}

func (m *mockPublisher) PublishTenantCreated(_ context.Context, _ uuid.UUID) error {
	m.events = append(m.events, "tenant.created")
	return nil
}
func (m *mockPublisher) PublishTenantDeleted(_ context.Context, _ uuid.UUID) error {
	m.events = append(m.events, "tenant.deleted")
	return nil
}
func (m *mockPublisher) PublishKeyRevoked(_ context.Context, _, _ uuid.UUID) error {
	m.events = append(m.events, "key.revoked")
	return nil
}

// --- Tests ---

func TestTenantService_Create(t *testing.T) {
	repo := newMockTenantRepo()
	pub := &mockPublisher{}
	svc := usecase.NewTenantService(repo, pub)

	tenant, err := svc.Create(context.Background(), "test-tenant", model.PlanStarter)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if tenant.Name != "test-tenant" {
		t.Errorf("Name: got %s, want test-tenant", tenant.Name)
	}
	if tenant.Plan != model.PlanStarter {
		t.Errorf("Plan: got %s, want starter", tenant.Plan)
	}
	if tenant.Config.MaxAPIKeys != 10 {
		t.Errorf("MaxAPIKeys: got %d, want 10 (starter plan)", tenant.Config.MaxAPIKeys)
	}
	if !tenant.Active {
		t.Error("new tenant should be active")
	}

	// Verify persisted
	found, err := svc.Get(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found.ID != tenant.ID {
		t.Error("persisted tenant ID mismatch")
	}
}

func TestTenantService_CreateDuplicate(t *testing.T) {
	repo := newMockTenantRepo()
	pub := &mockPublisher{}
	svc := usecase.NewTenantService(repo, pub)

	_, _ = svc.Create(context.Background(), "dup-tenant", model.PlanFree)
	_, err := svc.Create(context.Background(), "dup-tenant", model.PlanFree)

	if err != model.ErrDuplicateTenant {
		t.Errorf("expected ErrDuplicateTenant, got %v", err)
	}
}

func TestAPIKeyService_CreateAndValidate(t *testing.T) {
	tenantRepo := newMockTenantRepo()
	keyRepo := newMockKeyRepo()
	pub := &mockPublisher{}

	// Create tenant first
	tenantSvc := usecase.NewTenantService(tenantRepo, pub)
	tenant, _ := tenantSvc.Create(context.Background(), "key-test", model.PlanStarter)

	keySvc := usecase.NewAPIKeyService(keyRepo, tenantRepo, pub)

	// Create key
	key, plaintext, err := keySvc.Create(context.Background(), tenant.ID, "my-key", model.KeyScopeReadWrite)
	if err != nil {
		t.Fatalf("Create key: %v", err)
	}
	if key.Name != "my-key" {
		t.Errorf("key name: got %s, want my-key", key.Name)
	}
	if plaintext[:4] != "vnp_" {
		t.Error("plaintext should start with vnp_")
	}

	// Validate key
	validatedKey, validatedTenant, err := keySvc.Validate(context.Background(), plaintext)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if validatedKey.ID != key.ID {
		t.Error("validated key ID mismatch")
	}
	if validatedTenant.ID != tenant.ID {
		t.Error("validated tenant ID mismatch")
	}

	// Validate wrong key
	_, _, err = keySvc.Validate(context.Background(), "vnp_wrong_key_0000000000000000000000000000000000000000000000000000")
	if err != model.ErrAPIKeyInvalid {
		t.Errorf("expected ErrAPIKeyInvalid, got %v", err)
	}
}

func TestAPIKeyService_QuotaExceeded(t *testing.T) {
	tenantRepo := newMockTenantRepo()
	keyRepo := newMockKeyRepo()
	pub := &mockPublisher{}

	// Create free tenant (max 2 keys)
	tenantSvc := usecase.NewTenantService(tenantRepo, pub)
	tenant, _ := tenantSvc.Create(context.Background(), "free-tenant", model.PlanFree)

	keySvc := usecase.NewAPIKeyService(keyRepo, tenantRepo, pub)

	// Create 2 keys (should succeed)
	_, _, _ = keySvc.Create(context.Background(), tenant.ID, "key-1", model.KeyScopeReadOnly)
	_, _, _ = keySvc.Create(context.Background(), tenant.ID, "key-2", model.KeyScopeReadOnly)

	// Third key should fail
	_, _, err := keySvc.Create(context.Background(), tenant.ID, "key-3", model.KeyScopeReadOnly)
	if err != model.ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestAPIKeyService_RevokeAndRevalidate(t *testing.T) {
	tenantRepo := newMockTenantRepo()
	keyRepo := newMockKeyRepo()
	pub := &mockPublisher{}

	tenantSvc := usecase.NewTenantService(tenantRepo, pub)
	tenant, _ := tenantSvc.Create(context.Background(), "revoke-test", model.PlanStarter)

	keySvc := usecase.NewAPIKeyService(keyRepo, tenantRepo, pub)
	key, plaintext, _ := keySvc.Create(context.Background(), tenant.ID, "revokable", model.KeyScopeReadWrite)

	// Revoke
	if err := keySvc.Revoke(context.Background(), key.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// Validate should fail
	_, _, err := keySvc.Validate(context.Background(), plaintext)
	if err != model.ErrAPIKeyRevoked {
		t.Errorf("expected ErrAPIKeyRevoked, got %v", err)
	}
}
