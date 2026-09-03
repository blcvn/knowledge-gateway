# SOL-006 — Solution: Adaptive Memory, Profiles, Governance, Observability, Pipelines, Infrastructure, Org & SDK (CR-005 → CR-011)

| Field | Value |
|---|---|
| **Solution ID** | SOL-006 |
| **CRs** | CR-005, CR-006, CR-007, CR-008, CR-009, CR-010, CR-011 |
| **Architecture ref** | §2.2 Services Inventory · §4.2 Console Routes · §5.2 Domain Entities · §6.4 NATS · §7 Tech Stack |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Implemented** | 2026-06-17 |

---

## CR-005 — Adaptive Memory (Supermemory Engine)

### Architecture mapping

Theo §2.2: Supermemory gồm 9 services:
`sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-mcp, sm-auth, sm-analytics, sm-project`

Theo §5.2 Supermemory Domain:
```go
SMMemory   — ID, TenantID, Content, Tags[], Embedding
SMDocument — ID, TenantID, Title, Content, Type, URL
SMProfile  — UserID, TenantID, Memories[], Stats
```

Console route `FEAT-009` ↔ `/v1/console/adaptive/*`

### Backend Solution

```go
type ConsoleAdaptiveHandler struct {
    smMemory    SMMemoryClient     // gRPC → sm-memory (memories, versions)
    smConnector SMConnectorClient  // gRPC → sm-connector (external connectors)
    smAnalytics SMAnalyticsClient  // gRPC → sm-analytics (stats)
    nats        NATSClient         // trigger sync jobs
}

// GET /v1/console/adaptive/memories
// → sm-memory.ListMemories(tenantID, isLatest=true)
// Response: []AdaptiveMemory

// GET /v1/console/adaptive/memories/{id}/versions
// → sm-memory.GetVersionChain(id, tenantID)
// Response: []MemoryVersion (parent_id → root_id chain)

// GET /v1/console/adaptive/connectors
// → sm-connector.ListConnectors(tenantID)
// type ExternalConnector: google_drive | gmail | notion | onedrive | github

// POST /v1/console/adaptive/connectors/{id}/sync
// → Publish NATS: sm.connector.sync.{id}
// sm-connector picks up → triggers ingest job
// NATS event §6.4: sm.connector.synced on completion

// GET /v1/console/adaptive/analytics
// → sm-analytics.GetStats(tenantID)
// Response: AdaptiveAnalytics{creation_rate, deletion_rate, contradiction_count, ...}

// GET/PUT /v1/console/adaptive/forget-rules
// → sm-memory.GetForgetRules / SetForgetRules
// ForgetRule: {memory_type, forget_after, noise_filter, contradiction_resolution}
```

---

## CR-006 — User Profiles (Memobase Engine)

### Architecture mapping

Theo §2.2 + §5.2 Memobase Domain:
```go
UserContext — UserID, TenantID, Summary, Profiles[], Events[], Tokens
Profile     — Key, Value, Category(preference|fact|goal|habit), Score
Buffer      — UserID, Blobs[], TokenCount, FlushThreshold(default:20)
```

Services: `memobase-ingestion, memobase-engine, memobase-context`
Console route `FEAT-008` ↔ `/v1/console/profiles/*`

### Backend Solution

```go
type ConsoleProfilesHandler struct {
    memoCtx    MemobaseContextClient  // gRPC → memobase-context
    memoEngine MemobaseEngineClient   // gRPC → memobase-engine
    eventSvc   VNPEventClient         // gRPC → vnp-event
}

// GET /v1/console/profiles
// → memobase-engine.ListUsers(tenantID) [PostgreSQL]
// Response: []UserProfile (user_id, created_at, profiles=[])

// GET /v1/console/profiles/{user_id}
// → memobase-engine.GetUserProfiles(userID, tenantID)
// Response: UserProfile với profiles[] đầy đủ (topic/sub_topic/content)

// GET /v1/console/profiles/config | PUT /v1/console/profiles/config
// → memobase-engine.GetConfig / UpdateConfig
// Config: flush_threshold, buffer_token_limit, ttl_days

// GET /v1/console/profiles/{user_id}/buffers
// → memobase-engine.GetBuffer(userID, tenantID)
// Buffer: {token_count, token_threshold, idle_timeout, last_flush, flush_count}

// GET /v1/console/profiles/{user_id}/context
// → memobase-context.AssembleContext(userID, tenantID)
// § Flow §6.2: assembly < 100ms
// Response: ContextAssembly{context_string, token_count, profile_section_tokens, latency_ms}

// GET /v1/console/profiles/{user_id}/events
// → vnp-event.GetTimeline(tenantID, userID)
// UserEvent domain: {id, user_id, engine, event_type, action, gist_text}
```

