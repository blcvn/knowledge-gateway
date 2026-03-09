package batch

import (
	"context"
	"fmt"

	"kgs-platform/internal/data"
	"kgs-platform/internal/search"

	"github.com/google/uuid"
)

type QdrantIndexer struct {
	qdrant   *data.QdrantClient
	embedder search.EmbeddingClient
}

func NewQdrantIndexer(qdrant *data.QdrantClient, embedder search.EmbeddingClient) *QdrantIndexer {
	if embedder == nil {
		embedder = search.NewDeterministicEmbeddingClient(1536)
	}
	return &QdrantIndexer{qdrant: qdrant, embedder: embedder}
}

func (i *QdrantIndexer) IndexEntities(ctx context.Context, appID, tenantID string, entities []Entity) error {
	if i == nil || i.qdrant == nil || i.embedder == nil || len(entities) == 0 {
		return nil
	}
	_ = tenantID
	collection := buildCollectionName(appID)
	points := make([]data.VectorPoint, 0, len(entities))
	for _, entity := range entities {
		id, _ := entity.Properties["id"].(string)
		if id == "" {
			continue
		}
		pointID := toQdrantPointID(appID, tenantID, entity.Label, id)
		text := entityToText(entity)
		if text == "" {
			continue
		}
		vector, err := i.embedder.Embed(ctx, text)
		if err != nil {
			return fmt.Errorf("embed entity %s: %w", id, err)
		}
		points = append(points, data.VectorPoint{
			ID:     pointID,
			Vector: vector,
			Payload: map[string]any{
				"id":         id,
				"point_id":   pointID,
				"label":      entity.Label,
				"properties": entity.Properties,
				"app_id":     appID,
			},
		})
	}
	if len(points) == 0 {
		return nil
	}
	if err := i.qdrant.EnsureCollection(ctx, collection, len(points[0].Vector)); err != nil {
		return fmt.Errorf("ensure qdrant collection: %w", err)
	}
	if err := i.qdrant.UpsertVectors(ctx, collection, points); err != nil {
		return fmt.Errorf("upsert qdrant vectors: %w", err)
	}
	return nil
}

var _ VectorIndexer = (*QdrantIndexer)(nil)

func toQdrantPointID(appID, tenantID, label, entityID string) string {
	seed := fmt.Sprintf("%s|%s|%s|%s", appID, tenantID, label, entityID)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}
