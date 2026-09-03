package domain

type PromptTemplate struct {
	ID           string
	SystemPrompt string
	UserPrompt   string
	Model        string
	MaxTokens    int
}

type ModelConfig struct {
	Provider string
	Model    string
	APIKey   string
}

type EmbeddingDimension int

type TokenUsage struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	Model            string `json:"model"`
}

func (t *TokenUsage) Add(other TokenUsage) {
	t.PromptTokens += other.PromptTokens
	t.CompletionTokens += other.CompletionTokens
	t.TotalTokens += other.TotalTokens
}
