package domain

import "time"

type SearchResult struct {
	ID        string
	Content   string
	Score     float64
	Type      ResultType
	Strategy  SearchStrategy
	Metadata  map[string]interface{}
	Timestamp time.Time
}

type RetrieverConfig struct {
	MaxResults int
	Timeout    time.Duration
}

type RerankScore struct {
	ResultID string
	Score    float64
}

type SearchSession struct {
	SessionID string
	Query     string
	StartedAt time.Time
}
