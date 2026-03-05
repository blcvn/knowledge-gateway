package batch

import (
	"context"
	"fmt"

	"kgs-platform/internal/data"
)

type QdrantIndexer struct {
	qdrant *data.QdrantClient
}

func NewQdrantIndexer(qdrant *data.QdrantClient) *QdrantIndexer {
	return &QdrantIndexer{qdrant: qdrant}
}

func (i *QdrantIndexer) IndexEntities(ctx context.Context, appID, tenantID string, entities []Entity) error {
	if i == nil || i.qdrant == nil || len(entities) == 0 {
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
		text := entityToText(entity)
		if text == "" {
			continue
		}
		vector := embedDeterministic(text, 1536)
		points = append(points, data.VectorPoint{
			ID:     id,
			Vector: vector,
			Payload: map[string]any{
				"id":         id,
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
