package client

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
)

type StoreClientAdapter struct {
	conn *grpc.ClientConn
	cb   *gobreaker.CircuitBreaker
}

func NewStoreClientAdapter(conn *grpc.ClientConn) *StoreClientAdapter {
	cbSettings := gobreaker.Settings{
		Name:        "StoreService",
		MaxRequests: 10,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
	}

	return &StoreClientAdapter{
		conn: conn,
		cb:   gobreaker.NewCircuitBreaker(cbSettings),
	}
}

func (c *StoreClientAdapter) SaveBulk(ctx context.Context, groupID string, data map[string]interface{}) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		// Call gRPC SaveBulk
		return nil, nil
	})
	return err
}

func (c *StoreClientAdapter) RollbackBulk(ctx context.Context, groupID string, sagaID string) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		// Call gRPC RollbackBulk
		return nil, nil
	})
	return err
}
