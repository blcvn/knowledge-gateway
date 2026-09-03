package usecase

import (
	"context"
	"math"

	"vnp-memory/ov-search/internal/domain/model"
	"vnp-memory/ov-search/internal/domain/repository"
	"vnp-memory/ov-search/internal/usecase/dto"
	"vnp-memory/ov-search/internal/usecase/port"
)

type hierarchicalSearch struct {
	vectorRepo  repository.VectorRepository
	hotnessRepo repository.HotnessRepository
	embedder    port.EmbedderPort
	reranker    port.RerankPort
	fileReader  port.FileReaderPort
	
	propagationFactor float64
	convergenceThresh float64
}

func NewHierarchicalSearch(
	vr repository.VectorRepository,
	hr repository.HotnessRepository,
	emb port.EmbedderPort,
	rr port.RerankPort,
	fr port.FileReaderPort,
	propFactor float64,
	convThresh float64,
) port.SearchUseCase {
	return &hierarchicalSearch{
		vectorRepo:        vr,
		hotnessRepo:       hr,
		embedder:          emb,
		reranker:          rr,
		fileReader:        fr,
		propagationFactor: propFactor,
		convergenceThresh: convThresh,
	}
}

func (s *hierarchicalSearch) Search(ctx context.Context, req dto.SearchRequest) (*dto.SearchResponse, error) {
	// 1. Query Intent Analysis (simplified)
	_ = s.analyzeIntent(req.Query)

	// 2. Dense + Sparse Vector Search
	dense, sparse, err := s.embedder.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		return nil, err
	}

	results, err := s.vectorRepo.Search(ctx, dense, sparse, req.AccountID, req.MaxResults*2) // fetch more for reranking
	if err != nil {
		return nil, err
	}

	// 3. Hierarchical Score Propagation (Simplified for now - assumes tree loaded in memory or recursive fetch)
	// In production, this would build a tree of `results` and propagate scores up.
	s.propagateScores(results)

	// 4. Hotness Score Integration
	if req.EnableHotness {
		paths := make([]string, len(results))
		for i, r := range results {
			paths[i] = r.Path
		}
		
		hotnessMap, _ := s.hotnessRepo.GetMulti(ctx, req.AccountID, paths)
		for i := range results {
			if hs, ok := hotnessMap[results[i].Path]; ok {
				results[i].HotnessScore = model.Score(hs.ComputedHotness)
				// Combine semantic and hotness. Example logic:
				results[i].FinalScore = model.Score(float64(results[i].SemanticScore) * (1.0 + float64(results[i].HotnessScore)))
			} else {
				results[i].FinalScore = results[i].SemanticScore
			}
		}
	} else {
		for i := range results {
			results[i].FinalScore = results[i].SemanticScore
		}
	}

	// 5. Convergence Detection
	// Check if top N results have stabilized (delta < convergenceThresh)
	// (Skipped full loop implementation for brevity, assuming top results are stable)

	// 6. Cross-Encoder Reranking
	reranked, err := s.reranker.Rerank(ctx, req.Query, results, req.RerankStrategy)
	if err != nil {
		reranked = results // fallback
	}

	// Truncate to max results
	if len(reranked) > req.MaxResults {
		reranked = reranked[:req.MaxResults]
	}

	// 7. Tiered Context Loading
	for i := range reranked {
		content, _ := s.fileReader.ReadContext(ctx, reranked[i].Path, req.ContextLevel)
		reranked[i].MatchedContext = model.MatchedContext{
			Content:    content,
			DepthLevel: req.ContextLevel,
		}
	}

	return &dto.SearchResponse{Results: reranked}, nil
}

func (s *hierarchicalSearch) RetrieveContext(ctx context.Context, req dto.ContextRequest) (*dto.ContextResponse, error) {
	content, err := s.fileReader.ReadContext(ctx, req.Path, req.ContextLevel)
	if err != nil {
		return nil, err
	}
	return &dto.ContextResponse{Content: content}, nil
}

func (s *hierarchicalSearch) analyzeIntent(query string) []model.TypedQuery {
	// Simplified intent analysis
	return []model.TypedQuery{{RawQuery: query, Type: model.ContextTypeCode}}
}

func (s *hierarchicalSearch) propagateScores(results []model.SearchResult) {
	// Simplified score propagation
	// In a real implementation, find parent directories and update their scores = max(children) * propagationFactor
	for i := range results {
		results[i].SemanticScore = model.Score(math.Min(1.0, float64(results[i].SemanticScore)))
	}
}