**Memobase camelCase response mapping** (TypeScript `UserProfile`):
```go
type UserProfileResponse struct {
    UserID    string    `json:"user_id"`
    CreatedAt string    `json:"created_at"`
    UpdatedAt string    `json:"updated_at"`
    Profiles  []Profile `json:"profiles"`
}
type Profile struct {
    Topic    string `json:"topic"`
    SubTopic string `json:"sub_topic"`
    Content  string `json:"content"`
}
```

---

## CR-007 — Governance Center (vnp-admin + vnp-platform)

### Architecture mapping

Theo §5.2 Admin Domain:
```go
Tenant — ID, Name, Slug, Tier(free|pro|enterprise), Status(active|suspended|deleted)
User   — ID, TenantID, Email, Role(admin|editor|viewer)
APIKey — ID, TenantID, KeyHash(SHA-256), Permissions, ExpiresAt
```

NATS events §6.4: `admin.tenant.created`, `admin.tenant.deleted`
Console route `FEAT-011` ↔ `/v1/console/governance/*`

### Backend Solution

```go
type ConsoleGovernanceHandler struct {
    adminSvc   VNPAdminClient    // gRPC → vnp-admin service
    auditRepo  AuditLogRepo      // PostgreSQL audit_logs table
    eventSvc   VNPEventClient    // vnp-event để GDPR forget cascading
    searchHub  VNPSearchHubClient // cross-engine forget
}

// GET /v1/console/governance/tenants → vnp-admin.ListTenants()
// POST /v1/console/governance/tenants → vnp-admin.CreateTenant() + NATS: admin.tenant.created
// PUT  /v1/console/governance/tenants/{id} → vnp-admin.UpdateTenant()

// GET  /v1/console/governance/policies → PostgreSQL opa_policies table
// POST /v1/console/governance/policies → Store OPA rego_code in opa_policies
// PUT  /v1/console/governance/policies/{id} → Update + hot-reload OPA if enabled

// GET /v1/console/governance/audit
// → PostgreSQL audit_logs, filter by tenant, actor, action, entity_type, date range

// POST /v1/console/governance/gdpr/forget/preview
// → Dry-run: estimate count across all engines (no deletion)

// POST /v1/console/governance/gdpr/forget
// → Cascading delete via parallel gRPC:
//    memobase-engine.DeleteUser(userID)
//    zep-user.DeleteUser(userID)
//    sm-profile.DeleteUser(userID)
//    graphiti-store.DeleteEpisodesByUser(userID)
//    vnp-event.PurgeUserEvents(userID)
//    ov-fs.PurgeUserData(userID)
// → Audit log entry: action=GDPR_FORGET
```

**Audit log schema** (PostgreSQL in `vnp-platform`):
```sql
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    actor_id    UUID NOT NULL,
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT,
    result      TEXT DEFAULT 'success',
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id, created_at DESC);
```

---

## CR-008 — Observability (obs-service + OpenTelemetry)

### Architecture mapping

Theo §5: `obs-service/` — Observability infrastructure
Theo §7 Tech Stack: OpenTelemetry + Prometheus
Console route `FEAT-017` ↔ `/v1/console/observability/*`

### Backend Solution

