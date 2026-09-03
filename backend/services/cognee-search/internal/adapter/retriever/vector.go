package retriever

import (
	"context"
	"fmt"

	"vnp-memory/services/cognee-search/internal/usecase"
)

// VectorRetriever implements SIMILARITY search using Qdrant with NodeSet payload filtering.
type VectorRetriever struct {
	// qdrantClient qdrant.PointsClient  — injected in production
	// embedder     Embedder             — injected in production
}

// NewVectorRetriever creates a VectorRetriever.
func NewVectorRetriever() *VectorRetriever { return &VectorRetriever{} }

// Strategy returns the search strategy this retriever handles.
func (r *VectorRetriever) Strategy() usecase.SearchStrategy { return usecase.StrategySimilarity }

// Retrieve performs a vector similarity search with optional NodeSet payload filtering.
//
// Qdrant filter construction:
//   - dataset_id filter: exact match on payload.dataset_id
//   - node_sets filter:  payload.node_sets contains at least one of the requested tags
//     (Qdrant MatchAny — any point with matching NodeSet is returned)
func (r *VectorRetriever) Retrieve(ctx context.Context, req usecase.SearchRequest) ([]usecase.SearchResult, error) {
	// Build filter description for logging / stub purposes
	filterDesc := buildVectorFilterDesc(req)
	_ = filterDesc

	// Production implementation:
	// 1. vec, err := r.embedder.Embed(ctx, req.Query)
	// 2. filter := buildQdrantFilter(req)
	// 3. points, err := r.qdrantClient.Search(ctx, &qdrant.SearchPoints{
	//        CollectionName: fmt.Sprintf("cognee_%s", req.TenantID),
	//        Vector:         vec,
	//        Filter:         filter,        // [NEW] NodeSet filter applied here
	//        Limit:          uint64(req.TopK),
	//    })
	// 4. return mapPointsToResults(points), nil

	return []usecase.SearchResult{}, nil
}

// buildVectorFilterDesc describes the Qdrant filter for debugging.
func buildVectorFilterDesc(req usecase.SearchRequest) string {
	if len(req.NodeSets) == 0 {
		return fmt.Sprintf("dataset_id=%v", req.DatasetID)
	}
	return fmt.Sprintf("dataset_id=%v AND node_sets in %v", req.DatasetID, req.NodeSets)
}
