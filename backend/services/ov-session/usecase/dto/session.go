package dto

import (
	"github.com/vnp-memory/services/ov-session/domain/model"
)

type CreateSessionReq struct {
	AccountID string
	UserID    string
	AgentID   string
	Title     string
	Metadata  map[string]interface{}
}

type AddMessageReq struct {
	SessionID  string
	Role       model.MessageRole
	Content    string
	ToolCalls  []model.ToolCall
	TokenCount int
}

type GetMessagesReq struct {
	SessionID string
}
