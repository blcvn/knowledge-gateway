# Solution: SOL-007 — Admin Service & Observability Stack

**CR ID:** CR-GR-007  
**Solution ID:** SOL-007  
**Priority:** High (Wave 4)  
**Architecture:** NEW `services/graphiti-admin/` (service #36) + OTel instrumentation across all graphiti services

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- Monolith hiện có **35 services** trong `InProcessRegistry`.
- OTel Collector đã configured tại `:4317` (OTLP gRPC) — instrumenting Jaeger traces.
- Prometheus đã có `/metrics` endpoint per service.
- NATS embedded — `graphiti.tenant.created`, `graphiti.community.rebuilt` subjects.
- PostgreSQL — lưu tenant metadata.

**`graphiti-admin` là service #36 cần thêm vào InProcessRegistry.**

---

## 2. Service Structure — `services/graphiti-admin/`

```
services/graphiti-admin/
├── cmd/server/main.go            # (unused — monolith bootstrap)
├── internal/
│   ├── domain/
│   │   ├── tenant.go             # Tenant, TenantConfig, TenantStats
│   │   └── maintenance.go        # MaintenanceTask
│   ├── usecase/
│   │   ├── tenant_management.go  # CreateTenant, DeleteTenant, ListTenants, GetTenantStats
│   │   ├── rebuild_communities.go # Post-bulk community rebuild
│   │   ├── index_management.go   # BuildIndices, DeleteAllIndexes
│   │   └── token_usage_report.go # Cross-service token aggregation + cost estimation
│   ├── adapter/
│   │   ├── grpc/handler.go       # AdminService gRPC implementation
│   │   └── client/               # Clients for store, knowledge, search, ingestion
│   └── infra/
│       └── config/config.go
├── api/proto/graphiti/admin/v1/admin.proto
└── Makefile
```

---

## 3. Domain — `internal/domain/tenant.go`

```go
// services/graphiti-admin/internal/domain/tenant.go

package domain

import "time"

type Tenant struct {
    GroupID   string
    Name      string
    CreatedAt time.Time
    Config    TenantConfig
}

type TenantConfig struct {
    MaxEpisodes   int    // 0 = unlimited
    LLMProvider   string // override default (e.g., "anthropic")
    EmbedProvider string
}

type TenantStats struct {
    GroupID      string
    EpisodeCount int64
    EntityCount  int64
    EdgeCount    int64
    CommunityCount int64
    StorageSizeEstimate string  // human-readable, e.g. "2.3 MB"
}

type PostgresTenant struct {
    GroupID   string    `db:"group_id"`
    Name      string    `db:"name"`
    MaxEpisodes int     `db:"max_episodes"`
    LLMProvider string  `db:"llm_provider"`
    EmbedProvider string `db:"embed_provider"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}
```

---

## 4. PostgreSQL Schema

```sql
-- db/migrations/0024_graphiti_tenants.up.sql

CREATE TABLE graphiti_tenants (
    group_id      TEXT PRIMARY KEY,
    name          TEXT NOT NULL DEFAULT '',
    max_episodes  INTEGER NOT NULL DEFAULT 0,  -- 0 = unlimited
    llm_provider  TEXT NOT NULL DEFAULT '',
    embed_provider TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_graphiti_tenants_created ON graphiti_tenants(created_at DESC);
```

---

## 5. Tenant Management — `internal/usecase/tenant_management.go`

```go
// services/graphiti-admin/internal/usecase/tenant_management.go

package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/vnp-memory/services/graphiti-admin/internal/domain"
    "github.com/vnp-memory/services/graphiti-admin/internal/usecase/port"
)

type TenantManagementUseCase struct {
    tenantRepo  port.TenantRepository
    storePort   port.StorePort      // → graphiti-store
    publisher   port.EventPublisher // NATS
}

func (uc *TenantManagementUseCase) CreateTenant(ctx context.Context, req CreateTenantReq) (*domain.Tenant, error) {
    // Check if already exists
    existing, _ := uc.tenantRepo.Get(ctx, req.GroupID)
    if existing != nil {
        return existing, nil  // idempotent
    }

    tenant := domain.Tenant{
        GroupID:   req.GroupID,
        Name:      req.Name,
        CreatedAt: time.Now(),
        Config:    req.Config,
    }
    if err := uc.tenantRepo.Save(ctx, tenant); err != nil {
        return nil, fmt.Errorf("save tenant: %w", err)
    }

    // Publish event → Store can init schema (for Kuzu/FalkorDB per-group-id mode)
    uc.publisher.Publish(ctx, "graphiti.tenant.created", map[string]any{
        "group_id": req.GroupID,
        "name":     req.Name,
    })

    return &tenant, nil
}

