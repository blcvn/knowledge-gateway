package agentmemory

import (
    "time"
    "github.com/google/uuid"
)

type AuditEntry struct {
    ID          string
    Timestamp   time.Time
    Operation   string
    TargetIDs   []string
    PerformedBy string
    Project     string
    TenantID    string
    Details     map[string]any
    Reason      string
}

// Operation constants — 25 types
const (
    AuditObserve           = "observe"
    AuditRemember          = "remember"
    AuditSupersede         = "supersede"
    AuditForget            = "forget"
    AuditGovernanceDelete  = "governance_delete"
    AuditEvict             = "evict"
    AuditAutoForget        = "auto_forget"
    AuditCompress          = "compress"
    AuditSummarize         = "summarize"
    AuditConsolidate       = "consolidate"
    AuditSlotWrite         = "slot_write"
    AuditSlotDelete        = "slot_delete"
    AuditSessionStart      = "session_start"
    AuditSessionEnd        = "session_end"
    AuditSessionDelete     = "session_delete"
    AuditImportTranscript  = "import_transcript"
    AuditSearchQuery       = "search_query"
    AuditContextBuild      = "context_build"
    AuditSignalSend        = "signal_send"
    AuditLeaseAcquire      = "lease_acquire"
    AuditLeaseRelease      = "lease_release"
    AuditCheckpointCreate  = "checkpoint_create"
    AuditCheckpointResolve = "checkpoint_resolve"
    AuditSnapshotCreate    = "snapshot_create"
    AuditDecaySweep        = "decay_sweep"
)

func NewAuditEntry(tenantID, operation string, targetIDs []string) AuditEntry {
    return AuditEntry{
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Operation: operation,
        TargetIDs: targetIDs,
        TenantID:  tenantID,
    }
}
