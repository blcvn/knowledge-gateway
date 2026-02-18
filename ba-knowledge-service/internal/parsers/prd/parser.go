package prd

import (
	"context"
	"log"

	v32 "github.com/blcvn/backend/services/ba-agent-service/domain/v3.2"
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
func (p *Parser) ParsePRD(ctx context.Context, prdContent string) (*v32.StructuredPRD, error) {
	log.Printf("[PRDParser] STUB: ParsePRD called (Content Length: %d)", len(prdContent))

	// For now, return a minimal valid StructuredPRD
	// Full implementation requires AI Proxy Service integration with proper ChatRequest/Response proto definitions
	prd := &v32.StructuredPRD{
		Metadata: v32.PRDMetadata{
			ProductName: "Parsed Product",
			Version:     "1.0",
			Status:      "draft",
		},
		Features:    []v32.Feature{},
		UserStories: []v32.UserStory{},
	}

	log.Printf("[PRDParser] STUB: Returning minimal StructuredPRD")
	return prd, nil
}
