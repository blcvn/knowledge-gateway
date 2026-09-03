// Package admin implements tenant management usecases.
package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
)

// TenantService implements port.TenantUseCase.
type TenantService struct {
	repo      port.TenantRepository
	publisher port.EventPublisher
}

// NewTenantService creates a new TenantService.
func NewTenantService(repo port.TenantRepository, pub port.EventPublisher) *TenantService {
	return &TenantService{repo: repo, publisher: pub}
}

func (s *TenantService) CreateTenant(ctx context.Context, name, slug string, tier admin.SubscriptionTier) (*admin.Tenant, error) {
	tenant := &admin.Tenant{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Tier:      tier,
		Status:    admin.TenantActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	// Emit domain event for all engines to create their tenant contexts
	if err := s.publisher.PublishTenantCreated(ctx, tenant); err != nil {
		// Log but don't fail — eventually consistent
		fmt.Printf("warn: failed to publish tenant.created: %v\n", err)
	}

	return tenant, nil
}

func (s *TenantService) GetTenant(ctx context.Context, id uuid.UUID) (*admin.Tenant, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TenantService) UpdateTenant(ctx context.Context, id uuid.UUID, updates map[string]any) (*admin.Tenant, error) {
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Apply updates
	if name, ok := updates["name"].(string); ok {
		tenant.Name = name
	}
	if slug, ok := updates["slug"].(string); ok {
		tenant.Slug = slug
	}
	tenant.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}
	return tenant, nil
}

func (s *TenantService) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	// Emit cascade delete event
	if err := s.publisher.PublishTenantDeleted(ctx, id); err != nil {
		fmt.Printf("warn: failed to publish tenant.deleted: %v\n", err)
	}
	return nil
}

func (s *TenantService) ListTenants(ctx context.Context, offset, limit int) ([]*admin.Tenant, int, error) {
	return s.repo.List(ctx, offset, limit)
}

// --- API Key Service ---

// APIKeyService implements port.APIKeyUseCase.
type APIKeyService struct {
	repo port.APIKeyRepository
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(repo port.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

func (s *APIKeyService) CreateKey(ctx context.Context, tenantID uuid.UUID, name string, permissions []string) (*admin.APIKey, string, error) {
	// Generate random API key
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generate key: %w", err)
	}
	rawKey := "vnp_" + hex.EncodeToString(raw)

	// Hash for storage
	hash := sha256.Sum256([]byte(rawKey))

	key := &admin.APIKey{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		KeyHash:     hex.EncodeToString(hash[:]),
		KeyPrefix:   rawKey[:12],
		Permissions: permissions,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, "", fmt.Errorf("create key: %w", err)
	}

	return key, rawKey, nil // Return raw key only on creation
}

func (s *APIKeyService) ValidateKey(ctx context.Context, rawKey string) (*admin.APIKey, error) {
	hash := sha256.Sum256([]byte(rawKey))
	key, err := s.repo.FindByHash(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		return nil, fmt.Errorf("validate key: %w", err)
	}
	if key.IsRevoked() {
		return nil, fmt.Errorf("key is revoked")
	}
	if key.IsExpired() {
		return nil, fmt.Errorf("key has expired")
	}
	return key, nil
}

func (s *APIKeyService) RevokeKey(ctx context.Context, id uuid.UUID) error {
	return s.repo.Revoke(ctx, id)
}

func (s *APIKeyService) ListKeys(ctx context.Context, tenantID uuid.UUID) ([]*admin.APIKey, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// --- User Service ---

// UserService implements port.UserUseCase.
type UserService struct {
	repo port.UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(repo port.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, tenantID uuid.UUID, email, name string, role admin.UserRole) (*admin.User, error) {
	user := &admin.User{
		ID:       uuid.New(),
		TenantID: tenantID,
		Email:    email,
		Name:     name,
		Role:     role,
		Active:   true,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*admin.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]any) (*admin.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name, ok := updates["name"].(string); ok {
		user.Name = name
	}
	if role, ok := updates["role"].(string); ok {
		user.Role = admin.UserRole(role)
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*admin.User, int, error) {
	return s.repo.ListByTenant(ctx, tenantID, offset, limit)
}

// --- Health Service ---

// HealthService implements port.HealthUseCase.
type HealthService struct {
	checker port.ServiceHealthChecker
}

// NewHealthService creates a HealthService.
// checker may be nil — in that case it returns an empty health status list.
func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) AggregatedHealth(ctx context.Context) ([]*admin.HealthStatus, error) {
	if s.checker == nil {
		return []*admin.HealthStatus{
			{Service: "vnp-platform", Status: "SERVING"},
		}, nil
	}
	return s.checker.CheckAll(ctx)
}
