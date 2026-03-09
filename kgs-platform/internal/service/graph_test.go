package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/graph/v1"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/analytics"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/batch"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/overlay"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/projection"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/search"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/server/middleware"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"
)

type mockGraphUsecase struct {
	createNodeFn func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error)
	getNodeFn    func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error)
	getFullFn    func(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error)
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

func (m *mockGraphUsecase) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error) {
	if m.getFullFn == nil {
		return &biz.FullGraphResult{}, nil
	}
	return m.getFullFn(ctx, appID, tenantID, limit, offset)
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
	}, nil, nil, nil, nil, nil, nil)

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
	}, nil, nil, nil, nil, nil, nil)

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

func TestGraphServiceGetFullGraph(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
		getFullFn: func(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error) {
			if appID != "app-1" || tenantID != "tenant-1" {
				t.Fatalf("unexpected scope app=%s tenant=%s", appID, tenantID)
			}
			if limit != 200 || offset != 10 {
				t.Fatalf("unexpected pagination limit=%d offset=%d", limit, offset)
			}
			return &biz.FullGraphResult{
				Nodes: []biz.NodeResult{
					{
						ID:     "BR-001",
						Labels: []string{"Entity", "BUSINESS_RULE"},
						Properties: map[string]any{
							"id":         "BR-001",
							"given":      "logged in",
							"confidence": 0.97,
						},
					},
				},
				Edges: []biz.EdgeResult{
					{
						ID:           "E-001",
						RelationType: "RELATES_TO",
						SourceNodeID: "BR-001",
						TargetNodeID: "UC-001",
						Properties: map[string]any{
							"id": "E-001",
						},
					},
				},
				TotalNodes: 1,
				TotalEdges: 1,
			}, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read",
	})

	resp, err := svc.GetFullGraph(ctx, &pb.GetFullGraphRequest{
		NodeLimit:  200,
		NodeOffset: 10,
	})
	if err != nil {
		t.Fatalf("GetFullGraph error: %v", err)
	}
	if resp.TotalNodes != 1 || resp.TotalEdges != 1 {
		t.Fatalf("unexpected totals: nodes=%d edges=%d", resp.TotalNodes, resp.TotalEdges)
	}
	if len(resp.Nodes) != 1 || len(resp.Edges) != 1 {
		t.Fatalf("unexpected graph size: nodes=%d edges=%d", len(resp.Nodes), len(resp.Edges))
	}
	if resp.Nodes[0].Label != "BUSINESS_RULE" {
		t.Fatalf("expected primary label BUSINESS_RULE, got %q", resp.Nodes[0].Label)
	}
	if resp.Nodes[0].Properties["confidence"] != "0.97" {
		t.Fatalf("unexpected stringified property: %#v", resp.Nodes[0].Properties)
	}
	if resp.Edges[0].RelationType != "RELATES_TO" || resp.Edges[0].SourceNodeId != "BR-001" || resp.Edges[0].TargetNodeId != "UC-001" {
		t.Fatalf("unexpected edge response: %#v", resp.Edges[0])
	}
}

func TestGraphServiceGetFullGraphRejectsScopeMismatch(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read",
	})

	_, err := svc.GetFullGraph(ctx, &pb.GetFullGraphRequest{
		AppId:    "other-app",
		TenantId: "tenant-1",
	})
	if err == nil {
		t.Fatalf("expected scope mismatch error")
	}
}

