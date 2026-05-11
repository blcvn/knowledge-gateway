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

type accountUseCase struct {
	repo repository.AccountRepository
}

func NewAccountUseCase(repo repository.AccountRepository) port.AccountUseCase {
	return &accountUseCase{repo: repo}
}

func (u *accountUseCase) CreateAccount(ctx context.Context, name string, policy model.NamespacePolicy) (*model.Account, error) {
	acc := &model.Account{
		ID:              uuid.NewString(),
		Name:            name,
		NamespacePolicy: policy,
		Status:          "active",
		Metadata:        make(map[string]any),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := u.repo.Create(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (u *accountUseCase) GetAccount(ctx context.Context, id string) (*model.Account, error) {
	acc, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, domain.ErrAccountNotFound
	}
	return acc, nil
}

func (u *accountUseCase) ListAccounts(ctx context.Context, limit, offset int) ([]*model.Account, error) {
	return u.repo.List(ctx, limit, offset)
}

func (u *accountUseCase) DeleteAccount(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
