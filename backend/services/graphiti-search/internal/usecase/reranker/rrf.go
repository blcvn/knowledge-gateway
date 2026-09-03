package reranker

import (
	"context"
	"sort"

	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
)

type RRFReranker struct {
	kValue int
}

func NewRRFReranker(kValue int) usecase.Reranker {
	if kValue <= 0 {
		kValue = 60
	}
	return &RRFReranker{kValue: kValue}
}

func (r *RRFReranker) Type() domain.RerankerType {
	return domain.RerankerRRF
}

func (r *RRFReranker) Rerank(ctx context.Context, query string, results []domain.SearchResult) ([]domain.RankedResult, error) {
	if len(results) == 0 {
		return nil, nil
	}
	
	methodResults := make(map[domain.SearchMethod][]domain.SearchResult)
	for _, res := range results {
		methodResults[res.MethodUsed] = append(methodResults[res.MethodUsed], res)
	}

	for _, list := range methodResults {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Score > list[j].Score
		})
	}

	rrfScores := make(map[string]float64)
	resultMap := make(map[string]domain.SearchResult)

	for _, list := range methodResults {
		for rank, res := range list {
			rrfScores[res.EntityID] += 1.0 / float64(r.kValue+rank+1)
			resultMap[res.EntityID] = res
		}
	}

	ranked := make([]domain.RankedResult, 0, len(rrfScores))
	for id, score := range rrfScores {
		orig := resultMap[id]
		ranked = append(ranked, domain.RankedResult{
			EntityID: id,
			Score:    score,
			Content:  orig.Content,
			Metadata: orig.Metadata,
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	return ranked, nil
}
