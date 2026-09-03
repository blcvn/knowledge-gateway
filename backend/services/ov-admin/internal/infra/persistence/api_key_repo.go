package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/model"
)

type apiKeyRepository struct {
	db *sql.DB
}

func NewAPIKeyRepository(db *sql.DB) *apiKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) CreateHash(ctx context.Context, keyID, accountID, userID string, role model.Role, hash, prefix, label string, expiresAt *int64) error {
	var exp *time.Time
	if expiresAt != nil {
		t := time.Unix(*expiresAt, 0)
		exp = &t
	}
	
	query := `
		INSERT INTO ov_api_key_hashes (key_id, account_id, user_id, role, hash, prefix, label, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query, keyID, accountID, userID, role, hash, prefix, label, exp, time.Now())
	return err
}

func (r *apiKeyRepository) GetHashByPrefix(ctx context.Context, prefix string) (string, string, string, model.Role, error) {
	query := `
		SELECT hash, account_id, user_id, role 
		FROM ov_api_key_hashes 
		WHERE prefix = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`
	var hash, accountID, userID string
	var role model.Role
	err := r.db.QueryRowContext(ctx, query, prefix).Scan(&hash, &accountID, &userID, &role)
	if err != nil {
		return "", "", "", "", err
	}
	return hash, accountID, userID, role, nil
}

func (r *apiKeyRepository) Revoke(ctx context.Context, keyID string) error {
	query := `DELETE FROM ov_api_key_hashes WHERE key_id = $1`
	_, err := r.db.ExecContext(ctx, query, keyID)
	return err
}

func (r *apiKeyRepository) ListByAccount(ctx context.Context, accountID string) ([]*model.APIKey, error) {
	query := `
		SELECT key_id, account_id, user_id, role, label, prefix, expires_at, created_at
		FROM ov_api_key_hashes
		WHERE account_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		var key model.APIKey
		if err := rows.Scan(&key.KeyID, &key.AccountID, &key.UserID, &key.Role, &key.Label, &key.Prefix, &key.ExpiresAt, &key.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	return keys, nil
}
