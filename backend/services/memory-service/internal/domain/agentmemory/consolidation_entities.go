package agentmemory

import "time"

type SessionSummary struct {
    SessionID        string
    TenantID         string
    Title            string
    Narrative        string
    KeyDecisions     []string
    FilesModified    []string
    Concepts         []string
    ObservationCount int
    CreatedAt        time.Time
}

type ProceduralMemory struct {
    ID               string
    TenantID         string
    Project          string
    Name             string
    Steps            []string
    StepHash         string    // for dedup
    TriggerCondition string
    ExpectedOutcome  string
    Frequency        int
    Confidence       float64
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type Lesson struct {
    ID          string
    TenantID    string
    Project     string
    Content     string
    Confidence  float64
    Source      string
    Categories  []string
    AccessCount int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Insight struct {
    ID         string
    TenantID   string
    Content    string
    LessonIDs  []string
    Confidence float64
    CreatedAt  time.Time
}

type RawObs struct {
    ID                string
    SessionID         string
    TenantID          string
    HookType          string
    ToolName          string
    ToolInput         []byte
    ToolOutput        []byte
    UserPrompt        string
    AssistantResponse string
}

type CompressedObs struct {
    ID         string
    SessionID  string
    ObsType    string
    Title      string
    Subtitle   string
    Facts      []string
    Narrative  string
    Concepts   []string
    Files      []string
    Importance float64
    Confidence float64
}
