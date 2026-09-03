// Package usecase implements semantic event search for memobase-event.
// SOL-MB-004: Event Timeline & Semantic Search (CR-MB-004)
package usecase

import (
	"context"
	"fmt"

	"vnp-memory/services/memobase-event/internal/domain"
	"vnp-memory/services/memobase-event/internal/usecase/port"
)

const (
	defaultSearchTopK      = 10
	defaultSearchThreshold = 0.2
	defaultTimeRangeDays   = 21
)

// SearchEventsUseCase performs pgvector cosine similarity search on user_events.
type SearchEventsUseCase struct {
	eventRepo port.EventRepository
	embedder  port.Embedder
}

// NewSearchEventsUseCase constructs the use case.
func NewSearchEventsUseCase(eventRepo port.EventRepository, embedder port.Embedder) *SearchEventsUseCase {
	return &SearchEventsUseCase{eventRepo: eventRepo, embedder: embedder}
}

// SearchEventsRequest is the input for SearchEvents.
type SearchEventsRequest struct {
	UserID              string
	ProjectID           string
	Query               string
	TopK                int
	SimilarityThreshold float64
	TimeRangeDays       int
}

// Execute embeds the query and searches user_events by cosine similarity.
func (uc *SearchEventsUseCase) Execute(ctx context.Context, req SearchEventsRequest) ([]domain.SearchResult, error) {
	if !uc.embedder.IsEnabled() {
		return nil, domain.ErrEmbeddingDisabled
	}

	queryVec, err := uc.embedder.EmbedQuery(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	topK := req.TopK
	if topK <= 0 {
		topK = defaultSearchTopK
	}
	threshold := req.SimilarityThreshold
	if threshold <= 0 {
		threshold = defaultSearchThreshold
	}
	timeRange := req.TimeRangeDays
	if timeRange <= 0 {
		timeRange = defaultTimeRangeDays
	}

	return uc.eventRepo.SearchByEmbedding(ctx, port.SearchByEmbeddingQuery{
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		Vector:    queryVec,
		Threshold: threshold,
		TimeRange: timeRange,
		Limit:     topK,
	})
}

// SearchEventGistsUseCase performs pgvector cosine similarity search on user_event_gists.
type SearchEventGistsUseCase struct {
	gistRepo port.GistRepository
	embedder port.Embedder
}

// NewSearchEventGistsUseCase constructs the use case.
func NewSearchEventGistsUseCase(gistRepo port.GistRepository, embedder port.Embedder) *SearchEventGistsUseCase {
	return &SearchEventGistsUseCase{gistRepo: gistRepo, embedder: embedder}
}

// SearchEventGistsRequest is the input for SearchEventGists.
// Supports both text query (service embeds) and pre-computed embedding (from context service).
type SearchEventGistsRequest struct {
	UserID              string
	ProjectID           string
	Query               string    // Option A: text query
	PrecomputedEmbedding []float32 // Option B: pre-computed embedding (skips embed call)
	TopK                int
	SimilarityThreshold float64
}

// Execute searches event_gists by cosine similarity.
func (uc *SearchEventGistsUseCase) Execute(ctx context.Context, req SearchEventGistsRequest) ([]domain.GistSearchResult, error) {
	var vec []float32

	if len(req.PrecomputedEmbedding) > 0 {
		// Option B: use pre-computed embedding from context service (avoids double embedding)
		vec = req.PrecomputedEmbedding
	} else {
		// Option A: embed the text query
		if !uc.embedder.IsEnabled() {
			return nil, domain.ErrEmbeddingDisabled
		}
		var err error
		vec, err = uc.embedder.EmbedQuery(ctx, req.Query)
		if err != nil {
			return nil, fmt.Errorf("embed gist query: %w", err)
		}
	}

	topK := req.TopK
	if topK <= 0 {
		topK = defaultSearchTopK
	}
	threshold := req.SimilarityThreshold
	if threshold <= 0 {
		threshold = defaultSearchThreshold
	}

	return uc.gistRepo.SearchByEmbedding(ctx, port.GistSearchQuery{
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		Vector:    vec,
		Threshold: threshold,
		Limit:     topK,
	})
}
