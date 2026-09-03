package port

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/usecase/dto"
)

// BufferProcessor is the primary driving port for the processing pipeline.
type BufferProcessor interface {
	ProcessBuffer(ctx context.Context, req dto.ProcessBufferRequest) (dto.ProcessBufferResponse, error)
}

// ProfileExtractor is a driving port for extracting profiles from text.
type ProfileExtractor interface {
	ExtractProfile(ctx context.Context, req dto.ExtractProfileRequest) (dto.ExtractProfileResponse, error)
}

// ProfileMerger is a driving port for merging new facts with existing profiles.
type ProfileMerger interface {
	MergeProfile(ctx context.Context, req dto.MergeProfileRequest) (dto.MergeProfileResponse, error)
}

// EventProcessor is a driving port for processing and structuring user events.
type EventProcessor interface {
	ProcessEvent(ctx context.Context, req dto.ProcessEventRequest) error
}
