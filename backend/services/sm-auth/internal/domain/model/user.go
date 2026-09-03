package model

import "time"

// User represents an authenticated user in the system.
type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	PasswordHash   string    `json:"-"` // Not exposed in JSON
	AuthProvider   string    `json:"auth_provider"`
	AuthProviderID string    `json:"auth_provider_id"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
