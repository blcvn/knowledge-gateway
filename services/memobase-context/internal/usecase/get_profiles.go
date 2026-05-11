package usecase

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/port"
)

type getProfilesUseCase struct {
	profileRepo repository.ProfileReadRepository
	cache       port.ProfileCache
	cacheTTL    int
}

func NewGetProfilesUseCase(profileRepo repository.ProfileReadRepository, cache port.ProfileCache, cacheTTL int) port.ProfileFetcher {
	return &getProfilesUseCase{
		profileRepo: profileRepo,
		cache:       cache,
		cacheTTL:    cacheTTL,
	}
}

func (u *getProfilesUseCase) GetProfiles(ctx context.Context, req *dto.GetProfilesRequest) (*dto.ProfilesResponse, error) {
	profiles, err := u.cache.GetProfiles(ctx, req.UserID, req.ProjectID)
	if err != nil || len(profiles) == 0 {
		profiles, err = u.profileRepo.GetProfiles(ctx, req.UserID, req.ProjectID)
		if err != nil {
			return nil, err
		}
		_ = u.cache.SetProfiles(ctx, req.UserID, req.ProjectID, profiles, u.cacheTTL)
	}

	policy := &model.TruncationPolicy{
		OnlyTopics:      req.Topics,
		MaxSubtopicSize: req.MaxSubtopicSize,
		MaxTokenSize:    req.MaxTokenSize,
	}
	truncated := TruncateProfiles(profiles, policy)

	return &dto.ProfilesResponse{Profiles: truncated}, nil
}
