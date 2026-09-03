package model

import "time"

// APIKeyRole defines the role assigned to an API key
type APIKeyRole string

const (
	RoleRoot  APIKeyRole = "root"
	RoleAdmin APIKeyRole = "admin"
	RoleUser  APIKeyRole = "user"
	RoleAgent APIKeyRole = "agent"
)

// APIKeyStatus defines the state of the API Key
type APIKeyStatus string

const (
	APIKeyStatusActive  APIKeyStatus = "active"
	APIKeyStatusRevoked APIKeyStatus = "revoked"
)

// APIKeyHash represents a hashed API key for authentication
type APIKeyHash struct {
	ID         string
	AccountID  string
	UserID     string
	KeyHash    []byte
	KeyPrefix  string
	Role       APIKeyRole
	Label      string
	Status     APIKeyStatus
	LastUsedAt *time.Time
	CreatedAt  time.Time
	ExpiresAt  *time.Time
}
