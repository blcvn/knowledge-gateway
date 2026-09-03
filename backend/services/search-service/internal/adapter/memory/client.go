// Package memory implements MemoryClient — adapter to memory-service.
package memory

import (
	"context"

	"vnp-memory/services/search-service/internal/domain/search"
)

// NoopClient is a no-op MemoryClient.
type NoopClient struct{}

func (c *NoopClient) MemobaseSearch(_ context.Context, _ *search.Query) ([]*search.Item, error) {
	return nil, nil
}
func (c *NoopClient) SMSearch(_ context.Context, _ *search.Query) ([]*search.Item, error) {
	return nil, nil
}

// Client calls memory-service over HTTP.
type Client struct {
	baseURL string
}

// NewClient creates a memory-service client.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL}
}

// NewNoopClient returns a no-op MemoryClient.
func NewNoopClient() *NoopClient { return &NoopClient{} }

func (c *Client) MemobaseSearch(ctx context.Context, q *search.Query) ([]*search.Item, error) {
	if c.baseURL == "" {
		return nil, nil
	}
	// MVP: returns empty; full gRPC client in next iteration
	return nil, nil
}

func (c *Client) SMSearch(ctx context.Context, q *search.Query) ([]*search.Item, error) {
	if c.baseURL == "" {
		return nil, nil
	}
	return nil, nil
}
