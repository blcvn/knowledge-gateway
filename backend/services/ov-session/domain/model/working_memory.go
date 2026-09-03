package model

import (
	"time"
)

type WMState string

const (
	WMStateOngoing   WMState = "ongoing"
	WMStatePaused    WMState = "paused"
	WMStateCompleted WMState = "completed"
)

type Fact struct {
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

type ErrorState struct {
	Message  string `json:"message"`
	Resolved bool   `json:"resolved"`
}

type WorkingMemory struct {
	SessionID string                 `json:"session_id"`
	Title     string                 `json:"title"`
	State     WMState                `json:"state"`
	Goals     []string               `json:"goals"`
	Facts     []Fact                 `json:"facts"`
	Errors    []ErrorState           `json:"errors"`
	Context   map[string]interface{} `json:"context"`
	UpdatedAt time.Time              `json:"updated_at"`
}
