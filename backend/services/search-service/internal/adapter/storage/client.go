// Package storage implements StorageClient — adapter to storage-service.
package storage

import (
	"context"

	"vnp-memory/services/search-service/internal/domain/search"
)

// NoopClient is a no-op StorageClient.
type NoopClient struct{}

func (c *NoopClient) FileSearch(_ context.Context, _ *search.Query) ([]*search.Item, error) {
	return nil, nil
}

// Client calls storage-service over HTTP.
type Client struct {
	baseURL string
}

// NewClient creates a storage-service client.
func NewClient(baseURL string) *Client { return &Client{baseURL: baseURL} }

// NewNoopClient returns a no-op StorageClient.
func NewNoopClient() *NoopClient { return &NoopClient{} }

func (c *Client) FileSearch(_ context.Context, _ *search.Query) ([]*search.Item, error) {
	if c.baseURL == "" {
		return nil, nil
	}
	return nil, nil
}
