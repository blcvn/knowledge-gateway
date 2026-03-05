package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	pb "kgs-platform/api/graph/v1"
	"kgs-platform/internal/batch"
	"kgs-platform/internal/biz"
	"kgs-platform/internal/search"
	"kgs-platform/internal/server/middleware"
)

type mockGraphUsecase struct {
	createNodeFn func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error)
	getNodeFn    func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error)
}

func (m *mockGraphUsecase) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	return m.createNodeFn(ctx, appID, tenantID, label, properties)
}

func (m *mockGraphUsecase) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	return m.getNodeFn(ctx, appID, tenantID, nodeID)
}

func (m *mockGraphUsecase) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

func (m *mockGraphUsecase) GetContext(ctx context.Context, appID, tenantID string, nodeID string, depth int, direction string) (map[string]any, error) {
	return map[string]any{"data": []any{}}, nil
}

func (m *mockGraphUsecase) GetImpact(ctx context.Context, appID, tenantID string, nodeID string, maxDepth int) (map[string]any, error) {
	return map[string]any{"data": []any{}}, nil
}

func (m *mockGraphUsecase) GetCoverage(ctx context.Context, appID, tenantID string, nodeID string, maxDepth int) (map[string]any, error) {
	return map[string]any{"data": []any{}}, nil
}

func (m *mockGraphUsecase) GetSubgraph(ctx context.Context, appID, tenantID string, nodeIDs []string) (map[string]any, error) {
	return map[string]any{"data": []any{}}, nil
}

func TestGraphServiceCreateNode(t *testing.T) {
	var gotAppID, gotTenantID, gotLabel string
	var gotProps map[string]any

	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			gotAppID = appID
			gotTenantID = tenantID
			gotLabel = label
			gotProps = properties
			return map[string]any{"id": "node-1", "name": "alice"}, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, nil, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read,write",
	})

	resp, err := svc.CreateNode(ctx, &pb.CreateNodeRequest{
		Label:          "User",
		PropertiesJson: `{"username":"alice123","age":25}`,
	})
	if err != nil {
		t.Fatalf("CreateNode error: %v", err)
	}
	if resp.NodeId != "node-1" {
		t.Fatalf("unexpected node id: %s", resp.NodeId)
	}
	if resp.Label != "User" {
		t.Fatalf("unexpected label: %s", resp.Label)
	}
	if gotAppID != "app-1" || gotTenantID != "tenant-1" || gotLabel != "User" {
		t.Fatalf("unexpected usecase call args: app=%s tenant=%s label=%s", gotAppID, gotTenantID, gotLabel)
	}
	if gotProps["username"] != "alice123" {
		t.Fatalf("properties not parsed correctly: %#v", gotProps)
	}
}

func TestGraphServiceGetNode(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			if appID != "app-1" || tenantID != "tenant-1" || nodeID != "node-1" {
				t.Fatalf("unexpected args app=%s tenant=%s node=%s", appID, tenantID, nodeID)
			}
			return map[string]any{
				"id":    "node-1",
				"label": "User",
				"name":  "alice",
			}, nil
		},
	}, nil, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read",
	})

	resp, err := svc.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	if err != nil {
		t.Fatalf("GetNode error: %v", err)
	}
	if resp.NodeId != "node-1" || resp.Label != "User" {
		t.Fatalf("unexpected response: %#v", resp)
	}

	var props map[string]any
	if err := json.Unmarshal([]byte(resp.PropertiesJson), &props); err != nil {
		t.Fatalf("invalid properties json: %v", err)
	}
	if props["name"] != "alice" {
		t.Fatalf("unexpected properties: %#v", props)
	}
}

type fakeBatchWriter struct{}

func (w *fakeBatchWriter) BulkCreate(ctx context.Context, appID, tenantID string, entities []batch.Entity) (int, error) {
	return len(entities), nil
}

type fakeBatchDeduper struct{}

func (d *fakeBatchDeduper) Dedup(ctx context.Context, appID, tenantID string, entities []batch.Entity) ([]batch.Entity, int, error) {
	return entities, 0, nil
}

type fakeSearchEngine struct {
	results []search.Result
	err     error
}

func (f *fakeSearchEngine) HybridSearch(ctx context.Context, namespace, query string, opts search.Options) ([]search.Result, error) {
	return f.results, f.err
}

func TestGraphServiceBatchUpsertEntities(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, batch.NewUsecase(&fakeBatchWriter{}, &fakeBatchDeduper{}), nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "write",
	})

	resp, err := svc.BatchUpsertEntities(ctx, &pb.BatchUpsertRequest{
		Entities: []*pb.BatchEntity{
			{Label: "User", PropertiesJson: `{"name":"alice"}`},
			{Label: "User", PropertiesJson: `{"name":"bob"}`},
		},
	})
	if err != nil {
		t.Fatalf("BatchUpsertEntities error: %v", err)
	}
	if resp.Created != 2 || resp.Skipped != 0 {
		t.Fatalf("unexpected batch result: %#v", resp)
	}
}

func TestGraphServiceBatchUpsertEntities100(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, batch.NewUsecase(&fakeBatchWriter{}, &fakeBatchDeduper{}), nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "write",
	})

	entities := make([]*pb.BatchEntity, 0, 100)
	for i := 0; i < 100; i++ {
		entities = append(entities, &pb.BatchEntity{
			Label:          "User",
			PropertiesJson: fmt.Sprintf(`{"idx":%d}`, i),
		})
	}

	resp, err := svc.BatchUpsertEntities(ctx, &pb.BatchUpsertRequest{Entities: entities})
	if err != nil {
		t.Fatalf("BatchUpsertEntities(100) error: %v", err)
	}
	if resp.Created != 100 {
		t.Fatalf("expected created=100, got %d", resp.Created)
	}
}

func TestApplyPagination(t *testing.T) {
	reply := &pb.GraphReply{
		Nodes: []*pb.GraphNode{
			{Id: "n1"}, {Id: "n2"}, {Id: "n3"},
		},
		Edges: []*pb.GraphEdge{
			{Id: "e1", Source: "n1", Target: "n2"},
			{Id: "e2", Source: "n2", Target: "n3"},
		},
	}

	token := biz.EncodePageToken(1)
	got, err := applyPagination(context.Background(), reply, 1, token)
	if err != nil {
		t.Fatalf("applyPagination error: %v", err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].Id != "n2" {
		t.Fatalf("unexpected paged nodes: %#v", got.Nodes)
	}
}

func TestGraphServiceHybridSearch(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, nil, &fakeSearchEngine{
		results: []search.Result{
			{
				ID:            "n1",
				Label:         "Requirement",
				Properties:    map[string]any{"name": "Payment processing"},
				Score:         0.95,
				SemanticScore: 0.91,
				TextScore:     0.89,
			},
		},
	})

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read",
	})

	resp, err := svc.HybridSearch(ctx, &pb.HybridSearchRequest{
		Query: "payment",
		TopK:  5,
	})
	if err != nil {
		t.Fatalf("HybridSearch error: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].NodeId != "n1" || resp.Results[0].Score <= 0 {
		t.Fatalf("unexpected result: %#v", resp.Results[0])
	}
}
