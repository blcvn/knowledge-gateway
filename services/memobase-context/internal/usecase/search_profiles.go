package usecase

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/port"
)

type searchProfilesUseCase struct {
	profileRepo repository.ProfileReadRepository
	embedder    port.Embedder
}

func NewSearchProfilesUseCase(profileRepo repository.ProfileReadRepository, embedder port.Embedder) port.ProfileSearcher {
	return &searchProfilesUseCase{
		profileRepo: profileRepo,
		embedder:    embedder,
	}
}

func (u *searchProfilesUseCase) SearchProfiles(ctx context.Context, req *dto.SearchProfilesRequest) (*dto.SearchProfilesResponse, error) {
	emb, err := u.embedder.EmbedQuery(ctx, req.Query)
	if err != nil {
		return nil, err
	}
	profiles, err := u.profileRepo.SearchProfiles(ctx, req.UserID, req.ProjectID, emb, int(req.Limit))
	if err != nil {
		return nil, err
	}
	return &dto.SearchProfilesResponse{Profiles: profiles}, nil
}
