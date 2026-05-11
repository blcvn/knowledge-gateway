package llm

import (
	"context"
	"fmt"

	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/engine"
)

// BifrostClient implements port.LLMService using the VNP Bifrost gateway.
// In a real implementation, this interacts with HTTP/gRPC Bifrost endpoint.
type BifrostClient struct {
	endpoint string
}

func NewBifrostClient(endpoint string) *BifrostClient {
	return &BifrostClient{endpoint: endpoint}
}

func (b *BifrostClient) ExtractTopics(ctx context.Context, content string) ([]engine.TopicEntry, error) {
	// Simulated LLM Call #1 & #2 (Summarize + Extract)
	// Must enforce token constraints
	fmt.Printf("[Bifrost] Extracting topics via %s\n", b.endpoint)
	return []engine.TopicEntry{
		{Category: "General", Topic: "Extracted Topic", Weight: 1.0},
	}, nil
}

func (b *BifrostClient) GenerateGist(ctx context.Context, content string) (*engine.EventGist, error) {
	fmt.Printf("[Bifrost] Generating Gist via %s\n", b.endpoint)
	return &engine.EventGist{
		Summary:  "Generated summary of the event",
		KeyFacts: []string{"Fact 1", "Fact 2"},
	}, nil
}

func (b *BifrostClient) MergeTraits(ctx context.Context, existing map[string]any, newContent string) (map[string]any, error) {
	// Simulated LLM Call #3 (YOLO Merge)
	fmt.Printf("[Bifrost] Merging YOLO traits via %s\n", b.endpoint)
	if existing == nil {
		existing = make(map[string]any)
	}
	existing["last_merged_note"] = "Processed by Bifrost"
	return existing, nil
}
