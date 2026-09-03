package surrealdb

import (
	"context"
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

// ── Rules Repo ────────────────────────────────────────────────

type surrealRulesRepo struct {
	client *Client
	log    *log.Helper
}

func NewSurrealRulesRepo(client *Client, logger log.Logger) biz.RulesRepo {
	return &surrealRulesRepo{client: client, log: log.NewHelper(logger)}
}

func (r *surrealRulesRepo) CreateRule(ctx context.Context, rule *biz.Rule) (*biz.Rule, error) {
	sql := `CREATE kgs_rules SET
		app_id = $app_id, tenant_id = $tenant_id, name = $name,
		description = $description, trigger_type = $trigger_type,
		cron = $cron, cypher_query = $cypher_query,
		action = $action, payload = $payload,
		is_active = $is_active, created_at = time::now(), updated_at = time::now()`
	result, err := r.client.Query(ctx, sql, map[string]any{
		"app_id":       rule.AppID,
		"tenant_id":    rule.TenantID,
		"name":         rule.Name,
		"description":  rule.Description,
		"trigger_type": rule.TriggerType,
		"cron":         rule.Cron,
		"cypher_query": rule.CypherQuery,
		"action":       rule.Action,
		"is_active":    rule.IsActive,
		"payload":      nil,
	})
	if err != nil {
		return nil, err
	}
	created, err := unmarshalOne[biz.Rule](result)
	if err != nil || created == nil {
		return rule, nil
	}
	return created, nil
}

func (r *surrealRulesRepo) GetRule(ctx context.Context, id uint) (*biz.Rule, error) {
	sql := `SELECT * FROM kgs_rules WHERE id = $id LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	rule, err := unmarshalOne[biz.Rule](result)
	if err != nil || rule == nil {
		return nil, fmt.Errorf("rule not found: %d", id)
	}
	return rule, nil
}

func (r *surrealRulesRepo) ListRules(ctx context.Context, appID string) ([]*biz.Rule, error) {
	sql := `SELECT * FROM kgs_rules WHERE app_id = $app_id AND deleted_at IS NONE ORDER BY created_at DESC`
	result, err := r.client.Query(ctx, sql, map[string]any{"app_id": appID})
	if err != nil {
		return nil, err
	}
	rules, err := unmarshalSlice[biz.Rule](result)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.Rule, 0, len(rules))
	for i := range rules {
		out = append(out, &rules[i])
	}
	return out, nil
}

var _ biz.RulesRepo = (*surrealRulesRepo)(nil)

// ── Policy Repo ───────────────────────────────────────────────

type surrealPolicyRepo struct {
	client *Client
	log    *log.Helper
}

func NewSurrealPolicyRepo(client *Client, logger log.Logger) biz.PolicyRepo {
	return &surrealPolicyRepo{client: client, log: log.NewHelper(logger)}
}

func (r *surrealPolicyRepo) CreatePolicy(ctx context.Context, policy *biz.Policy) (*biz.Policy, error) {
	sql := `CREATE kgs_policies SET
		app_id = $app_id, tenant_id = $tenant_id, name = $name,
		description = $description, rego_content = $rego_content,
		is_active = $is_active, created_at = time::now(), updated_at = time::now()`
	result, err := r.client.Query(ctx, sql, map[string]any{
		"app_id":       policy.AppID,
		"tenant_id":    policy.TenantID,
		"name":         policy.Name,
		"description":  policy.Description,
		"rego_content": policy.RegoContent,
		"is_active":    policy.IsActive,
	})
	if err != nil {
		return nil, err
	}
	created, err := unmarshalOne[biz.Policy](result)
	if err != nil || created == nil {
		return policy, nil
	}
	return created, nil
}

func (r *surrealPolicyRepo) GetPolicy(ctx context.Context, id uint) (*biz.Policy, error) {
	sql := `SELECT * FROM kgs_policies WHERE id = $id LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	policy, err := unmarshalOne[biz.Policy](result)
	if err != nil || policy == nil {
		return nil, fmt.Errorf("policy not found: %d", id)
	}
	return policy, nil
}

func (r *surrealPolicyRepo) ListPolicies(ctx context.Context, appID string) ([]*biz.Policy, error) {
	sql := `SELECT * FROM kgs_policies WHERE app_id = $app_id AND deleted_at IS NONE ORDER BY created_at DESC`
	result, err := r.client.Query(ctx, sql, map[string]any{"app_id": appID})
	if err != nil {
		return nil, err
	}
	policies, err := unmarshalSlice[biz.Policy](result)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.Policy, 0, len(policies))
	for i := range policies {
		out = append(out, &policies[i])
	}
	return out, nil
}

var _ biz.PolicyRepo = (*surrealPolicyRepo)(nil)
