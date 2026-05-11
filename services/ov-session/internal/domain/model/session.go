package model

import (
	"time"
)

type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCommitted SessionStatus = "committed"
	SessionStatusArchived  SessionStatus = "archived"
)

type Session struct {
	ID                 string
	AccountID          string
	UserID             string
	AgentID            string
	Title              string
	Status             SessionStatus
	ArchivePath        string
	MemoriesCount      int
	CompressionVersion string
	Metadata           map[string]interface{}
	CreatedAt          time.Time
	CommittedAt        *time.Time
}

type SessionMeta struct {
	ID        string
	Title     string
	Status    SessionStatus
	CreatedAt time.Time
}
