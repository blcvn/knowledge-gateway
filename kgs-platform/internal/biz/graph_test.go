package biz

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeGraphRepo struct {
	queryCalls      int
	createNodeCalls int
	createEdgeCalls int
}

func (r *fakeGraphRepo) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	r.createNodeCalls++
	return map[string]any{"id": properties["id"], "label": label}, nil
}
func (r *fakeGraphRepo) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	return nil, nil
}
func (r *fakeGraphRepo) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	r.createEdgeCalls++
	return map[string]any{"id": properties["id"], "type": relationType}, nil
}
func (r *fakeGraphRepo) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	r.queryCalls++
	return map[string]any{"data": []map[string]any{}}, nil
}

func TestGraphUsecaseGetContextDepth5UsesBatchedTraversal(t *testing.T) {
	repo := &fakeGraphRepo{}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, nil, log.NewStdLogger(io.Discard))

	_, err := uc.GetContext(context.Background(), "app-1", "tenant-1", "node-1", 5, "OUTGOING")
	if err != nil {
		t.Fatalf("GetContext error: %v", err)
	}
	// depth=5 with window=3 should split into 2 queries.
	if repo.queryCalls != 2 {
		t.Fatalf("expected 2 batched query calls, got %d", repo.queryCalls)
	}
}

type fakeOverlayWriter struct {
	entityCalls int
	edgeCalls   int
}

func (w *fakeOverlayWriter) AddEntityDelta(ctx context.Context, overlayID, namespace, label string, properties map[string]any) (map[string]any, error) {
	w.entityCalls++
	return map[string]any{"id": properties["id"], "overlay_id": overlayID, "label": label}, nil
}

func (w *fakeOverlayWriter) AddEdgeDelta(ctx context.Context, overlayID, namespace, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error) {
	w.edgeCalls++
	return map[string]any{"id": properties["id"], "overlay_id": overlayID, "relation_type": relationType}, nil
}

func TestGraphUsecaseRoutesWriteToOverlayWhenOverlayIDProvided(t *testing.T) {
	repo := &fakeGraphRepo{}
	overlay := &fakeOverlayWriter{}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, overlay, log.NewStdLogger(io.Discard))

	nodeProps := map[string]any{"overlay_id": "ov-1", "name": "N1"}
	nodeOut, err := uc.CreateNode(context.Background(), "app-1", "tenant-1", "Requirement", nodeProps)
	if err != nil {
		t.Fatalf("CreateNode error: %v", err)
	}
	if nodeOut["overlay_id"] != "ov-1" {
		t.Fatalf("expected overlay output, got %#v", nodeOut)
	}
	if repo.createNodeCalls != 0 {
		t.Fatalf("CreateNode should not write base graph when overlay_id provided")
	}
	if overlay.entityCalls != 1 {
		t.Fatalf("expected 1 overlay entity write, got %d", overlay.entityCalls)
	}

	edgeProps := map[string]any{"overlay_id": "ov-1", "strength": 0.8}
	edgeOut, err := uc.CreateEdge(context.Background(), "app-1", "tenant-1", "DEPENDS_ON", "n1", "n2", edgeProps)
	if err != nil {
		t.Fatalf("CreateEdge error: %v", err)
	}
	if edgeOut["overlay_id"] != "ov-1" {
		t.Fatalf("expected overlay output, got %#v", edgeOut)
	}
	if repo.createEdgeCalls != 0 {
		t.Fatalf("CreateEdge should not write base graph when overlay_id provided")
	}
	if overlay.edgeCalls != 1 {
		t.Fatalf("expected 1 overlay edge write, got %d", overlay.edgeCalls)
	}
}
