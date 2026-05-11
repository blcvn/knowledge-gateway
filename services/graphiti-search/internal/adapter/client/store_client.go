package client

import (
	"context"

	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
)

type StoreClientAdapter struct {
	conn   *grpc.ClientConn
	cb     *gobreaker.CircuitBreaker
	tracer trace.Tracer
}

func NewStoreClientAdapter(conn *grpc.ClientConn) usecase.StoreSearchClient {
	st := gobreaker.Settings{
		Name:        "store-grpc-client",
		MaxRequests: 1,
		Interval:    0,
		Timeout:     5 * 1000 * 1000 * 1000, // 5s
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	}

	return &StoreClientAdapter{
		conn:   conn,
		cb:     gobreaker.NewCircuitBreaker(st),
		tracer: otel.Tracer("graphiti-search/store-client"),
	}
}

func (c *StoreClientAdapter) executeWithCBAndTrace(ctx context.Context, operationName string, fallbackFn func() ([]domain.SearchResult, error)) ([]domain.SearchResult, error) {
	ctx, span := c.tracer.Start(ctx, operationName)
	defer span.End()

	res, err := c.cb.Execute(func() (interface{}, error) {
		return fallbackFn()
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return res.([]domain.SearchResult), nil
}

func (c *StoreClientAdapter) CosineSimilaritySearch(ctx context.Context, queryVector []float32, limit int) ([]domain.SearchResult, error) {
	return c.executeWithCBAndTrace(ctx, "CosineSimilaritySearch", func() ([]domain.SearchResult, error) {
		return []domain.SearchResult{}, nil
	})
}

func (c *StoreClientAdapter) FulltextSearch(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	return c.executeWithCBAndTrace(ctx, "FulltextSearch", func() ([]domain.SearchResult, error) {
		return []domain.SearchResult{}, nil
	})
}

func (c *StoreClientAdapter) BFSSearch(ctx context.Context, startNodeID string, maxDepth int) ([]domain.SearchResult, error) {
	return c.executeWithCBAndTrace(ctx, "BFSSearch", func() ([]domain.SearchResult, error) {
		return []domain.SearchResult{}, nil
	})
}

func (c *StoreClientAdapter) NodeSearch(ctx context.Context, labels []string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error) {
	return c.executeWithCBAndTrace(ctx, "NodeSearch", func() ([]domain.SearchResult, error) {
		return []domain.SearchResult{}, nil
	})
}

func (c *StoreClientAdapter) EdgeSearch(ctx context.Context, edgeType string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error) {
	return c.executeWithCBAndTrace(ctx, "EdgeSearch", func() ([]domain.SearchResult, error) {
		return []domain.SearchResult{}, nil
	})
}

func (c *StoreClientAdapter) CommunitySearch(ctx context.Context, query string) ([]domain.SearchResult, error) {
	return c.executeWithCBAndTrace(ctx, "CommunitySearch", func() ([]domain.SearchResult, error) {
		return []domain.SearchResult{}, nil
	})
}