```go
type ConsoleObservabilityHandler struct {
    promClient PromClient      // Prometheus HTTP API client
    traceRepo  TraceRepository // OTEL span store (Jaeger/Tempo backend)
    errorAgg   ErrorAggRepo    // Aggregated errors từ slog + PostgreSQL
    costSvc    CostService     // Bifrost LLM token cost tracking
}

// GET /v1/console/observability/metrics
// → Prometheus queries:
//   rate(vnp_requests_total[5m])              — request rate
//   histogram_quantile(0.95, vnp_latency_ms)  — p95 latency
//   rate(vnp_errors_total[5m])                — error rate
// Response: MetricPoint[] với time-series data

// GET /v1/console/observability/traces?service=&status=&limit=
// → OTel collector/Jaeger query hoặc PostgreSQL trace table
// Response: TraceSpan[]

// GET /v1/console/observability/errors?service=&severity=
// → Aggregated errors từ structured slog (slog → PostgreSQL error_aggregates)
// Response: ErrorEntry{id, message, service, count, lastOccurrence, stack}

// GET /v1/console/observability/costs
// → Bifrost: total LLM tokens per tenant per model, per engine
```

**Error aggregation table** (PostgreSQL in `obs-service`):
```sql
CREATE TABLE error_aggregates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    service         TEXT NOT NULL,
    message         TEXT NOT NULL,
    message_hash    TEXT NOT NULL,      -- MD5 để group identical errors
    count           INT DEFAULT 1,
    last_occurrence TIMESTAMPTZ DEFAULT NOW(),
    stack           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, message_hash)
);
```

---

## CR-009 — Pipelines Monitor (pipeline-service + NATS JetStream)

### Architecture mapping

Theo §5.2 Pipeline Domain:
```go
Pipeline         — Engine, Status(idle|running|paused|error), JobCount, Workers[]
Job              — ID, Engine, Type(ingest|index|sync|cognify), Status, Priority
Queue            — Name, Engine, Size, MaxSize, Workers
Worker           — ID, Engine, Status(idle|busy|offline)
PipelineTemplate — ID, Name, Engine, Config
```

NATS events §6.4 per engine: `cognee.pipeline.completed`, `memobase.pipeline.completed`, ...
Console route `FEAT-015` ↔ `/v1/console/pipelines/*`

### Backend Solution

```go
type ConsolePipelinesHandler struct {
    pipelineSvc PipelineServiceClient  // gRPC → pipeline-service
    nats        NATSJetStreamClient     // JetStream management API
}

// GET /v1/console/pipelines/queues
// → NATS JetStream: js.StreamInfo per engine stream
// → Tổng hợp: pending messages = queue depth, messages/s = throughput
// QueueMetrics: {depth, throughput, retry_count}

// GET /v1/console/pipelines/status
// → pipeline-service.GetAllPipelineStatus(tenantID)
// → Aggregate: per-engine idle/running/error

// GET /v1/console/pipelines/workers
// → pipeline-service.ListWorkers(tenantID)
// Worker: {id, engine, status(idle|busy|offline)}

// GET /v1/console/pipelines/templates
// → pipeline-service.ListTemplates(tenantID)
// PipelineTemplate: preconfigured pipeline configs per engine

// GET /v1/console/pipelines/{engine}/jobs
// → pipeline-service.ListJobs(engine, tenantID)
// PipelineJob: {id, engine, type, status, progress, created_at, updated_at}

// GET /v1/console/pipelines/{engine}/jobs/{id}
// → pipeline-service.GetJob(engine, jobID, tenantID)
```

---

## CR-010 — Infrastructure Health (InProcessRegistry + DB pings)

### Architecture mapping

Theo §2.1: InProcessRegistry biết uptime của tất cả 35 services.
Theo §8: PostgreSQL, Neo4j, Redis, NATS, MinIO, Qdrant.
Console route `FEAT-016` ↔ `/v1/console/infra/*`

### Backend Solution

```go
type ConsoleInfraHandler struct {
    registry   InProcessRegistry    // Tất cả 35 services (§2.2)
    db         *sql.DB              // PostgreSQL
    neo4j      neo4j.Driver         // Neo4j ping
    redis      *redis.Client        // Redis ping
    nats       *nats.Conn           // NATS status
    minio      *minio.Client        // MinIO ping
}

// GET /v1/console/infra/services
// → Loop qua registry.GetAll() → gRPC health check từng service
// ServiceInfo: {name, version, status(Healthy/Warning/Critical), uptime}

// GET /v1/console/infra/databases
// → Parallel ping:
//   db.PingContext(ctx) → PostgreSQL latency
//   neo4j.VerifyConnectivity(ctx) → Neo4j latency
//   redis.Ping(ctx) → Redis latency
//   nats.Status() → NATS connection status
// DatabaseHealth: {name, type, status, latency_ms}

// GET /v1/console/infra/resources
// → Đọc từ Prometheus node_exporter:
//   process_cpu_seconds_total{job="vnp-memory"}
//   process_resident_memory_bytes
//   disk_usage_percent
// ResourceMetrics: {service, cpu_usage_pct, memory_usage_mb, disk_usage_pct}

// GET /v1/console/infra/topology
// → Trả về graph topology: monolith mode → single node với 35 children
//   gateway mode → topology graph với gateway + distributed services

// GET /v1/console/infra/deployments
// → Deployment info: build version, git commit, start time
```

