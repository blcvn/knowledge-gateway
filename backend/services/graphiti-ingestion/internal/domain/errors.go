package domain

import "errors"

var (
	ErrInvalidHash       = errors.New("invalid content hash")
	ErrEpisodeNotFound   = errors.New("episode not found")
	ErrDuplicateEpisode  = errors.New("duplicate episode detected")
	ErrInvalidSagaState  = errors.New("invalid saga state transition")
	ErrSagaNotFound      = errors.New("saga not found")
	ErrExtractionFailed  = errors.New("entity/edge extraction failed")
	ErrResolutionFailed  = errors.New("entity/edge resolution failed")
	ErrUpdateFailed      = errors.New("community update failed")
	ErrStoreFailed       = errors.New("store bulk save failed")
)
