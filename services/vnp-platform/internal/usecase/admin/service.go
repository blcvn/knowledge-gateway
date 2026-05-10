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
