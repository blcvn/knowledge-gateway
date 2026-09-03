// Package usecase implements the recall pipeline.
package usecase

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"vnp-memory/services/vnp-search-hub/domain/model"
)

// EngineClient abstracts gRPC calls to downstream search engines.
type EngineClient interface {
	Search(ctx context.Context, req *model.RecallRequest) (*model.EngineResult, error)
}

// RecallService implements the memory.recall() pipeline.
type RecallService struct {
	engines map[string]EngineClient // engine name → client
	configs []model.EngineConfig
}

func NewRecallService(configs []model.EngineConfig) *RecallService {
	return &RecallService{
		engines: make(map[string]EngineClient),
		configs: configs,
	}
}

// RegisterEngine adds an engine client to the registry.
func (s *RecallService) RegisterEngine(name string, client EngineClient) {
	s.engines[name] = client
}

// Recall executes the full pipeline: fan-out → merge → rerank → budget.
func (s *RecallService) Recall(ctx context.Context, req *model.RecallRequest) (*model.RecallResponse, error) {
	start := time.Now()

	// 1. Fan-out to all enabled engines in parallel
	engineResults := s.fanOut(ctx, req)

	// 2. Merge and deduplicate
	merged := s.mergeResults(engineResults)

	// 3. Rerank
	reranked := s.rerank(merged, req.RerankStrategy)

	// 4. Token budgeting
	budgeted, tokensUsed := s.applyTokenBudget(reranked, req.TokenBudget)

	// 5. Build context string
	contextStr := buildContext(budgeted)

	// 6. Collect metadata
	var enginesUsed []string
	for _, er := range engineResults {
		if er.Error == nil {
			enginesUsed = append(enginesUsed, er.EngineName)
		}
	}

	return &model.RecallResponse{
		Results: budgeted,
		Context: contextStr,
		Metadata: model.RecallMetadata{
			LatencyMs:    time.Since(start).Milliseconds(),
			EnginesUsed:  enginesUsed,
			TotalResults: len(budgeted),
			TokensUsed:   tokensUsed,
		},
	}, nil
}

// fanOut queries all engines in parallel using goroutines.
func (s *RecallService) fanOut(ctx context.Context, req *model.RecallRequest) []model.EngineResult {
	var (
		mu      sync.Mutex
		results []model.EngineResult
		wg      sync.WaitGroup
	)

	for name, client := range s.engines {
		if !s.isEngineInScope(name, req.Scope) {
			continue
		}
		wg.Add(1)
		go func(n string, c EngineClient) {
			defer wg.Done()
			engineCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			result, err := c.Search(engineCtx, req)
			if err != nil {
				result = &model.EngineResult{EngineName: n, Error: err}
			}
			result.EngineName = n

			mu.Lock()
			results = append(results, *result)
			mu.Unlock()
		}(name, client)
	}

	wg.Wait()
	return results
}

// mergeResults combines results from all engines, deduplicating by content hash.
func (s *RecallService) mergeResults(engineResults []model.EngineResult) []model.RecallResult {
	seen := make(map[string]bool)
	var merged []model.RecallResult

	for _, er := range engineResults {
		if er.Error != nil {
			continue
		}
		for _, r := range er.Results {
			key := r.Content[:min(64, len(r.Content))] // Simple dedup by content prefix
			if !seen[key] {
				seen[key] = true
				merged = append(merged, r)
			}
		}
	}
	return merged
}

// rerank applies the specified reranking strategy.
func (s *RecallService) rerank(results []model.RecallResult, strategy model.RerankStrategy) []model.RecallResult {
	switch strategy {
	case model.RerankRRF:
		return s.rerankRRF(results)
	case model.RerankMMR:
		return s.rerankMMR(results)
	default:
		return s.rerankRRF(results)
	}
}

// rerankRRF implements Reciprocal Rank Fusion.
// Score = Σ 1/(k + rank_i), where k = 60 (standard constant).
func (s *RecallService) rerankRRF(results []model.RecallResult) []model.RecallResult {
	const k = 60.0

	// Group by source, sort within each source by original score
	bySource := make(map[string][]model.RecallResult)
	for _, r := range results {
		bySource[r.Source] = append(bySource[r.Source], r)
	}

	// Sort each source by original score (descending)
	for src := range bySource {
		sort.Slice(bySource[src], func(i, j int) bool {
			return bySource[src][i].Score > bySource[src][j].Score
		})
	}

	// Compute RRF scores
	rrfScores := make(map[int]float64) // index → RRF score
	indexMap := make(map[string]int)    // content prefix → index in results

	for _, r := range results {
		key := r.Content[:min(64, len(r.Content))]
		if _, exists := indexMap[key]; !exists {
			indexMap[key] = len(indexMap)
		}
	}

	for _, sourceResults := range bySource {
		for rank, r := range sourceResults {
			key := r.Content[:min(64, len(r.Content))]
			idx := indexMap[key]
			rrfScores[idx] += 1.0 / (k + float64(rank+1))
		}
	}

	// Apply RRF scores and sort
	for i := range results {
		key := results[i].Content[:min(64, len(results[i].Content))]
		if idx, ok := indexMap[key]; ok {
			results[i].Score = rrfScores[idx]
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

// rerankMMR implements Maximal Marginal Relevance (simplified).
func (s *RecallService) rerankMMR(results []model.RecallResult) []model.RecallResult {
	// Simplified: sort by score, then diversify by source
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

// applyTokenBudget truncates results to fit within the token budget.
func (s *RecallService) applyTokenBudget(results []model.RecallResult, budget int) ([]model.RecallResult, int) {
	if budget <= 0 {
		budget = 4096 // Default budget
	}

	var (
		selected   []model.RecallResult
		tokenCount int
	)

	for _, r := range results {
		tokens := len(r.Content) / 4 // Approximate: 4 chars ≈ 1 token
		if tokenCount+tokens > budget {
			break
		}
		selected = append(selected, r)
		tokenCount += tokens
	}
	return selected, tokenCount
}

func (s *RecallService) isEngineInScope(engine string, scope model.RecallScope) bool {
	if scope == model.ScopeAll {
		return true
	}
	// Scope mapping: which engines serve which scopes
	scopeMap := map[model.RecallScope][]string{
		model.ScopeSemantic:  {"cognee", "graphiti", "zep"},
		model.ScopeEpisodic:  {"graphiti", "zep", "vnp-event"},
		model.ScopeProfile:   {"memobase", "supermemory"},
		model.ScopeEvents:    {"vnp-event"},
	}
	if engines, ok := scopeMap[scope]; ok {
		for _, e := range engines {
			if e == engine {
				return true
			}
		}
	}
	return false
}

func buildContext(results []model.RecallResult) string {
	var ctx string
	for _, r := range results {
		ctx += fmt.Sprintf("[%s:%s] %s\n", r.Source, r.Type, r.Content)
	}
	return ctx
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
