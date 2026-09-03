package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// PolicyRepo defines the persistence interface for Policies
type PolicyRepo interface {
	CreatePolicy(ctx context.Context, policy *Policy) (*Policy, error)
	GetPolicy(ctx context.Context, id uint) (*Policy, error)
	ListPolicies(ctx context.Context, appID string) ([]*Policy, error)
}

type PolicyUsecase struct {
	repo PolicyRepo
	log  *log.Helper
}

func NewPolicyUsecase(repo PolicyRepo, logger log.Logger) *PolicyUsecase {
	return &PolicyUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *PolicyUsecase) CreatePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	// TODO: rego policy syntax validation before saving
	return uc.repo.CreatePolicy(ctx, policy)
}

func (uc *PolicyUsecase) GetPolicy(ctx context.Context, id uint) (*Policy, error) {
	return uc.repo.GetPolicy(ctx, id)
}

func (uc *PolicyUsecase) ListPolicies(ctx context.Context, appID string) ([]*Policy, error) {
	return uc.repo.ListPolicies(ctx, appID)
}
