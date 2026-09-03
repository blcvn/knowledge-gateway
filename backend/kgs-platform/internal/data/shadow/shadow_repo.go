package shadow

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

// ShadowGraphRepo wraps a primary and secondary GraphRepo to enable shadow mode.
// - Writes: primary (sync) + secondary (async, best-effort)
// - Reads: primary (sync) + secondary (async comparison, log diffs)
// The primary result is always returned to the caller.
type ShadowGraphRepo struct {
	primary   biz.GraphRepo
	secondary biz.GraphRepo
	log       *log.Helper
	mu        sync.Mutex
	diffCount int
}

func NewShadowGraphRepo(primary, secondary biz.GraphRepo, logger log.Logger) biz.GraphRepo {
	return &ShadowGraphRepo{
		primary:   primary,
		secondary: secondary,
		log:       log.NewHelper(logger),
	}
}

// ── Write operations: primary sync + secondary async ──────────

func (s *ShadowGraphRepo) CreateNode(ctx context.Context, appID, tenantID, label string, props map[string]any) (map[string]any, error) {
	result, err := s.primary.CreateNode(ctx, appID, tenantID, label, props)
	go s.asyncWrite("CreateNode", func() error {
		_, e := s.secondary.CreateNode(ctx, appID, tenantID, label, props)
		return e
	})
	return result, err
}

func (s *ShadowGraphRepo) CreateEdge(ctx context.Context, appID, tenantID, relType, src, dst string, props map[string]any) (map[string]any, error) {
	result, err := s.primary.CreateEdge(ctx, appID, tenantID, relType, src, dst, props)
	go s.asyncWrite("CreateEdge", func() error {
		_, e := s.secondary.CreateEdge(ctx, appID, tenantID, relType, src, dst, props)
		return e
	})
	return result, err
}

func (s *ShadowGraphRepo) DeleteNode(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
	result, err := s.primary.DeleteNode(ctx, appID, tenantID, nodeID)
	go s.asyncWrite("DeleteNode", func() error {
		_, e := s.secondary.DeleteNode(ctx, appID, tenantID, nodeID)
		return e
	})
	return result, err
}

func (s *ShadowGraphRepo) DeleteEdge(ctx context.Context, appID, tenantID, edgeID string) error {
	err := s.primary.DeleteEdge(ctx, appID, tenantID, edgeID)
	go s.asyncWrite("DeleteEdge", func() error {
		return s.secondary.DeleteEdge(ctx, appID, tenantID, edgeID)
	})
	return err
}

func (s *ShadowGraphRepo) BatchDeleteNodes(ctx context.Context, appID, tenantID string, nodeIDs []string) (int, int, error) {
	n, e, err := s.primary.BatchDeleteNodes(ctx, appID, tenantID, nodeIDs)
	go s.asyncWrite("BatchDeleteNodes", func() error {
		_, _, e := s.secondary.BatchDeleteNodes(ctx, appID, tenantID, nodeIDs)
		return e
	})
	return n, e, err
}

// ── Read operations: primary sync + secondary async compare ──

func (s *ShadowGraphRepo) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	result, err := s.primary.GetNode(ctx, appID, tenantID, nodeID)
	go s.asyncRead("GetNode", result, func() (any, error) {
		return s.secondary.GetNode(ctx, appID, tenantID, nodeID)
	})
	return result, err
}

func (s *ShadowGraphRepo) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	result, err := s.primary.ExecuteQuery(ctx, cypher, params)
	go s.asyncRead("ExecuteQuery", result, func() (any, error) {
		return s.secondary.ExecuteQuery(ctx, cypher, params)
	})
	return result, err
}

func (s *ShadowGraphRepo) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error) {
	result, err := s.primary.GetFullGraph(ctx, appID, tenantID, limit, offset)
	go s.asyncRead("GetFullGraph", result, func() (any, error) {
		return s.secondary.GetFullGraph(ctx, appID, tenantID, limit, offset)
	})
	return result, err
}

// ── Async helpers ─────────────────────────────────────────────

func (s *ShadowGraphRepo) asyncWrite(op string, fn func() error) {
	if err := fn(); err != nil {
		s.log.Warnf("[Shadow] %s secondary write failed: %v", op, err)
	}
}

func (s *ShadowGraphRepo) asyncRead(op string, primaryResult any, fn func() (any, error)) {
	secondaryResult, err := fn()
	if err != nil {
		s.log.Warnf("[Shadow] %s secondary read failed: %v", op, err)
		return
	}
	s.compareResults(op, primaryResult, secondaryResult)
}

func (s *ShadowGraphRepo) compareResults(op string, primary, secondary any) {
	if reflect.DeepEqual(primary, secondary) {
		return
	}

	s.mu.Lock()
	s.diffCount++
	count := s.diffCount
	s.mu.Unlock()

	primaryJSON, _ := json.Marshal(primary)
	secondaryJSON, _ := json.Marshal(secondary)

	s.log.Warnf("[Shadow] DIFF #%d op=%s primary=%s secondary=%s",
		count,
		op,
		truncateBytes(primaryJSON, 500),
		truncateBytes(secondaryJSON, 500),
	)
}

func truncateBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

var _ biz.GraphRepo = (*ShadowGraphRepo)(nil)
