package tokenizer

import (
	"fmt"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// TiktokenTokenizer implements Tokenizer using tiktoken-go (cl100k_base / o200k_base).
type TiktokenTokenizer struct {
	enc   *tiktoken.Tiktoken
	model string
}

// New creates a TiktokenTokenizer for the given model name.
// Example: "gpt-4o", "gpt-4", "gpt-3.5-turbo"
func New(model string) (*TiktokenTokenizer, error) {
	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return nil, fmt.Errorf("tokenizer: unsupported model %q: %w", model, err)
	}
	return &TiktokenTokenizer{enc: enc, model: model}, nil
}

// Count returns the number of tokens in text.
func (t *TiktokenTokenizer) Count(text string) int {
	if text == "" {
		return 0
	}
	return len(t.enc.Encode(text, nil, nil))
}

// CountMessages returns the total token count for a chat conversation.
// Follows OpenAI's message formatting overhead: 3 base + 4 per message.
func (t *TiktokenTokenizer) CountMessages(messages []ChatMessage) int {
	total := 3 // base overhead per conversation
	for _, msg := range messages {
		total += 4 // overhead per message
		total += t.Count(msg.Role)
		total += t.Count(msg.Content)
	}
	return total
}

// TruncateToTokens truncates text to at most maxTokens tokens,
// returning the decoded string of the truncated token list.
func (t *TiktokenTokenizer) TruncateToTokens(text string, maxTokens int) string {
	tokens := t.enc.Encode(text, nil, nil)
	if len(tokens) <= maxTokens {
		return text
	}
	return t.enc.Decode(tokens[:maxTokens])
}

// CountBlob counts tokens for a blob of a given type.
// Supported blobTypes: "chat", "doc", "summary".
func (t *TiktokenTokenizer) CountBlob(blobData any, blobType string) int {
	switch blobType {
	case "chat":
		if data, ok := blobData.(ChatBlobData); ok {
			return t.CountMessages(data.Messages)
		}
		if data, ok := blobData.(*ChatBlobData); ok {
			return t.CountMessages(data.Messages)
		}
		return 0
	case "doc", "summary":
		if s, ok := blobData.(string); ok {
			return t.Count(s)
		}
		return 0
	default:
		return 0
	}
}
