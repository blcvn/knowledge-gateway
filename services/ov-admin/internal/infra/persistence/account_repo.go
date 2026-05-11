package persistence

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/model"
)

type accountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *accountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(ctx context.Context, account *model.Account) error {
	policyBytes, err := json.Marshal(account.NamespacePolicy)
	if err != nil {
		return err
	}
	metadataBytes, err := json.Marshal(account.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO ov_accounts (id, name, namespace_policy, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = r.db.ExecContext(ctx, query, account.ID, account.Name, policyBytes, account.Status, metadataBytes, account.CreatedAt, account.UpdatedAt)
	return err
}

func (r *accountRepository) GetByID(ctx context.Context, id string) (*model.Account, error) {
	query := `
		SELECT id, name, namespace_policy, status, metadata, created_at, updated_at
		FROM ov_accounts
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var acc model.Account
	var policyBytes, metadataBytes []byte
	err := row.Scan(&acc.ID, &acc.Name, &policyBytes, &acc.Status, &metadataBytes, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Or domain.ErrAccountNotFound
		}
		return nil, err
	}

	if len(policyBytes) > 0 {
		_ = json.Unmarshal(policyBytes, &acc.NamespacePolicy)
	}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &acc.Metadata)
	}

	return &acc, nil
}

func (r *accountRepository) List(ctx context.Context, limit, offset int) ([]*model.Account, error) {
	query := `
		SELECT id, name, namespace_policy, status, metadata, created_at, updated_at
		FROM ov_accounts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var acc model.Account
		var policyBytes, metadataBytes []byte
		if err := rows.Scan(&acc.ID, &acc.Name, &policyBytes, &acc.Status, &metadataBytes, &acc.CreatedAt, &acc.UpdatedAt); err != nil {
			return nil, err
		}
		if len(policyBytes) > 0 {
			_ = json.Unmarshal(policyBytes, &acc.NamespacePolicy)
		}
		if len(metadataBytes) > 0 {
			_ = json.Unmarshal(metadataBytes, &acc.Metadata)
		}
		accounts = append(accounts, &acc)
	}
	return accounts, nil
}

func (r *accountRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM ov_accounts WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
