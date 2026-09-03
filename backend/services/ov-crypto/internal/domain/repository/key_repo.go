package repository

import (
	"context"

	"vnp-memory/services/ov-crypto/internal/domain/model"
)

// KeyRepository defines the storage interface for key metadata and API keys
type KeyRepository interface {
	// Account Keys
	GetActiveAccountKey(ctx context.Context, accountID string) (*model.AccountKey, error)
	CreateAccountKey(ctx context.Context, key *model.AccountKey) error
	UpdateAccountKeyStatus(ctx context.Context, accountID string, version model.KeyVersion, status model.KeyStatus) error
	
	// Key Rotation Log
	RecordRotation(ctx context.Context, log *model.KeyRotationLog) error
	UpdateRotationStatus(ctx context.Context, logID string, status string, durationMs int) error

	// API Keys
	GetAPIKeyByPrefix(ctx context.Context, accountID string, prefix string) (*model.APIKeyHash, error)
	CreateAPIKey(ctx context.Context, apiKey *model.APIKeyHash) error
	UpdateAPIKeyLastUsed(ctx context.Context, id string) error
}
