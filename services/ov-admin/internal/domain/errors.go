package domain

import "errors"

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrAgentNotFound   = errors.New("agent not found")
	ErrDuplicateUser   = errors.New("duplicate user")
	ErrInvalidKey      = errors.New("invalid api key")
	ErrPermissionDenied = errors.New("permission denied")
)
