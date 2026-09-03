package model

type Score float64

type SearchResult struct {
	ID             string         `json:"id"`
	Path           string         `json:"path"`
	SemanticScore  Score          `json:"semantic_score"`
	HotnessScore   Score          `json:"hotness_score"`
	FinalScore     Score          `json:"final_score"`
	MatchedContext MatchedContext `json:"matched_context"`
}

type MatchedContext struct {
	Content     string `json:"content"`
	ContextType string `json:"context_type"`
	DepthLevel  string `json:"depth_level"`
}
