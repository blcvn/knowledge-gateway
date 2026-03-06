package analytics

import (
	"context"
	"testing"
)

func TestTraceabilityMatrixGroupsBySource(t *testing.T) {
	executor := &stubQueryExecutor{
		fn: func(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
			return map[string]any{
				"data": []map[string]any{
					{
						"source_id":   "S1",
						"source_name": "FR-001",
						"source_type": "Requirement",
						"target_id":   "T1",
						"target_name": "UC-001",
						"target_type": "UseCase",
						"hops":        int64(1),
						"path":        []any{"IMPLEMENTS"},
					},
					{
						"source_id":   "S1",
						"source_name": "FR-001",
						"source_type": "Requirement",
						"target_id":   "T2",
						"target_name": "API-001",
						"target_type": "APIEndpoint",
						"hops":        int64(2),
						"path":        []any{"IMPLEMENTS", "CALLS"},
					},
				},
			}, nil
		},
	}
	engine := NewEngine(executor, NewCache(nil))

	report, err := engine.TraceabilityMatrix(context.Background(), "graph/app-1/tenant-1", []string{"Requirement"}, []string{"UseCase", "APIEndpoint"}, 3)
	if err != nil {
		t.Fatalf("TraceabilityMatrix error: %v", err)
	}
	if report.TotalSources != 1 {
		t.Fatalf("unexpected total sources: %d", report.TotalSources)
	}
	if report.TotalTargets != 2 {
		t.Fatalf("unexpected total targets: %d", report.TotalTargets)
	}
	if len(report.Matrix) != 1 || len(report.Matrix[0].Targets) != 2 {
		t.Fatalf("unexpected matrix: %#v", report.Matrix)
	}
	if report.Matrix[0].Targets[1].Hops != 2 {
		t.Fatalf("unexpected hops: %#v", report.Matrix[0].Targets)
	}
}

func TestTraceabilityMatrixEmptyFiltersSkipsQuery(t *testing.T) {
	executor := &stubQueryExecutor{
		fn: func(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
			t.Fatalf("query should not be executed for empty filters")
			return nil, nil
		},
	}
	engine := NewEngine(executor, nil)

	report, err := engine.TraceabilityMatrix(context.Background(), "graph/app-1/tenant-1", nil, []string{"UseCase"}, 0)
	if err != nil {
		t.Fatalf("TraceabilityMatrix error: %v", err)
	}
	if len(report.Matrix) != 0 || executor.calls != 0 {
		t.Fatalf("expected empty matrix without query, matrix=%#v calls=%d", report.Matrix, executor.calls)
	}
}
