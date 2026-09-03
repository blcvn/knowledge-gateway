package model

import "time"

type WatchStatus string

const (
	WatchStatusActive  WatchStatus = "active"
	WatchStatusPaused  WatchStatus = "paused"
	WatchStatusDeleted WatchStatus = "deleted"
)

type EventType string

const (
	EventTypeCreated  EventType = "CREATED"
	EventTypeModified EventType = "MODIFIED"
	EventTypeDeleted  EventType = "DELETED"
)

type WatchTask struct {
	ID             string
	AccountID      string
	SourcePath     string
	TargetPath     string
	Patterns       []string
	PollIntervalMs int64
	Status         WatchStatus
	LastPollAt     time.Time
	FilesTracked   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WatchEvent struct {
	Type      EventType
	Path      string
	Timestamp time.Time
}
