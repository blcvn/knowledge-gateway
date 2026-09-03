package domain

import "time"

type Action struct {
    ID           string
    TenantID     string
    Project      string
    AgentID      string
    Title        string
    Description  string
    Status       ActionStatus
    Priority     int
    Requires     []string  // action IDs
    ConflictsWith []string
    Tags         []string
    Result       string
    CreatedAt    time.Time
    UpdatedAt    time.Time
    CompletedAt  *time.Time
}

type Lease struct {
    ID         string
    ActionID   string
    AgentID    string
    Status     string    // "active" | "expired" | "released"
    AcquiredAt time.Time
    ExpiresAt  time.Time
    RenewedAt  *time.Time
}

type Signal struct {
    ID         string
    TenantID   string
    FromAgent  string
    ToAgent    string
    SignalType string    // handoff|update|cancel|request|response|alert
    Content    string
    ThreadID   string
    ReplyTo    string
    IsRead     bool
    ExpiresAt  time.Time
    CreatedAt  time.Time
}

type Routine struct {
    ID          string
    TenantID    string
    Project     string
    Name        string
    Description string
    Steps       []string  // JSONB: ordered step definitions
    CreatedAt   time.Time
}

type Checkpoint struct {
    ID          string
    TenantID    string
    Project     string
    AgentID     string
    ActionID    string
    Title       string
    Description string
    Status      string    // "pending" | "approved" | "rejected" | "expired"
    ApprovedBy  string
    RejectedBy  string
    Reason      string
    ExpiresAt   time.Time
    CreatedAt   time.Time
    ResolvedAt  *time.Time
}

type Sentinel struct {
    ID          string
    TenantID    string
    Name        string
    Condition   SentinelCondition
    ActionID    string
    SignalTo    string
    Status      string    // "watching" | "triggered" | "expired"
    ExpiresAt   time.Time
    CreatedAt   time.Time
    TriggeredAt *time.Time
}

type SentinelCondition struct {
    Type   string  // "action_done" | "signal_received" | "time"
    Target string
    Value  string
}

type Sketch struct {
    ID        string
    TenantID  string
    Project   string
    SessionID string
    Title     string
    ActionIDs []string
    Status    string    // "active" | "promoted" | "expired"
    ExpiresAt time.Time
    CreatedAt time.Time
}

type Crystal struct {
    ID              string
    TenantID        string
    SourceActionIDs []string
    Narrative       string
    KeyOutcomes     []string
    FilesAffected   []string
    Lessons         []string
    CreatedAt       time.Time
}
