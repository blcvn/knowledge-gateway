package tokenizer

// ChatMessage is a single message in a conversation.
type ChatMessage struct {
	Role    string
	Content string
}

// ChatBlobData wraps a conversation for blob token counting.
type ChatBlobData struct {
	Messages []ChatMessage
}

// Tokenizer is the interface for counting and truncating tokens.
type Tokenizer interface {
	Count(text string) int
	CountMessages(messages []ChatMessage) int
	TruncateToTokens(text string, maxTokens int) string
	CountBlob(blobData any, blobType string) int
}
