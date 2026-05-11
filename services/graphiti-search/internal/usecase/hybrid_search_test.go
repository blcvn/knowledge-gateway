package usecase

import (
	"context"
	"testing"
	"time"

	"vnp-memory/services/graphiti-search/internal/domain"
)

type mockStoreClient struct{}
func (m *mockStoreClient) CosineSimilaritySearch(ctx context.Context, queryVector []float32, limit int) ([]domain.SearchResult, error) {
	return []domain.SearchResult{{EntityID: "1", Score: 0.9}}, nil
}
func (m *mockStoreClient) FulltextSearch(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	return []domain.SearchResult{{EntityID: "2", Score: 0.8}}, nil
}
func (m *mockStoreClient) BFSSearch(ctx context.Context, startNodeID string, maxDepth int) ([]domain.SearchResult, error) {
	return []domain.SearchResult{{EntityID: "3", Score: 0.7}}, nil
}
func (m *mockStoreClient) NodeSearch(ctx context.Context, labels []string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error) {
	return nil, nil
}
func (m *mockStoreClient) EdgeSearch(ctx context.Context, edgeType string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error) {
	return nil, nil
}
func (m *mockStoreClient) CommunitySearch(ctx context.Context, query string) ([]domain.SearchResult, error) {
	return nil, nil
}

type mockEmbedder struct{}
func (m *mockEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

type mockCache struct{}
func (m *mockCache) Get(ctx context.Context, key string) ([]domain.RankedResult, error) {
	return nil, domain.ErrCacheUnavailable
}
func (m *mockCache) Set(ctx context.Context, key string, results []domain.RankedResult, ttl time.Duration) error {
	return nil
}
func (m *mockCache) InvalidateGroup(ctx context.Context, groupID string) error {
	return nil
}

type mockReranker struct{}
func (m *mockReranker) Rerank(ctx context.Context, query string, results []domain.SearchResult) ([]domain.RankedResult, error) {
	return []domain.RankedResult{{EntityID: "1", Score: 1.0, Rank: 1}}, nil
}
func (m *mockReranker) Type() domain.RerankerType {
	return domain.RerankerRRF
}

func TestHybridSearchUseCase(t *testing.T) {
	uc := NewHybridSearchUseCase(
		&mockStoreClient{},
		&mockEmbedder{},
		&mockCache{},
		[]Reranker{&mockReranker{}},
		time.Minute,
	)

	q := domain.SearchQuery{
		Query:     "test",
		GroupID:   "tenant1",
		Methods:   []domain.SearchMethod{domain.MethodCosine, domain.MethodBM25},
		Rerankers: []domain.RerankerType{domain.RerankerRRF},
		Limit:     10,
	}

	res, err := uc.Execute(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected results")
	}
}