func (uc *TenantManagementUseCase) DeleteTenant(ctx context.Context, groupID string) error {
    // 1. Clear all graph data for this tenant
    if err := uc.storePort.ClearData(ctx, []string{groupID}); err != nil {
        return fmt.Errorf("clear data: %w", err)
    }

    // 2. Remove tenant record
    if err := uc.tenantRepo.Delete(ctx, groupID); err != nil {
        return fmt.Errorf("delete tenant record: %w", err)
    }

    uc.publisher.Publish(ctx, "graphiti.tenant.deleted", map[string]any{"group_id": groupID})
    return nil
}

func (uc *TenantManagementUseCase) ListTenants(ctx context.Context) ([]*domain.Tenant, error) {
    return uc.tenantRepo.List(ctx)
}

func (uc *TenantManagementUseCase) GetTenantStats(ctx context.Context, groupID string) (*domain.TenantStats, error) {
    stats := &domain.TenantStats{GroupID: groupID}

    // Query counts from store
    counts, err := uc.storePort.GetGroupStats(ctx, groupID)
    if err != nil { return stats, nil }  // non-fatal: return empty stats

    stats.EpisodeCount   = counts.EpisodeCount
    stats.EntityCount    = counts.EntityCount
    stats.EdgeCount      = counts.EdgeCount
    stats.CommunityCount = counts.CommunityCount
    return stats, nil
}

type CreateTenantReq struct {
    GroupID string
    Name    string
    Config  domain.TenantConfig
}
```

---

## 6. Community Rebuild — `internal/usecase/rebuild_communities.go`

```go
// services/graphiti-admin/internal/usecase/rebuild_communities.go

type RebuildCommunitiesUseCase struct {
    knowledgePort port.KnowledgePort
    storePort     port.StorePort
    publisher     port.EventPublisher
}

func (uc *RebuildCommunitiesUseCase) Execute(ctx context.Context, groupID string) error {
    // 1. Remove existing communities for group
    if err := uc.storePort.RemoveCommunities(ctx, groupID); err != nil {
        return fmt.Errorf("remove existing communities: %w", err)
    }

    // 2. Trigger full community detection via knowledge service
    if _, err := uc.knowledgePort.BuildCommunities(ctx, port.BuildCommunitiesReq{GroupID: groupID}); err != nil {
        return fmt.Errorf("build communities: %w", err)
    }

    // 3. Publish event → search service invalidates community caches
    uc.publisher.Publish(ctx, "graphiti.community.rebuilt", map[string]any{
        "group_id": groupID,
        "trigger":  "admin_manual",
    })

    return nil
}
```

---

## 7. Token Usage Report — `internal/usecase/token_usage_report.go`

```go
// services/graphiti-admin/internal/usecase/token_usage_report.go

type TokenUsageReportUseCase struct {
    knowledgePort port.KnowledgePort
    pricingConfig PricingConfig
}

type PricingConfig struct {
    Providers map[string]ProviderPricing
}

type ProviderPricing struct {
    InputPricePerMToken  float64  // $ per million input tokens
    OutputPricePerMToken float64  // $ per million output tokens
}

var DefaultPricingConfig = PricingConfig{
    Providers: map[string]ProviderPricing{
        "openai_gpt4o":       {InputPricePerMToken: 5.00, OutputPricePerMToken: 15.00},
        "openai_gpt4o_mini":  {InputPricePerMToken: 0.15, OutputPricePerMToken: 0.60},
        "anthropic_claude35": {InputPricePerMToken: 3.00, OutputPricePerMToken: 15.00},
        "gemini_flash":       {InputPricePerMToken: 0.075, OutputPricePerMToken: 0.30},
    },
}

type TokenUsageReport struct {
    Period           string
    ByPrompt         map[string]PromptUsage
    Total            PromptUsage
    EstimatedCostUSD float64
}

