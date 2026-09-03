package consolidation

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	"vnp-memory/services/memory-service/internal/consolidation/port"
)

const summarizeSystemPrompt = `Summarize this AI agent coding session.
Return JSON:
{
  "title": "1 sentence session title",
  "narrative": "2-3 paragraph description",
  "key_decisions": ["decision 1", "decision 2"],
  "files_modified": ["/path/to/file"],
  "concepts": ["concept1", "concept2"]
}`

type Summarizer struct {
	obsRepo port.IObservationRepo
	llm     port.ILLMProvider
	cb      *CircuitBreaker
}

func NewSummarizer(obsRepo port.IObservationRepo, llm port.ILLMProvider) *Summarizer {
	return &Summarizer{
		obsRepo: obsRepo, llm: llm,
		cb: NewCircuitBreaker(3, 5*time.Minute),
	}
}

func (s *Summarizer) Summarize(ctx context.Context, sessionID string) *agentmemory.SessionSummary {
	compObs, _ := s.obsRepo.ListCompressed(ctx, sessionID)
	if len(compObs) == 0 {
		return nil
	}

	summary := &agentmemory.SessionSummary{
		SessionID:        sessionID,
		ObservationCount: len(compObs),
	}

	// Build input for LLM
	var lines []string
	for _, obs := range compObs {
		lines = append(lines, fmt.Sprintf("- [%s] %s", obs.ObsType, obs.Title))
		if len(obs.Facts) > 0 {
			lines = append(lines, "  Facts: "+strings.Join(obs.Facts, "; "))
		}
	}
	input := strings.Join(lines, "\n")

	// Try LLM
	if s.llm != nil && s.cb.Allow() {
		resp, err := s.llm.Chat(ctx, summarizeSystemPrompt, input)
		if err == nil {
			var result struct {
				Title         string   `json:"title"`
				Narrative     string   `json:"narrative"`
				KeyDecisions  []string `json:"key_decisions"`
				FilesModified []string `json:"files_modified"`
				Concepts      []string `json:"concepts"`
			}
			if json.Unmarshal([]byte(resp), &result) == nil {
				s.cb.RecordSuccess()
				summary.Title = result.Title
				summary.Narrative = result.Narrative
				summary.KeyDecisions = result.KeyDecisions
				summary.FilesModified = result.FilesModified
				summary.Concepts = result.Concepts
				return summary
			}
		}
		s.cb.RecordFailure()
	}

	// Synthetic summary: top 3 by importance
	sortByImportance(compObs)
	if len(compObs) > 0 {
		summary.Title = compObs[0].Title
	}
	top := compObs
	if len(top) > 3 {
		top = top[:3]
	}
	narratives := make([]string, len(top))
	for i, obs := range top {
		narratives[i] = obs.Narrative
	}
	summary.Narrative = strings.Join(narratives, " ")
	return summary
}

func sortByImportance(obs []agentmemory.CompressedObs) {
	sort.Slice(obs, func(i, j int) bool { return obs[i].Importance > obs[j].Importance })
}
