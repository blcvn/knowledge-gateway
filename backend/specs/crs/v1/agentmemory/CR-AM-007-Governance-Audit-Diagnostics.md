# Change Request: CR-AM-007 — Governance, Audit & Diagnostics

**CR ID:** CR-AM-007  
**Component:** `services/memory-service` [EXTEND] | `services/admin-service` [EXTEND] | `services/observe-service` [EXTEND]  
**Priority:** Medium  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.5, SRS FR-GOV-001..003, FR-DIAG-001..003  
**Spec:** `references/agentmemory/specs/services/admin-service/spec.md`

---

## 1. Mô tả

Triển khai **Governance, Audit & Diagnostics** layer:
1. **Cascade Delete** — xóa memory kéo theo xóa khỏi BM25, vector index, và graph nodes.
2. **Audit Trail** — mọi delete/forget operation tạo `AuditEntry` với 40+ operation types.
3. **Git Snapshots** — snapshot memory state vào Git định kỳ.
4. **Privacy & PII Redaction** — strip sensitive data patterns trước khi lưu.
5. **Health Monitor** — `/health` endpoint với `HealthSnapshot` struct.
6. **Circuit Breaker** — bảo vệ khỏi LLM provider failures.

---

## 2. Vấn đề hiện tại

- Không có cascade delete (xóa memory khỏi DB không xóa khỏi search indexes).
- Không có audit log cho delete operations.
- Không có privacy redaction (sensitive data lưu raw vào DB).
- `GET /health` hiện tại chỉ trả về `{status: "ok"}` — thiếu chi tiết.

---

## 3. Thay đổi đề xuất

### 3.1. Cascade Delete (Governance Delete)

```go
// services/memory-service/internal/usecase/governance.go

type GovernanceDeleteRequest struct {
    MemoryID   string `json:"memory_id"`
    Reason     string `json:"reason,omitempty"`
    PerformedBy string `json:"performed_by,omitempty"`
}

func (s *GovernanceService) Delete(ctx context.Context, req GovernanceDeleteRequest) error {
    // 1. Load memory metadata
    var mem AgentMemory
    if err := s.db.Get(ctx, req.MemoryID, &mem); err != nil {
        return err
    }
    
    // 2. Delete from PostgreSQL
    s.db.Delete(ctx, req.MemoryID)
    
    // 3. Remove from BM25 index (HTTP call to search-service)
    s.searchClient.RemoveIndex(ctx, req.MemoryID)
    
    // 4. Remove from Vector index (HTTP call to search-service)
    s.searchClient.RemoveVector(ctx, req.MemoryID)
    
    // 5. Remove graph edges (HTTP call to graph-service)
    s.graphClient.RemoveBySourceID(ctx, req.MemoryID)
    
    // 6. Create AuditEntry
    s.auditService.Record(ctx, AuditEntry{
        Operation:   "memory_governance_delete",
        TargetIDs:   []string{req.MemoryID},
        PerformedBy: req.PerformedBy,
        Reason:      reason,
        Timestamp:   time.Now(),
    })
    
    return nil
}
```

### 3.2. Audit Trail

```go
// services/memory-service/internal/domain/audit.go

type AuditEntry struct {
    ID           string    `json:"id"`
    Timestamp    time.Time `json:"timestamp"`
    Operation    string    `json:"operation"`   // see OperationTypes below
    FunctionID   string    `json:"function_id,omitempty"`
    TargetIDs    []string  `json:"target_ids"`
    PerformedBy  string    `json:"performed_by,omitempty"`
    Project      string    `json:"project,omitempty"`
    TenantID     string    `json:"tenant_id"`
    Details      map[string]any `json:"details,omitempty"`
    QualityScore float64   `json:"quality_score,omitempty"`  // for remember ops
    Reason       string    `json:"reason,omitempty"`
}

// Operation types (40+):
const (
    AuditObserve            = "observe"
    AuditRemember           = "remember"
    AuditSupersede          = "supersede"
    AuditForget             = "forget"
    AuditGovernanceDelete   = "governance_delete"
    AuditEvict              = "evict"
    AuditAutoForget         = "auto_forget"
    AuditCompress           = "compress"
    AuditSummarize          = "summarize"
    AuditConsolidate        = "consolidate"
    AuditSlotWrite          = "slot_write"
    AuditSlotDelete         = "slot_delete"
    AuditSessionStart       = "session_start"
    AuditSessionEnd         = "session_end"
    AuditSessionDelete      = "session_delete"
    AuditSearchQuery        = "search_query"
    AuditContextBuild       = "context_build"
    AuditSignalSend         = "signal_send"
    AuditLeaseAcquire       = "lease_acquire"
    AuditLeaseRelease       = "lease_release"
    AuditCheckpointCreate   = "checkpoint_create"
    AuditCheckpointResolve  = "checkpoint_resolve"
    AuditSnapshotCreate     = "snapshot_create"
    AuditImportTranscript   = "import_transcript"
    // ... total 40+ types
)
```

