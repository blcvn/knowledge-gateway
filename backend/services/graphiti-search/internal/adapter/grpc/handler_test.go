package grpc

import (
	"context"
	"testing"
	"time"

	pb "vnp-memory/services/graphiti-search/internal/adapter/grpc/pb"
	"google.golang.org/grpc/metadata"
	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
	"vnp-memory/services/graphiti-search/internal/usecase/reranker"
)

type mockStore struct{}

func (m *mockStore) CosineSimilaritySearch(ctx context.Context, queryVector []float32, limit int) ([]domain.SearchResult, error) {
	return []domain.SearchResult{{EntityID: "1", Score: 0.9, Content: "test"}}, nil
}
func (m *mockStore) FulltextSearch(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	return []domain.SearchResult{{EntityID: "1", Score: 0.8, Content: "test"}}, nil
}
func (m *mockStore) BFSSearch(ctx context.Context, startNodeID string, maxDepth int) ([]domain.SearchResult, error) {
	return nil, nil
}
func (m *mockStore) NodeSearch(ctx context.Context, labels []string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error) {
	return nil, nil
}
func (m *mockStore) EdgeSearch(ctx context.Context, edgeType string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error) {
	return nil, nil
}
func (m *mockStore) CommunitySearch(ctx context.Context, query string) ([]domain.SearchResult, error) {
	return nil, nil
}

type mockCache struct{}

func (m *mockCache) Get(ctx context.Context, key string) ([]domain.RankedResult, error) { return nil, nil }
func (m *mockCache) Set(ctx context.Context, key string, val []domain.RankedResult, exp time.Duration) error { return nil }
func (m *mockCache) InvalidateGroup(ctx context.Context, groupID string) error { return nil }

type mockEmbedder struct{}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func TestHandler_Search(t *testing.T) {
	s := &mockStore{}
	c := &mockCache{}
	e := &mockEmbedder{}
	r := reranker.NewRRFReranker(60)

	hybridUC := usecase.NewHybridSearchUseCase(s, e, c, []usecase.Reranker{r}, time.Second*5)
	
	handler := NewSearchServiceServer(hybridUC, nil, nil, nil)

	// Missing tenant ID
	ctx := context.Background()
	_, err := handler.Search(ctx, &pb.GraphSearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("Expected error due to missing tenant ID")
	}

	// Valid tenant ID
	md := metadata.Pairs("x-tenant-id", "tenant-1")
	ctx = metadata.NewIncomingContext(ctx, md)

	res, err := handler.Search(ctx, &pb.GraphSearchRequest{
		Query: "test",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(res.Nodes) == 0 {
		t.Errorf("Expected at least 1 node in response")
	}
}
