package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/dto"
)

type ContextAssembler interface {
	GetContext(ctx context.Context, req *dto.GetContextRequest) (*dto.ContextResponse, error)
}

type ProfileFetcher interface {
	GetProfiles(ctx context.Context, req *dto.GetProfilesRequest) (*dto.ProfilesResponse, error)
}

type ProfileSearcher interface {
	SearchProfiles(ctx context.Context, req *dto.SearchProfilesRequest) (*dto.SearchProfilesResponse, error)
}
