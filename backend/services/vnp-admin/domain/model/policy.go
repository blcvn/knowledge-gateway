// Package model defines core domain entities for vnp-admin.
package model

import (
	"time"

	"github.com/google/uuid"
)

// PolicyStatus represents the lifecycle status of a policy.
type PolicyStatus string

const (
	PolicyStatusActive   PolicyStatus = "active"
	PolicyStatusInactive PolicyStatus = "inactive"
	PolicyStatusDraft    PolicyStatus = "draft"
)

// Policy represents an OPA policy definition for governance.
type Policy struct {
	ID          uuid.UUID    `json:"id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	RegoCode    string       `json:"rego_code"`
	Scope       string       `json:"scope"`
	Status      PolicyStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// NewPolicy creates a new policy in draft status.
func NewPolicy(tenantID uuid.UUID, name, description, regoCode, scope string) *Policy {
	now := time.Now().UTC()
	return &Policy{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		RegoCode:    regoCode,
		Scope:       scope,
		Status:      PolicyStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Sentinel errors for policy operations.
var (
	ErrPolicyNotFound  = &DomainError{Code: "POLICY_NOT_FOUND", Message: "policy not found"}
	ErrDuplicatePolicy = &DomainError{Code: "DUPLICATE_POLICY", Message: "policy with this name already exists"}
)
