package model

import "errors"

// DomainError provides structured error codes for API responses.
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }

var (
	ErrTenantNotFound   = errors.New("tenant not found")
	ErrDuplicateTenant  = errors.New("tenant with this name already exists")
	ErrTenantInactive   = errors.New("tenant is inactive")
	ErrAPIKeyNotFound   = errors.New("api key not found")
	ErrAPIKeyRevoked    = errors.New("api key has been revoked")
	ErrAPIKeyInvalid    = errors.New("api key validation failed")
	ErrQuotaExceeded    = errors.New("tenant quota exceeded")
	ErrUserNotFound     = errors.New("user not found")
	ErrDuplicateUser    = errors.New("user with this email already exists in tenant")
	ErrAuditLogNotFound = errors.New("audit log not found")
)

