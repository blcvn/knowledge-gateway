package model

import "time"

type Agent struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	AccountID string         `json:"account_id"`
	Name      string         `json:"name"`
	Config    map[string]any `json:"config"`
	Status    string         `json:"status"` // active, disabled
	CreatedAt time.Time      `json:"created_at"`
}
