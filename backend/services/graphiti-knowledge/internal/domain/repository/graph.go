package repository

import (
\t"context"

\tpb "vnp-memory/services/graphiti-knowledge/adapter/grpc/pb"
)

// GraphRepository defines the contract for persisting and querying Knowledge Graph data.
// In a strict Clean Architecture, methods should accept Domain Models instead of 'pb' structs,
// but we use 'pb' here temporarily for rapid prototyping and mapping.
type GraphRepository interface {
\t// SearchSimilarEntities finds existing nodes that match the name for Deduplication.
\tSearchSimilarEntities(ctx context.Context, groupID string, entityName string, topK int) ([]*pb.ExtractedEntity, error)

\t// UpsertEntity creates a new entity node or updates an existing one (based on groupID + Name).
\tUpsertEntity(ctx context.Context, groupID string, entity *pb.ExtractedEntity) error

\t// UpsertEdge creates a directed relationship between two existing entities.
\tUpsertEdge(ctx context.Context, groupID string, edge *pb.ExtractedEdge) error
}