**New API Endpoints:**
```
GET  /v1/memory/audit                          # List audit entries
GET  /v1/memory/audit?operation=governance_delete&project=...
DELETE /v1/memory/agent/{id}/governance        # Cascade delete + audit
```

### 3.3. Privacy Redaction Package

```go
// pkg/privacy/redact.go
// Applied in observe-service pipeline step 3 (before KV write)

var sensitivePatterns = []redactPattern{
    {name: "bearer_token",  re: regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._-]{20,}`)},
    {name: "openai_key",    re: regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`)},
    {name: "aws_key",       re: regexp.MustCompile(`AKIA[A-Z0-9]{16}`)},
    {name: "private_key",   re: regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
    {name: "jwt_token",     re: regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)},
    {name: "github_pat",    re: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`)},
    {name: "env_secret",    re: regexp.MustCompile(`(?m)^[A-Z_]+=["']?[A-Za-z0-9+/=]{20,}["']?$`)},
}

func Strip(jsonStr string) string {
    for _, p := range sensitivePatterns {
        jsonStr = p.re.ReplaceAllStringFunc(jsonStr, func(match string) string {
            return "[REDACTED:" + p.name + "]"
        })
    }
    return jsonStr
}
```

### 3.4. Health Monitor

```go
// services/admin-service/internal/health/monitor.go

type HealthSnapshot struct {
    Status        string          `json:"status"`   // "healthy" | "degraded" | "critical"
    Timestamp     time.Time       `json:"timestamp"`
    UptimeSeconds float64         `json:"uptime_seconds"`
    Workers       []WorkerStatus  `json:"workers"`
    Memory        MemoryUsage     `json:"memory"`
    CPU           CPUUsage        `json:"cpu"`
    Indexes       IndexHealth     `json:"indexes"`
    Connections   ConnHealth      `json:"connections"`
    Alerts        []string        `json:"alerts,omitempty"`
}

type WorkerStatus struct {
    Name    string `json:"name"`
    Status  string `json:"status"`  // "running" | "stopped" | "error"
    LastRun time.Time `json:"last_run,omitempty"`
}

// GET /v1/health → aggregate health across all services
// Workers checked: observe-pipeline, consolidation-pipeline, lease-sweeper, 
//                  index-persistence, signal-sweeper
// Alerts when: memory > 80%, index lag > 5min, LLM circuit open
```

### 3.5. Git Snapshots (opt-in)

```go
// When AGENTMEMORY_SNAPSHOT_ENABLED=true:
// Triggered: after consolidation pipeline OR manually via API

type SnapshotMeta struct {
    CommitHash    string    `json:"commit_hash"`
    Stats         SnapshotStats `json:"stats"`
    CreatedAt     time.Time `json:"created_at"`
}

type SnapshotStats struct {
    Sessions     int `json:"sessions"`
    Observations int `json:"observations"`
    Memories     int `json:"memories"`
    GraphNodes   int `json:"graph_nodes"`
}

// POST /v1/admin/snapshot       # Create Git snapshot
// GET  /v1/admin/snapshots      # List snapshots
// GET  /v1/admin/snapshots/{h}/diff # Diff between two commits
```

### 3.6. Doctor Command (CLI)

```
GET /v1/admin/doctor → diagnostic report

Checks:
- Database connectivity (PostgreSQL)
- Search service connectivity (observe-search:8082)
- Graph service connectivity (graph-service:8084)
- Orchestration service connectivity (orchestration-service:8085)
- Index health (BM25 + vector load status)
- LLM provider status (circuit breaker state)
- NATS connectivity
- Data directory write access
- Port availability

Returns: [{check, status, message, suggestion}]
```

---

## 4. Acceptance Criteria

- [x] `DELETE /v1/memory/agent/{id}/governance` xóa memory khỏi: PostgreSQL, BM25 index, vector index (verify: subsequent search không trả về memory đó).
- [x] Mọi governance delete tạo `AuditEntry` với `operation: "governance_delete"` trong `audit` table.
- [x] `GET /v1/memory/audit?operation=governance_delete` trả về đúng list delete operations.
- [x] Hook payload chứa `sk-abc123456789` → sau pipeline: stored observation có `[REDACTED:openai_key]`.
- [x] `GET /v1/health` trả về `{status: "healthy", workers: [...], indexes: {...}}` với HTTP 200.
- [x] Khi PostgreSQL không kết nối được: `GET /v1/health` trả về `{status: "critical", alerts: ["db connection failed"]}`.
- [x] `GET /v1/admin/doctor` trả về diagnostic report đầy đủ với tất cả service checks.
- [x] Snapshot tạo Git commit với đúng stats (sessions, observations, memories count).