func TestGraphServiceGetFullGraphEmptyGraph(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
		getFullFn: func(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error) {
			return &biz.FullGraphResult{
				Nodes:      []biz.NodeResult{},
				Edges:      []biz.EdgeResult{},
				TotalNodes: 0,
				TotalEdges: 0,
			}, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-empty",
		TenantID: "tenant-empty",
		Scopes:   "read",
	})

	resp, err := svc.GetFullGraph(ctx, &pb.GetFullGraphRequest{})
	if err != nil {
		t.Fatalf("GetFullGraph error: %v", err)
	}
	if resp.TotalNodes != 0 || resp.TotalEdges != 0 || len(resp.Nodes) != 0 || len(resp.Edges) != 0 {
		t.Fatalf("expected empty response, got %#v", resp)
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

type fakeAnalyticsEngine struct {
	coverageFn     func(ctx context.Context, namespace, domain string) (*analytics.CoverageReport, error)
	traceabilityFn func(ctx context.Context, namespace string, sourceTypes, targetTypes []string, maxHops int) (*analytics.TraceabilityMatrix, error)
}

func (f *fakeAnalyticsEngine) CoverageReport(ctx context.Context, namespace, domain string) (*analytics.CoverageReport, error) {
	return f.coverageFn(ctx, namespace, domain)
}

func (f *fakeAnalyticsEngine) TraceabilityMatrix(ctx context.Context, namespace string, sourceTypes, targetTypes []string, maxHops int) (*analytics.TraceabilityMatrix, error) {
	return f.traceabilityFn(ctx, namespace, sourceTypes, targetTypes, maxHops)
}

func (f *fakeAnalyticsEngine) ClusterAnalysis(ctx context.Context, namespace, entityType string) (*analytics.ClusterReport, error) {
	return &analytics.ClusterReport{}, nil
}

type fakeProjectionEngine struct {
	applyFn  func(ctx context.Context, namespace, role string, rawData map[string]any) (map[string]any, error)
	createFn func(ctx context.Context, namespace string, view projection.ViewDefinition) (*projection.ViewDefinition, error)
	getFn    func(ctx context.Context, namespace, viewID string) (*projection.ViewDefinition, error)
	listFn   func(ctx context.Context, namespace string) ([]projection.ViewDefinition, error)
	deleteFn func(ctx context.Context, namespace, viewID string) error
}

func (f *fakeProjectionEngine) Apply(ctx context.Context, namespace, role string, rawData map[string]any) (map[string]any, error) {
	if f.applyFn == nil {
		return rawData, nil
	}
	return f.applyFn(ctx, namespace, role, rawData)
}

func (f *fakeProjectionEngine) CreateViewDefinition(ctx context.Context, namespace string, view projection.ViewDefinition) (*projection.ViewDefinition, error) {
	if f.createFn == nil {
		return &view, nil
	}
	return f.createFn(ctx, namespace, view)
}

func (f *fakeProjectionEngine) GetViewDefinition(ctx context.Context, namespace, viewID string) (*projection.ViewDefinition, error) {
	if f.getFn == nil {
		return &projection.ViewDefinition{ID: viewID}, nil
	}
	return f.getFn(ctx, namespace, viewID)
}

func (f *fakeProjectionEngine) ListViewDefinitions(ctx context.Context, namespace string) ([]projection.ViewDefinition, error) {
	if f.listFn == nil {
		return []projection.ViewDefinition{}, nil
	}
	return f.listFn(ctx, namespace)
}

func (f *fakeProjectionEngine) DeleteViewDefinition(ctx context.Context, namespace, viewID string) error {
	if f.deleteFn == nil {
		return nil
	}
	return f.deleteFn(ctx, namespace, viewID)
}

type fakeOverlayManager struct {
	createFn  func(ctx context.Context, namespace, sessionID, baseVersionID string) (*overlay.OverlayGraph, error)
	commitFn  func(ctx context.Context, overlayID, conflictPolicy string) (*overlay.CommitResult, error)
	discardFn func(ctx context.Context, overlayID string) error
}

func (f *fakeOverlayManager) Create(ctx context.Context, namespace, sessionID, baseVersionID string) (*overlay.OverlayGraph, error) {
	return f.createFn(ctx, namespace, sessionID, baseVersionID)
}

func (f *fakeOverlayManager) Get(ctx context.Context, overlayID string) (*overlay.OverlayGraph, error) {
	return nil, nil
}

func (f *fakeOverlayManager) Commit(ctx context.Context, overlayID, conflictPolicy string) (*overlay.CommitResult, error) {
	return f.commitFn(ctx, overlayID, conflictPolicy)
}

func (f *fakeOverlayManager) Discard(ctx context.Context, overlayID string) error {
	return f.discardFn(ctx, overlayID)
}

func (f *fakeOverlayManager) DiscardBySession(ctx context.Context, sessionID string) error {
	return nil
}

type fakeVersionManager struct {
	listFn     func(ctx context.Context, namespace string) ([]version.GraphVersion, error)
	diffFn     func(ctx context.Context, namespace, fromVersionID, toVersionID string) (*version.DiffResult, error)
	rollbackFn func(ctx context.Context, namespace, targetVersionID, reason string) (string, error)
}

func (f *fakeVersionManager) CreateDelta(ctx context.Context, namespace string, changes version.ChangeSet) (string, error) {
	return "", nil
}

func (f *fakeVersionManager) GetVersion(ctx context.Context, namespace, versionID string) (*version.GraphVersion, error) {
	return nil, nil
}

func (f *fakeVersionManager) ListVersions(ctx context.Context, namespace string) ([]version.GraphVersion, error) {
	return f.listFn(ctx, namespace)
}

func (f *fakeVersionManager) DiffVersions(ctx context.Context, namespace, fromVersionID, toVersionID string) (*version.DiffResult, error) {
	return f.diffFn(ctx, namespace, fromVersionID, toVersionID)
}

func (f *fakeVersionManager) Rollback(ctx context.Context, namespace, targetVersionID, reason string) (string, error) {
	return f.rollbackFn(ctx, namespace, targetVersionID, reason)
}

func TestGraphServiceBatchUpsertEntities(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, batch.NewUsecase(&fakeBatchWriter{}, &fakeBatchDeduper{}), nil, nil, nil, nil, nil)

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
	}, batch.NewUsecase(&fakeBatchWriter{}, &fakeBatchDeduper{}), nil, nil, nil, nil, nil)

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
	}, nil, nil, nil, nil)

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

