package domain

import "errors"

var (
    ErrMissingFields       = errors.New("session_id and hook_type are required")
    ErrSessionNotFound     = errors.New("session not found")
    ErrSessionLimitExceeded = errors.New("session observation limit exceeded (500)")
    ErrSessionEnded        = errors.New("session already ended")
)
