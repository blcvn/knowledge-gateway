package consolidation

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SessionSummarizerUseCase creates a natural-language narrative summary of an ended session.
type SessionSummarizerUseCase struct{}

func NewSessionSummarizerUseCase() *SessionSummarizerUseCase {
	return &SessionSummarizerUseCase{}
}

// SummarizeRequest is the input for session summarization.
type SummarizeRequest struct {
	TenantID   string
	SessionID  string
	Project    string
	AgentID    string
	Narratives []string       // compressed observation narratives
	Duration   time.Duration
	ObsCount   int
}

// SummarizeResult is the result of a session summary operation.
type SummarizeResult struct {
	SessionID  string
	Narrative  string
	Topics     []string
	Duration   time.Duration
	ObsCount   int
	TokenCount int
}

// Execute summarizes an ended session into a compact narrative.
func (uc *SessionSummarizerUseCase) Execute(_ context.Context, req SummarizeRequest) (*SummarizeResult, error) {
	if len(req.Narratives) == 0 {
		return &SummarizeResult{
			SessionID: req.SessionID,
			Narrative: fmt.Sprintf("Empty session %s in project %s", req.SessionID, req.Project),
			Duration:  req.Duration,
			ObsCount:  0,
		}, nil
	}

	// Build a consolidated narrative from all observation narratives
	combined := strings.Join(req.Narratives, " ")
	topics := extractTopics(combined)

	narrative := fmt.Sprintf("Session %s (%s, %d observations): %s",
		req.SessionID[:min(8, len(req.SessionID))],
		req.Project,
		req.ObsCount,
		combined[:min(500, len(combined))],
	)

	return &SummarizeResult{
		SessionID:  req.SessionID,
		Narrative:  narrative,
		Topics:     topics,
		Duration:   req.Duration,
		ObsCount:   req.ObsCount,
		TokenCount: len(narrative) / 4,
	}, nil
}

// extractTopics extracts key topics from a narrative using simple heuristics.
func extractTopics(narrative string) []string {
	words := strings.Fields(narrative)
	seen := make(map[string]bool)
	var topics []string
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ".,;:!?\"'()[]{}"))
		if len(w) > 5 && !seen[w] && !isCommonWord(w) {
			seen[w] = true
			if len(topics) < 10 {
				topics = append(topics, w)
			}
		}
	}
	return topics
}

var commonWords = map[string]bool{
	"the": true, "and": true, "that": true, "this": true, "with": true,
	"from": true, "have": true, "been": true, "were": true, "they": true,
}

func isCommonWord(w string) bool { return commonWords[w] }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
