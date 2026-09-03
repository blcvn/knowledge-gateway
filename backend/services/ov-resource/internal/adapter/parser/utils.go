package parser

import "strings"

func estimateTokens(text string) int {
	// A rough estimation: 1 token ≈ 4 characters
	return len(strings.TrimSpace(text)) / 4
}
