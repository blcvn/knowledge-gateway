package model

import "time"

type Role string

const (
	RoleRoot  Role = "ROOT"
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
	RoleAgent Role = "AGENT"
)

type User struct {
	ID        string         `json:"id"`
	AccountID string         `json:"account_id"`
	Name      string         `json:"name"`
	Role      Role           `json:"role"`
	Status    string         `json:"status"` // active, suspended
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
