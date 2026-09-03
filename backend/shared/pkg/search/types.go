package search

type BM25Result struct {
    DocID     string
    SessionID string
    AgentID   string
    Score     float64
}

type VectorResult struct {
    DocID     string
    SessionID string
    Score     float64
}

type GraphResult struct {
    DocID string
    Score float64
}

type HybridResult struct {
    DocID         string
    SessionID     string
    CombinedScore float64
    BM25Score     float64
    VectorScore   float64
    GraphScore    float64
    BM25Rank      int
    VectorRank    int
}

type ScoreWeights struct {
    BM25   float64  // default 0.4
    Vector float64  // default 0.6
    Graph  float64  // default 0.0
}

var DefaultWeights = ScoreWeights{BM25: 0.4, Vector: 0.6}

type DocMeta struct {
    SessionID string
    AgentID   string
    TenantID  string
}
