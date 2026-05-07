package biz

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type deleteRepoMock struct {
	deleteNodeFn       func(ctx context.Context, appID, tenantID, nodeID string) (int, error)
	deleteEdgeFn       func(ctx context.Context, appID, tenantID, edgeID string) error
	batchDeleteNodesFn func(ctx context.Context, appID, tenantID string, nodeIDs []string) (int, int, error)
}

func (m *deleteRepoMock) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

func (m *deleteRepoMock) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (m *deleteRepoMock) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

func (m *deleteRepoMock) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	return map[string]any{"data": []map[string]any{}}, nil
}

func (m *deleteRepoMock) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*FullGraphResult, error) {
	return &FullGraphResult{}, nil
}

func (m *deleteRepoMock) DeleteNode(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
	if m.deleteNodeFn == nil {
		return 0, nil
	}
	return m.deleteNodeFn(ctx, appID, tenantID, nodeID)
}

func (m *deleteRepoMock) DeleteEdge(ctx context.Context, appID, tenantID, edgeID string) error {
	if m.deleteEdgeFn == nil {
		return nil
	}
	return m.deleteEdgeFn(ctx, appID, tenantID, edgeID)
}

func (m *deleteRepoMock) BatchDeleteNodes(ctx context.Context, appID, tenantID string, nodeIDs []string) (int, int, error) {
	if m.batchDeleteNodesFn == nil {
		return len(nodeIDs), 0, nil
	}
	return m.batchDeleteNodesFn(ctx, appID, tenantID, nodeIDs)
}

type deleteOverlayWriterMock struct {
	deleteNodeCalls int
	deleteEdgeCalls int
}

func (m *deleteOverlayWriterMock) AddEntityDelta(ctx context.Context, overlayID, namespace, label string, properties map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

func (m *deleteOverlayWriterMock) AddEdgeDelta(ctx context.Context, overlayID, namespace, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

func (m *deleteOverlayWriterMock) DeleteEntityDelta(ctx context.Context, overlayID, nodeID string) error {
	m.deleteNodeCalls++
	return nil
}

func (m *deleteOverlayWriterMock) DeleteEdgeDelta(ctx context.Context, overlayID, edgeID string) error {
	m.deleteEdgeCalls++
	return nil
}

type deleteLockSpy struct {
	acquireNodeCalls      int
	acquireNamespaceCalls int
	releaseCalls          int
	lastNamespace         string
}

func (m *deleteLockSpy) AcquireNodeLock(ctx context.Context, namespace, nodeID string, ttl time.Duration) (string, error) {
	m.acquireNodeCalls++
	return "node:" + nodeID, nil
}

func (m *deleteLockSpy) AcquireSubgraphLock(ctx context.Context, namespace, rootID string, depth int, ttl time.Duration) (string, error) {
	return "", nil
}

func (m *deleteLockSpy) AcquireVersionLock(ctx context.Context, namespace string, ttl time.Duration) (string, error) {
	return "", nil
}

func (m *deleteLockSpy) AcquireNamespaceLock(ctx context.Context, namespace string, ttl time.Duration) (string, error) {
	m.acquireNamespaceCalls++
	m.lastNamespace = namespace
	return "ns:" + namespace, nil
}

func (m *deleteLockSpy) Release(ctx context.Context, lockToken string) error {
	m.releaseCalls++
	return nil
}

type serialNodeLockManager struct {
	mu        sync.Mutex
	cond      *sync.Cond
	locked    map[string]bool
	tokenNode map[string]string
}

func newSerialNodeLockManager() *serialNodeLockManager {
	m := &serialNodeLockManager{
		locked:    map[string]bool{},
		tokenNode: map[string]string{},
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *serialNodeLockManager) AcquireNodeLock(ctx context.Context, namespace, nodeID string, ttl time.Duration) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for m.locked[nodeID] {
		m.cond.Wait()
	}
	m.locked[nodeID] = true
	token := "tok:" + nodeID + ":" + time.Now().String()
	m.tokenNode[token] = nodeID
	return token, nil
}

func (m *serialNodeLockManager) AcquireSubgraphLock(ctx context.Context, namespace, rootID string, depth int, ttl time.Duration) (string, error) {
	return "", nil
}

func (m *serialNodeLockManager) AcquireVersionLock(ctx context.Context, namespace string, ttl time.Duration) (string, error) {
	return "", nil
}

func (m *serialNodeLockManager) AcquireNamespaceLock(ctx context.Context, namespace string, ttl time.Duration) (string, error) {
	return "ns:" + namespace, nil
}

func (m *serialNodeLockManager) Release(ctx context.Context, lockToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	nodeID := m.tokenNode[lockToken]
	delete(m.tokenNode, lockToken)
	if nodeID != "" {
		delete(m.locked, nodeID)
		m.cond.Broadcast()
	}
	return nil
}

func TestGraphUsecaseDeleteNodeAcquireDeleteRelease(t *testing.T) {
	lockMgr := &deleteLockSpy{}
	repo := &deleteRepoMock{
		deleteNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
			return 3, nil
		},
	}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, lockMgr, nil, log.NewStdLogger(io.Discard))

	edgesRemoved, err := uc.DeleteNode(context.Background(), "app-1", "tenant-1", "node-1")
	if err != nil {
		t.Fatalf("DeleteNode error: %v", err)
	}
	if edgesRemoved != 3 {
		t.Fatalf("expected edgesRemoved=3, got %d", edgesRemoved)
	}
	if lockMgr.acquireNodeCalls != 1 {
		t.Fatalf("expected acquire node lock once, got %d", lockMgr.acquireNodeCalls)
	}
	if lockMgr.releaseCalls != 1 {
		t.Fatalf("expected release once, got %d", lockMgr.releaseCalls)
	}
}

