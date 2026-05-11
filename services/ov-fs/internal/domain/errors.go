package domain

import "errors"

var (
	ErrPathNotFound      = errors.New("path not found")
	ErrPathAlreadyExists = errors.New("path already exists")
	ErrLockContention    = errors.New("lock contention")
	ErrInvalidPath       = errors.New("invalid path")
	ErrPermissionDenied  = errors.New("permission denied")
)
