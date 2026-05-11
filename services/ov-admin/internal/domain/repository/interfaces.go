package repository

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/model"
)

type AccountRepository interface {
	Create(ctx context.Context, account *model.Account) error
	GetByID(ctx context.Context, id string) (*model.Account, error)
	List(ctx context.Context, limit, offset int) ([]*model.Account, error)
	Delete(ctx context.Context, id string) error
}

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, accountID, id string) (*model.User, error)
	ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*model.User, error)
	Delete(ctx context.Context, accountID, id string) error
}

type APIKeyRepository interface {
	CreateHash(ctx context.Context, keyID, accountID, userID string, role model.Role, hash, prefix, label string, expiresAt *int64) error
	GetHashByPrefix(ctx context.Context, prefix string) (hash string, accountID, userID string, role model.Role, err error)
	Revoke(ctx context.Context, keyID string) error
	ListByAccount(ctx context.Context, accountID string) ([]*model.APIKey, error)
}
