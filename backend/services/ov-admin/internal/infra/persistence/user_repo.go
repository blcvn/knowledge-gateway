package persistence

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/model"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	metadataBytes, err := json.Marshal(user.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO ov_users (id, account_id, name, role, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = r.db.ExecContext(ctx, query, user.ID, user.AccountID, user.Name, user.Role, user.Status, metadataBytes, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, accountID, id string) (*model.User, error) {
	query := `
		SELECT id, account_id, name, role, status, metadata, created_at, updated_at
		FROM ov_users
		WHERE account_id = $1 AND id = $2
	`
	row := r.db.QueryRowContext(ctx, query, accountID, id)

	var user model.User
	var metadataBytes []byte
	err := row.Scan(&user.ID, &user.AccountID, &user.Name, &user.Role, &user.Status, &metadataBytes, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &user.Metadata)
	}
	return &user, nil
}

func (r *userRepository) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*model.User, error) {
	query := `
		SELECT id, account_id, name, role, status, metadata, created_at, updated_at
		FROM ov_users
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		var metadataBytes []byte
		if err := rows.Scan(&user.ID, &user.AccountID, &user.Name, &user.Role, &user.Status, &metadataBytes, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		if len(metadataBytes) > 0 {
			_ = json.Unmarshal(metadataBytes, &user.Metadata)
		}
		users = append(users, &user)
	}
	return users, nil
}

func (r *userRepository) Delete(ctx context.Context, accountID, id string) error {
	query := `DELETE FROM ov_users WHERE account_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, accountID, id)
	return err
}
