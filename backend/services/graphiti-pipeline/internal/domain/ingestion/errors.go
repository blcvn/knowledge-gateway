package ingestion

import "errors"

var (
	ErrDuplicateEpisode = errors.New("duplicate episode")
	ErrPipelineFailed   = errors.New("pipeline failed")
	ErrInvalidEpisode   = errors.New("invalid episode")
	ErrInvalidSaga      = errors.New("invalid saga")
)
