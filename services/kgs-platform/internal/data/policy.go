package data

import (
	"context"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type policyRepo struct {
	data *Data
	log  *log.Helper
}

// NewPolicyRepo .
func NewPolicyRepo(data *Data, logger log.Logger) biz.PolicyRepo {
	return &policyRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *policyRepo) CreatePolicy(ctx context.Context, policy *biz.Policy) (*biz.Policy, error) {
	if err := r.data.db.WithContext(ctx).Create(policy).Error; err != nil {
		r.log.Errorf("failed to create policy: %v", err)
		return nil, err
	}
	return policy, nil
}

func (r *policyRepo) GetPolicy(ctx context.Context, id uint) (*biz.Policy, error) {
	var policy biz.Policy
	if err := r.data.db.WithContext(ctx).First(&policy, id).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *policyRepo) ListPolicies(ctx context.Context, appID string) ([]*biz.Policy, error) {
	var policies []*biz.Policy
	if err := r.data.db.WithContext(ctx).Where("app_id = ?", appID).Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}
