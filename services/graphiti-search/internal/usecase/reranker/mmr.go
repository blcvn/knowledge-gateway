package reranker

import (
	"context"
	"strings"

	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
)

type MMRReranker struct {
	lambda float64
}

func NewMMRReranker(lambda float64) usecase.Reranker {
	if lambda <= 0 || lambda > 1 {
		lambda = 0.7
	}
	return &MMRReranker{lambda: lambda}
}

func (r *MMRReranker) Type() domain.RerankerType {
	return domain.RerankerMMR
}

func (r *MMRReranker) Rerank(ctx context.Context, query string, results []domain.SearchResult) ([]domain.RankedResult, error) {
	if len(results) == 0 {
		return nil, nil
	}

	var selected []domain.SearchResult
	var unselected []domain.SearchResult
	unselected = append(unselected, results...)

	for len(unselected) > 0 {
		bestIdx := -1
		bestScore := -1.0

		for i, cand := range unselected {
			rel := cand.Score
			maxSim := 0.0
			for _, sel := range selected {
				sim := jaccardSim(cand.Content, sel.Content)
				if sim > maxSim {
					maxSim = sim
				}
			}

			score := r.lambda*rel - (1.0-r.lambda)*maxSim
			if score > bestScore || bestIdx == -1 {
				bestScore = score
				bestIdx = i
			}
		}

		selected = append(selected, unselected[bestIdx])
		unselected = append(unselected[:bestIdx], unselected[bestIdx+1:]...)
	}

	ranked := make([]domain.RankedResult, len(selected))
	for i, res := range selected {
		ranked[i] = domain.RankedResult{
			EntityID: res.EntityID,
			Score:    res.Score,
			Rank:     i + 1,
			Content:  res.Content,
			Metadata: res.Metadata,
		}
	}
	return ranked, nil
}

func jaccardSim(s1, s2 string) float64 {
	w1 := strings.Fields(strings.ToLower(s1))
	w2 := strings.Fields(strings.ToLower(s2))
	if len(w1) == 0 && len(w2) == 0 {
		return 1.0
	}
	m1 := make(map[string]bool)
	for _, w := range w1 {
		m1[w] = true
	}
	intersection := 0
	for _, w := range w2 {
		if m1[w] {
			intersection++
			delete(m1, w)
		}
	}
	union := len(w1) + len(w2) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
