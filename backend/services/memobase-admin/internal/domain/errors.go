package domain

import "errors"

var (
	ErrInvalidTokenFormat  = errors.New("invalid token format: must start with 'sk-proj-'")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrProjectSuspended    = errors.New("project is suspended")
	ErrProjectNotFound     = errors.New("project not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidProfileConfig = errors.New("invalid profile config")
)
