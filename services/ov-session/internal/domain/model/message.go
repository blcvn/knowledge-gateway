package model

import (
	"time"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleTool      MessageRole = "tool"
)

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Message struct {
	ID         string
	SessionID  string
	Role       MessageRole
	Content    string
	ToolCalls  []ToolCall
	TokenCount int
	Sequence   int
	CreatedAt  time.Time
}
