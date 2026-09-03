package usecase

import (
    "context"
    "time"

    pkgsearch "github.com/vnp-memory/pkg/search"
    "vnp-memory/services/search-service/internal/domain"
    "vnp-memory/services/search-service/internal/usecase/port"
)

type SmartSearchUseCase struct {
    bm25     *pkgsearch.BM25Index
    vector   *pkgsearch.VectorIndex
    embedder port.IEmbedder
    weights  pkgsearch.ScoreWeights
    obsStore port.IObservationStore
}

type SmartSearchRequest struct {
    Query     string
    TenantID  string
    Project   string
    SessionFilter string
    Limit     int
    Weights   pkgsearch.ScoreWeights
}

type SmartSearchResponse struct {
    Results []domain.SearchResult
    TookMs  int64
}

func (uc *SmartSearchUseCase) Execute(ctx context.Context, req SmartSearchRequest) (*SmartSearchResponse, error) {
    start := time.Now()
    if req.Limit == 0 { req.Limit = 10 }

    // BM25 search (always)
    bm25Results := uc.bm25.Search(req.Query, req.Limit*3)

    // Vector search (if embedder configured)
    var vectorResults []pkgsearch.VectorResult
    if uc.embedder != nil {
        if vec, err := uc.embedder.Embed(ctx, req.Query); err == nil && vec != nil {
            vectorResults = uc.vector.Search(vec, req.Limit*3)
        }
    }

    // RRF fusion
    weights := req.Weights
    if weights.BM25 == 0 && weights.Vector == 0 { weights = uc.weights }
    fused := pkgsearch.RRFFuse(bm25Results, vectorResults, nil, weights, req.Limit)

    // Enrich with observation data from PostgreSQL
    results, _ := uc.enrichResults(ctx, fused, req)

    return &SmartSearchResponse{
        Results: results,
        TookMs:  time.Since(start).Milliseconds(),
    }, nil
}

func (uc *SmartSearchUseCase) enrichResults(ctx context.Context, fused []pkgsearch.HybridResult, req SmartSearchRequest) ([]domain.SearchResult, error) {
    docIDs := make([]string, len(fused))
    for i, r := range fused { docIDs[i] = r.DocID }

    obsMap, _ := uc.obsStore.GetByIDs(ctx, docIDs)

    results := make([]domain.SearchResult, 0, len(fused))
    for _, r := range fused {
        sr := domain.SearchResult{
            DocID:         r.DocID,
            SessionID:     r.SessionID,
            CombinedScore: r.CombinedScore,
            BM25Score:     r.BM25Score,
            VectorScore:   r.VectorScore,
        }
        if obs, ok := obsMap[r.DocID]; ok {
            sr.ObsType   = obs.ObsType
            sr.Title     = obs.Title
            sr.Narrative = obs.Narrative
            sr.Facts     = obs.Facts
            sr.Concepts  = obs.Concepts
        }
        results = append(results, sr)
    }
    return results, nil
}