---

## CR-011 — Org Settings & API SDK (vnp-admin + vnp-platform)

### Architecture mapping

Theo §5.2 Admin Domain: `Tenant, User, APIKey`
Không có FEAT ID riêng trong architecture — cần route mới trong `/v1/console/org/*` và `/v1/console/sdk/*`.

### Backend Solution

Tạo thêm 2 handler groups mới:

```go
// === Org Handler ===
// GET /v1/console/org/settings → PostgreSQL tenants table (own tenant)
// PUT /v1/console/org/settings → Update tenant config (name, slug, timezone, limits)
// GET /v1/console/org/members → PostgreSQL users table filtered by tenant_id
// GET /v1/console/org/roles → Static roles + permissions config

// OrgSettings response:
type OrgSettings struct {
    Name                 string `json:"name"`
    Slug                 string `json:"slug"`
    Timezone             string `json:"timezone"`
    MaxAgents            int    `json:"maxAgents"`
    MaxMemoriesPerUser   int    `json:"maxMemoriesPerUser"`
}

// === SDK Handler ===
// GET /v1/console/sdk/keys → PostgreSQL api_keys table (§5.2 APIKey domain)
// POST /v1/console/sdk/keys → Create API Key (random prefix + SHA-256 hash stored)
// GET /v1/console/sdk/rate-limits → Tenant rate tier config từ Redis/PostgreSQL
// GET/POST /v1/console/sdk/webhooks → PostgreSQL webhooks table

// APIKey domain (§5.2): ID, TenantID, KeyHash(SHA-256), Permissions, ExpiresAt
// Khi list: trả về masked key (first 8 chars + "...")
// Khi tạo mới: trả về raw key một lần duy nhất
```

**Webhook schema** (PostgreSQL in `vnp-platform`):
```sql
CREATE TABLE webhooks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    url          TEXT NOT NULL,
    events       TEXT[] NOT NULL,   -- ['memory.created', 'profile.updated', ...]
    secret_hash  TEXT,              -- HMAC signing key hash
    status       TEXT DEFAULT 'active',
    success_rate FLOAT DEFAULT 100.0,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Tóm tắt mapping Service ↔ Console Handler

```
Console Handler             ← gRPC (bufconn) →  Engine Service
─────────────────────────────────────────────────────────────────
ConsoleAdaptiveHandler      ←──────────────────  sm-memory
                            ←──────────────────  sm-connector
                            ←──────────────────  sm-analytics

ConsoleProfilesHandler      ←──────────────────  memobase-context
                            ←──────────────────  memobase-engine
                            ←──────────────────  vnp-event

ConsoleGovernanceHandler    ←──────────────────  vnp-admin
                            ←── PostgreSQL ─────  audit_logs
                            ←──────────────────  (all engines, GDPR)

ConsoleObservabilityHandler ←── Prometheus ─────  HTTP API
                            ←── PostgreSQL ─────  error_aggregates
                            ←── OTEL ───────────  trace store

ConsolePipelinesHandler     ←──────────────────  pipeline-service
                            ←── NATS ───────────  JetStream mgmt API

ConsoleInfraHandler         ←──────────────────  InProcessRegistry (35 services)
                            ←── PostgreSQL ─────  ping
                            ←── Neo4j ──────────  ping
                            ←── Redis ──────────  ping
                            ←── NATS ───────────  status

ConsoleOrgHandler           ←── PostgreSQL ─────  tenants, users tables
ConsoleSDKHandler           ←── PostgreSQL ─────  api_keys, webhooks tables
                            ←── Redis ──────────  rate_limit config
```
