package domain

import "time"

type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

type EpisodeIngested struct {
	EpisodeID  string
	GroupID    string
	NodesCount int
	EdgesCount int
	Timestamp  time.Time
}

func (e EpisodeIngested) EventName() string     { return "graphiti.episode.ingested" }
func (e EpisodeIngested) OccurredAt() time.Time { return e.Timestamp }

type EpisodeFailed struct {
	EpisodeID string
	GroupID   string
	Reason    string
	Timestamp time.Time
}

func (e EpisodeFailed) EventName() string     { return "graphiti.episode.failed" }
func (e EpisodeFailed) OccurredAt() time.Time { return e.Timestamp }

type SagaStepCompleted struct {
	SagaID    string
	Step      PipelineStep
	Timestamp time.Time
}

func (e SagaStepCompleted) EventName() string     { return "graphiti.saga.step_completed" }
func (e SagaStepCompleted) OccurredAt() time.Time { return e.Timestamp }
