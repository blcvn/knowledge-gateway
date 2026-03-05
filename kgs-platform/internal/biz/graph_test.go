package biz

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeGraphRepo struct {
	queryCalls int
}

func (r *fakeGraphRepo) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	return nil, nil
}
func (r *fakeGraphRepo) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	return nil, nil
}
func (r *fakeGraphRepo) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	return nil, nil
}
func (r *fakeGraphRepo) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	r.queryCalls++
	return map[string]any{"data": []map[string]any{}}, nil
}

func TestGraphUsecaseGetContextDepth5UsesBatchedTraversal(t *testing.T) {
	repo := &fakeGraphRepo{}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, log.NewStdLogger(io.Discard))

	_, err := uc.GetContext(context.Background(), "app-1", "tenant-1", "node-1", 5, "OUTGOING")
	if err != nil {
		t.Fatalf("GetContext error: %v", err)
	}
	// depth=5 with window=3 should split into 2 queries.
	if repo.queryCalls != 2 {
		t.Fatalf("expected 2 batched query calls, got %d", repo.queryCalls)
	}
}
