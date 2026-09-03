package domain

import "errors"

type SearchQuery struct {
	Query          string
	GroupID        string
	Methods        []SearchMethod
	Rerankers      []RerankerType
	Limit          int
	TemporalFilter *TemporalWindow
	EntityLabels   []string
}

func (q *SearchQuery) Validate() error {
	if q.Query == "" {
		return errors.New("query cannot be empty")
	}
	if len(q.Methods) == 0 {
		return errors.New("at least one search method must be provided")
	}
	if q.Limit <= 0 {
		return errors.New("limit must be greater than 0")
	}
	for _, m := range q.Methods {
		if !m.IsValid() {
			return errors.New("invalid search method")
		}
	}
	for _, r := range q.Rerankers {
		if !r.IsValid() {
			return errors.New("invalid reranker type")
		}
	}
	if q.TemporalFilter != nil {
		if err := q.TemporalFilter.Validate(); err != nil {
			return errors.New("invalid temporal filter")
		}
	}
	return nil
}

type SearchResult struct {
	EntityID   string
	Score      float64
	MethodUsed SearchMethod
	Content    string
	Metadata   map[string]any
}

type RankedResult struct {
	EntityID string
	Score    float64
	Rank     int
	Content  string
	Metadata map[string]any
}
