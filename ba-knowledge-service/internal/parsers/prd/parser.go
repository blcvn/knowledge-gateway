package prd

import (
	"context"
	"log"

	"github.com/blcvn/backend/services/pkg/domain"
	aiproxy "github.com/blcvn/kratos-proto/go/ai-proxy"
	prompt "github.com/blcvn/kratos-proto/go/prompt"
)

// Parser handles PRD parsing
type Parser struct {
	aiProxyClient aiproxy.AIProxyServiceClient
	promptClient  prompt.PromptServiceClient
}

// NewParser creates a new PRD parser
func NewParser(aiProxyClient aiproxy.AIProxyServiceClient, promptClient prompt.PromptServiceClient) *Parser {
	return &Parser{
		aiProxyClient: aiProxyClient,
		promptClient:  promptClient,
	}
}

// ParsePRD parses a PRD document into structured format
// TODO: Implement full PRD parsing logic using AI Proxy Service
func (p *Parser) ParsePRD(ctx context.Context, prdContent string) (*domain.StructuredPRD, error) {
	log.Printf("[PRDParser] STUB: ParsePRD called (Content Length: %d)", len(prdContent))

	// For now, return a minimal valid StructuredPRD
	// Full implementation requires AI Proxy Service integration with proper ChatRequest/Response proto definitions
	prd := &domain.StructuredPRD{
		Metadata: domain.PRDMetadata{
			ProductName: "Parsed Product",
			Version:     "1.0",
			Status:      "draft",
		},
		Features:    []domain.Feature{},
		UserStories: []domain.UserStory{},
	}

	log.Printf("[PRDParser] STUB: Returning minimal StructuredPRD")
	return prd, nil
}
