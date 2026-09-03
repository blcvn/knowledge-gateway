package consolidation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	"vnp-memory/services/memory-service/internal/consolidation/port"
)

const compressionSystemPrompt = `You are a memory system. Compress this AI agent observation.
Return ONLY valid JSON:
{
  "title": "one-line action summary (max 80 chars)",
  "subtitle": "key detail or null",
  "facts": ["key fact 1", "key fact 2", "key fact 3"],
  "narrative": "2-3 sentence description",
  "concepts": ["entity1", "entity2"],
  "files": ["/path/to/file"],
  "importance": 0.7
}`

type Compressor struct {
	llm          port.ILLMProvider
	cb           *CircuitBreaker
	autoCompress bool
}

func NewCompressor(llm port.ILLMProvider, autoCompress bool) *Compressor {
	return &Compressor{
		llm:          llm,
		cb:           NewCircuitBreaker(3, 5*time.Minute),
		autoCompress: autoCompress,
	}
}

func (c *Compressor) Compress(ctx context.Context, raw agentmemory.RawObs) agentmemory.CompressedObs {
	if c.autoCompress && c.llm != nil && c.cb.Allow() {
		comp, err := c.compressWithLLM(ctx, raw)
		if err == nil {
			c.cb.RecordSuccess()
			return comp
		}
		c.cb.RecordFailure()
	}
	return syntheticCompress(raw)
}

func (c *Compressor) compressWithLLM(ctx context.Context, raw agentmemory.RawObs) (agentmemory.CompressedObs, error) {
	userMsg := fmt.Sprintf("HookType: %s\nToolName: %s\nOutput: %s",
		raw.HookType, raw.ToolName, string(raw.ToolOutput))

	resp, err := c.llm.Chat(ctx, compressionSystemPrompt, userMsg)
	if err != nil {
		return agentmemory.CompressedObs{}, err
	}

	var result struct {
		Title      string   `json:"title"`
		Subtitle   string   `json:"subtitle"`
		Facts      []string `json:"facts"`
		Narrative  string   `json:"narrative"`
		Concepts   []string `json:"concepts"`
		Files      []string `json:"files"`
		Importance float64  `json:"importance"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return agentmemory.CompressedObs{}, err
	}
	return agentmemory.CompressedObs{
		SessionID: raw.SessionID, Title: result.Title, Subtitle: result.Subtitle,
		Facts: result.Facts, Narrative: result.Narrative, Concepts: result.Concepts,
		Files: result.Files, Importance: result.Importance,
	}, nil
}

// syntheticCompress — same as observe-service pipeline, zero LLM
func syntheticCompress(raw agentmemory.RawObs) agentmemory.CompressedObs {
	return agentmemory.CompressedObs{}
}
