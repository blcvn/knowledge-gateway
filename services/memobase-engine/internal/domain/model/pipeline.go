package model

// MergeDecision captures the YOLO merge result output by LLM #3.
type MergeDecision struct {
	Add    []MergeAdd    `json:"add"`
	Update []MergeUpdate `json:"update"`
	Delete []int         `json:"delete"`
}

// MergeAdd represents a new profile to be created.
type MergeAdd struct {
	Topic    string `json:"topic"`
	SubTopic string `json:"sub_topic"`
	Memo     string `json:"memo"`
}

// MergeUpdate represents an existing profile to be updated.
type MergeUpdate struct {
	Index int    `json:"index"`
	Memo  string `json:"memo"`
}

// PipelineResult represents the final outcome of processing a buffer.
type PipelineResult struct {
	ProfilesAdded   int
	ProfilesUpdated int
	ProfilesDeleted int
	EventsCreated   int
	TokensConsumed  int
}
