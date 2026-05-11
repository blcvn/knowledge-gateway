package usecase

import (
	"context"

	"vnp-memory/services/ov-crypto/internal/usecase/dto"
)

func (kr *keyRotator) GetKeyStatus(ctx context.Context, accountID string) (*dto.KeyStatusResponse, error) {
	keyMeta, err := kr.repo.GetActiveAccountKey(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &dto.KeyStatusResponse{
		Version:     int(keyMeta.KeyVersion),
		Provider:    keyMeta.ProviderType,
		CreatedAt:   keyMeta.CreatedAt,
		LastRotated: keyMeta.RotatedAt,
		Status:      string(keyMeta.Status),
	}, nil
}
