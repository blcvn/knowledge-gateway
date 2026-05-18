// Package usecase implements unified search orchestration for vnp-search-hub.
package usecase

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// SearchRequest represents a unified cross-engine search request.
type SearchRequest struct {
	Query     string            `json:"query"`
	Engines   []string          `json:"engines,omitempty"` // empty = all
	Filters   map[string]string `json:"filters,omitempty"`
	Limit     int               `json:"limit"`
	Offset    int               `json:"offset"`
	Reranking string            `json:"reranking"` // semantic | temporal | hybrid | none
}

// SearchResult represents a single search result from any engine.
type SearchResult struct {
	ID       string         `json:"id"`
	Engine   string         `json:"engine"`
	Type     string         `json:"type"`
	Content  string         `json:"content"`
	Score    float64        `json:"score"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SearchResponse represents the unified search response.
type SearchResponse struct {
	Results   []SearchResult       `json:"results"`
	Total     int                  `json:"total"`
	Facets    SearchFacets         `json:"facets"`
	LatencyMs int64                `json:"latency_ms"`
}

// SearchFacets provides aggregation counts.
type SearchFacets struct {
	ByEngine map[string]int `json:"by_engine"`
	ByType   map[string]int `json:"by_type"`
}

// EngineSearcher interface that each engine client implements.
type EngineSearcher interface {
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	Name() string
}

// SearchOrchestrator coordinates fan-out search across multiple engines.
type SearchOrchestrator struct {
	engines map[string]EngineSearcher
	logger  *slog.Logger
}

// NewSearchOrchestrator creates a search orchestrator.
func NewSearchOrchestrator(engines map[string]EngineSearcher, logger *slog.Logger) *SearchOrchestrator {
	return &SearchOrchestrator{engines: engines, logger: logger}
}

// Search executes fan-out search across selected engines.
func (o *SearchOrchestrator) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	start := time.Now()

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	// Select engines
	targetEngines := req.Engines
	if len(targetEngines) == 0 {
		targetEngines = make([]string, 0, len(o.engines))
		for name := range o.engines {
			targetEngines = append(targetEngines, name)
		}
	}

	// Fan-out with 5s timeout per engine
	searchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type engineResults struct {
		engine  string
		results []SearchResult
		err     error
	}

	ch := make(chan engineResults, len(targetEngines))
	var wg sync.WaitGroup

	for _, name := range targetEngines {
		eng, ok := o.engines[name]
		if !ok {
			o.logger.Warn("unknown engine requested", "engine", name)
			continue
		}
		wg.Add(1)
		go func(e EngineSearcher) {
			defer wg.Done()
			results, err := e.Search(searchCtx, req.Query, req.Limit)
			ch <- engineResults{engine: e.Name(), results: results, err: err}
		}(eng)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Merge results
	var allResults []SearchResult
	facets := SearchFacets{
		ByEngine: make(map[string]int),
		ByType:   make(map[string]int),
	}

	for er := range ch {
		if er.err != nil {
			o.logger.Warn("engine search failed", "engine", er.engine, "error", er.err)
			continue // Graceful degradation — partial results
		}
		for _, r := range er.results {
			allResults = append(allResults, r)
			facets.ByEngine[r.Engine]++
			facets.ByType[r.Type]++
		}
	}

	// Sort by score descending
	sortByScore(allResults)

	// Apply offset + limit
	total := len(allResults)
	if req.Offset < len(allResults) {
		allResults = allResults[req.Offset:]
	}
	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}

	return &SearchResponse{
		Results:   allResults,
		Total:     total,
		Facets:    facets,
		LatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

// sortByScore sorts results by score descending.
func sortByScore(results []SearchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}
