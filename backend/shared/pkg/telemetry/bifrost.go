package telemetry

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

// LLMUsage captures token consumption for a single LLM call.
// SOL-ENT-005 / TASK-ENT-010
type LLMUsage struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	LatencyMs        int64
	TenantID         string
	Service          string
}

// LLMCostAccumulator tracks cumulative LLM token usage across a service instance.
type LLMCostAccumulator struct {
	promptTokens     atomic.Int64
	completionTokens atomic.Int64
	totalCalls       atomic.Int64
	logger           *slog.Logger
}

// NewLLMCostAccumulator creates a new LLMCostAccumulator.
func NewLLMCostAccumulator(logger *slog.Logger) *LLMCostAccumulator {
	return &LLMCostAccumulator{logger: logger}
}

// Record records a single LLM call's usage.
func (a *LLMCostAccumulator) Record(usage LLMUsage) {
	a.promptTokens.Add(int64(usage.PromptTokens))
	a.completionTokens.Add(int64(usage.CompletionTokens))
	a.totalCalls.Add(1)

	a.logger.Info("llm.usage",
		"model", usage.Model,
		"tenant_id", usage.TenantID,
		"service", usage.Service,
		"prompt_tokens", usage.PromptTokens,
		"completion_tokens", usage.CompletionTokens,
		"total_tokens", usage.TotalTokens,
		"latency_ms", usage.LatencyMs,
	)
}

// Summary returns cumulative totals.
func (a *LLMCostAccumulator) Summary() (promptTokens, completionTokens, totalCalls int64) {
	return a.promptTokens.Load(), a.completionTokens.Load(), a.totalCalls.Load()
}

// LLMCostInterceptor wraps an LLM call and records token usage.
// Use this as a decorator around any LLMClient.Complete/CompleteJSON call.
//
//	usage, result, err := telemetry.LLMCostInterceptor(ctx, acc, "gpt-4o", tenantID, serviceName, func() (string, int, int, error) {
//	    resp, err := llm.Complete(ctx, prompt)
//	    return resp, promptCount, completionCount, err
//	})
type LLMCallFn func() (result string, promptTokens, completionTokens int, err error)

// LLMCostInterceptor wraps an LLM call function and records cost metrics.
func LLMCostInterceptor(
	ctx context.Context,
	acc *LLMCostAccumulator,
	model, tenantID, service string,
	fn LLMCallFn,
) (string, error) {
	_ = ctx // reserved for future span propagation
	start := time.Now()

	result, promptTokens, completionTokens, err := fn()
	if err != nil {
		return "", err
	}

	acc.Record(LLMUsage{
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		LatencyMs:        time.Since(start).Milliseconds(),
		TenantID:         tenantID,
		Service:          service,
	})

	return result, nil
}
