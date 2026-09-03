package data

import (
	"context"
	"errors"
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"
	"gorm.io/gorm"
)

type EntityReader struct {
	graphRepo biz.GraphRepo
	db        *gorm.DB
}

func NewEntityReader(graphRepo biz.GraphRepo, db *gorm.DB) *EntityReader {
	return &EntityReader{
		graphRepo: graphRepo,
		db:        db,
	}
}

func (r *EntityReader) GetEntity(ctx context.Context, appID, tenantID, entityID string) (map[string]any, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("entity reader is not configured")
	}

	if r.graphRepo != nil {
		neo, err := r.graphRepo.GetNode(ctx, appID, tenantID, entityID)
		if err == nil && neo != nil {
			pgVersion, versionErr := getEntityVersionPG(ctx, r.db, entityID)
			if versionErr == nil {
				if pgVersion > intAny(neo["version"]) {
					fallback, pgErr := getEntityPG(ctx, r.db, entityID)
					if pgErr != nil {
						return nil, pgErr
					}
					observability.IncReadPGFallback("stale")
					return kgEntityToMap(*fallback), nil
				}
				return normalizeEntityMap(neo), nil
			}
			if !errors.Is(versionErr, gorm.ErrRecordNotFound) {
				return nil, versionErr
			}
			return normalizeEntityMap(neo), nil
		}
	}

	entity, err := getEntityPG(ctx, r.db, entityID)
	if err != nil {
		return nil, err
	}
	observability.IncReadPGFallback("not_found")
	return kgEntityToMap(*entity), nil
}

func (r *EntityReader) EnrichWithFreshVersions(ctx context.Context, appID, tenantID string, entities []map[string]any) ([]map[string]any, error) {
	if len(entities) == 0 {
		return []map[string]any{}, nil
	}
	if r == nil || r.db == nil {
		return entities, nil
	}

	ids := make([]string, 0, len(entities))
	neoVersions := make(map[string]int, len(entities))
	for _, entity := range entities {
		id := strAny(entity["id"])
		if id == "" {
			continue
		}
		ids = append(ids, id)
		neoVersions[id] = intAny(entity["version"])
	}
	if len(ids) == 0 {
		return entities, nil
	}

	pgVersions, err := getEntityVersionsBatchPG(ctx, r.db, ids)
	if err != nil {
		return nil, err
	}

	staleIDs := make([]string, 0)
	staleSet := make(map[string]struct{})
	for id, pgVersion := range pgVersions {
		if neoVersion, ok := neoVersions[id]; !ok || pgVersion > neoVersion {
			staleSet[id] = struct{}{}
			staleIDs = append(staleIDs, id)
		}
	}
	if len(staleIDs) == 0 {
		out := make([]map[string]any, 0, len(entities))
		for _, entity := range entities {
			out = append(out, normalizeEntityMap(entity))
		}
		return out, nil
	}

	rows, err := getEntitiesBatchPG(ctx, r.db, staleIDs)
	if err != nil {
		return nil, err
	}
	freshByID := make(map[string]map[string]any, len(rows))
	for _, row := range rows {
		freshByID[row.EntityID] = kgEntityToMap(row)
	}

	out := make([]map[string]any, 0, len(entities))
	for _, entity := range entities {
		id := strAny(entity["id"])
		if id == "" {
			out = append(out, normalizeEntityMap(entity))
			continue
		}
		if _, isStale := staleSet[id]; !isStale {
			out = append(out, normalizeEntityMap(entity))
			continue
		}
		if fresh, ok := freshByID[id]; ok {
			observability.IncReadPGFallback("stale")
			out = append(out, fresh)
			continue
		}
		out = append(out, normalizeEntityMap(entity))
	}
	return out, nil
}

func kgEntityToMap(entity KGEntity) map[string]any {
	out := map[string]any(entity.Properties)
	if out == nil {
		out = map[string]any{}
	} else {
		cloned := make(map[string]any, len(out))
		for k, v := range out {
			cloned[k] = v
		}
		out = cloned
	}

	out["id"] = entity.EntityID
	out["label"] = entity.EntityType
	out["entity_type"] = entity.EntityType
	out["name"] = entity.Name
	out["app_id"] = entity.AppID
	out["tenant_id"] = entity.TenantID
	out["version"] = entity.Version
	out["is_deleted"] = entity.IsDeleted
	return out
}

func normalizeEntityMap(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	if out["id"] == nil {
		if out["node_id"] != nil {
			out["id"] = out["node_id"]
		}
	}
	if out["entity_type"] == nil && out["label"] != nil {
		out["entity_type"] = out["label"]
	}
	if out["label"] == nil && out["entity_type"] != nil {
		out["label"] = out["entity_type"]
	}
	return out
}

func strAny(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprint(v)
	}
}

func intAny(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int8:
		return int(x)
	case int16:
		return int(x)
	case int32:
		return int(x)
	case int64:
		return int(x)
	case uint:
		return int(x)
	case uint8:
		return int(x)
	case uint16:
		return int(x)
	case uint32:
		return int(x)
	case uint64:
		return int(x)
	case float32:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}
