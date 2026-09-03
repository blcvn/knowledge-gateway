package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/usecase/port"
)

type apiKeyUseCase struct {
	repo repository.APIKeyRepository
	hash port.HashPort
}

func NewAPIKeyUseCase(repo repository.APIKeyRepository, hash port.HashPort) port.APIKeyUseCase {
	return &apiKeyUseCase{repo: repo, hash: hash}
}

func (u *apiKeyUseCase) CreateAPIKey(ctx context.Context, accountID, userID string, role model.Role, label string, expiresAt *int64) (*model.APIKey, error) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, err
	}
	rawSecret := hex.EncodeToString(secretBytes)
	prefix := rawSecret[:8]
	
	hashed, err := u.hash.HashPassword(ctx, rawSecret)
	if err != nil {
		return nil, err
	}

	keyID := uuid.NewString()
	
	err = u.repo.CreateHash(ctx, keyID, accountID, userID, role, hashed, prefix, label, expiresAt)
	if err != nil {
		return nil, err
	}

	var exp *time.Time
	if expiresAt != nil {
		t := time.Unix(*expiresAt, 0)
		exp = &t
	}

	return &model.APIKey{
		KeyID:     keyID,
		AccountID: accountID,
		UserID:    userID,
		Role:      role,
		Label:     label,
		Prefix:    prefix,
		ExpiresAt: exp,
		CreatedAt: time.Now(),
		// Raw key returned once, in an actual scenario we would return it wrapped in response
	}, nil
}

func (u *apiKeyUseCase) ValidateAPIKey(ctx context.Context, rawKey string) (*model.ValidateResult, error) {
	if len(rawKey) < 8 {
		return nil, domain.ErrInvalidKey
	}
	prefix := rawKey[:8]
	
	hash, accountID, userID, role, err := u.repo.GetHashByPrefix(ctx, prefix)
	if err != nil {
		return nil, domain.ErrInvalidKey
	}

	valid, err := u.hash.ComparePassword(ctx, hash, rawKey)
	if err != nil || !valid {
		return nil, domain.ErrInvalidKey
	}

	return &model.ValidateResult{
		Valid:     true,
		AccountID: accountID,
		UserID:    userID,
		Role:      role,
	}, nil
}

func (u *apiKeyUseCase) RevokeAPIKey(ctx context.Context, keyID string) error {
	return u.repo.Revoke(ctx, keyID)
}

func (u *apiKeyUseCase) ListAPIKeys(ctx context.Context, accountID string) ([]*model.APIKey, error) {
	return u.repo.ListByAccount(ctx, accountID)
}
