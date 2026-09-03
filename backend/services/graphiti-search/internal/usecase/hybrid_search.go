package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"vnp-memory/services/graphiti-search/internal/domain"
)

type HybridSearchUseCase struct {
	storeClient StoreSearchClient
	embedder    EmbedderClient
	cache       CacheRepo
	rerankers   map[domain.RerankerType]Reranker
	cacheTTL    time.Duration
}

func NewHybridSearchUseCase(
	storeClient StoreSearchClient,
	embedder EmbedderClient,
	cache CacheRepo,
	rerankers []Reranker,
	cacheTTL time.Duration,
) *HybridSearchUseCase {
	rm := make(map[domain.RerankerType]Reranker)
	for _, r := range rerankers {
		rm[r.Type()] = r
	}
	return &HybridSearchUseCase{
		storeClient: storeClient,
		embedder:    embedder,
		cache:       cache,
		rerankers:   rm,
		cacheTTL:    cacheTTL,
	}
}

func (uc *HybridSearchUseCase) Execute(ctx context.Context, query domain.SearchQuery) ([]domain.RankedResult, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	cacheKey := uc.generateCacheKey(query)
	if cached, err := uc.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	var queryVector []float32
	var err error
	if uc.hasMethod(query.Methods, domain.MethodCosine) {
		queryVector, err = uc.embedder.EmbedQuery(ctx, query.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to embed query: %w", err)
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	allResults := make([]domain.SearchResult, 0)
	errs := make([]error, 0)

	for _, method := range query.Methods {
		wg.Add(1)
		go func(m domain.SearchMethod) {
			defer wg.Done()
			var res []domain.SearchResult
			var e error

			switch m {
			case domain.MethodCosine:
				res, e = uc.storeClient.CosineSimilaritySearch(ctx, queryVector, query.Limit)
			case domain.MethodBM25:
				res, e = uc.storeClient.FulltextSearch(ctx, query.Query, query.Limit)
			case domain.MethodBFS:
				nodes, e2 := uc.storeClient.FulltextSearch(ctx, query.Query, 1)
				if e2 == nil && len(nodes) > 0 {
					res, e = uc.storeClient.BFSSearch(ctx, nodes[0].EntityID, 2)
				}
			}

			mu.Lock()
			if e != nil {
				errs = append(errs, e)
			} else {
				allResults = append(allResults, res...)
			}
			mu.Unlock()
		}(method)
	}

	wg.Wait()

	if len(allResults) == 0 {
		return nil, domain.ErrNoResults
	}

	uniqueResults := uc.deduplicate(allResults)

	rankedResults := uc.applyReranking(ctx, query.Query, uniqueResults, query.Rerankers)

	if len(rankedResults) > query.Limit {
		rankedResults = rankedResults[:query.Limit]
	}

	_ = uc.cache.Set(ctx, cacheKey, rankedResults, uc.cacheTTL)

	return rankedResults, nil
}

func (uc *HybridSearchUseCase) generateCacheKey(query domain.SearchQuery) string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%+v", query)))
	return fmt.Sprintf("search:%s:%s", query.GroupID, hex.EncodeToString(hash.Sum(nil)))
}

func (uc *HybridSearchUseCase) hasMethod(methods []domain.SearchMethod, target domain.SearchMethod) bool {
	for _, m := range methods {
		if m == target {
			return true
		}
	}
	return false
}

func (uc *HybridSearchUseCase) deduplicate(results []domain.SearchResult) []domain.SearchResult {
	seen := make(map[string]bool)
	unique := make([]domain.SearchResult, 0, len(results))
	for _, r := range results {
		if !seen[r.EntityID] {
			seen[r.EntityID] = true
			unique = append(unique, r)
		}
	}
	return unique
}

func (uc *HybridSearchUseCase) applyReranking(ctx context.Context, query string, results []domain.SearchResult, rerankers []domain.RerankerType) []domain.RankedResult {
	if len(results) == 0 {
		return nil
	}

	if len(rerankers) == 0 {
		ranked := make([]domain.RankedResult, len(results))
		for i, r := range results {
			ranked[i] = domain.RankedResult{
				EntityID: r.EntityID,
				Score:    r.Score,
				Rank:     i + 1,
				Content:  r.Content,
				Metadata: r.Metadata,
			}
		}
		return ranked
	}

	currentResults := results
	var finalRanked []domain.RankedResult

	for i, rt := range rerankers {
		if reranker, ok := uc.rerankers[rt]; ok {
			ranked, err := reranker.Rerank(ctx, query, currentResults)
			if err != nil {
				continue // Skip failed reranker
			}
			finalRanked = ranked
			
			// If not the last reranker, convert RankedResult back to SearchResult for the next one
			if i < len(rerankers)-1 {
				nextInput := make([]domain.SearchResult, len(ranked))
				for j, r := range ranked {
					nextInput[j] = domain.SearchResult{
						EntityID: r.EntityID,
						Score:    r.Score,
						Content:  r.Content,
						Metadata: r.Metadata,
					}
				}
				currentResults = nextInput
			}
		}
	}

	if finalRanked == nil {
		// Fallback
		return uc.applyReranking(ctx, query, results, nil)
	}

	return finalRanked
}
