package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/repository"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/usecase/port"
)

// APIKeyService implements port.APIKeyUseCase.
type APIKeyService struct {
	keyRepo    repository.APIKeyRepository
	tenantRepo repository.TenantRepository
	pub        port.EventPublisherPort
}

func NewAPIKeyService(keyRepo repository.APIKeyRepository, tenantRepo repository.TenantRepository, pub port.EventPublisherPort) *APIKeyService {
	return &APIKeyService{keyRepo: keyRepo, tenantRepo: tenantRepo, pub: pub}
}

// Create generates a new API key and returns the plaintext ONCE.
func (s *APIKeyService) Create(ctx context.Context, tenantID uuid.UUID, name string, scope model.KeyScope) (*model.APIKey, string, error) {
	// Verify tenant exists and is active
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, "", model.ErrTenantNotFound
	}
	if !tenant.Active {
		return nil, "", model.ErrTenantInactive
	}

	// Check quota
	existing, _ := s.keyRepo.ListByTenant(ctx, tenantID)
	if len(existing) >= tenant.Config.MaxAPIKeys {
		return nil, "", model.ErrQuotaExceeded
	}

	key, plaintext, err := model.GenerateAPIKey(tenantID, name, scope)
	if err != nil {
		return nil, "", fmt.Errorf("generate key: %w", err)
	}

	if err := s.keyRepo.Create(ctx, key); err != nil {
		return nil, "", fmt.Errorf("persist key: %w", err)
	}

	return key, plaintext, nil
}

// Validate checks a plaintext API key and returns the key + tenant if valid.
func (s *APIKeyService) Validate(ctx context.Context, plaintext string) (*model.APIKey, *model.Tenant, error) {
	hash := sha256.Sum256([]byte(plaintext))
	hashStr := hex.EncodeToString(hash[:])

	key, err := s.keyRepo.FindByHash(ctx, hashStr)
	if err != nil {
		return nil, nil, model.ErrAPIKeyInvalid
	}
	if !key.Active {
		return nil, nil, model.ErrAPIKeyRevoked
	}

	tenant, err := s.tenantRepo.FindByID(ctx, key.TenantID)
	if err != nil {
		return nil, nil, model.ErrTenantNotFound
	}

	return key, tenant, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, id uuid.UUID) error {
	key, err := s.keyRepo.FindByID(ctx, id)
	if err != nil {
		return model.ErrAPIKeyNotFound
	}
	if err := s.keyRepo.Revoke(ctx, id); err != nil {
		return fmt.Errorf("revoke key: %w", err)
	}
	_ = s.pub.PublishKeyRevoked(ctx, key.TenantID, id)
	return nil
}

func (s *APIKeyService) List(ctx context.Context, tenantID uuid.UUID) ([]*model.APIKey, error) {
	return s.keyRepo.ListByTenant(ctx, tenantID)
}
