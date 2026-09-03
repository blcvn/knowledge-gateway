package client

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
	"google.golang.org/grpc"
)

type KnowledgeClientAdapter struct {
	conn *grpc.ClientConn
	cb   *gobreaker.CircuitBreaker
}

func NewKnowledgeClientAdapter(conn *grpc.ClientConn) *KnowledgeClientAdapter {
	cbSettings := gobreaker.Settings{
		Name:        "KnowledgeService",
		MaxRequests: 10,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
	}

	return &KnowledgeClientAdapter{
		conn: conn,
		cb:   gobreaker.NewCircuitBreaker(cbSettings),
	}
}

func (c *KnowledgeClientAdapter) ExtractEntities(ctx context.Context, episode domain.Episode) ([]map[string]interface{}, error) {
	res, err := c.cb.Execute(func() (interface{}, error) {
		// Mock gRPC call
		// client := pb.NewKnowledgeServiceClient(c.conn)
		// return client.ExtractEntities(ctx, &req)
		return []map[string]interface{}{}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("ExtractEntities failed: %w", err)
	}
	return res.([]map[string]interface{}), nil
}

func (c *KnowledgeClientAdapter) ResolveEntities(ctx context.Context, groupID string, entities []map[string]interface{}) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		return nil, nil
	})
	return err
}

func (c *KnowledgeClientAdapter) ExtractEdges(ctx context.Context, episode domain.Episode, entities []map[string]interface{}) ([]map[string]interface{}, error) {
	res, err := c.cb.Execute(func() (interface{}, error) {
		return []map[string]interface{}{}, nil
	})
	if err != nil {
		return nil, err
	}
	return res.([]map[string]interface{}), nil
}

func (c *KnowledgeClientAdapter) ResolveEdges(ctx context.Context, groupID string, edges []map[string]interface{}) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		return nil, nil
	})
	return err
}

func (c *KnowledgeClientAdapter) UpdateCommunity(ctx context.Context, groupID string) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		return nil, nil
	})
	return err
}
