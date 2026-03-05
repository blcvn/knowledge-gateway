package integration

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"kgs-platform/internal/analytics"
	"kgs-platform/internal/batch"
	"kgs-platform/internal/overlay"
	"kgs-platform/internal/search"
	"kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPhase5Integration_Batch(t *testing.T) {
	uc := batch.NewUsecaseWithIndexer(&integrationWriter{}, &integrationDeduper{}, &integrationIndexer{})
	out, err := uc.Execute(context.Background(), batch.BatchUpsertRequest{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Entities: []batch.Entity{{Label: "Requirement", Properties: map[string]any{"name": "FR-001"}}},
	})
	if err != nil {
		t.Fatalf("batch execute failed: %v", err)
	}
	if out.Created != 1 || out.Skipped != 0 {
		t.Fatalf("unexpected batch output: %#v", out)
	}
}

func TestPhase5Integration_Search(t *testing.T) {
	engine := search.NewEngine(&integrationVector{}, &integrationText{}, &integrationCentrality{})
	results, err := engine.HybridSearch(context.Background(), "graph/app-1/tenant-1", "payment", search.Options{TopK: 5, Alpha: 0.5, Beta: 0.2})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected non-empty search results")
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected positive score: %#v", results[0])
	}
}

func TestPhase5Integration_OverlayVersion(t *testing.T) {
	ctx := context.Background()
	versionMgr := newIntegrationVersionManager(t)
	store := newIntegrationOverlayStore()
	mgr := overlay.NewManager(store, versionMgr, nil, log.NewStdLogger(io.Discard))

	item, err := mgr.Create(ctx, "graph/app-1/tenant-1", "session-1", "current")
	if err != nil {
		t.Fatalf("create overlay failed: %v", err)
	}
	if _, err := mgr.AddEntityDelta(ctx, item.OverlayID, item.Namespace, "Requirement", map[string]any{"name": "FR-001"}); err != nil {
		t.Fatalf("add entity delta failed: %v", err)
	}
	if _, err := mgr.AddEdgeDelta(ctx, item.OverlayID, item.Namespace, "IMPLEMENTS", "n1", "n2", map[string]any{}); err != nil {
		t.Fatalf("add edge delta failed: %v", err)
	}
	commit, err := mgr.Commit(ctx, item.OverlayID, "KEEP_OVERLAY")
	if err != nil {
		t.Fatalf("commit overlay failed: %v", err)
	}
	if commit.NewVersionID == "" || commit.EntitiesCommitted != 1 || commit.EdgesCommitted != 1 {
		t.Fatalf("unexpected commit result: %#v", commit)
	}

	versions, err := versionMgr.ListVersions(ctx, item.Namespace)
	if err != nil {
		t.Fatalf("list versions failed: %v", err)
	}
	if len(versions) == 0 {
		t.Fatalf("expected versions after overlay commit")
	}
}

func TestPhase5Integration_Analytics(t *testing.T) {
	query := &integrationQuery{}
	engine := analytics.NewEngine(query, analytics.NewCache(nil))

	coverage, err := engine.CoverageReport(context.Background(), "graph/app-1/tenant-1", "payment")
	if err != nil {
		t.Fatalf("coverage report failed: %v", err)
	}
	if coverage.TotalEntities != 10 || coverage.CoveredEntities != 8 {
		t.Fatalf("unexpected coverage report: %#v", coverage)
	}

	trace, err := engine.TraceabilityMatrix(context.Background(), "graph/app-1/tenant-1", []string{"Requirement"}, []string{"UseCase"}, 3)
	if err != nil {
		t.Fatalf("traceability failed: %v", err)
	}
	if trace.TotalSources != 1 || trace.TotalTargets != 1 {
		t.Fatalf("unexpected traceability report: %#v", trace)
	}
}

type integrationWriter struct{}

func (w *integrationWriter) BulkCreate(ctx context.Context, appID, tenantID string, entities []batch.Entity) (int, error) {
	return len(entities), nil
}

type integrationDeduper struct{}

func (d *integrationDeduper) Dedup(ctx context.Context, appID, tenantID string, entities []batch.Entity) ([]batch.Entity, int, error) {
	return entities, 0, nil
}

type integrationIndexer struct{}

func (d *integrationIndexer) IndexEntities(ctx context.Context, appID, tenantID string, entities []batch.Entity) error {
	return nil
}

type integrationVector struct{}

func (v *integrationVector) Search(ctx context.Context, namespace, query string, topK int) ([]search.Result, error) {
	return []search.Result{{ID: "n1", Label: "Requirement", Score: 0.9, Properties: map[string]any{"name": "FR-001"}}}, nil
}

type integrationText struct{}

func (v *integrationText) Search(ctx context.Context, namespace, query string, topK int) ([]search.Result, error) {
	return []search.Result{{ID: "n1", Label: "Requirement", Score: 0.8, Properties: map[string]any{"name": "FR-001"}}}, nil
}

type integrationCentrality struct{}

func (c *integrationCentrality) Scores(ctx context.Context, namespace string, nodeIDs []string) (map[string]float64, error) {
	return map[string]float64{"n1": 0.5}, nil
}

type integrationQuery struct{}

func (q *integrationQuery) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	switch {
	case strings.Contains(cypher, "covered_entities"):
		return map[string]any{"data": []map[string]any{{"entity_type": "Requirement", "total_entities": int64(10), "covered_entities": int64(8)}}}, nil
	case strings.Contains(cypher, "MATCH p=(s)-[*1..$max_hops]->(t)"):
		return map[string]any{"data": []map[string]any{{"source_id": "S1", "source_name": "FR-001", "source_type": "Requirement", "target_id": "T1", "target_name": "UC-001", "target_type": "UseCase", "hops": int64(1), "path": []any{"IMPLEMENTS"}}}}, nil
	default:
		return map[string]any{"data": []map[string]any{}}, nil
	}
}

type integrationOverlayStore struct {
	mu       sync.Mutex
	overlays map[string]*overlay.OverlayGraph
	sessions map[string]string
}

func newIntegrationOverlayStore() *integrationOverlayStore {
	return &integrationOverlayStore{overlays: map[string]*overlay.OverlayGraph{}, sessions: map[string]string{}}
}

func (s *integrationOverlayStore) Save(ctx context.Context, item *overlay.OverlayGraph, ttl time.Duration) error {
	_ = ctx
	_ = ttl
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *item
	s.overlays[item.OverlayID] = &cp
	return nil
}

func (s *integrationOverlayStore) Get(ctx context.Context, overlayID string) (*overlay.OverlayGraph, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.overlays[overlayID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *item
	return &cp, nil
}

func (s *integrationOverlayStore) Delete(ctx context.Context, overlayID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.overlays, overlayID)
	return nil
}

func (s *integrationOverlayStore) BindSession(ctx context.Context, sessionID, overlayID string, ttl time.Duration) error {
	_ = ctx
	_ = ttl
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = overlayID
	return nil
}

func (s *integrationOverlayStore) UnbindSession(ctx context.Context, sessionID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

func (s *integrationOverlayStore) FindBySession(ctx context.Context, sessionID string) (string, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[sessionID], nil
}

func newIntegrationVersionManager(t *testing.T) *version.Manager {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&version.GraphVersion{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return version.NewManager(db, log.NewStdLogger(io.Discard))
}
