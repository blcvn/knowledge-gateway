package domain

// SearchResult represents a single search result from any search method.
type SearchResult struct {
	UUID       string  `json:"uuid"`
	NodeLabel  string  `json:"node_label"`  // Entity, Episodic, Community
	Name       string  `json:"name"`
	Summary    string  `json:"summary,omitempty"`
	Fact       string  `json:"fact,omitempty"`       // For edge results
	Score      float64 `json:"score"`               // Relevance score (cosine, BM25, or distance)
	Distance   int     `json:"distance,omitempty"`   // For BFS results (hop count)
	GroupID    string  `json:"group_id"`
}

// SearchParams provides common parameters for search operations.
type SearchParams struct {
	GroupID string `json:"group_id"`
	Limit   int    `json:"limit"`
}

// Validate checks required search parameters.
func (p *SearchParams) Validate() error {
	if p.GroupID == "" {
		return ErrMissingGroupID
	}
	if p.Limit <= 0 {
		p.Limit = 10 // default
	}
	if p.Limit > 100 {
		p.Limit = 100 // cap
	}
	return nil
}
