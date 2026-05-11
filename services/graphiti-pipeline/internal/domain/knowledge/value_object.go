package knowledge

import "errors"

type PromptTemplate struct {
	Name     string `json:"name"`
	Template string `json:"template"`
}

func (p PromptTemplate) Validate() error {
	if p.Name == "" || p.Template == "" {
		return errors.New("prompt template name and template cannot be empty")
	}
	return nil
}

type TokenUsage struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Model            string `json:"model"`
}

type ModelConfig struct {
	ModelName   string  `json:"model_name"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}
