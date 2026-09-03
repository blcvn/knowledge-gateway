// Package usecase implements token usage reporting for graphiti-admin.
// SOL-007: Admin Service & Observability Stack (CR-GR-007)
package usecase

import (
	"context"

	"vnp-memory/services/graphiti-admin/internal/usecase/port"
)

// PricingConfig holds per-provider token pricing.
type PricingConfig struct {
	Providers map[string]ProviderPricing
}

// ProviderPricing holds input/output cost per million tokens.
type ProviderPricing struct {
	InputPricePerMToken  float64 // $ per million input tokens
	OutputPricePerMToken float64 // $ per million output tokens
}

// DefaultPricingConfig is the default token cost table.
var DefaultPricingConfig = PricingConfig{
	Providers: map[string]ProviderPricing{
		"openai_gpt4o":       {InputPricePerMToken: 5.00, OutputPricePerMToken: 15.00},
		"openai_gpt4o_mini":  {InputPricePerMToken: 0.15, OutputPricePerMToken: 0.60},
		"anthropic_claude35": {InputPricePerMToken: 3.00, OutputPricePerMToken: 15.00},
		"gemini_flash":       {InputPricePerMToken: 0.075, OutputPricePerMToken: 0.30},
	},
}

// TokenUsageReport aggregates usage across all prompts.
type TokenUsageReport struct {
	Period           string
	ByPrompt         map[string]PromptUsage
	Total            PromptUsage
	EstimatedCostUSD float64
}

// PromptUsage contains aggregated token counts for a prompt type.
type PromptUsage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	CallCount    int64
}

// TokenUsageReportUseCase aggregates LLM token usage and calculates cost.
type TokenUsageReportUseCase struct {
	knowledgePort port.KnowledgePort
	pricingConfig PricingConfig
}

// NewTokenUsageReportUseCase constructs the use case.
func NewTokenUsageReportUseCase(knowledgePort port.KnowledgePort, cfg PricingConfig) *TokenUsageReportUseCase {
	return &TokenUsageReportUseCase{knowledgePort: knowledgePort, pricingConfig: cfg}
}

// Execute returns a token usage report for the given group and period.
func (uc *TokenUsageReportUseCase) Execute(ctx context.Context, groupID, period string) (*TokenUsageReport, error) {
	raw, err := uc.knowledgePort.GetTokenUsage(ctx, port.GetTokenUsageReq{GroupID: groupID})
	if err != nil {
		return nil, err
	}

	report := &TokenUsageReport{
		Period:   period,
		ByPrompt: make(map[string]PromptUsage),
	}

	for promptName, usage := range raw {
		pu := PromptUsage{
			InputTokens:  usage.PromptTokens,
			OutputTokens: usage.CompletionTokens,
			TotalTokens:  usage.TotalTokens,
			CallCount:    usage.CallCount,
		}
		report.ByPrompt[promptName] = pu
		report.Total.InputTokens  += pu.InputTokens
		report.Total.OutputTokens += pu.OutputTokens
		report.Total.TotalTokens  += pu.TotalTokens
		report.Total.CallCount    += pu.CallCount
	}

	// Cost estimation using default provider (gpt-4o)
	pricing := uc.pricingConfig.Providers["openai_gpt4o"]
	report.EstimatedCostUSD = float64(report.Total.InputTokens)/1_000_000*pricing.InputPricePerMToken +
		float64(report.Total.OutputTokens)/1_000_000*pricing.OutputPricePerMToken

	return report, nil
}
