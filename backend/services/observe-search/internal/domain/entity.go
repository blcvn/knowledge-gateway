package domain

// SearchResult represents a single search result from the hybrid engine.
type SearchResult struct {
    DocID         string
    SessionID     string
    AgentID       string
    Narrative     string
    Title         string
    CombinedScore float64
    BM25Score     float64
    VectorScore   float64
}

// ContextBlock is a single block of context assembled for injection.
type ContextBlock struct {
    Type    string  // "memory" | "summary" | "observation"
    Content string
    Tokens  int
    Recency float64
    Source  string
}

// ContextResponse is the result of a context build request.
type ContextResponse struct {
    Blocks      []ContextBlock
    TotalTokens int
    Formatted   string
}
