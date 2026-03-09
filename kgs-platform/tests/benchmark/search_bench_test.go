package benchmark

import (
	"context"
	"testing"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/batch"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/search"
)

func BenchmarkHybridSearch(b *testing.B) {
	engine := search.NewEngine(&benchVector{}, &benchText{}, &benchCentrality{})
	ctx := context.Background()
	opts := search.Options{TopK: 20, Alpha: 0.6, Beta: 0.2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.HybridSearch(ctx, "graph/app-1/tenant-1", "payment requirements", opts)
		if err != nil {
			b.Fatalf("HybridSearch failed: %v", err)
		}
	}
}

func BenchmarkBatchUpsert(b *testing.B) {
	uc := batch.NewUsecaseWithIndexer(&benchWriter{}, &benchDeduper{}, &benchIndexer{})
	entities := make([]batch.Entity, 0, 200)
	for i := 0; i < 200; i++ {
		entities = append(entities, batch.Entity{Label: "Requirement", Properties: map[string]any{"name": "FR"}})
	}
	req := batch.BatchUpsertRequest{AppID: "app-1", TenantID: "tenant-1", Entities: entities}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.Execute(ctx, req)
		if err != nil {
			b.Fatalf("Batch Execute failed: %v", err)
		}
	}
}

type benchVector struct{}

func (v *benchVector) Search(ctx context.Context, namespace, query string, topK int) ([]search.Result, error) {
	out := make([]search.Result, 0, topK)
	for i := 0; i < topK; i++ {
		out = append(out, search.Result{ID: "n", Label: "Requirement", Score: 0.7})
	}
	return out, nil
}

type benchText struct{}

func (v *benchText) Search(ctx context.Context, namespace, query string, topK int) ([]search.Result, error) {
	out := make([]search.Result, 0, topK)
	for i := 0; i < topK; i++ {
		out = append(out, search.Result{ID: "n", Label: "Requirement", Score: 0.6})
	}
	return out, nil
}

type benchCentrality struct{}

func (c *benchCentrality) Scores(ctx context.Context, namespace string, nodeIDs []string) (map[string]float64, error) {
	return map[string]float64{"n": 0.4}, nil
}

type benchWriter struct{}

func (w *benchWriter) BulkCreate(ctx context.Context, appID, tenantID string, entities []batch.Entity) (int, error) {
	return len(entities), nil
}

type benchDeduper struct{}

func (d *benchDeduper) Dedup(ctx context.Context, appID, tenantID string, entities []batch.Entity) ([]batch.Entity, int, error) {
	return entities, 0, nil
}

type benchIndexer struct{}

func (d *benchIndexer) IndexEntities(ctx context.Context, appID, tenantID string, entities []batch.Entity) error {
	return nil
}