func TestGraphServiceOverlayAndVersionRPCs(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, nil, nil, &fakeOverlayManager{
		createFn: func(ctx context.Context, namespace, sessionID, baseVersionID string) (*overlay.OverlayGraph, error) {
			return &overlay.OverlayGraph{
				OverlayID:     "ov-1",
				Status:        overlay.StatusActive,
				BaseVersionID: "v1",
				CreatedAt:     fixedTime(0),
				ExpiresAt:     fixedTime(3600),
			}, nil
		},
		commitFn: func(ctx context.Context, overlayID, conflictPolicy string) (*overlay.CommitResult, error) {
			return &overlay.CommitResult{
				NewVersionID:      "v2",
				EntitiesCommitted: 2,
				EdgesCommitted:    1,
				ConflictsResolved: 1,
			}, nil
		},
		discardFn: func(ctx context.Context, overlayID string) error {
			return nil
		},
	}, &fakeVersionManager{
		listFn: func(ctx context.Context, namespace string) ([]version.GraphVersion, error) {
			return []version.GraphVersion{
				{ID: "v2", ParentID: "v1", CommitMessage: "commit", CreatedAt: fixedTime(10)},
			}, nil
		},
		diffFn: func(ctx context.Context, namespace, fromVersionID, toVersionID string) (*version.DiffResult, error) {
			return &version.DiffResult{
				FromVersionID: fromVersionID,
				ToVersionID:   toVersionID,
				EntitiesAdded: 5,
			}, nil
		},
		rollbackFn: func(ctx context.Context, namespace, targetVersionID, reason string) (string, error) {
			return "v3", nil
		},
	}, nil, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read,write",
	})

	createResp, err := svc.CreateOverlay(ctx, &pb.CreateOverlayRequest{SessionId: "s1", BaseVersion: "current"})
	if err != nil || createResp.OverlayId != "ov-1" {
		t.Fatalf("CreateOverlay failed resp=%#v err=%v", createResp, err)
	}

	commitResp, err := svc.CommitOverlay(ctx, &pb.CommitOverlayRequest{OverlayId: "ov-1", ConflictPolicy: "KEEP_OVERLAY"})
	if err != nil || commitResp.NewVersionId != "v2" {
		t.Fatalf("CommitOverlay failed resp=%#v err=%v", commitResp, err)
	}

	if _, err := svc.DiscardOverlay(ctx, &pb.DiscardOverlayRequest{OverlayId: "ov-1"}); err != nil {
		t.Fatalf("DiscardOverlay failed: %v", err)
	}

	versionsResp, err := svc.ListVersions(ctx, &pb.ListVersionsRequest{})
	if err != nil || len(versionsResp.Versions) != 1 {
		t.Fatalf("ListVersions failed resp=%#v err=%v", versionsResp, err)
	}

	diffResp, err := svc.DiffVersions(ctx, &pb.DiffVersionsRequest{FromVersionId: "v1", ToVersionId: "v2"})
	if err != nil || diffResp.EntitiesAdded != 5 {
		t.Fatalf("DiffVersions failed resp=%#v err=%v", diffResp, err)
	}

	rollbackResp, err := svc.RollbackVersion(ctx, &pb.RollbackVersionRequest{VersionId: "v1", Reason: "test"})
	if err != nil || rollbackResp.RollbackVersionId != "v3" {
		t.Fatalf("RollbackVersion failed resp=%#v err=%v", rollbackResp, err)
	}
}

