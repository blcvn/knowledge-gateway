package biz

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

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
func (r *fakeGraphRepo) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*FullGraphResult, error) {
	return &FullGraphResult{}, nil
}

func TestGraphUsecaseGetContextDepth5UsesBatchedTraversal(t *testing.T) {
	repo := &fakeGraphRepo{}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, nil, nil, log.NewStdLogger(io.Discard))

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

type captureLockManager struct {
	releaseCalls  int
	releaseCtxErr error
}

type orderingLockManager struct {
	mu          sync.Mutex
	acquireSeq  []string
	acquireTTLs map[string]time.Duration
}

func (m *captureLockManager) AcquireNodeLock(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (m *captureLockManager) AcquireSubgraphLock(context.Context, string, string, int, time.Duration) (string, error) {
	return "", nil
}

func (m *captureLockManager) AcquireVersionLock(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (m *captureLockManager) AcquireNamespaceLock(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (m *captureLockManager) Release(ctx context.Context, lockToken string) error {
	m.releaseCalls++
	m.releaseCtxErr = ctx.Err()
	return nil
}

func (m *orderingLockManager) AcquireNodeLock(_ context.Context, _ string, nodeID string, ttl time.Duration) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acquireTTLs == nil {
		m.acquireTTLs = make(map[string]time.Duration)
	}
	m.acquireSeq = append(m.acquireSeq, nodeID)
	m.acquireTTLs[nodeID] = ttl
	return "tok-" + nodeID, nil
}

func (m *orderingLockManager) AcquireSubgraphLock(context.Context, string, string, int, time.Duration) (string, error) {
	return "", nil
}

func (m *orderingLockManager) AcquireVersionLock(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (m *orderingLockManager) AcquireNamespaceLock(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (m *orderingLockManager) Release(context.Context, string) error {
	return nil
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
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, nil, overlay, log.NewStdLogger(io.Discard))

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

func TestGraphUsecaseReleaseLockUsesDetachedContext(t *testing.T) {
	repo := &fakeGraphRepo{}
	lockMgr := &captureLockManager{}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, lockMgr, nil, log.NewStdLogger(io.Discard))

	lockCtx, cancel := context.WithCancel(context.Background())
	cancel()
	uc.releaseLock(lockCtx, "token-1")

	if lockMgr.releaseCalls != 1 {
		t.Fatalf("expected release call, got %d", lockMgr.releaseCalls)
	}
	if lockMgr.releaseCtxErr != nil {
		t.Fatalf("expected detached release context, got err=%v", lockMgr.releaseCtxErr)
	}
}

func TestGraphUsecaseCreateEdgeAcquiresNodeLocksInStableOrder(t *testing.T) {
	t.Setenv(nodeLockTTLEnvKey, "45s")

	repo := &fakeGraphRepo{}
	lockMgr := &orderingLockManager{}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, lockMgr, nil, log.NewStdLogger(io.Discard))

	_, err := uc.CreateEdge(context.Background(), "app-1", "tenant-1", "RELATES_TO", "node-b", "node-a", map[string]any{"id": "edge-1"})
	if err != nil {
		t.Fatalf("CreateEdge error: %v", err)
	}

	lockMgr.mu.Lock()
	defer lockMgr.mu.Unlock()
	if len(lockMgr.acquireSeq) != 2 {
		t.Fatalf("expected 2 lock acquisitions, got %d", len(lockMgr.acquireSeq))
	}
	if lockMgr.acquireSeq[0] != "node-a" || lockMgr.acquireSeq[1] != "node-b" {
		t.Fatalf("expected deterministic lexicographic lock order [node-a node-b], got %#v", lockMgr.acquireSeq)
	}
	if lockMgr.acquireTTLs["node-a"] != 45*time.Second || lockMgr.acquireTTLs["node-b"] != 45*time.Second {
		t.Fatalf("expected configured lock TTL=45s, got %#v", lockMgr.acquireTTLs)
	}
}

func TestLockTTLFromEnv_DefaultWhenInvalid(t *testing.T) {
	t.Setenv(nodeLockTTLEnvKey, "invalid")
	if got := lockTTLFromEnv(); got != defaultNodeLockTTL {
		t.Fatalf("expected default lock TTL for invalid env, got %v", got)
	}
}
