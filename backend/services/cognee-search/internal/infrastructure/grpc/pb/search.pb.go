package pb

import "context"

type SearchRequest struct {
	Query      string
	Strategies []string
	TopK       int32
	Rerank     bool
	Filters    *SearchFilters
}

type SearchFilters struct {
	DatasetId string
}

type SearchResponse struct {
	Results []*SearchResult
}

type SearchResult struct {
	Id       string
	Content  string
	Score    float32
}

type RAGRequest struct {
	Query      string
	Strategies []string
	TopK       int32
}

type RAGResponse struct {
	Answer  string
	Sources []*SearchResult
}

type GetChunksRequest struct{}
type ChunksResponse struct{}

type CogneeSearchServiceServer interface {
	Search(context.Context, *SearchRequest) (*SearchResponse, error)
	RAGComplete(context.Context, *RAGRequest) (*RAGResponse, error)
	GetChunks(context.Context, *GetChunksRequest) (*ChunksResponse, error)
}
