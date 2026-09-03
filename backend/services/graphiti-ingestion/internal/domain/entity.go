package domain

import "time"

type EpisodeType string

const (
	SourceChat EpisodeType = "chat"
	SourceText EpisodeType = "text"
	SourceAPI  EpisodeType = "api"
)

type PipelineStep string

const (
	StepExtractEntities PipelineStep = "extract_entities"
	StepResolveEntities PipelineStep = "resolve_entities"
	StepExtractEdges    PipelineStep = "extract_edges"
	StepResolveEdges    PipelineStep = "resolve_edges"
	StepUpdateCommunity PipelineStep = "update_community"
	StepSaveBulk        PipelineStep = "save_bulk"
	StepCompleted       PipelineStep = "completed"
)

type SagaStatus string

const (
	SagaStatusPending   SagaStatus = "pending"
	SagaStatusRunning   SagaStatus = "running"
	SagaStatusCompleted SagaStatus = "completed"
	SagaStatusFailed    SagaStatus = "failed"
	SagaStatusRollback  SagaStatus = "rollback"
)

type StepEntry struct {
	Step       PipelineStep
	Status     SagaStatus
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

type Episode struct {
	UUID          string            `json:"uuid"`
	Name          string            `json:"name"`
	GroupID       string            `json:"group_id"`
	Body          string            `json:"body"`
	Source        EpisodeType       `json:"source"`
	ReferenceTime time.Time         `json:"reference_time"`
	ContentHash   string            `json:"content_hash"`
	SagaID        *string           `json:"saga_id,omitempty"`
	EntityTypes   map[string]string `json:"entity_types,omitempty"`
	EdgeTypes     map[string]string `json:"edge_types,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

func (e *Episode) Validate() error {
	if e.Name == "" {
		return ErrInvalidEpisode("name is required")
	}
	if e.Body == "" {
		return ErrInvalidEpisode("body is required")
	}
	if e.GroupID == "" {
		return ErrInvalidEpisode("group_id is required")
	}
	if e.ReferenceTime.IsZero() {
		return ErrInvalidEpisode("reference_time is required")
	}
	return nil
}

type SagaState struct {
	ID           string       `json:"id"`
	EpisodeID    string       `json:"episode_id"`
	GroupID      string       `json:"group_id"`
	CurrentStep  PipelineStep `json:"current_step"`
	Status       SagaStatus   `json:"status"`
	StepHistory  []StepEntry  `json:"step_history"`
	RetryCount   int          `json:"retry_count"`
	ErrorMessage string       `json:"error_message,omitempty"`
	StartedAt    time.Time    `json:"started_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
}

func (s *SagaState) Transition(nextStep PipelineStep, status SagaStatus, errMsg string) error {
	if s.Status == SagaStatusCompleted || s.Status == SagaStatusRollback {
		return ErrInvalidSagaTransition("saga is already in a terminal state")
	}

	entry := StepEntry{
		Step:       s.CurrentStep,
		Status:     s.Status,
		FinishedAt: time.Now(),
	}
	if s.Status == SagaStatusFailed {
		entry.Error = s.ErrorMessage
	}
	s.StepHistory = append(s.StepHistory, entry)

	s.CurrentStep = nextStep
	s.Status = status
	if errMsg != "" {
		s.ErrorMessage = errMsg
	}
	if status == SagaStatusCompleted || status == SagaStatusRollback {
		now := time.Now()
		s.CompletedAt = &now
	}
	return nil
}
