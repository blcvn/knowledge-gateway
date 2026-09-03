package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"vnp-memory/services/cognee-search/internal/domain"
)

// merge implements Reciprocal Rank Fusion (RRF) for deduplicating and re-scoring search results.
func merge(results []domain.SearchResult, topK int) []domain.SearchResult {
	scores := make(map[string]float64)
	resultMap := make(map[string]domain.SearchResult)
	k := 60.0 // RRF constant

	// Group results by strategy to calculate rank within each strategy
	strategyGroups := make(map[domain.SearchStrategy][]domain.SearchResult)
	for _, res := range results {
		strategyGroups[res.Strategy] = append(strategyGroups[res.Strategy], res)
	}

	for _, group := range strategyGroups {
		// Sort within strategy group by original score descending
		sort.SliceStable(group, func(i, j int) bool {
			return group[i].Score > group[j].Score
		})

		for rank, res := range group {
			// Create a unique hash for deduplication based on content
			hash := sha256.Sum256([]byte(res.Content))
			contentHash := hex.EncodeToString(hash[:])

			// RRF calculation: 1 / (k + rank)
			rrfScore := 1.0 / (k + float64(rank+1))

			if existing, ok := resultMap[contentHash]; ok {
				// Accumulate RRF score if content already exists
				scores[contentHash] += rrfScore
				// Update existing metadata if necessary, but keep the primary content
				existing.Metadata["merged_strategies"] = append(existing.Metadata["merged_strategies"].([]string), string(res.Strategy))
				resultMap[contentHash] = existing
			} else {
				scores[contentHash] = rrfScore
				res.Metadata = map[string]interface{}{
					"merged_strategies": []string{string(res.Strategy)},
				}
				resultMap[contentHash] = res
			}
		}
	}

	// Reconstruct the final list
	var merged []domain.SearchResult
	for contentHash, score := range scores {
		res := resultMap[contentHash]
		res.Score = score // update with fused score
		merged = append(merged, res)
	}

	// Sort globally by the new fused score descending
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	if len(merged) > topK {
		merged = merged[:topK]
	}

	return merged
}
