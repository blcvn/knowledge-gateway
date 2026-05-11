package dto

type GetContextRequest struct {
	UserID            string
	ProjectID         string
	MaxTokenSize      int32
	PreferTopics      []string
	OnlyTopics        []string
	ProfileEventRatio float32
	ChatsContext      string
}

type GetProfilesRequest struct {
	UserID          string
	ProjectID       string
	Topics          []string
	MaxSubtopicSize int32
	MaxTokenSize    int32
}

type SearchProfilesRequest struct {
	UserID       string
	ProjectID    string
	Query        string
	Limit        int32
}
