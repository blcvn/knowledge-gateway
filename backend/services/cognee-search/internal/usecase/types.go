package usecase

import "github.com/google/uuid"

// SearchStrategy identifies which retriever to use.
type SearchStrategy string

const (
	StrategySimilarity      SearchStrategy = "SIMILARITY"
	StrategyGraphCompletion SearchStrategy = "GRAPH_COMPLETION"
	StrategyGraphSummary    SearchStrategy = "GRAPH_SUMMARY"
	StrategyKeyword         SearchStrategy = "KEYWORD"
	StrategyChunks          SearchStrategy = "CHUNKS"
	StrategyTemporal        SearchStrategy = "TEMPORAL"
	StrategyMultiHop        SearchStrategy = "MULTI_HOP"
	StrategyHybrid          SearchStrategy = "HYBRID"
	StrategyFeelingLucky    SearchStrategy = "FEELING_LUCKY"
	StrategyFeedback        SearchStrategy = "FEEDBACK"
)

// SearchRequest is the input to SearchUseCase.Execute().
type SearchRequest struct {
	Query           string
	Strategies      []SearchStrategy
	DatasetID       *uuid.UUID
	DatasetName     string
	TenantID        string
	NodeSets        []string   // [NEW] CR-002 — filter by NodeSet tags
	TopK            int
	SaveInteraction bool
	SessionID       *string
	FeedbackFor     *string
	FeedbackScore   *float64
	FeedbackText    string
}

// SearchResult is a single result from a retriever.
type SearchResult struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]string
}

// SearchResponse is the output of SearchUseCase.Execute().
type SearchResponse struct {
	Results       []SearchResult
	InteractionID *string
	Metadata      map[string]string
}
