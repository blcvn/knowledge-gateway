---
id: MERGE-P1-T3
title: "vnp-platform: Absorb vnp-event + vnp-dashboard + ov-admin + zep-admin + sm-analytics + sm-project"
phase: P1
service: vnp-platform
priority: P0
status: Done
estimated: 6h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P1-T2]
---

## Mục Tiêu

Absorb 6 services còn lại vào `vnp-platform` để hoàn thiện Platform domain. Tất cả đều là stub (no routes), chỉ cần implement domain + usecase + wire routes.

**Services bị absorb:**
| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `vnp-event` | 985 | NATS event streaming |
| `vnp-dashboard` | 228 | Dashboard metrics aggregation |
| `ov-admin` | 965 | OpenViking Account + Agent mgmt |
| `zep-admin` | 308 | Zep Project/session admin |
| `sm-analytics` | 127 | Analytics aggregation |
| `sm-project` | 361 | Supermemory Space management |

## Thay Đổi Cần Thực Hiện

### A. vnp-event → `domain/event/` + `usecase/event/`

```go
// domain/event/entity.go
type Event struct {
    ID        string
    Subject   string    // NATS subject
    TenantID  string
    Payload   []byte
    CreatedAt time.Time
}

// usecase/event/service.go
type EventUseCase interface {
    Publish(ctx context.Context, subject string, payload any) error
    Subscribe(ctx context.Context, subject string, handler EventHandler) error
    ListEvents(ctx context.Context, tenantID string, filter EventFilter) ([]*Event, int, error)
}
```

**ForwardService Routes (event):**
```go
router.Handle("GET",  "/v1/memobase/users/*/events",  eventForward.GetUserEvents)
```

**NATS Subjects exposed:**
```
vnp.events.memory.*       → memory operations
vnp.events.tenant.*       → tenant operations
vnp.events.auth.*         → auth operations
```

---

### B. vnp-dashboard → `domain/dashboard/` + `usecase/dashboard/`

```go
// domain/dashboard/entity.go
type DashboardMetrics struct {
    TotalMemories    int64
    ActiveSessions   int64
    RequestsPerMin   float64
    StorageUsedBytes int64
    LastUpdatedAt    time.Time
}

type ThroughputPoint struct {
    Timestamp time.Time
    RPM       float64
    ErrorRate float64
}

// usecase/dashboard/service.go
type DashboardUseCase interface {
    GetHealth(ctx context.Context) (*HealthStatus, error)
    GetMetrics(ctx context.Context) (*DashboardMetrics, error)
    GetThroughput(ctx context.Context, window time.Duration) ([]*ThroughputPoint, error)
    GetHeatmap(ctx context.Context, from, to time.Time) (*Heatmap, error)
}
```

**ForwardService Routes (dashboard):**
```go
router.Handle("GET", "/v1/console/dashboard/health",     dashboardForward.Health)
router.Handle("GET", "/v1/console/dashboard/metrics",    dashboardForward.Metrics)
router.Handle("GET", "/v1/console/dashboard/throughput", dashboardForward.Throughput)
router.Handle("GET", "/v1/console/dashboard/heatmap",    dashboardForward.Heatmap)
```

---

### C. ov-admin → `domain/account/` (OpenViking Account + Agent)

```go
// domain/account/entity.go
type Account struct {
    ID        string
    TenantID  string
    Name      string
    Email     string
    Plan      string
    CreatedAt time.Time
}

type Agent struct {
    ID        string
    AccountID string
    Name      string
    Type      string  // "assistant" | "worker" | "indexer"
    Config    map[string]any
}

// usecase/account/service.go — minimal: CRUD for accounts + agents
// Routes: all under /v1/admin/* via AdminHandler (already registered)
// No additional routes needed — ov-admin functionality folded into admin domain
```

---

### D. zep-admin → `domain/zepproject/` (Zep Project management)

```go
// domain/zepproject/entity.go
type ZepProject struct {
    ID        string
    TenantID  string
    Name      string
    Settings  map[string]any
    CreatedAt time.Time
}

// Zep project management goes through /v1/admin/* or /v1/console/governance/*
// No additional gateway routes needed — maps to existing governance routes
```

---

### E. sm-analytics → `domain/analytics/` + `usecase/analytics/`

```go
// domain/analytics/entity.go
type AnalyticsReport struct {
    TenantID     string
    Period       string
    MemoryCount  int64
    SearchCount  int64
    IngestCount  int64
    TopTopics    []TopicStat
    GeneratedAt  time.Time
}

// usecase/analytics/service.go
type AnalyticsUseCase interface {
    GetReport(ctx context.Context, tenantID string, period string) (*AnalyticsReport, error)
    GetForgetRules(ctx context.Context, tenantID string) ([]*ForgetRule, error)
    UpdateForgetRules(ctx context.Context, tenantID string, rules []*ForgetRule) error
}
```

