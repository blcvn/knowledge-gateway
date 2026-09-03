package dto

import "github.com/vnp-community/vnp-memory/services/memobase-context/domain/model"

type ContextResponse struct {
	Context      string
	ProfileCount int32
	EventCount   int32
	TotalTokens  int32
}

type ProfilesResponse struct {
	Profiles []*model.Profile
}

type SearchProfilesResponse struct {
	Profiles []*model.Profile
}
