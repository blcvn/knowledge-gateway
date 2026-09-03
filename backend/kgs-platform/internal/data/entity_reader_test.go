package data

import (
	"context"
	"errors"
	"testing"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
)

type readerGraphRepoStub struct {
	getNodeFn func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error)
}

func (s *readerGraphRepoStub) CreateNode(context.Context, string, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (s *readerGraphRepoStub) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	if s.getNodeFn == nil {
		return nil, errors.New("not found")
	}
	return s.getNodeFn(ctx, appID, tenantID, nodeID)
}
func (s *readerGraphRepoStub) CreateEdge(context.Context, string, string, string, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (s *readerGraphRepoStub) ExecuteQuery(context.Context, string, map[string]any) (map[string]any, error) {
	return map[string]any{"data": []map[string]any{}}, nil
}
func (s *readerGraphRepoStub) GetFullGraph(context.Context, string, string, int, int) (*biz.FullGraphResult, error) {
	return &biz.FullGraphResult{}, nil
}
func (s *readerGraphRepoStub) DeleteNode(context.Context, string, string, string) (int, error) {
	return 0, nil
}
func (s *readerGraphRepoStub) DeleteEdge(context.Context, string, string, string) error {
	return nil
}
func (s *readerGraphRepoStub) BatchDeleteNodes(context.Context, string, string, []string) (int, int, error) {
	return 0, 0, nil
}

func TestEntityReaderGetEntityNeo4jFresh(t *testing.T) {
	db := newKGTestDB(t)
	entity := KGEntity{
		EntityID:   "e1",
		AppID:      "app",
		TenantID:   "tenant",
		EntityType: "Requirement",
		Name:       "FR-1",
		Version:    2,
		Properties: JSONMap{"id": "e1", "name": "FR-1", "version": 2},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	repo := &readerGraphRepoStub{
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return map[string]any{"id": "e1", "label": "Requirement", "version": 2, "name": "neo"}, nil
		},
	}
	reader := NewEntityReader(repo, db)

	got, err := reader.GetEntity(context.Background(), "app", "tenant", "e1")
	if err != nil {
		t.Fatalf("GetEntity error: %v", err)
	}
	if got["name"] != "neo" {
		t.Fatalf("expected Neo4j data, got %#v", got["name"])
	}
}

func TestEntityReaderGetEntityNeo4jStaleFallbackPG(t *testing.T) {
	db := newKGTestDB(t)
	entity := KGEntity{
		EntityID:   "e2",
		AppID:      "app",
		TenantID:   "tenant",
		EntityType: "Requirement",
		Name:       "PG Name",
		Version:    3,
		Properties: JSONMap{"id": "e2", "name": "PG Name", "version": 3},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	repo := &readerGraphRepoStub{
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return map[string]any{"id": "e2", "label": "Requirement", "version": 1, "name": "neo"}, nil
		},
	}
	reader := NewEntityReader(repo, db)

	got, err := reader.GetEntity(context.Background(), "app", "tenant", "e2")
	if err != nil {
		t.Fatalf("GetEntity error: %v", err)
	}
	if got["name"] != "PG Name" {
		t.Fatalf("expected Postgres fallback, got %#v", got["name"])
	}
}

func TestEntityReaderGetEntityNeo4jMissFallbackPG(t *testing.T) {
	db := newKGTestDB(t)
	entity := KGEntity{
		EntityID:   "e3",
		AppID:      "app",
		TenantID:   "tenant",
		EntityType: "Requirement",
		Name:       "From PG",
		Version:    1,
		Properties: JSONMap{"id": "e3", "name": "From PG"},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	reader := NewEntityReader(&readerGraphRepoStub{
		getNodeFn: func(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
			return nil, errors.New("not found")
		},
	}, db)

	got, err := reader.GetEntity(context.Background(), "app", "tenant", "e3")
	if err != nil {
		t.Fatalf("GetEntity error: %v", err)
	}
	if got["name"] != "From PG" {
		t.Fatalf("expected PG fallback, got %#v", got["name"])
	}
}

func TestEntityReaderEnrichWithFreshVersions(t *testing.T) {
	db := newKGTestDB(t)
	seed := []KGEntity{
		{
			EntityID:   "n1",
			AppID:      "app",
			TenantID:   "tenant",
			EntityType: "Requirement",
			Name:       "N1",
			Version:    2,
			Properties: JSONMap{"id": "n1", "name": "N1", "version": 2},
		},
		{
			EntityID:   "n2",
			AppID:      "app",
			TenantID:   "tenant",
			EntityType: "Requirement",
			Name:       "N2",
			Version:    1,
			Properties: JSONMap{"id": "n2", "name": "N2", "version": 1},
		},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed entity %s: %v", seed[i].EntityID, err)
		}
	}

	reader := NewEntityReader(nil, db)
	input := []map[string]any{
		{"id": "n1", "version": 1, "name": "neo-n1"},
		{"id": "n2", "version": 1, "name": "neo-n2"},
	}
	out, err := reader.EnrichWithFreshVersions(context.Background(), "app", "tenant", input)
	if err != nil {
		t.Fatalf("EnrichWithFreshVersions error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("unexpected len(out)=%d", len(out))
	}
	if out[0]["name"] != "N1" {
		t.Fatalf("expected stale entity replaced from PG, got %#v", out[0]["name"])
	}
	if out[1]["name"] != "neo-n2" {
		t.Fatalf("expected fresh entity unchanged, got %#v", out[1]["name"])
	}
}

func TestEntityReaderEnrichWithFreshVersionsEmpty(t *testing.T) {
	reader := NewEntityReader(nil, newKGTestDB(t))
	out, err := reader.EnrichWithFreshVersions(context.Background(), "app", "tenant", nil)
	if err != nil {
		t.Fatalf("EnrichWithFreshVersions error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty result, got %d", len(out))
	}
}
