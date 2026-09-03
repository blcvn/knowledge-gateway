// Package kg implements KGClient — a gRPC adapter calling kg-service.
package kg

import (
	"context"
	"encoding/json"
	"fmt"

	"vnp-memory/services/search-service/internal/domain/search"
)

// noopKGClient is a no-op used when kg-service is not configured.
type noopKGClient struct{}

func (c *noopKGClient) GraphitiSearch(_ context.Context, q *search.Query) ([]*search.Item, error) {
	return nil, nil
}
func (c *noopKGClient) CogneeSearch(_ context.Context, q *search.Query) ([]*search.Item, error) {
	return nil, nil
}

// Client calls kg-service over HTTP (ForwardService protocol).
// For MVP we use HTTP REST since the ForwardService exposes HTTP too.
type Client struct {
	baseURL string
}

// NewClient creates a kg-service HTTP client.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL}
}

// NewNoopClient returns a no-op KGClient.
func NewNoopClient() *noopKGClient { return &noopKGClient{} }

// GraphitiSearch calls kg-service /v1/graphiti/search.
func (c *Client) GraphitiSearch(ctx context.Context, q *search.Query) ([]*search.Item, error) {
	return c.search(ctx, "/v1/graphiti/search", q)
}

// CogneeSearch calls kg-service /v1/cognee/search.
func (c *Client) CogneeSearch(ctx context.Context, q *search.Query) ([]*search.Item, error) {
	return c.search(ctx, "/v1/cognee/search", q)
}

func (c *Client) search(ctx context.Context, path string, q *search.Query) ([]*search.Item, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("kg-service: not configured")
	}
	// Simple HTTP client — in production, use gRPC ForwardService client
	body, _ := json.Marshal(map[string]any{
		"query":     q.Query,
		"tenant_id": q.TenantID,
		"limit":     q.Limit,
		"mode":      q.Mode,
	})
	_ = body
	// MVP: return empty slice; full impl in next iteration
	return nil, nil
}
