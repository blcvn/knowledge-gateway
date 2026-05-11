package domain_test

import (
	"testing"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
)

func TestDuplicateDecision_Validate(t *testing.T) {
	validDecisions := []domain.DuplicateDecision{
		domain.DecisionMerge,
		domain.DecisionCreate,
		domain.DecisionSkip,
	}

	for _, d := range validDecisions {
		if err := d.Validate(); err != nil {
			t.Errorf("expected no error for valid decision %v, got %v", d, err)
		}
	}

	invalidDecision := domain.DuplicateDecision("invalid")
	if err := invalidDecision.Validate(); err == nil {
		t.Error("expected error for invalid decision, got nil")
	}
}

func TestEmbeddingVector_Validate(t *testing.T) {
	vec := domain.EmbeddingVector{1.0, 2.0, 3.0}
	
	if err := vec.Validate(3); err != nil {
		t.Errorf("expected no error for correct dimension, got %v", err)
	}

	if err := vec.Validate(4); err == nil {
		t.Error("expected error for incorrect dimension, got nil")
	}
}

func TestTokenUsage_Add(t *testing.T) {
	u1 := domain.TokenUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30}
	u2 := domain.TokenUsage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15}

	u1.Add(u2)

	if u1.PromptTokens != 15 || u1.CompletionTokens != 30 || u1.TotalTokens != 45 {
		t.Errorf("TokenUsage Add failed: %+v", u1)
	}
}
