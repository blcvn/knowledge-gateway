package data

import (
	"context"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type rulesRepo struct {
	data *Data
	log  *log.Helper
}

// NewRulesRepo .
func NewRulesRepo(data *Data, logger log.Logger) biz.RulesRepo {
	return &rulesRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *rulesRepo) CreateRule(ctx context.Context, rule *biz.Rule) (*biz.Rule, error) {
	if err := r.data.db.WithContext(ctx).Create(rule).Error; err != nil {
		r.log.Errorf("failed to create rule: %v", err)
		return nil, err
	}
	return rule, nil
}

func (r *rulesRepo) GetRule(ctx context.Context, id uint) (*biz.Rule, error) {
	var rule biz.Rule
	if err := r.data.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *rulesRepo) ListRules(ctx context.Context, appID string) ([]*biz.Rule, error) {
	var rules []*biz.Rule
	if err := r.data.db.WithContext(ctx).Where("app_id = ?", appID).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}
