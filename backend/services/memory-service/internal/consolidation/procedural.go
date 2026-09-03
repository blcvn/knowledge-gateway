package consolidation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	"vnp-memory/services/memory-service/internal/consolidation/port"

	"github.com/google/uuid"
)

type ProceduralExtractor struct {
	summaryRepo    port.ISessionSummaryRepo
	proceduralRepo port.IProceduralRepo
	llm            port.ILLMProvider
	minFrequency   int
}

func (e *ProceduralExtractor) ExtractAll(ctx context.Context) {
	summaries, _ := e.summaryRepo.ListAll(ctx)
	sequences := e.buildSequences(summaries)

	for seqKey, sessionIDs := range sequences {
		if len(sessionIDs) < e.minFrequency {
			continue
		}

		stepHash := fmt.Sprintf("%x", sha256.Sum256([]byte(seqKey)))
		existing, _ := e.proceduralRepo.FindByStepHash(ctx, stepHash)
		if existing != nil {
			e.proceduralRepo.IncrementFrequency(ctx, existing.ID)
			continue
		}

		steps := strings.Split(seqKey, "|")
		proc := agentmemory.ProceduralMemory{
			ID:         uuid.New().String(),
			Steps:      steps,
			StepHash:   stepHash,
			Frequency:  len(sessionIDs),
			Confidence: float64(len(sessionIDs)) / 10.0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if e.llm != nil {
			e.enrichWithLLM(ctx, &proc, steps)
		} else {
			proc.Name = fmt.Sprintf("Workflow: %s → %s", steps[0], steps[len(steps)-1])
		}

		e.proceduralRepo.Save(ctx, proc)
	}
}

// buildSequences: extract key_decisions from summaries, find repeated patterns
func (e *ProceduralExtractor) buildSequences(summaries []agentmemory.SessionSummary) map[string][]string {
	sequences := make(map[string][]string)
	for _, s := range summaries {
		if len(s.KeyDecisions) < 2 {
			continue
		}
		key := strings.Join(s.KeyDecisions, "|")
		sequences[key] = append(sequences[key], s.SessionID)
	}
	return sequences
}

func (e *ProceduralExtractor) enrichWithLLM(ctx context.Context, proc *agentmemory.ProceduralMemory, steps []string) {
	prompt := fmt.Sprintf("Name this coding workflow: %s. Return JSON: {\"name\": \"...\", \"trigger\": \"...\", \"outcome\": \"...\"}", strings.Join(steps, " → "))
	resp, err := e.llm.Chat(ctx, "You name coding workflows concisely.", prompt)
	if err != nil {
		return
	}
	var result struct {
		Name    string `json:"name"`
		Trigger string `json:"trigger"`
		Outcome string `json:"outcome"`
	}
	if json.Unmarshal([]byte(resp), &result) == nil {
		proc.Name = result.Name
		proc.TriggerCondition = result.Trigger
		proc.ExpectedOutcome = result.Outcome
	}
}
