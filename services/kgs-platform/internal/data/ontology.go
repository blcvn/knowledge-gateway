package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

type ontologyRepo struct {
	data *Data
	log  *log.Helper
}

// NewOntologyRepo .
func NewOntologyRepo(data *Data, logger log.Logger) *ontologyRepo {
	return &ontologyRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

const (
	entityCachePrefix   = "ontology:entity:"
	relationCachePrefix = "ontology:relation:"
	cacheTTL            = 5 * time.Minute
)

func (r *ontologyRepo) GetEntityType(ctx context.Context, appID, name string) (*biz.EntityType, error) {
	cacheKey := fmt.Sprintf("%s%s:%s", entityCachePrefix, appID, name)

	val, err := r.data.rc.Get(ctx, cacheKey).Result()
	if err == nil {
		var entityType biz.EntityType
		if json.Unmarshal([]byte(val), &entityType) == nil {
			return &entityType, nil
		}
	} else if err != redis.Nil {
		r.log.Errorf("Redis Get error: %v", err)
	}

	var entity biz.EntityType
	if err := r.data.db.WithContext(ctx).Where("app_id = ? AND name = ?", appID, name).First(&entity).Error; err != nil {
		return nil, err
	}

	if data, err := json.Marshal(entity); err == nil {
		r.data.rc.Set(ctx, cacheKey, data, cacheTTL)
	}

	return &entity, nil
}

func (r *ontologyRepo) GetRelationType(ctx context.Context, appID, name string) (*biz.RelationType, error) {
	cacheKey := fmt.Sprintf("%s%s:%s", relationCachePrefix, appID, name)

	val, err := r.data.rc.Get(ctx, cacheKey).Result()
	if err == nil {
		var relationType biz.RelationType
		if json.Unmarshal([]byte(val), &relationType) == nil {
			return &relationType, nil
		}
	} else if err != redis.Nil {
		r.log.Errorf("Redis Get error: %v", err)
	}

	var relation biz.RelationType
	if err := r.data.db.WithContext(ctx).Where("app_id = ? AND name = ?", appID, name).First(&relation).Error; err != nil {
		return nil, err
	}

	if data, err := json.Marshal(relation); err == nil {
		r.data.rc.Set(ctx, cacheKey, data, cacheTTL)
	}

	return &relation, nil
}
