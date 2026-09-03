package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	pb "vnp-memory/services/graphiti-knowledge/adapter/grpc/pb"
)

// OpenAIClient implements the LLMClient interface for generating JSON outputs.
type OpenAIClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIClient initializes a new client targeting OpenAI APIs (or compatible like Groq/Ollama).
func NewOpenAIClient(apiKey string, model string) *OpenAIClient {
	if model == "" {
		model = openai.GPT4o // Default to latest GPT-4o
	}
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// GenerateJSON sends a prompt and guarantees a JSON-formatted response.
func (c *OpenAIClient) GenerateJSON(ctx context.Context, systemPrompt, userPrompt string) ([]byte, error) {
	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.1, // Low temperature for consistent, analytical extraction
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from LLM")
	}

	content := resp.Choices[0].Message.Content

	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &raw); err == nil {
		if len(raw) == 1 {
			for _, val := range raw {
				return val, nil
			}
		}
	}

	return []byte(content), nil
}

// EvaluateSimilarity evaluates a newly extracted entity against existing candidates to resolve duplicates.
func (c *OpenAIClient) EvaluateSimilarity(ctx context.Context, newEntity *pb.ExtractedEntity, candidates []*pb.ExtractedEntity) (*pb.Resolution, error) {
	// Construct the prompt
	systemPrompt := `You are an Entity Resolution Expert.
Determine if the NEW entity refers to the EXACT SAME real-world concept as one of the CANDIDATE entities.
Output a JSON object with:
- 'decision': strictly one of ["MERGE", "CREATE_NEW"]
- 'existing_entity_name': the name of the matched candidate (only if decision is MERGE)
- 'confidence': float between 0.0 and 1.0`

	var candidatesStr string
	for i, cand := range candidates {
		candidatesStr += fmt.Sprintf("[%d] Name: %s, Label: %s, Summary: %s\n", i, cand.Name, cand.Label, cand.Summary)
	}

	userPrompt := fmt.Sprintf("NEW ENTITY:\nName: %s, Label: %s, Summary: %s\n\nCANDIDATES:\n%s", newEntity.Name, newEntity.Label, newEntity.Summary, candidatesStr)

	responseBytes, err := c.GenerateJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var decision struct {
		Decision           string  `json:"decision"`
		ExistingEntityName string  `json:"existing_entity_name"`
		Confidence         float64 `json:"confidence"`
	}
	if err := json.Unmarshal(responseBytes, &decision); err != nil {
		return nil, err
	}

	res := &pb.Resolution{
		ExtractedEntity: newEntity,
		Decision:        decision.Decision,
		Confidence:      decision.Confidence,
	}

	if decision.Decision == "MERGE" {
		res.ExistingEntityId = decision.ExistingEntityName // In Neo4j we're using Name as matching ID for now
	}

	return res, nil
}
