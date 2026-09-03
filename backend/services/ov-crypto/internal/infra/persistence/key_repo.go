package persistence

import (
	"context"
	"database/sql"
	"errors"

	"vnp-memory/services/ov-crypto/internal/domain"
	"vnp-memory/services/ov-crypto/internal/domain/model"
	"vnp-memory/services/ov-crypto/internal/domain/repository"
)

type keyRepoImpl struct {
	db *sql.DB
}

func NewKeyRepository(db *sql.DB) repository.KeyRepository {
	return &keyRepoImpl{db: db}
}

func (r *keyRepoImpl) GetActiveAccountKey(ctx context.Context, accountID string) (*model.AccountKey, error) {
	query := `SELECT id, account_id, key_version, provider_type, encrypted_root_key, key_reference, status, created_at, rotated_at, expires_at 
	          FROM ov_account_keys 
	          WHERE account_id = $1 AND status = 'active' ORDER BY key_version DESC LIMIT 1`
	
	var key model.AccountKey
	err := r.db.QueryRowContext(ctx, query, accountID).Scan(
		&key.ID, &key.AccountID, &key.KeyVersion, &key.ProviderType, &key.EncryptedRootKey,
		&key.KeyReference, &key.Status, &key.CreatedAt, &key.RotatedAt, &key.ExpiresAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *keyRepoImpl) CreateAccountKey(ctx context.Context, key *model.AccountKey) error {
	// Implementation omitted
	return nil
}

func (r *keyRepoImpl) UpdateAccountKeyStatus(ctx context.Context, accountID string, version model.KeyVersion, status model.KeyStatus) error {
	// Implementation omitted
	return nil
}

func (r *keyRepoImpl) RecordRotation(ctx context.Context, log *model.KeyRotationLog) error {
	// Implementation omitted
	return nil
}

func (r *keyRepoImpl) UpdateRotationStatus(ctx context.Context, logID string, status string, durationMs int) error {
	// Implementation omitted
	return nil
}

func (r *keyRepoImpl) GetAPIKeyByPrefix(ctx context.Context, accountID string, prefix string) (*model.APIKeyHash, error) {
	// Implementation omitted
	return nil, domain.ErrNotFound
}

func (r *keyRepoImpl) CreateAPIKey(ctx context.Context, apiKey *model.APIKeyHash) error {
	// Implementation omitted
	return nil
}

func (r *keyRepoImpl) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	// Implementation omitted
	return nil
}
