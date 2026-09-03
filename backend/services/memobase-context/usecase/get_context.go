package usecase

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/vnp-community/vnp-memory/services/memobase-context/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-context/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-context/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-context/usecase/port"
)

type getContextUseCase struct {
	profileFetcher      port.ProfileFetcher
	eventRepo           repository.EventGistSearchRepository
	defaultMaxTokenSize int32
	profileEventRatio   float32
	eventThreshold      float32
	eventWindowDays     int
	eventTopK           int
}

func NewGetContextUseCase(
	profileFetcher port.ProfileFetcher,
	eventRepo repository.EventGistSearchRepository,
	defaultMaxTokenSize int32,
	profileEventRatio float32,
	eventThreshold float32,
	eventWindowDays int,
	eventTopK int,
) port.ContextAssembler {
	return &getContextUseCase{
		profileFetcher:      profileFetcher,
		eventRepo:           eventRepo,
		defaultMaxTokenSize: defaultMaxTokenSize,
		profileEventRatio:   profileEventRatio,
		eventThreshold:      eventThreshold,
		eventWindowDays:     eventWindowDays,
		eventTopK:           eventTopK,
	}
}

func (u *getContextUseCase) GetContext(ctx context.Context, req *dto.GetContextRequest) (*dto.ContextResponse, error) {
	maxTokenSize := req.MaxTokenSize
	if maxTokenSize <= 0 {
		maxTokenSize = u.defaultMaxTokenSize
	}
	ratio := req.ProfileEventRatio
	if ratio <= 0 {
		ratio = u.profileEventRatio
	}

	profileBudget := int32(float32(maxTokenSize) * ratio)

	g, ctx := errgroup.WithContext(ctx)

	var profilesResp *dto.ProfilesResponse
	g.Go(func() error {
		var err error
		profilesResp, err = u.profileFetcher.GetProfiles(ctx, &dto.GetProfilesRequest{
			UserID:       req.UserID,
			ProjectID:    req.ProjectID,
			Topics:       req.OnlyTopics,
			MaxTokenSize: profileBudget,
		})
		return err
	})

	var events []*model.EventGist
	g.Go(func() error {
		var err error
		events, err = u.eventRepo.SearchBySimilarity(ctx, req.UserID, req.ProjectID, u.eventThreshold, u.eventWindowDays, u.eventTopK)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var sb strings.Builder
	var profileTokens int32 = 0
	for _, p := range profilesResp.Profiles {
		line := fmt.Sprintf("- %s::%s: %s\n", p.Topic, p.SubTopic, p.Content)
		sb.WriteString(line)
		profileTokens += estimateTokens(line)
	}
	profileSection := sb.String()

	eventBudget := maxTokenSize - profileTokens
	var eventTokens int32 = 0
	sb.Reset()
	var finalEvents []*model.EventGist
	for _, e := range events {
		line := fmt.Sprintf("- [%s] %s\n", e.CreatedAt.Format("2006-01-02"), e.GistData)
		t := estimateTokens(line)
		if eventTokens+t > eventBudget {
			break
		}
		sb.WriteString(line)
		eventTokens += t
		finalEvents = append(finalEvents, e)
	}
	eventSection := sb.String()

	pt := model.DefaultPromptTemplate()
	finalCtx := pt.Assemble(profileSection, eventSection)

	return &dto.ContextResponse{
		Context:      finalCtx,
		ProfileCount: int32(len(profilesResp.Profiles)),
		EventCount:   int32(len(finalEvents)),
		TotalTokens:  profileTokens + eventTokens,
	}, nil
}
