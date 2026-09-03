package telemetry

import (
	"sync"
)

// PromptUsageAgg accumulates token usage per prompt type
type PromptUsageAgg struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CallCount        int64
}

// TokenUsage for interface compatibility
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// TokenTracker tracks LLM token usage per prompt type. Thread-safe.
type TokenTracker struct {
	mu    sync.RWMutex
	usage map[string]*PromptUsageAgg
}

func NewTokenTracker() *TokenTracker {
	return &TokenTracker{usage: make(map[string]*PromptUsageAgg)}
}

// Track adds token usage for a given prompt name
func (t *TokenTracker) Track(promptName string, usage TokenUsage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	agg, ok := t.usage[promptName]
	if !ok {
		agg = &PromptUsageAgg{}
		t.usage[promptName] = agg
	}
	agg.PromptTokens += int64(usage.PromptTokens)
	agg.CompletionTokens += int64(usage.CompletionTokens)
	agg.TotalTokens += int64(usage.TotalTokens)
	agg.CallCount++
}

// GetAll returns a snapshot of all prompt usage (copy, safe to read without lock)
func (t *TokenTracker) GetAll() map[string]PromptUsageAgg {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make(map[string]PromptUsageAgg, len(t.usage))
	for k, v := range t.usage {
		result[k] = *v
	}
	return result
}

// Reset clears all accumulated token usage
func (t *TokenTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usage = make(map[string]*PromptUsageAgg)
}
