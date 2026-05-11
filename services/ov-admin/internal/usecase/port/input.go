package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/model"
)

type AccountUseCase interface {
	CreateAccount(ctx context.Context, name string, policy model.NamespacePolicy) (*model.Account, error)
	GetAccount(ctx context.Context, id string) (*model.Account, error)
	ListAccounts(ctx context.Context, limit, offset int) ([]*model.Account, error)
	DeleteAccount(ctx context.Context, id string) error
}

type UserUseCase interface {
	CreateUser(ctx context.Context, accountID, name string, role model.Role) (*model.User, error)
	GetUser(ctx context.Context, accountID, id string) (*model.User, error)
	ListUsers(ctx context.Context, accountID string, limit, offset int) ([]*model.User, error)
	DeleteUser(ctx context.Context, accountID, id string) error
}

type APIKeyUseCase interface {
	CreateAPIKey(ctx context.Context, accountID, userID string, role model.Role, label string, expiresAt *int64) (*model.APIKey, error)
	ValidateAPIKey(ctx context.Context, rawKey string) (*model.ValidateResult, error)
	RevokeAPIKey(ctx context.Context, keyID string) error
	ListAPIKeys(ctx context.Context, accountID string) ([]*model.APIKey, error)
}

type HealthUseCase interface {
	GetHealth(ctx context.Context) (map[string]string, error)
}
