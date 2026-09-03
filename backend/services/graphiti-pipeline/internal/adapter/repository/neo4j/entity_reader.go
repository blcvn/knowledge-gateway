package neo4j

import (
	"context"

	"graphiti-pipeline/internal/domain/knowledge"
)

type EntityReader struct {
	// driver neo4j.Driver
}

func NewEntityReader() *EntityReader {
	return &EntityReader{}
}

func (r *EntityReader) FindSimilarEntities(ctx context.Context, groupID string, embedding knowledge.EmbeddingVector, limit int) ([]knowledge.ExtractedEntity, error) {
	// Cypher query utilizing cosine index on name_embedding within group_id scope
	return nil, nil
}

func (r *EntityReader) GetEntityByName(ctx context.Context, groupID string, name string) (*knowledge.ExtractedEntity, error) {
	return nil, nil
}
