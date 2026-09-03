package ingestion

import (
	"time"
)

type EpisodeType string

const (
	EpisodeTypeMessage    EpisodeType = "message"
	EpisodeTypeJSON       EpisodeType = "json"
	EpisodeTypeText       EpisodeType = "text"
	EpisodeTypeFactTriple EpisodeType = "fact_triple"
)

type Episode struct {
	ID            EpisodeID   `json:"id"`
	GroupID       GroupID     `json:"group_id"`
	Type          EpisodeType `json:"type"`
	Content       string      `json:"content"`
	ContentHash   ContentHash `json:"content_hash"`
	ReferenceTime time.Time   `json:"reference_time"`
	CreatedAt     time.Time   `json:"created_at"`
}

type PipelineStep string

const (
	StepExtractEntities    PipelineStep = "EXTRACT_ENTITIES"
	StepResolveEntities    PipelineStep = "RESOLVE_ENTITIES"
	StepExtractEdges       PipelineStep = "EXTRACT_EDGES"
	StepResolveEdges       PipelineStep = "RESOLVE_EDGES"
	StepGenerateEmbeddings PipelineStep = "GENERATE_EMBEDDINGS"
	StepSaveBulk           PipelineStep = "SAVE_BULK"
	StepUpdateCommunity    PipelineStep = "UPDATE_COMMUNITY"
)

type SagaState string

const (
	SagaStateQueued     SagaState = "QUEUED"
	SagaStateProcessing SagaState = "PROCESSING"
	SagaStateCompleted  SagaState = "COMPLETED"
	SagaStateFailed     SagaState = "FAILED"
)

type Saga struct {
	ID           string       `json:"id"`
	EpisodeID    EpisodeID    `json:"episode_id"`
	GroupID      GroupID      `json:"group_id"`
	State        SagaState    `json:"state"`
	CurrentStep  PipelineStep `json:"current_step"`
	ErrorDetails string       `json:"error_details,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
