// Package usecase implements the append_event use case for memobase-event.
// SOL-MB-004: Event Timeline & Semantic Search (CR-MB-004)
//
// AppendEvent is called by memobase-engine after a successful LLM flush.
// The engine pre-computes embeddings so this service only stores them.
package usecase

import (
	"context"
	"fmt"
	"strings"

	"vnp-memory/services/memobase-event/internal/domain"
	"vnp-memory/services/memobase-event/internal/usecase/port"
)

// AppendEventUseCase stores a new event and its gists.
type AppendEventUseCase struct {
	eventRepo port.EventRepository
	gistRepo  port.GistRepository
}

// NewAppendEventUseCase constructs the use case.
func NewAppendEventUseCase(eventRepo port.EventRepository, gistRepo port.GistRepository) *AppendEventUseCase {
	return &AppendEventUseCase{eventRepo: eventRepo, gistRepo: gistRepo}
}

// AppendEventRequest is the input for AppendEvent.
type AppendEventRequest struct {
	UserID         string
	ProjectID      string
	EventData      domain.EventData
	Embedding      []float32   // pre-computed by memobase-engine
	GistEmbeddings [][]float32 // per-gist embeddings, len == number of gists in EventTip
}

// AppendEventResult is the output from AppendEvent.
type AppendEventResult struct {
	EventID   string
	GistCount int
}

// Execute stores the event and its parsed gists.
func (uc *AppendEventUseCase) Execute(ctx context.Context, req AppendEventRequest) (*AppendEventResult, error) {
	// 1. Store event with pre-computed embedding from engine
	event, err := uc.eventRepo.Save(ctx, domain.Event{
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		EventData: req.EventData,
		Embedding: req.Embedding,
	})
	if err != nil {
		return nil, fmt.Errorf("save event: %w", err)
	}

	// 2. Parse gists from event_tip ("- line1\n- line2" → ["line1", "line2"])
	gistContents := parseGistsFromEventTip(req.EventData.EventTip)
	var gists []domain.EventGist
	for i, content := range gistContents {
		var embedding []float32
		if i < len(req.GistEmbeddings) {
			embedding = req.GistEmbeddings[i]
		}
		gists = append(gists, domain.EventGist{
			EventID:   event.ID,
			UserID:    req.UserID,
			ProjectID: req.ProjectID,
			GistData:  domain.GistData{GistContent: content},
			Embedding: embedding,
		})
	}

	// 3. Bulk insert gists
	if len(gists) > 0 {
		if err := uc.gistRepo.SaveBulk(ctx, gists); err != nil {
			// non-fatal: event is stored, gists are best-effort
			return &AppendEventResult{EventID: event.ID, GistCount: 0}, nil
		}
	}

	return &AppendEventResult{EventID: event.ID, GistCount: len(gists)}, nil
}

// parseGistsFromEventTip splits "- line1\n- line2" → ["line1", "line2"].
func parseGistsFromEventTip(tip string) []string {
	var gists []string
	for _, line := range strings.Split(tip, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			gists = append(gists, strings.TrimPrefix(line, "- "))
		}
	}
	return gists
}
