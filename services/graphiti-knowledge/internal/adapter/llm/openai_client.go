package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
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

	// Pre-process the output: ensure it's a valid JSON payload.
	// Some models might wrap JSON in Markdown code blocks even when JSON mode is enforced.
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// Since we asked for an array, but JSON mode requires an object, 
	// the LLM might return `{"entities": [...]}`. 
	// We handle this loosely by checking if it's an object wrapping an array.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &raw); err == nil {
		// If there's exactly one key pointing to an array, extract it.
		if len(raw) == 1 {
			for _, val := range raw {
				return val, nil
			}
		}
	}

	// Fallback: return the raw content if it was a flat array (though strict JSON mode forbids this, 
	// compatible endpoints like Anthropic via translation proxy might return it).
	return []byte(content), nil
}
