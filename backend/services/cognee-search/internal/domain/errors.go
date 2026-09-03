package domain

import (
	"errors"
)

var (
	ErrStrategyNotFound = errors.New("search strategy not found")
	ErrEmptyQuery       = errors.New("query cannot be empty")
)
