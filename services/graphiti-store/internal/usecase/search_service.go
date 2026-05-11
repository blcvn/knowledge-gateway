package usecase

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/usecase/port"
)

// SearchServiceImpl implements port.SearchService.
type SearchServiceImpl struct {
	driver domain.GraphDriver
}

// NewSearchService creates a SearchService backed by the given graph driver.
func NewSearchService(driver domain.GraphDriver) *SearchServiceImpl {
	return &SearchServiceImpl{driver: driver}
}

// CosineSimilaritySearch delegates vector search to the driver.
func (s *SearchServiceImpl) CosineSimilaritySearch(ctx context.Context, req port.VectorSearchRequest) ([]domain.SearchResult, error) {
	params := domain.SearchParams{GroupID: req.GroupID, Limit: req.Limit}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.driver.CosineSimilaritySearch(ctx, req.GroupID, req.Embedding, params.Limit)
}

// FulltextSearch delegates BM25 text search to the driver.
func (s *SearchServiceImpl) FulltextSearch(ctx context.Context, req port.TextSearchRequest) ([]domain.SearchResult, error) {
	params := domain.SearchParams{GroupID: req.GroupID, Limit: req.Limit}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.driver.FulltextSearch(ctx, req.GroupID, req.Query, params.Limit)
}

// BFSSearch delegates graph traversal to the driver.
func (s *SearchServiceImpl) BFSSearch(ctx context.Context, req port.BFSSearchRequest) ([]domain.SearchResult, error) {
	if req.MaxDepth <= 0 {
		req.MaxDepth = 2
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	return s.driver.BFSSearch(ctx, req.StartNodeID, req.MaxDepth, req.Limit)
}
