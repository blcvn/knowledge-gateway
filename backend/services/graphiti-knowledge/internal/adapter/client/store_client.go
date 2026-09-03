package client

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"vnp-memory/services/graphiti-knowledge/domain"
	"vnp-memory/services/graphiti-knowledge/usecase/port"
)

type storeClient struct {
	// grpcClient pb.GraphitiStoreServiceClient
	cb     *gobreaker.CircuitBreaker
	tracer trace.Tracer
}

func NewStoreClient() port.GraphReader {
	st := gobreaker.Settings{
		Name:        "store-client",
		MaxRequests: 20,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	}
	return &storeClient{
		cb:     gobreaker.NewCircuitBreaker(st),
		tracer: otel.Tracer("store-client"),
	}
}

func (c *storeClient) FindSimilarEntities(ctx context.Context, embedding domain.EmbeddingVector, threshold float64) ([]domain.ExtractedEntity, error) {
	ctx, span := c.tracer.Start(ctx, "FindSimilarEntities")
	defer span.End()

	res, err := c.cb.Execute(func() (interface{}, error) {
		var entities []domain.ExtractedEntity
		err := retry.Do(
			func() error {
				// gRPC call to store service
				entities = []domain.ExtractedEntity{} // Mocked for now until proto exists
				return nil
			},
			retry.Attempts(3),
			retry.Context(ctx),
		)
		return entities, err
	})

	if err != nil {
		return nil, err
	}
	return res.([]domain.ExtractedEntity), nil
}

func (c *storeClient) GetEntityByName(ctx context.Context, name string, groupID string) (*domain.ExtractedEntity, error) {
	ctx, span := c.tracer.Start(ctx, "GetEntityByName")
	defer span.End()

	res, err := c.cb.Execute(func() (interface{}, error) {
		var entity *domain.ExtractedEntity
		err := retry.Do(
			func() error {
				// gRPC call to store service
				entity = nil
				return nil
			},
			retry.Attempts(3),
			retry.Context(ctx),
		)
		return entity, err
	})

	if err != nil {
		return nil, err
	}
	return res.(*domain.ExtractedEntity), nil
}

func (c *storeClient) FindSimilarEdges(ctx context.Context, embedding domain.EmbeddingVector, threshold float64) ([]domain.ExtractedEdge, error) {
	ctx, span := c.tracer.Start(ctx, "FindSimilarEdges")
	defer span.End()

	res, err := c.cb.Execute(func() (interface{}, error) {
		var edges []domain.ExtractedEdge
		err := retry.Do(
			func() error {
				// gRPC call
				edges = []domain.ExtractedEdge{}
				return nil
			},
			retry.Attempts(3),
			retry.Context(ctx),
		)
		return edges, err
	})

	if err != nil {
		return nil, err
	}
	return res.([]domain.ExtractedEdge), nil
}
