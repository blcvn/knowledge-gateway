package outbox

import (
	"context"
	"fmt"
	stdlog "log"
	"regexp"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/search"
	"github.com/google/uuid"
)

type QdrantSyncer struct {
	qdrant *data.QdrantClient
	embed  search.EmbeddingClient
}

func NewQdrantSyncer(qdrant *data.QdrantClient, embed search.EmbeddingClient) *QdrantSyncer {
	if embed == nil {
		embed = search.NewDeterministicEmbeddingClient(1536)
	}
	return &QdrantSyncer{qdrant: qdrant, embed: embed}
}

func (s *QdrantSyncer) UpsertVector(ctx context.Context, entity data.KGEntity) error {
	if s == nil || s.qdrant == nil || s.embed == nil {
		return nil
	}
	started := time.Now()
	text := entityToText(entity)
	if text == "" {
		stdlog.Printf("[KGS][QdrantSyncer] UpsertVector skip empty text app_id=%s tenant_id=%s entity_id=%s",
			entity.AppID, entity.TenantID, entity.EntityID)
		return nil
	}
	stdlog.Printf("[KGS][QdrantSyncer] UpsertVector start app_id=%s tenant_id=%s entity_id=%s entity_type=%s",
		entity.AppID, entity.TenantID, entity.EntityID, entity.EntityType)
	vector, err := s.embed.Embed(ctx, text)
	if err != nil {
		stdlog.Printf("[KGS][QdrantSyncer] UpsertVector embed failed app_id=%s tenant_id=%s entity_id=%s err=%v",
			entity.AppID, entity.TenantID, entity.EntityID, err)
		return err
	}
	collection := qdrantCollection(entity.AppID)
	if err := s.qdrant.EnsureCollection(ctx, collection, len(vector)); err != nil {
		stdlog.Printf("[KGS][QdrantSyncer] UpsertVector ensure collection failed app_id=%s tenant_id=%s entity_id=%s collection=%s err=%v",
			entity.AppID, entity.TenantID, entity.EntityID, collection, err)
		return err
	}
	pointID := qdrantPointID(entity.AppID, entity.TenantID, entity.EntityID)
	err = s.qdrant.UpsertVectors(ctx, collection, []data.VectorPoint{{
		ID:     pointID,
		Vector: vector,
		Payload: map[string]any{
			"id":         entity.EntityID,
			"point_id":   pointID,
			"label":      entity.EntityType,
			"properties": map[string]any(entity.Properties),
			"app_id":     entity.AppID,
			"tenant_id":  entity.TenantID,
			"version":    entity.Version,
		},
	}})
	if err != nil {
		stdlog.Printf("[KGS][QdrantSyncer] UpsertVector failed app_id=%s tenant_id=%s entity_id=%s collection=%s point_id=%s err=%v",
			entity.AppID, entity.TenantID, entity.EntityID, collection, pointID, err)
		return err
	}
	stdlog.Printf("[KGS][QdrantSyncer] UpsertVector done app_id=%s tenant_id=%s entity_id=%s collection=%s point_id=%s duration=%s",
		entity.AppID, entity.TenantID, entity.EntityID, collection, pointID, time.Since(started))
	return nil
}

func (s *QdrantSyncer) DeleteVector(ctx context.Context, entityID, tenantID, appID string) error {
	if s == nil || s.qdrant == nil {
		return nil
	}
	started := time.Now()
	collection := qdrantCollection(appID)
	pointID := qdrantPointID(appID, tenantID, entityID)
	stdlog.Printf("[KGS][QdrantSyncer] DeleteVector start app_id=%s tenant_id=%s entity_id=%s collection=%s point_id=%s",
		appID, tenantID, entityID, collection, pointID)
	err := s.qdrant.DeleteVectors(ctx, collection, []string{pointID})
	if err != nil {
		stdlog.Printf("[KGS][QdrantSyncer] DeleteVector failed app_id=%s tenant_id=%s entity_id=%s collection=%s point_id=%s err=%v",
			appID, tenantID, entityID, collection, pointID, err)
		return err
	}
	stdlog.Printf("[KGS][QdrantSyncer] DeleteVector done app_id=%s tenant_id=%s entity_id=%s collection=%s point_id=%s duration=%s",
		appID, tenantID, entityID, collection, pointID, time.Since(started))
	return nil
}

var qdrantSanitizePattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func qdrantCollection(appID string) string {
	app := strings.ToLower(strings.TrimSpace(appID))
	if app == "" {
		app = "default"
	}
	app = qdrantSanitizePattern.ReplaceAllString(app, "_")
	app = strings.Trim(app, "_")
	if app == "" {
		app = "default"
	}
	return "kgs-vectors-" + app
}

func qdrantPointID(appID, tenantID, entityID string) string {
	seed := fmt.Sprintf("%s|%s|%s", strings.TrimSpace(appID), strings.TrimSpace(tenantID), strings.TrimSpace(entityID))
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}

func entityToText(entity data.KGEntity) string {
	parts := []string{entity.EntityType, entity.Name}
	props := map[string]any(entity.Properties)
	for _, key := range []string{"name", "title", "content", "description"} {
		value, ok := props[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			parts = append(parts, value)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
