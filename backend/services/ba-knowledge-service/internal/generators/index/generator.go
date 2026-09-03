package index

import (
	"context"
	"fmt"

	aiproxy "github.com/blcvn/kratos-proto/go/ai-proxy"
	promptpb "github.com/blcvn/kratos-proto/go/prompt"
)

// Generator generates URD Index documents
type Generator struct {
	aiProxyClient aiproxy.AIProxyServiceClient
	promptClient  promptpb.PromptServiceClient
}

// NewGenerator creates a new index generator
func NewGenerator(aiProxyClient aiproxy.AIProxyServiceClient, promptClient promptpb.PromptServiceClient) *Generator {
	return &Generator{
		aiProxyClient: aiProxyClient,
		promptClient:  promptClient,
	}
}

// GenerateIndex generates an index from PRD content
func (g *Generator) GenerateIndex(ctx context.Context, prdContent, title string) (string, error) {
	// TODO: Implement full index generation logic
	// For now, return a placeholder

	fmt.Printf("[IndexGenerator] Generating index for: %s\n", title)

	// In the full implementation, this would:
	// 1. Get the index generation prompt from Prompt Service
	// 2. Call AI Proxy with the PRD content
	// 3. Parse and structure the response

	return fmt.Sprintf("# URD Index for %s\n\n## Generated from PRD\n\nContent: %d characters", title, len(prdContent)), nil
}
