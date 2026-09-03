package usecase_test

import (
	"context"
	"testing"

	"vnp-memory/services/vnp-search-hub/domain/model"
	"vnp-memory/services/vnp-search-hub/usecase"
)

type mockEngineClient struct {
	name    string
	results []model.RecallResult
	err     error
}

func (m *mockEngineClient) Search(_ context.Context, _ *model.RecallRequest) (*model.EngineResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.EngineResult{
		EngineName: m.name,
		Results:    m.results,
		LatencyMs:  50,
	}, nil
}

func TestRecallService_FanOutAndRerank(t *testing.T) {
	configs := []model.EngineConfig{
		{Name: "cognee", Enabled: true},
		{Name: "graphiti", Enabled: true},
		{Name: "zep", Enabled: true},
	}
	svc := usecase.NewRecallService(configs)

	// Register mock engines
	svc.RegisterEngine("cognee", &mockEngineClient{
		name: "cognee",
		results: []model.RecallResult{
			{Content: "Cognee fact about AI memory systems", Score: 0.95, Source: "cognee", Type: "fact"},
			{Content: "Cognee knowledge graph entry for LLM", Score: 0.85, Source: "cognee", Type: "fact"},
		},
	})
	svc.RegisterEngine("graphiti", &mockEngineClient{
		name: "graphiti",
		results: []model.RecallResult{
			{Content: "Graphiti episode: user discussed project timeline", Score: 0.90, Source: "graphiti", Type: "episode"},
		},
	})
	svc.RegisterEngine("zep", &mockEngineClient{
		name: "zep",
		results: []model.RecallResult{
			{Content: "Zep memory: user prefers dark mode", Score: 0.80, Source: "zep", Type: "memory"},
			{Content: "Cognee fact about AI memory systems", Score: 0.70, Source: "zep", Type: "memory"}, // duplicate content
		},
	})

	req := &model.RecallRequest{
		Query:          "What do I know about AI?",
		Scope:          model.ScopeAll,
		MaxResults:     10,
		RerankStrategy: model.RerankRRF,
		TokenBudget:    4096,
	}

	resp, err := svc.Recall(context.Background(), req)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}

	// Should have results from all 3 engines
	if len(resp.Metadata.EnginesUsed) != 3 {
		t.Errorf("EnginesUsed: got %d, want 3", len(resp.Metadata.EnginesUsed))
	}

	// Should have 4 results (5 total - 1 duplicate removed)
	if len(resp.Results) != 4 {
		t.Errorf("Results count: got %d, want 4 (1 dedup)", len(resp.Results))
	}

	// All results should have RRF scores > 0
	for i, r := range resp.Results {
		if r.Score <= 0 {
			t.Errorf("Result[%d] score should be > 0, got %f", i, r.Score)
		}
	}

	// Results should be sorted by RRF score (descending)
	for i := 1; i < len(resp.Results); i++ {
		if resp.Results[i].Score > resp.Results[i-1].Score {
			t.Errorf("Results not sorted by RRF score: [%d]=%f > [%d]=%f",
				i, resp.Results[i].Score, i-1, resp.Results[i-1].Score)
		}
	}

	// Context string should be non-empty
	if resp.Context == "" {
		t.Error("Context should not be empty")
	}

	// Latency should be recorded
	if resp.Metadata.LatencyMs <= 0 {
		t.Error("LatencyMs should be > 0")
	}
}

func TestRecallService_ScopeFiltering(t *testing.T) {
	configs := []model.EngineConfig{
		{Name: "cognee", Enabled: true},
		{Name: "memobase", Enabled: true},
		{Name: "vnp-event", Enabled: true},
	}
	svc := usecase.NewRecallService(configs)

	cogneeHit := false
	memobaseHit := false
	eventHit := false

	svc.RegisterEngine("cognee", &mockEngineClient{
		name: "cognee",
		results: []model.RecallResult{
			{Content: "semantic result from cognee", Score: 0.9, Source: "cognee"},
		},
	})
	svc.RegisterEngine("memobase", &mockEngineClient{
		name: "memobase",
		results: []model.RecallResult{
			{Content: "profile result from memobase", Score: 0.8, Source: "memobase"},
		},
	})
	svc.RegisterEngine("vnp-event", &mockEngineClient{
		name: "vnp-event",
		results: []model.RecallResult{
			{Content: "event result from vnp-event", Score: 0.7, Source: "vnp-event"},
		},
	})

	// Test semantic scope — should only use cognee (not memobase or vnp-event)
	resp, _ := svc.Recall(context.Background(), &model.RecallRequest{
		Scope:          model.ScopeSemantic,
		RerankStrategy: model.RerankRRF,
		TokenBudget:    4096,
	})

	for _, r := range resp.Results {
		switch r.Source {
		case "cognee":
			cogneeHit = true
		case "memobase":
			memobaseHit = true
		case "vnp-event":
			eventHit = true
		}
	}

	if !cogneeHit {
		t.Error("semantic scope should include cognee")
	}
	if memobaseHit {
		t.Error("semantic scope should NOT include memobase")
	}
	if eventHit {
		t.Error("semantic scope should NOT include vnp-event")
	}
}

func TestRecallService_TokenBudgeting(t *testing.T) {
	svc := usecase.NewRecallService(nil)
	svc.RegisterEngine("test", &mockEngineClient{
		name: "test",
		results: func() []model.RecallResult {
			var r []model.RecallResult
			for i := 0; i < 100; i++ {
				// Each result ~50 chars = ~12 tokens
				r = append(r, model.RecallResult{
					Content: "This is a moderately sized test result number " + string(rune('A'+i%26)),
					Score:   float64(100-i) / 100.0,
					Source:  "test",
				})
			}
			return r
		}(),
	})

	resp, _ := svc.Recall(context.Background(), &model.RecallRequest{
		Scope:          model.ScopeAll,
		RerankStrategy: model.RerankRRF,
		TokenBudget:    50, // Very small budget — should truncate
	})

	if resp.Metadata.TokensUsed > 50 {
		t.Errorf("TokensUsed %d exceeds budget 50", resp.Metadata.TokensUsed)
	}
	if len(resp.Results) >= 100 {
		t.Error("Token budgeting should have truncated results")
	}
}

func TestRecallService_EngineFailure_GracefulDegradation(t *testing.T) {
	svc := usecase.NewRecallService(nil)
	svc.RegisterEngine("good", &mockEngineClient{
		name: "good",
		results: []model.RecallResult{
			{Content: "good result", Score: 0.9, Source: "good"},
		},
	})
	svc.RegisterEngine("bad", &mockEngineClient{
		name: "bad",
		err:  context.DeadlineExceeded,
	})

	resp, err := svc.Recall(context.Background(), &model.RecallRequest{
		Scope:          model.ScopeAll,
		RerankStrategy: model.RerankRRF,
		TokenBudget:    4096,
	})

	if err != nil {
		t.Fatalf("Recall should not error when one engine fails: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Should have 1 result from good engine, got %d", len(resp.Results))
	}
	if len(resp.Metadata.EnginesUsed) != 1 {
		t.Errorf("Should report 1 engine used, got %d", len(resp.Metadata.EnginesUsed))
	}
}
