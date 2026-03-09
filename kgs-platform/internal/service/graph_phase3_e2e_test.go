package service

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/graph/v1"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/overlay"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/server/middleware"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type phase3Repo struct{}

func (r *phase3Repo) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	return map[string]any{"id": properties["id"], "label": label}, nil
}
func (r *phase3Repo) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	return map[string]any{"id": nodeID, "label": "Requirement"}, nil
}
func (r *phase3Repo) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	return map[string]any{"id": properties["id"], "type": relationType}, nil
}
func (r *phase3Repo) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	return map[string]any{"data": []map[string]any{}}, nil
}
func (r *phase3Repo) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error) {
	return &biz.FullGraphResult{}, nil
}

type memOverlayStore struct {
	items       map[string]*overlay.OverlayGraph
	sessionBind map[string]string
}

func newMemOverlayStore() *memOverlayStore {
	return &memOverlayStore{
		items:       map[string]*overlay.OverlayGraph{},
		sessionBind: map[string]string{},
	}
}

func (s *memOverlayStore) Save(ctx context.Context, item *overlay.OverlayGraph, ttl time.Duration) error {
	_ = ctx
	_ = ttl
	cp := *item
	s.items[item.OverlayID] = &cp
	return nil
}
func (s *memOverlayStore) Get(ctx context.Context, overlayID string) (*overlay.OverlayGraph, error) {
	_ = ctx
	item, ok := s.items[overlayID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *item
	return &cp, nil
}
func (s *memOverlayStore) Delete(ctx context.Context, overlayID string) error {
	_ = ctx
	delete(s.items, overlayID)
	return nil
}
func (s *memOverlayStore) BindSession(ctx context.Context, sessionID, overlayID string, ttl time.Duration) error {
	_ = ctx
	_ = ttl
	s.sessionBind[sessionID] = overlayID
	return nil
}
func (s *memOverlayStore) UnbindSession(ctx context.Context, sessionID string) error {
	_ = ctx
	delete(s.sessionBind, sessionID)
	return nil
}
func (s *memOverlayStore) FindBySession(ctx context.Context, sessionID string) (string, error) {
	_ = ctx
	return s.sessionBind[sessionID], nil
}

func TestPhase3OverlayLifecycleViaAPI(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:phase3_api_e2e?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&version.GraphVersion{}); err != nil {
		t.Fatalf("migrate graph versions: %v", err)
	}

	versionMgr := version.NewManager(db, log.DefaultLogger)
	namespace := biz.ComputeNamespace("app-1", "tenant-1")
	baseVersionID, err := versionMgr.CreateDelta(context.Background(), namespace, version.ChangeSet{
		CommitMessage: "base",
	})
	if err != nil {
		t.Fatalf("seed base version: %v", err)
	}

	store := newMemOverlayStore()
	overlayMgr := overlay.NewManager(store, versionMgr, nil, log.DefaultLogger)
	repo := &phase3Repo{}
	graphUC := biz.NewGraphUsecase(repo, biz.NewQueryPlanner(), nil, nil, nil, overlayMgr, log.NewStdLogger(io.Discard))
	svc := NewGraphService(graphUC, nil, nil, overlayMgr, versionMgr, nil, nil)

	ctx := context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read,write",
	})

	createOverlayResp, err := svc.CreateOverlay(ctx, &pb.CreateOverlayRequest{
		SessionId:   "session-api-1",
		BaseVersion: "current",
	})
	if err != nil {
		t.Fatalf("CreateOverlay: %v", err)
	}
	if createOverlayResp.BaseVersionId != baseVersionID {
		t.Fatalf("expected base version %s, got %s", baseVersionID, createOverlayResp.BaseVersionId)
	}

	nodeProps, _ := json.Marshal(map[string]any{
		"overlay_id": createOverlayResp.OverlayId,
		"name":       "FR-001",
	})
	createNodeResp, err := svc.CreateNode(ctx, &pb.CreateNodeRequest{
		Label:          "Requirement",
		PropertiesJson: string(nodeProps),
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if createNodeResp.NodeId == "" {
		t.Fatalf("expected node id")
	}

	edgeProps, _ := json.Marshal(map[string]any{
		"overlay_id": createOverlayResp.OverlayId,
		"strength":   0.9,
	})
	createEdgeResp, err := svc.CreateEdge(ctx, &pb.CreateEdgeRequest{
		SourceNodeId:   createNodeResp.NodeId,
		TargetNodeId:   createNodeResp.NodeId,
		RelationType:   "DEPENDS_ON",
		PropertiesJson: string(edgeProps),
	})
	if err != nil {
		t.Fatalf("CreateEdge: %v", err)
	}
	if createEdgeResp.EdgeId == "" {
		t.Fatalf("expected edge id")
	}

	commitResp, err := svc.CommitOverlay(ctx, &pb.CommitOverlayRequest{
		OverlayId:      createOverlayResp.OverlayId,
		ConflictPolicy: overlay.PolicyKeepOverlay,
	})
	if err != nil {
		t.Fatalf("CommitOverlay: %v", err)
	}
	if commitResp.NewVersionId == "" {
		t.Fatalf("expected new version id")
	}

	listResp, err := svc.ListVersions(ctx, &pb.ListVersionsRequest{})
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(listResp.Versions) < 2 {
		t.Fatalf("expected >=2 versions, got %d", len(listResp.Versions))
	}

	diffResp, err := svc.DiffVersions(ctx, &pb.DiffVersionsRequest{
		FromVersionId: baseVersionID,
		ToVersionId:   commitResp.NewVersionId,
	})
	if err != nil {
		t.Fatalf("DiffVersions: %v", err)
	}
	if diffResp.EntitiesAdded < 1 || diffResp.EdgesAdded < 1 {
		t.Fatalf("unexpected diff result: %#v", diffResp)
	}

	rollbackResp, err := svc.RollbackVersion(ctx, &pb.RollbackVersionRequest{
		VersionId: baseVersionID,
		Reason:    "phase3 e2e",
	})
	if err != nil {
		t.Fatalf("RollbackVersion: %v", err)
	}
	if rollbackResp.RollbackVersionId == "" {
		t.Fatalf("expected rollback version id")
	}
}
