package outline

import (
	"context"
	"fmt"

	aiproxy "github.com/blcvn/kratos-proto/go/ai-proxy"
	promptpb "github.com/blcvn/kratos-proto/go/prompt"
)

// Generator generates URD Outline documents
type Generator struct {
	aiProxyClient aiproxy.AIProxyServiceClient
	promptClient  promptpb.PromptServiceClient
}

// NewGenerator creates a new outline generator
func NewGenerator(aiProxyClient aiproxy.AIProxyServiceClient, promptClient promptpb.PromptServiceClient) *Generator {
	return &Generator{
		aiProxyClient: aiProxyClient,
		promptClient:  promptClient,
	}
}

// GenerateOutline generates an outline from index content
func (g *Generator) GenerateOutline(ctx context.Context, indexContent, title string) (string, error) {
	// TODO: Implement full outline generation logic
	// For now, return a placeholder

	fmt.Printf("[OutlineGenerator] Generating outline for: %s\n", title)

	// In the full implementation, this would:
	// 1. Get the outline generation prompt from Prompt Service
	// 2. Call AI Proxy with the index content
	// 3. Parse and structure the response

	return fmt.Sprintf("# URD Outline for %s\n\n## Generated from Index\n\nContent: %d characters", title, len(indexContent)), nil
}
