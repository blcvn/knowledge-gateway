package agentmemory

import (
    "time"
    "github.com/google/uuid"
)

type AgentMemory struct {
    ID                   string
    TenantID             string
    Project              string
    Type                 MemoryType
    Title                string
    Content              string
    Concepts             []string
    Files                []string
    SessionIDs           []string
    Strength             float64    // 0.0 - 1.0
    Version              int
    ParentID             string
    Supersedes           []string
    RelatedIDs           []string
    SourceObservationIDs []string
    IsLatest             bool
    ForgetAfter          *time.Time
    AgentID              string
    FlaggedEviction      bool
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

type MemorySlot struct {
    TenantID    string
    Project     string
    Scope       string    // "project" | "global"
    Label       string
    Content     string
    Description string
    SizeLimit   int
    Pinned      bool     // immune to eviction
    ReadOnly    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func NewAgentMemory(tenantID, project string, memType MemoryType, title, content string) AgentMemory {
    return AgentMemory{
        ID:       uuid.New().String(),
        TenantID: tenantID,
        Project:  project,
        Type:     memType,
        Title:    title,
        Content:  content,
        Strength: 0.7,
        Version:  1,
        IsLatest: true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