func TestGraphUsecaseDeleteNodeOverlay(t *testing.T) {
	overlay := &deleteOverlayWriterMock{}
	repoCalled := 0
	repo := &deleteRepoMock{
		deleteNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
			repoCalled++
			return 0, nil
		},
	}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, nil, overlay, log.NewStdLogger(io.Discard))

	ctx := context.WithValue(context.Background(), "overlay_id", "ov-1")
	if _, err := uc.DeleteNode(ctx, "app-1", "tenant-1", "node-1"); err != nil {
		t.Fatalf("DeleteNode overlay error: %v", err)
	}
	if overlay.deleteNodeCalls != 1 {
		t.Fatalf("expected overlay delete node call, got %d", overlay.deleteNodeCalls)
	}
	if repoCalled != 0 {
		t.Fatalf("expected base repo not called in overlay mode, got %d", repoCalled)
	}
}

func TestGraphUsecaseBatchDeleteNodesAcquireNamespaceLock(t *testing.T) {
	lockMgr := &deleteLockSpy{}
	repo := &deleteRepoMock{
		batchDeleteNodesFn: func(ctx context.Context, appID, tenantID string, nodeIDs []string) (int, int, error) {
			return len(nodeIDs), 5, nil
		},
	}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, lockMgr, nil, log.NewStdLogger(io.Discard))

	deleted, edgesRemoved, err := uc.BatchDeleteNodes(context.Background(), "app-1", "tenant-1", []string{"n1", "n2"})
	if err != nil {
		t.Fatalf("BatchDeleteNodes error: %v", err)
	}
	if deleted != 2 || edgesRemoved != 5 {
		t.Fatalf("unexpected result deleted=%d edgesRemoved=%d", deleted, edgesRemoved)
	}
	if lockMgr.acquireNamespaceCalls != 1 {
		t.Fatalf("expected acquire namespace lock once, got %d", lockMgr.acquireNamespaceCalls)
	}
	if lockMgr.lastNamespace != "graph/app-1/tenant-1" {
		t.Fatalf("unexpected namespace lock target: %s", lockMgr.lastNamespace)
	}
}

func TestGraphUsecaseConcurrentDeleteNodeLockPreventsRace(t *testing.T) {
	lockMgr := newSerialNodeLockManager()
	var inFlight int32
	var maxInFlight int32

	repo := &deleteRepoMock{
		deleteNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
			current := atomic.AddInt32(&inFlight, 1)
			for {
				max := atomic.LoadInt32(&maxInFlight)
				if current <= max || atomic.CompareAndSwapInt32(&maxInFlight, max, current) {
					break
				}
			}
			time.Sleep(40 * time.Millisecond)
			atomic.AddInt32(&inFlight, -1)
			return 0, nil
		},
	}
	uc := NewGraphUsecase(repo, NewQueryPlanner(), nil, nil, nil, lockMgr, nil, log.NewStdLogger(io.Discard))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = uc.DeleteNode(context.Background(), "app-1", "tenant-1", "node-1")
	}()
	go func() {
		defer wg.Done()
		_, _ = uc.DeleteNode(context.Background(), "app-1", "tenant-1", "node-1")
	}()
	wg.Wait()

	if atomic.LoadInt32(&maxInFlight) > 1 {
		t.Fatalf("expected serialized delete by lock, max in-flight=%d", atomic.LoadInt32(&maxInFlight))
	}
}
