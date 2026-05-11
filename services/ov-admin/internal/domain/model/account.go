package model

import "time"

type NamespacePolicy struct {
	MaxUsers       int `json:"max_users"`
	StorageQuotaMB int `json:"storage_quota_mb"`
}

type Account struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	NamespacePolicy NamespacePolicy `json:"namespace_policy"`
	Status          string          `json:"status"` // active, suspended, deleted
	Metadata        map[string]any  `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
