package usecase

import (
	"context"
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/argon2"

	"vnp-memory/services/ov-crypto/internal/domain/repository"
	"vnp-memory/services/ov-crypto/internal/usecase/dto"
)

type apiKeyUseCase struct {
	repo repository.KeyRepository
	// Argon2id parameters
	time    uint32
	memory  uint32
	threads uint8
}

func NewAPIKeyUseCase(repo repository.KeyRepository, time uint32, memory uint32, threads uint8) *apiKeyUseCase {
	return &apiKeyUseCase{
		repo:    repo,
		time:    time,
		memory:  memory,
		threads: threads,
	}
}

func (uc *apiKeyUseCase) ValidateAPIKey(ctx context.Context, req dto.ValidateAPIKeyRequest) (*dto.ValidateAPIKeyResponse, error) {
	if len(req.RawKey) < 8 {
		return nil, fmt.Errorf("invalid API key format")
	}

	prefix := req.RawKey[:8]
	
	// Fetch hash by prefix and account
	apiKeyMeta, err := uc.repo.GetAPIKeyByPrefix(ctx, req.AccountID, prefix)
	if err != nil {
		return &dto.ValidateAPIKeyResponse{IsValid: false}, nil // Do not leak error
	}

	// Re-hash the provided raw key
	// In a real implementation, a salt should be retrieved from the apiKeyMeta or encoded with the hash
	// Here we use a dummy salt for demonstration
	salt := []byte("constant-salt-for-demo") // DO NOT USE IN PRODUCTION
	
	hash := argon2.IDKey([]byte(req.RawKey), salt, uc.time, uc.memory, uc.threads, 32)

	// Compare hashes securely
	if subtle.ConstantTimeCompare(hash, apiKeyMeta.KeyHash) == 1 {
		// Valid
		_ = uc.repo.UpdateAPIKeyLastUsed(ctx, apiKeyMeta.ID)
		return &dto.ValidateAPIKeyResponse{
			IsValid: true,
			Role:    string(apiKeyMeta.Role),
		}, nil
	}

	return &dto.ValidateAPIKeyResponse{IsValid: false}, nil
}