func fixedTime(offsetSec int64) time.Time {
	return time.Unix(1700000000+offsetSec, 0).UTC()
}

func TestGraphServiceCoverageAndTraceabilityAPI(t *testing.T) {
	svc := NewGraphService(&mockGraphUsecase{
		createNodeFn: func(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
			return nil, nil
		},
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, &fakeAnalyticsEngine{
		coverageFn: func(ctx context.Context, namespace, domain string) (*analytics.CoverageReport, error) {
			if namespace != "graph/app-1/tenant-1" || domain != "payment" {
				t.Fatalf("unexpected coverage args namespace=%s domain=%s", namespace, domain)
			}
			return &analytics.CoverageReport{
				Domain:          "payment",
				TotalEntities:   10,
				CoveredEntities: 8,
				CoveragePercent: 80,
				UncoveredTypes:  []string{"NFR"},
				GeneratedAt:     fixedTime(0),
				ByType: []analytics.CoverageByType{
					{EntityType: "Requirement", TotalEntities: 10, CoveredEntities: 8, CoveragePercent: 80},
				},
			}, nil
		},
		traceabilityFn: func(ctx context.Context, namespace string, sourceTypes, targetTypes []string, maxHops int) (*analytics.TraceabilityMatrix, error) {
			if namespace != "graph/app-1/tenant-1" || maxHops != 3 {
				t.Fatalf("unexpected trace args namespace=%s maxHops=%d", namespace, maxHops)
			}
			return &analytics.TraceabilityMatrix{
				Matrix: []analytics.TraceabilityRow{
					{
						SourceID:   "S1",
						SourceName: "FR-001",
						SourceType: "Requirement",
						Targets: []analytics.TraceabilityTarget{
							{EntityID: "T1", Name: "UC-001", Type: "UseCase", Hops: 1, Path: []string{"IMPLEMENTS"}},
						},
					},
				},
				TotalSources:      1,
				TotalTargets:      1,
				ComputeDurationMs: 120,
			}, nil
		},
	}, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read",
	})

	coverageResp, err := svc.GetCoverageReport(ctx, &pb.GetCoverageReportRequest{Domain: "payment"})
	if err != nil {
		t.Fatalf("GetCoverageReport error: %v", err)
	}
	if coverageResp.TotalEntities != 10 || coverageResp.CoveredEntities != 8 || coverageResp.CoveragePercent != 80 {
		t.Fatalf("unexpected coverage response: %#v", coverageResp)
	}
	if len(coverageResp.ByType) != 1 || coverageResp.ByType[0].EntityType != "Requirement" {
		t.Fatalf("unexpected coverage by_type: %#v", coverageResp.ByType)
	}

	traceResp, err := svc.GetTraceabilityMatrix(ctx, &pb.GetTraceabilityMatrixRequest{
		SourceTypes: []string{"Requirement"},
		TargetTypes: []string{"UseCase"},
		MaxHops:     3,
	})
	if err != nil {
		t.Fatalf("GetTraceabilityMatrix error: %v", err)
	}
	if traceResp.TotalSources != 1 || traceResp.TotalTargets != 1 {
		t.Fatalf("unexpected traceability totals: %#v", traceResp)
	}
	if len(traceResp.Matrix) != 1 || len(traceResp.Matrix[0].Targets) != 1 {
		t.Fatalf("unexpected traceability matrix: %#v", traceResp.Matrix)
	}
}
