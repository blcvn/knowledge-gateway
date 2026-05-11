package llm

import "strings"

// ResponseParser extracts JSON from LLM markdown responses
type ResponseParser struct{}

func NewResponseParser() *ResponseParser {
	return &ResponseParser{}
}

func (p *ResponseParser) ParseJSON(markdown string) string {
	// Strip markdown code fences
	markdown = strings.TrimSpace(markdown)
	if strings.HasPrefix(markdown, "```json") {
		markdown = strings.TrimPrefix(markdown, "```json")
		markdown = strings.TrimSuffix(markdown, "```")
	} else if strings.HasPrefix(markdown, "```") {
		markdown = strings.TrimPrefix(markdown, "```")
		markdown = strings.TrimSuffix(markdown, "```")
	}
	return strings.TrimSpace(markdown)
}
