package domain

import (
    "time"
    "github.com/google/uuid"
)

type Session struct {
    ID               string
    TenantID         string
    Project          string
    CWD              string
    Model            string
    AgentID          string
    Status           string  // "active" | "completed" | "abandoned"
    FirstPrompt      string
    Summary          string
    ObservationCount int
    Tags             []string
    CommitSHAs       []string
    StartedAt        time.Time
    EndedAt          *time.Time
    LastActiveAt     time.Time
}

type RawObservation struct {
    ID                string
    SessionID         string
    TenantID          string
    HookType          string
    ToolName          string
    ToolInput         []byte  // JSON
    ToolOutput        []byte  // JSON
    UserPrompt        string
    AssistantResponse string
    Modality          string  // "text" | "image"
    ImageData         string
    AgentID           string
    Raw               []byte  // full JSON payload
    Timestamp         time.Time
}

type CompressedObservation struct {
    ID         string
    SessionID  string
    TenantID   string
    ObsType    string
    Title      string
    Subtitle   string
    Facts      []string
    Narrative  string
    Concepts   []string
    Files      []string
    Importance float64
    Confidence float64
    ImageRef   string
    AgentID    string
    Timestamp  time.Time
}

func NewSession(tenantID, project, cwd, model, agentID string) Session {
    return Session{
        ID:           uuid.New().String(),
        TenantID:     tenantID,
        Project:      project,
        CWD:          cwd,
        Model:        model,
        AgentID:      agentID,
        Status:       "active",
        StartedAt:    time.Now(),
        LastActiveAt: time.Now(),
    }
}