type PromptUsage struct {
    InputTokens      int64
    OutputTokens     int64
    TotalTokens      int64
    CallCount        int64
}

func (uc *TokenUsageReportUseCase) Execute(ctx context.Context, groupID, period string) (*TokenUsageReport, error) {
    raw, err := uc.knowledgePort.GetTokenUsage(ctx, port.GetTokenUsageReq{GroupID: groupID})
    if err != nil { return nil, err }

    report := &TokenUsageReport{
        Period:   period,
        ByPrompt: make(map[string]PromptUsage),
    }

    for promptName, usage := range raw {
        pu := PromptUsage{
            InputTokens:  usage.PromptTokens,
            OutputTokens: usage.CompletionTokens,
            TotalTokens:  usage.TotalTokens,
            CallCount:    usage.CallCount,
        }
        report.ByPrompt[promptName] = pu
        report.Total.InputTokens  += pu.InputTokens
        report.Total.OutputTokens += pu.OutputTokens
        report.Total.TotalTokens  += pu.TotalTokens
        report.Total.CallCount    += pu.CallCount
    }

    // Cost estimation using default provider (gpt-4o)
    pricing := uc.pricingConfig.Providers["openai_gpt4o"]
    report.EstimatedCostUSD = float64(report.Total.InputTokens)/1_000_000*pricing.InputPricePerMToken +
        float64(report.Total.OutputTokens)/1_000_000*pricing.OutputPricePerMToken

    return report, nil
}
```

---

## 8. OTel Instrumentation — `pkg/observability/`

### 8.1. Tracer Setup

```go
// pkg/observability/tracer.go

package observability

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type OTelConfig struct {
    ServiceName  string
    Endpoint     string   // OTLP gRPC: "otel-collector:4317"
    SamplingRate float64  // 0.0-1.0, default 1.0
}

