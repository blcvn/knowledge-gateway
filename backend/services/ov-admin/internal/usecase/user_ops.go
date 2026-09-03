package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/usecase/port"
)

type userUseCase struct {
	repo repository.UserRepository
}

func NewUserUseCase(repo repository.UserRepository) port.UserUseCase {
	return &userUseCase{repo: repo}
}

func (u *userUseCase) CreateUser(ctx context.Context, accountID, name string, role model.Role) (*model.User, error) {
	user := &model.User{
		ID:        uuid.NewString(),
		AccountID: accountID,
		Name:      name,
		Role:      role,
		Status:    "active",
		Metadata:  make(map[string]any),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := u.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *userUseCase) GetUser(ctx context.Context, accountID, id string) (*model.User, error) {
	user, err := u.repo.GetByID(ctx, accountID, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (u *userUseCase) ListUsers(ctx context.Context, accountID string, limit, offset int) ([]*model.User, error) {
	return u.repo.ListByAccount(ctx, accountID, limit, offset)
}

func (u *userUseCase) DeleteUser(ctx context.Context, accountID, id string) error {
	return u.repo.Delete(ctx, accountID, id)
}
