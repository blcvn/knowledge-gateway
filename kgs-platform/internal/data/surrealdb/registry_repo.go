package surrealdb

import (
	"context"
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type surrealRegistryRepo struct {
	client *Client
	log    *log.Helper
}

func NewSurrealRegistryRepo(client *Client, logger log.Logger) biz.RegistryRepo {
	return &surrealRegistryRepo{
		client: client,
		log:    log.NewHelper(logger),
	}
}

func (r *surrealRegistryRepo) CreateApp(ctx context.Context, app *biz.App) error {
	sql := `CREATE type::thing('kgs_apps', $app_id) SET
		app_id = $app_id,
		app_name = $app_name,
		description = $description,
		owner = $owner,
		status = $status,
		created_at = time::now(),
		updated_at = time::now()`
	_, err := r.client.Query(ctx, sql, map[string]any{
		"app_id":      app.AppID,
		"app_name":    app.AppName,
		"description": app.Description,
		"owner":       app.Owner,
		"status":      app.Status,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] CreateApp failed app_id=%s err=%v", app.AppID, err)
		return err
	}
	r.log.Infof("[KGS][SurrealDB] CreateApp done app_id=%s", app.AppID)
	return nil
}

func (r *surrealRegistryRepo) GetApp(ctx context.Context, appID string) (*biz.App, error) {
	sql := `SELECT * FROM kgs_apps WHERE app_id = $app_id AND deleted_at IS NONE LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{"app_id": appID})
	if err != nil {
		return nil, err
	}
	apps, err := unmarshalSlice[biz.App](result)
	if err != nil || len(apps) == 0 {
		return nil, fmt.Errorf("app not found: %s", appID)
	}
	return &apps[0], nil
}

func (r *surrealRegistryRepo) GetAppByExternalID(ctx context.Context, externalID string) (*biz.App, error) {
	sql := `SELECT * FROM kgs_apps WHERE external_id = $external_id AND deleted_at IS NONE LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{"external_id": externalID})
	if err != nil {
		return nil, err
	}
	apps, err := unmarshalSlice[biz.App](result)
	if err != nil || len(apps) == 0 {
		return nil, fmt.Errorf("app not found by external_id: %s", externalID)
	}
	return &apps[0], nil
}

func (r *surrealRegistryRepo) ListApps(ctx context.Context) ([]*biz.App, error) {
	sql := `SELECT * FROM kgs_apps WHERE deleted_at IS NONE ORDER BY created_at DESC`
	result, err := r.client.Query(ctx, sql, nil)
	if err != nil {
		return nil, err
	}
	apps, err := unmarshalSlice[biz.App](result)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.App, 0, len(apps))
	for i := range apps {
		out = append(out, &apps[i])
	}
	return out, nil
}

func (r *surrealRegistryRepo) CreateAPIKey(ctx context.Context, key *biz.APIKey) error {
	sql := `CREATE type::thing('kgs_api_keys', $key_hash) SET
		key_hash = $key_hash,
		app_id = $app_id,
		key_prefix = $key_prefix,
		name = $name,
		scopes = $scopes,
		is_revoked = false,
		expires_at = $expires_at,
		created_at = time::now()`
	_, err := r.client.Query(ctx, sql, map[string]any{
		"key_hash":   key.KeyHash,
		"app_id":     key.AppID,
		"key_prefix": key.KeyPrefix,
		"name":       key.Name,
		"scopes":     key.Scopes,
		"expires_at": key.ExpiresAt,
	})
	return err
}

func (r *surrealRegistryRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (*biz.APIKey, error) {
	sql := `SELECT * FROM kgs_api_keys WHERE key_hash = $key_hash LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{"key_hash": keyHash})
	if err != nil {
		return nil, err
	}
	keys, err := unmarshalSlice[biz.APIKey](result)
	if err != nil || len(keys) == 0 {
		return nil, fmt.Errorf("api key not found")
	}
	return &keys[0], nil
}

func (r *surrealRegistryRepo) RevokeAPIKey(ctx context.Context, keyHash string) error {
	sql := `UPDATE kgs_api_keys SET is_revoked = true, updated_at = time::now() WHERE key_hash = $key_hash`
	_, err := r.client.Query(ctx, sql, map[string]any{"key_hash": keyHash})
	return err
}

func (r *surrealRegistryRepo) GetQuota(ctx context.Context, appID, quotaType string) (*biz.Quota, error) {
	sql := `SELECT * FROM kgs_quotas WHERE app_id = $app_id AND quota_type = $quota_type LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{
		"app_id":     appID,
		"quota_type": quotaType,
	})
	if err != nil {
		return nil, err
	}
	quotas, err := unmarshalSlice[biz.Quota](result)
	if err != nil || len(quotas) == 0 {
		return nil, fmt.Errorf("quota not found for app=%s type=%s", appID, quotaType)
	}
	return &quotas[0], nil
}

var _ biz.RegistryRepo = (*surrealRegistryRepo)(nil)