func InitTracer(ctx context.Context, cfg OTelConfig) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithInsecure(),
        otlptracegrpc.WithEndpoint(cfg.Endpoint),
    )
    if err != nil { return nil, err }

    sampler := sdktrace.AlwaysSample()
    if cfg.SamplingRate < 1.0 {
        sampler = sdktrace.TraceIDRatioBased(cfg.SamplingRate)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithSampler(sampler),
        sdktrace.WithResource(newResource(cfg.ServiceName)),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### 8.2. Ingestion Pipeline Instrumentation

```go
// services/graphiti-ingestion/internal/usecase/ingest_episode.go (EXTEND)
// Add OpenTelemetry spans to each pipeline step

func (uc *IngestEpisodeUseCase) Execute(ctx context.Context, req domain.RawEpisode) (*domain.IngestResult, error) {
    tracer := otel.Tracer("graphiti-ingestion")
    ctx, rootSpan := tracer.Start(ctx, "ingestion.pipeline",
        trace.WithAttributes(
            attribute.String("group_id", req.GroupID),
            attribute.String("source", string(req.Source)),
        ),
    )
    defer rootSpan.End()

    // Step 1: Chunk
    ctx, chunkSpan := tracer.Start(ctx, "ingestion.chunk_content")
    chunks, err := uc.chunker.Chunk(req.Body, ...)
    chunkSpan.SetAttributes(attribute.Int("chunk_count", len(chunks)))
    chunkSpan.End()
    if err != nil { rootSpan.RecordError(err); return nil, err }

    // Step 2: Context retrieval
    ctx, ctxSpan := tracer.Start(ctx, "ingestion.retrieve_context")
    prevEpisodes, _ := uc.storePort.RetrieveEpisodes(ctx, ...)
    ctxSpan.SetAttributes(attribute.Int("prev_episodes_count", len(prevEpisodes)))
    ctxSpan.End()

    // Step 3: Entity extraction
    ctx, extractSpan := tracer.Start(ctx, "ingestion.extract_entities")
    entities, tokenUsage, err := uc.knowledgePort.ExtractEntities(ctx, ...)
    extractSpan.SetAttributes(
        attribute.Int("entity_count", len(entities)),
        attribute.Int("tokens_used", tokenUsage.TotalTokens),
    )
    extractSpan.End()

    // Steps 4-9: similar pattern...
    // Step 8: Persist
    ctx, persistSpan := tracer.Start(ctx, "ingestion.persist")
    err = uc.storePort.SaveBulk(ctx, ...)
    persistSpan.End()

    return result, nil
}
```

### 8.3. Knowledge Service Instrumentation

```go
// services/graphiti-knowledge/internal/adapter/client/llm/bifrost.go (EXTEND)

func (c *BifrostLLMClient) GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error) {
    tracer := otel.Tracer("graphiti-knowledge")
    ctx, span := tracer.Start(ctx, "knowledge.llm.generate",
        trace.WithAttributes(
            attribute.String("prompt_name", opts.PromptName),
            attribute.Bool("cached", false),
            attribute.Int("model_size", int(opts.ModelSize)),
        ),
    )
    defer span.End()

    // Check cache
    if cached, ok := c.cache.Get(ctx, computeCacheKey(messages, opts)); ok {
        span.SetAttributes(attribute.Bool("cached", true))
        return cached, nil
    }

    resp, err := c.doGenerate(ctx, messages, opts)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }

    span.SetAttributes(
        attribute.Int("input_tokens", resp.TokenUsage.PromptTokens),
        attribute.Int("output_tokens", resp.TokenUsage.CompletionTokens),
    )
    return resp, nil
}
```

### 8.4. Search Service Instrumentation

```go
// services/graphiti-search/internal/usecase/search.go (EXTEND)

func (uc *SearchUseCase) Execute(ctx context.Context, req SearchRequest) (*SearchResults, error) {
    tracer := otel.Tracer("graphiti-search")
    ctx, span := tracer.Start(ctx, "search.query",
        trace.WithAttributes(
            attribute.String("query", req.Query[:min(len(req.Query), 100)]),
            attribute.StringSlice("group_ids", req.Filters.GroupIDs),
        ),
    )
    defer span.End()

    // Embed query
    ctx, embedSpan := tracer.Start(ctx, "search.embed_query")
    queryEmb, _ := uc.knowledgeClient.GenerateEmbedding(ctx, req.Query)
    embedSpan.End()

    // Parallel search steps with child spans
    // (cosine, bm25, bfs each get their own span)

    span.SetAttributes(
        attribute.Int("result_count", len(results.Edges)),
        attribute.Int64("latency_ms", results.LatencyMs),
    )
    return results, nil
}
```

---

## 9. Prometheus Metrics — Per-Service

```go
// pkg/observability/metrics.go

// Graphiti-specific Prometheus metrics per service

// Ingestion metrics
var (
    IngestEpisodeTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_ingestion_episodes_total"},
        []string{"group_id", "source", "status"},
    )
    IngestPipelineDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "graphiti_ingestion_pipeline_duration_seconds"},
        []string{"step"},
    )
    IngestEntitiesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_ingestion_entities_extracted_total"},
        []string{"group_id"},
    )
    IngestTokensUsed = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_ingestion_tokens_used_total"},
        []string{"prompt_type"},
    )
    IngestQueueDepth = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{Name: "graphiti_ingestion_queue_depth"},
        []string{"group_id"},
    )
)

// Knowledge metrics
var (
    LLMCallsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_knowledge_llm_calls_total"},
        []string{"provider", "model", "prompt_type", "status"},
    )
    LLMDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "graphiti_knowledge_llm_duration_seconds",
            Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60},
        },
        []string{"provider", "prompt_type"},
    )
    LLMTokensTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_knowledge_llm_tokens_total"},
        []string{"prompt_type", "token_type"},  // token_type: input|output
    )
    LLMCacheHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_knowledge_llm_cache_hits_total"},
        []string{"prompt_type"},
    )
    ResolutionDecisions = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_knowledge_resolution_decisions_total"},
        []string{"decision_type"},  // merge|new|DUPLICATE|NEW|CONTRADICTION|UPDATE
    )
)

// Search metrics
var (
    SearchQueriesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_search_queries_total"},
        []string{"config", "status"},
    )
    SearchDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "graphiti_search_duration_seconds",
            Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
        },
        []string{"config"},
    )
    SearchCacheHits = prometheus.NewCounter(
        prometheus.CounterOpts{Name: "graphiti_search_cache_hits_total"},
    )
    RerankerDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "graphiti_search_rerank_duration_seconds"},
        []string{"reranker"},
    )
)

// Store metrics
var (
    StoreQueriesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_store_queries_total"},
        []string{"operation", "object_type", "status"},
    )
    StoreQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "graphiti_store_query_duration_seconds"},
        []string{"operation", "object_type"},
    )
    StoreTransactionsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphiti_store_transactions_total"},
        []string{"status"},  // committed|rolled_back
    )
)
```

---

## 10. Anonymous Telemetry — `pkg/telemetry/posthog.go`

```go
// pkg/telemetry/posthog.go

package telemetry

import (
    "os"
    "sync"

    "github.com/posthog/posthog-go"
)

var (
    once   sync.Once
    client posthog.Client
    disabled bool
)

func Init(apiKey string) {
    once.Do(func() {
        // Opt-out via env var
        if os.Getenv("GRAPHITI_TELEMETRY_DISABLED") == "true" {
            disabled = true
            return
        }
        client, _ = posthog.NewWithConfig(apiKey, posthog.Config{
            Endpoint: "https://app.posthog.com",
        })
    })
}

type TelemetryEvent struct {
    Name       string
    Properties map[string]any
}

// Track captures anonymous usage events.
// NEVER captures content, queries, entity names, or user data.
func Track(event TelemetryEvent) {
    if disabled || client == nil { return }

    // Use stable anonymous ID (hash of hostname, not user ID)
    anonID := getAnonymousID()

    go client.Enqueue(posthog.Capture{
        DistinctId: anonID,
        Event:      event.Name,
        Properties: sanitizeProperties(event.Properties),
    })
}

// sanitizeProperties ensures no PII escapes
func sanitizeProperties(props map[string]any) posthog.Properties {
    safe := posthog.NewProperties()
    allowedKeys := map[string]bool{
        "source_type": true, "entity_count": true, "edge_count": true,
        "backend_type": true, "search_config": true, "result_count": true,
        "cluster_count": true, "group_size": true, "tool_name": true,
    }
    for k, v := range props {
        if allowedKeys[k] { safe.Set(k, v) }
    }
    return safe
}
```

---

## 11. gRPC Service — `internal/adapter/grpc/handler.go`

```protobuf
// api/proto/graphiti/admin/v1/admin.proto

syntax = "proto3";
package graphiti.admin.v1;

service AdminService {
    // Tenant management
    rpc CreateTenant(CreateTenantRequest) returns (CreateTenantResponse);
    rpc DeleteTenant(DeleteTenantRequest) returns (DeleteTenantResponse);
    rpc ListTenants(ListTenantsRequest) returns (ListTenantsResponse);
    rpc GetTenantStats(GetTenantStatsRequest) returns (GetTenantStatsResponse);

    // Community & Index management
    rpc RebuildCommunities(RebuildCommunitiesRequest) returns (RebuildCommunitiesResponse);
    rpc BuildIndicesAndConstraints(BuildIndicesRequest) returns (BuildIndicesResponse);
    rpc DeleteAllIndexes(DeleteAllIndexesRequest) returns (DeleteAllIndexesResponse);

    // Reporting
    rpc GetTokenUsageReport(GetTokenUsageReportRequest) returns (GetTokenUsageReportResponse);

    // Store-level stats
    rpc GetGroupStats(GetGroupStatsRequest) returns (GetGroupStatsResponse);

    // Health
    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message CreateTenantRequest {
    string group_id      = 1;
    string name          = 2;
    TenantConfigProto config = 3;
}

message TenantConfigProto {
    int32  max_episodes   = 1;
    string llm_provider   = 2;
    string embed_provider = 3;
}

message GetTokenUsageReportRequest {
    string group_id = 1;  // optional: all tenants if empty
    string period   = 2;  // "all_time" | "24h" | "7d"
}

message GetTokenUsageReportResponse {
    map<string, PromptUsageProto> by_prompt = 1;
    PromptUsageProto total = 2;
    double estimated_cost_usd = 3;
}
```

---

## 12. Bootstrap — Add service #36 to InProcessRegistry

```go
// apps/memory/internal/bootstrap/graphiti_admin.go

func InitGraphitiAdmin(reg *bus.InProcessRegistry,
    knowledgeConn, storeConn *bufconn.Listener,
    db *pgxpool.Pool, natsConn *nats.Conn) {

    knowledgeClient := adminport.NewKnowledgeClient(knowledgeConn)
    storeClient     := adminport.NewStoreClient(storeConn)
    publisher       := natevent.NewPublisher(natsConn, "graphiti")

    tenantRepo    := postgres.NewTenantRepository(db)
    tenantUC      := usecase.NewTenantManagementUseCase(tenantRepo, storeClient, publisher)
    rebuildUC     := usecase.NewRebuildCommunitiesUseCase(knowledgeClient, storeClient, publisher)
    indexUC       := usecase.NewIndexManagementUseCase(storeClient)
    tokenReportUC := usecase.NewTokenUsageReportUseCase(knowledgeClient, DefaultPricingConfig)

    handler := grpchandler.NewAdminHandler(tenantUC, rebuildUC, indexUC, tokenReportUC)

    pb.RegisterAdminServiceServer(reg.GRPCServer(), handler)
    reg.Register("graphiti-admin", handler)  // service #36
}
```

---

## 13. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/graphiti-admin/internal/domain/tenant.go` | Tenant, TenantConfig, TenantStats |
| `services/graphiti-admin/internal/usecase/tenant_management.go` | CRUD + stats |
| `services/graphiti-admin/internal/usecase/rebuild_communities.go` | Post-bulk rebuild |
| `services/graphiti-admin/internal/usecase/index_management.go` | Build/delete indices |
| `services/graphiti-admin/internal/usecase/token_usage_report.go` | Token aggregation + cost |
| `services/graphiti-admin/internal/adapter/grpc/handler.go` | All AdminService RPCs |
| `services/graphiti-admin/internal/adapter/repository/postgres/tenant_repo.go` | PostgreSQL tenant storage |
| `api/proto/graphiti/admin/v1/admin.proto` | Full gRPC contract |
| `db/migrations/0024_graphiti_tenants.up.sql` | graphiti_tenants table |
| `pkg/observability/tracer.go` | OTel TracerProvider setup |
| `pkg/observability/metrics.go` | All Prometheus metric vars |
| `pkg/telemetry/posthog.go` | Anonymous telemetry (opt-out) |
| `apps/memory/internal/bootstrap/graphiti_admin.go` | Service #36 bootstrap |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/graphiti-ingestion/internal/usecase/ingest_episode.go` | + OTel spans per step |
| `services/graphiti-knowledge/internal/adapter/client/llm/bifrost.go` | + OTel span + Prometheus metrics |
| `services/graphiti-search/internal/usecase/search.go` | + OTel span + Prometheus metrics |
| `services/graphiti-store/internal/adapter/driver/neo4j/bulk_repo.go` | + OTel span + Prometheus metrics |
| `gateway/internal/adapter/handler/health_handler.go` | + admin service health check |
| `gateway/internal/adapter/handler/router.go` | + admin tenant CRUD routes |
| `apps/memory/internal/bootstrap/graphiti.go` | + call InitGraphitiAdmin() |
| `apps/memory/configs/config.yaml` | + telemetry config |

---

## 14. Acceptance Criteria Mapping

| AC từ CR-GR-007 | Covered by |
|----------------|-----------|
| POST /v1/graphiti/admin/tenants → created, GET lists it | tenantManagementUC + tenantRepo |
| DELETE /v1/graphiti/admin/tenants/tenant-a → data cleared | DeleteTenant() → storePort.ClearData() |
| POST /v1/graphiti/episodes/bulk then POST /v1/graphiti/admin/communities → communities built | rebuildCommunitiesUC |
| GET /v1/graphiti/admin/token-usage → {by_prompt: {extract_nodes: {total_tokens: N}}} | tokenUsageReportUC |
| estimated_cost_usd = correct calculation | tokenUsageReportUC + PricingConfig |
| OTel traces in Jaeger: ingestion.pipeline with child spans | OTel instrumentation in ingest_episode.go |
| graphiti_knowledge_llm_calls_total increments per LLM call | LLMCallsTotal counter |
| GRAPHITI_TELEMETRY_DISABLED=true → no PostHog calls | disabled flag in telemetry.Init() |
| gRPC health check graphiti-knowledge:9003 → SERVING | gRPC health server in all services |
| POST /v1/graphiti/admin/indices → Neo4j vector index created | indexManagementUC → storePort.BuildIndicesAndConstraints() |
