package client

import (
	"context"
	"errors"
	"testing"

	"github.com/sony/gobreaker"
	"vnp-memory/services/graphiti-search/internal/domain"
	"google.golang.org/grpc"
)

func TestStoreClient_CircuitBreaker(t *testing.T) {
	conn := &grpc.ClientConn{} // Mock conn
	client := NewStoreClientAdapter(conn).(*StoreClientAdapter)

	// Simulate failures
	mockErr := errors.New("grpc connection error")
	var err error
	for i := 0; i < 5; i++ {
		_, err = client.executeWithCBAndTrace(context.Background(), "TestSearch", func() ([]domain.SearchResult, error) {
			return nil, mockErr
		})
		if err != mockErr {
			t.Fatalf("Expected mockErr, got: %v", err)
		}
	}

	// 6th request should trip circuit breaker
	_, err = client.executeWithCBAndTrace(context.Background(), "TestSearch", func() ([]domain.SearchResult, error) {
		return nil, mockErr
	})

	if err != gobreaker.ErrOpenState {
		t.Fatalf("Expected circuit breaker to be open, got: %v", err)
	}

	// Wait for timeout (5s timeout is too long for unit test, but conceptually we test the state)
	if client.cb.State() != gobreaker.StateOpen {
		t.Fatalf("Expected state Open, got: %v", client.cb.State())
	}
}
