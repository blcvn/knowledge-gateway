package search

import (
    "context"
    "time"

    pkg_search "github.com/vnp-memory/pkg/search"
)

// SmartSearchRequest is the input for the smart hybrid search.
type SmartSearchRequest struct {
    Query     string
    TenantID  string
    SessionID string
    AgentID   string
    Limit     int
    Weights   pkg_search.ScoreWeights
}

// SmartSearchResponse is the output of the smart hybrid search.
type SmartSearchResponse struct {
    Results []pkg_search.HybridResult
    TookMs  int64
}

// SmartSearch orchestrates BM25 + vector + RRF search.
type SmartSearch struct {
    bm25     *pkg_search.BM25Index
    vector   *pkg_search.VectorIndex
    embedder IEmbedder
    weights  pkg_search.ScoreWeights
}

// IEmbedder is a port for generating embeddings.
type IEmbedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}

func NewSmartSearch(bm25 *pkg_search.BM25Index, vector *pkg_search.VectorIndex, embedder IEmbedder, weights pkg_search.ScoreWeights) *SmartSearch {
    return &SmartSearch{bm25: bm25, vector: vector, embedder: embedder, weights: weights}
}

func (s *SmartSearch) Execute(ctx context.Context, req SmartSearchRequest) (*SmartSearchResponse, error) {
    start := time.Now()

    limit := req.Limit
    if limit == 0 {
        limit = 10
    }

    bm25Results := s.bm25.Search(req.Query, limit*3)

    var vectorResults []pkg_search.VectorResult
    if s.embedder != nil {
        vec, err := s.embedder.Embed(ctx, req.Query)
        if err == nil && vec != nil {
            vectorResults = s.vector.Search(vec, limit*3)
        }
    }

    weights := req.Weights
    if weights.BM25 == 0 {
        weights = s.weights
    }

    hybrid := pkg_search.RRFFuse(bm25Results, vectorResults, nil, weights, limit)

    return &SmartSearchResponse{
        Results: hybrid,
        TookMs:  time.Since(start).Milliseconds(),
    }, nil
}