**ForwardService Routes (analytics):**
```go
router.Handle("GET", "/v1/console/adaptive/analytics",     analyticsForward.GetAnalytics)
router.Handle("GET", "/v1/console/adaptive/forget-rules",  analyticsForward.GetForgetRules)
router.Handle("PUT", "/v1/console/adaptive/forget-rules",  analyticsForward.UpdateForgetRules)
```

---

### F. sm-project → `domain/space/` + `usecase/space/`

```go
// domain/space/entity.go
type Space struct {
    ID        string
    TenantID  string
    Name      string
    Config    map[string]any
    CreatedAt time.Time
}

// usecase/space/service.go
type SpaceUseCase interface {
    CreateSpace(ctx context.Context, tenantID, name string, config map[string]any) (*Space, error)
    GetSpace(ctx context.Context, id string) (*Space, error)
    ListSpaces(ctx context.Context, tenantID string) ([]*Space, error)
}
```

**ForwardService Routes (space):**
```go
router.Handle("POST", "/v1/sm/projects/spaces", spaceForward.CreateSpace)
```

---

### G. Profile + Debugger + Session Console Routes

Trong task này cũng wire các console routes còn thiếu:

```go
// Profile routes (từ memobase-context data)
router.Handle("GET", "/v1/console/profiles",              profileForward.ListProfiles)
router.Handle("GET", "/v1/console/profiles/config",       profileForward.GetConfig)
router.Handle("PUT", "/v1/console/profiles/config",       profileForward.UpdateConfig)
router.Handle("GET", "/v1/console/profiles/*",            profileForward.GetProfile)
router.Handle("GET", "/v1/console/profiles/*/events",     profileForward.GetEvents)
router.Handle("GET", "/v1/console/profiles/*/context",    profileForward.GetContext)
router.Handle("GET", "/v1/console/profiles/*/buffers",    profileForward.GetBuffers)

// Debugger routes
router.Handle("POST", "/v1/console/debugger/trace",       debuggerForward.CreateTrace)
router.Handle("GET",  "/v1/console/debugger/traces/*",    debuggerForward.GetTrace)
router.Handle("GET",  "/v1/console/debugger/traces",      debuggerForward.ListTraces)

// Session routes
router.Handle("GET", "/v1/console/sessions",               sessionForward.ListSessions)
router.Handle("GET", "/v1/console/sessions/live",          sessionForward.ListLiveSessions)
router.Handle("GET", "/v1/console/sessions/*",             sessionForward.GetSession)
router.Handle("GET", "/v1/console/sessions/*/timeline",    sessionForward.GetTimeline)
router.Handle("GET", "/v1/console/sessions/*/diff",        sessionForward.GetDiff)
router.Handle("GET", "/v1/console/sessions/*/working-memory", sessionForward.GetWorkingMemory)
router.Handle("GET", "/v1/console/sessions/*/user-summary",   sessionForward.GetUserSummary)
```

## Database Migrations

```sql
-- migrations/004_platform_analytics.sql
CREATE TABLE IF NOT EXISTS vnp_spaces (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES vnp_tenants(id),
    name       TEXT NOT NULL,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vnp_analytics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    period      TEXT NOT NULL,
    data        JSONB NOT NULL DEFAULT '{}',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vnp_forget_rules (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rules     JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Acceptance Criteria

- [ ] `GET /v1/console/dashboard/health` returns `{"status":"ok"}`
- [ ] `GET /v1/console/dashboard/metrics` returns metrics JSON
- [ ] `POST /v1/sm/projects/spaces` tạo space trong PostgreSQL
- [ ] `GET /v1/console/adaptive/analytics` returns analytics report
- [ ] `GET /v1/console/governance/audit` returns audit log entries
- [ ] Tất cả session console routes trả về valid JSON (có thể empty list)
- [ ] Tất cả debugger console routes trả về valid JSON
- [ ] `go build ./services/vnp-platform/...` passes
- [ ] Unit tests cho từng usecase pass

## Ghi Chú

- Dashboard metrics lấy từ PostgreSQL aggregate queries — không cần Prometheus cho MVP
- debugger + session routes: implement với `stub` responses trước nếu data chưa có
- Tất cả 6 services gốc giữ nguyên folder cho đến P4 cleanup
- Console routes từ `vnp-platform` cần forward đến chính `vnp-platform` (self-forward hoặc direct handler)
