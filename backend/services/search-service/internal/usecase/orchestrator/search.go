// Package orchestrator implements the cross-engine search orchestrator.
//
// Implements: SearchOrchestrator (fan-out + RRF reranking) and RAG pipeline.
// Re-uses patterns from existing vnp-search-hub/usecase/search_orchestrator.go.
// (MERGE-P2-T4)
package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"vnp-memory/services/search-service/internal/domain/search"
)

// SearchOrchestrator fans out search to multiple engine clients.
type SearchOrchestrator struct {
	kg      KGClientInterface
	memory  MemoryClientInterface
	storage StorageClientInterface
}

// NewSearchOrchestrator creates a SearchOrchestrator.
func NewSearchOrchestrator(kg KGClientInterface, mem MemoryClientInterface, store StorageClientInterface) *SearchOrchestrator {
	return &SearchOrchestrator{kg: kg, memory: mem, storage: store}
}

type engineResult struct {
	engine string
	items  []*search.Item
	err    error
}

// Search executes concurrent fan-out search and merges results.
func (o *SearchOrchestrator) Search(ctx context.Context, q *search.Query) (*search.Result, error) {
	start := time.Now()
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	// Default: search all engines
	engines := q.Engines
	if len(engines) == 0 {
		engines = []string{"graphiti", "cognee", "memobase", "sm", "storage"}
	}

	searchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	ch := make(chan engineResult, len(engines))
	var wg sync.WaitGroup

	for _, eng := range engines {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			items, err := o.searchEngine(searchCtx, name, q)
			ch <- engineResult{engine: name, items: items, err: err}
		}(eng)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect per-engine results
	engineResults := make(map[string][]*search.Item)
	enginesUsed := make([]string, 0)
	for er := range ch {
		if er.err != nil {
			// Graceful degradation: skip failed engines
			continue
		}
		if len(er.items) > 0 {
			engineResults[er.engine] = er.items
			enginesUsed = append(enginesUsed, er.engine)
		}
	}

	// Rerank
	var allItems []*search.Item
	switch q.RankStrategy {
	case "rrf":
		allItems = rrfMerge(engineResults, 60)
	default:
		// Simple score merge
		for _, items := range engineResults {
			allItems = append(allItems, items...)
		}
		sort.Slice(allItems, func(i, j int) bool {
			return allItems[i].Score > allItems[j].Score
		})
	}

	// Apply offset + limit
	total := len(allItems)
	if q.Offset < len(allItems) {
		allItems = allItems[q.Offset:]
	}
	if len(allItems) > q.Limit {
		allItems = allItems[:q.Limit]
	}

	return &search.Result{
		Items:   allItems,
		Total:   total,
		Took:    time.Since(start),
		Engines: enginesUsed,
	}, nil
}

func (o *SearchOrchestrator) searchEngine(ctx context.Context, engine string, q *search.Query) ([]*search.Item, error) {
	switch engine {
	case "graphiti":
		if o.kg == nil {
			return nil, nil
		}
		return o.kg.GraphitiSearch(ctx, q)
	case "cognee":
		if o.kg == nil {
			return nil, nil
		}
		return o.kg.CogneeSearch(ctx, q)
	case "memobase":
		if o.memory == nil {
			return nil, nil
		}
		return o.memory.MemobaseSearch(ctx, q)
	case "sm":
		if o.memory == nil {
			return nil, nil
		}
		return o.memory.SMSearch(ctx, q)
	case "storage":
		if o.storage == nil {
			return nil, nil
		}
		return o.storage.FileSearch(ctx, q)
	default:
		return nil, fmt.Errorf("unknown engine: %s", engine)
	}
}

// RAG builds retrieval-augmented context from top search results.
func (o *SearchOrchestrator) RAG(ctx context.Context, query, tenantID string) (*search.RAGResponse, error) {
	q := &search.Query{
		Query:        query,
		TenantID:     tenantID,
		Engines:      []string{"graphiti", "memobase", "sm"},
		Limit:        10,
		RankStrategy: "rrf",
	}
	result, err := o.Search(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("rag search: %w", err)
	}

	topN := result.Items
	if len(topN) > 5 {
		topN = topN[:5]
	}

	var parts []string
	for _, item := range topN {
		parts = append(parts, item.Content)
	}
	contextText := strings.Join(parts, "\n\n")

	return &search.RAGResponse{
		Context: contextText,
		Sources: topN,
		Tokens:  len(contextText) / 4,
	}, nil
}

// rrfMerge merges engine results using Reciprocal Rank Fusion.
func rrfMerge(engineResults map[string][]*search.Item, k int) []*search.Item {
	scores := make(map[string]float64)
	byID := make(map[string]*search.Item)

	for _, items := range engineResults {
		for rank, item := range items {
			scores[item.ID] += 1.0 / float64(k+rank+1)
			if _, exists := byID[item.ID]; !exists {
				byID[item.ID] = item
			}
		}
	}

	merged := make([]*search.Item, 0, len(byID))
	for id, item := range byID {
		item.Score = scores[id]
		merged = append(merged, item)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	return merged
}
