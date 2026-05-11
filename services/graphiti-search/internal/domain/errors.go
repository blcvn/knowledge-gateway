package domain

import "errors"

var (
	ErrNoResults        = errors.New("no results found")
	ErrInvalidQuery     = errors.New("invalid search query")
	ErrCacheUnavailable = errors.New("cache is unavailable")
)
