package port

import (
	"context"

	"vnp-memory/services/ov-crypto/internal/usecase/dto"
)

// CryptoUseCase defines the inbound operations for the crypto service.
type CryptoUseCase interface {
	Encrypt(ctx context.Context, req dto.EncryptRequest) (*dto.EncryptResponse, error)
	Decrypt(ctx context.Context, req dto.DecryptRequest) (*dto.DecryptResponse, error)
	RotateKey(ctx context.Context, req dto.RotateKeyRequest) (*dto.RotateKeyResponse, error)
	GetKeyStatus(ctx context.Context, accountID string) (*dto.KeyStatusResponse, error)
	
	// API Key features
	ValidateAPIKey(ctx context.Context, req dto.ValidateAPIKeyRequest) (*dto.ValidateAPIKeyResponse, error)
}
