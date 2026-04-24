package analytics

import (
	"context"
	"testing"
)

type stubQueryExecutor struct {
	calls int
	fn    func(ctx context.Context, cypher string, params map[string]any) (map[string]any, error)
}

func (s *stubQueryExecutor) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	s.calls++
	return s.fn(ctx, cypher, params)
}

func TestCoverageReportComputesTotalsAndCaches(t *testing.T) {
	executor := &stubQueryExecutor{
		fn: func(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
			return map[string]any{
				"data": []map[string]any{
					{"entity_type": "Requirement", "total_entities": int64(10), "covered_entities": int64(8)},
					{"entity_type": "UseCase", "total_entities": int64(5), "covered_entities": int64(5)},
				},
			}, nil
		},
	}
	engine := NewEngine(executor, NewCache(nil))

	report, err := engine.CoverageReport(context.Background(), "graph/app-1/tenant-1", "payment")
	if err != nil {
		t.Fatalf("CoverageReport error: %v", err)
	}
	if report.TotalEntities != 15 {
		t.Fatalf("unexpected total entities: %d", report.TotalEntities)
	}
	if report.CoveredEntities != 13 {
		t.Fatalf("unexpected covered entities: %d", report.CoveredEntities)
	}
	if len(report.UncoveredTypes) != 1 || report.UncoveredTypes[0] != "Requirement" {
		t.Fatalf("unexpected uncovered types: %#v", report.UncoveredTypes)
	}
	if executor.calls != 1 {
		t.Fatalf("expected one query execution, got %d", executor.calls)
	}

	_, err = engine.CoverageReport(context.Background(), "graph/app-1/tenant-1", "payment")
	if err != nil {
		t.Fatalf("CoverageReport cached call error: %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("expected cached report to skip query, got %d calls", executor.calls)
	}
}
