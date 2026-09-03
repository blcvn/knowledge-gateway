package full

import (
	"context"
	"fmt"

	aiproxy "github.com/blcvn/kratos-proto/go/ai-proxy"
	promptpb "github.com/blcvn/kratos-proto/go/prompt"
)

// Generator generates Full URD documents
type Generator struct {
	aiProxyClient aiproxy.AIProxyServiceClient
	promptClient  promptpb.PromptServiceClient
}

// NewGenerator creates a new full URD generator
func NewGenerator(aiProxyClient aiproxy.AIProxyServiceClient, promptClient promptpb.PromptServiceClient) *Generator {
	return &Generator{
		aiProxyClient: aiProxyClient,
		promptClient:  promptClient,
	}
}

// GenerateFull generates a full URD from outline content
func (g *Generator) GenerateFull(ctx context.Context, outlineContent, title string) (string, error) {
	// TODO: Implement full URD generation logic
	// For now, return a placeholder

	fmt.Printf("[FullGenerator] Generating full URD for: %s\n", title)

	// In the full implementation, this would:
	// 1. Get the full URD generation prompt from Prompt Service
	// 2. Call AI Proxy with the outline content
	// 3. Parse and structure the response

	return fmt.Sprintf("# Full URD for %s\n\n## Generated from Outline\n\nContent: %d characters", title, len(outlineContent)), nil
}
