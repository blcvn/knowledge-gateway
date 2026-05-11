package model

import "time"

type APIKey struct {
	KeyID     string     `json:"key_id"`
	AccountID string     `json:"account_id"`
	UserID    string     `json:"user_id"`
	Role      Role       `json:"role"`
	Label     string     `json:"label"`
	Prefix    string     `json:"prefix"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type ValidateResult struct {
	Valid     bool   `json:"valid"`
	AccountID string `json:"account_id"`
	UserID    string `json:"user_id"`
	Role      Role   `json:"role"`
	AgentID   string `json:"agent_id"`
}
