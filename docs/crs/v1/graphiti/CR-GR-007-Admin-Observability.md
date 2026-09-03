# Change Request: CR-GR-007 — Admin Service (Tenant Management, Index Ops, Observability)

**CR ID:** CR-GR-007  
**Component:** `services/graphiti-admin` [NEW SERVICE] | Observability stack  
**Priority:** High  
**Status:** In Progress
**Reference:** graphiti PRD §5.6, SRS §8, specs/services/06-admin-service.md  
**Maps to Python:** `graphiti_core/telemetry.py`, `tracer.py`, `utils/maintenance/`

---

## 1. Mô tả

Xây dựng **graphiti-admin** service và observability stack:
1. **Tenant Management** — create/delete/list tenants (group_ids) với auto-provisioning.
2. **Index Operations** — build/rebuild Neo4j indices, community detection trigger.
3. **OpenTelemetry Tracing** — distributed tracing across all graphiti services.
4. **Token Usage Tracking** — per-prompt-type LLM token consumption monitoring.
5. **Health Monitoring** — gRPC Health v1 + HTTP `/healthz` trên tất cả services.
6. **Telemetry** — anonymous usage statistics (opt-out via env var).

---

## 2. Vấn đề hiện tại

- ❌ Không có tenant management API (create/delete group_id provisioning).
- ❌ Không có trigger để rebuild community detection sau bulk import.
- ❌ OpenTelemetry chỉ configured một phần, thiếu instrumentation trong graphiti pipeline.
- ❌ Token usage không được tracked per prompt type.
- ❌ Không có anonymous telemetry opt-out mechanism.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/graphiti-admin/`

**Port:** `9005` (gRPC internal)

```
services/graphiti-admin/
├── internal/
│   ├── domain/
│   │   ├── tenant.go           # Tenant, TenantConfig
│   │   └── maintenance.go      # MaintenanceTask
│   ├── usecase/
│   │   ├── tenant_management.go # Create/delete/list tenants
│   │   ├── rebuild_communities.go # Trigger community rebuild
│   │   ├── index_management.go  # Build/delete indices
│   │   └── token_usage_report.go # Cross-service token aggregation
│   └── adapter/
│       ├── grpc/handler.go
│       └── client/             # Clients for store, knowledge, search
```

### 3.2. Tenant Management

```go
// internal/usecase/tenant_management.go

type TenantManager interface {
    // Create new tenant (provisions DB indices if not exists)
    CreateTenant(ctx, req CreateTenantReq) (*Tenant, error)

    // Delete tenant (cascade: clear all data for group_id)
    DeleteTenant(ctx, groupID string) error

    // List all active tenants
    ListTenants(ctx) ([]*Tenant, error)

    // Get tenant stats (episode count, entity count, edge count)
    GetTenantStats(ctx, groupID string) (*TenantStats, error)
}

type Tenant struct {
    GroupID   string
    Name      string
    CreatedAt time.Time
    Config    TenantConfig
}

type TenantConfig struct {
    MaxEpisodes   int    // 0 = unlimited
    LLMProvider   string // override default LLM for this tenant
    EmbedProvider string
}

// CreateTenant publishes: graphiti.tenant.created
// → Store: initialize schema (if Kuzu/FalkorDB per-group-id mode)
// → Search: initialize indices
```

### 3.3. Community Rebuild (Post-Bulk-Import)

```go
// internal/usecase/rebuild_communities.go
// Called after IngestEpisodeBulk (community detection was skipped during bulk)

func (uc *RebuildCommunitiesUseCase) Execute(ctx, groupID string) error {
    // 1. Remove existing communities for group (store.RemoveCommunities)
    // 2. Trigger knowledge.BuildCommunities(groupID)
    //    - Label Propagation in-memory
    //    - LLM summarization per cluster
    // 3. Persist via store.SaveCommunityNode + SaveCommunityEdge (batch)
    // 4. Publish: graphiti.community.rebuilt
    //    → search.InvalidateCache(groupID)
}
```

### 3.4. Index Management

```go
// POST /v1/graphiti/admin/indices
// Triggers: store.BuildIndicesAndConstraints()
// Creates: vector indices, fulltext indices, uniqueness constraints

// DELETE /v1/graphiti/admin/indices
// Triggers: store.DeleteAllIndexes()
// Warning: destructive operation, requires "admin" scope

// Useful for:
// - Initial setup
// - After schema migration
// - After Neo4j upgrade
```

### 3.5. OpenTelemetry Instrumentation

```go
// pkg/observability/tracer.go

type OTelConfig struct {
    ServiceName  string
    Endpoint     string  // OTLP endpoint: "otel-collector:4317"
    SamplingRate float64 // 0.0-1.0, default: 1.0 (100%)
}

// Instrumented operations per service:

// graphiti-ingestion:
// - Span: "ingestion.pipeline" (root)
//   - Child: "ingestion.chunk_content"
//   - Child: "ingestion.retrieve_context" (store call)
//   - Child: "ingestion.extract_entities" (knowledge call)
//   - Child: "ingestion.resolve_entities" (knowledge + store calls)
//   - Child: "ingestion.extract_edges" (knowledge call)
//   - Child: "ingestion.resolve_edges" (knowledge + store calls)
//   - Child: "ingestion.persist" (store call)
//   - Child: "ingestion.update_community" (knowledge call)

// graphiti-knowledge:
// - Span: "knowledge.llm.generate" (attributes: provider, model, prompt_name, cached)
// - Span: "knowledge.embedder.create" (attributes: provider, dimensions)
// - Span: "knowledge.reranker.rank" (attributes: provider)
// - Span: "knowledge.community.build" (attributes: cluster_count)

// graphiti-search:
// - Span: "search.query" (root)
//   - Child: "search.embed_query" (knowledge call)
//   - Child: "search.cosine_similarity" (store call) ─── parallel
//   - Child: "search.bm25_fulltext" (store call)    ─── parallel
//   - Child: "search.bfs_traversal" (store call)    ─── parallel
//   - Child: "search.rerank"
//   - Child: "search.filter"

// graphiti-store:
// - Span: "store.query" (attributes: operation, object_type, driver)
// - Span: "store.transaction"
```

### 3.6. Token Usage Tracking & Reporting

```go
// Aggregation endpoint: GET /v1/graphiti/admin/token-usage
// Collects from knowledge.GetTokenUsage RPC

type TokenUsageReport struct {
    GroupID   string                     `json:"group_id,omitempty"`
    Period    string                     `json:"period"`  // "all_time" | "24h" | "7d"
    ByPrompt  map[string]PromptUsage    `json:"by_prompt"`
    Total     PromptUsage               `json:"total"`
    EstimatedCostUSD float64            `json:"estimated_cost_usd"`
}

type PromptUsage struct {
    InputTokens      int64   `json:"input_tokens"`
    OutputTokens     int64   `json:"output_tokens"`
    TotalTokens      int64   `json:"total_tokens"`
    CallCount        int64   `json:"call_count"`
}

// Cost estimation based on provider pricing:
// OpenAI gpt-4o: $5/1M input, $15/1M output
// gpt-4o-mini: $0.15/1M input, $0.60/1M output
// (configurable via admin config)
```

### 3.7. Telemetry (Anonymous, Opt-Out)

```go
// pkg/telemetry/posthog.go
// Anonymous usage statistics — tracks product adoption patterns
// Opt-out: set GRAPHITI_TELEMETRY_DISABLED=true

type TelemetryEvent struct {
    EventName  string         `json:"event"`
    Properties map[string]any `json:"properties"`
}

// Events tracked (no PII):
// - "ingestion_completed": {source_type, entity_count, edge_count, backend_type}
// - "search_executed": {search_config, result_count}
// - "community_built": {cluster_count, group_size}
// - "mcp_tool_called": {tool_name}

// NEVER tracks: content, queries, entity names, or any user data
```

### 3.8. gRPC API (Admin Service)

```protobuf
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
    rpc GetTokenUsageReport(GetTokenUsageRequest) returns (GetTokenUsageResponse);

    // Health
    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

### 3.9. Prometheus Metrics (cross-service)

```yaml
# All services export to prometheus on :9090/metrics

# Ingestion metrics:
graphiti_ingestion_episodes_total{group_id, source, status}
graphiti_ingestion_pipeline_duration_seconds{step}
graphiti_ingestion_entities_extracted_total{group_id}
graphiti_ingestion_tokens_used_total{prompt_type}

# Knowledge metrics:
graphiti_knowledge_llm_calls_total{provider, model, prompt_type, status}
graphiti_knowledge_llm_duration_seconds{provider, prompt_type}
graphiti_knowledge_llm_tokens_total{prompt_type, token_type}
graphiti_knowledge_llm_cache_hits_total{prompt_type}
graphiti_knowledge_embedding_calls_total{provider, status}
graphiti_knowledge_resolution_decisions{decision_type}

# Search metrics:
graphiti_search_queries_total{config, status}
graphiti_search_duration_seconds{config}
graphiti_search_cache_hits_total{}
graphiti_search_rerank_duration_seconds{reranker}

# Store metrics:
graphiti_store_queries_total{operation, object_type, status}
graphiti_store_query_duration_seconds{operation, object_type}
graphiti_store_transactions_total{status}
graphiti_store_connection_pool_size{}
```

### 3.10. NATS Events (Admin)

| Subject | Publisher | Subscribers |
|---|---|---|
| `graphiti.tenant.created` | Admin | Store (init schema), Search (init index) |
| `graphiti.community.rebuilt` | Admin/Knowledge | Search (invalidate cache) |
| `graphiti.health.degraded` | Any service | Admin (alerting) |

---

## 4. Configuration

```yaml
server:
  grpc_port: 9005

services:
  knowledge: { address: "graphiti-knowledge:9003" }
  store:     { address: "graphiti-store:9004" }
  search:    { address: "graphiti-search:9002" }
  ingestion: { address: "graphiti-ingestion:9001" }

telemetry:
  disabled: false              # GRAPHITI_TELEMETRY_DISABLED=true to opt out
  posthog_key: "${POSTHOG_KEY}"

observability:
  otel_endpoint: "otel-collector:4317"
  sampling_rate: 1.0           # 100% in dev, 0.1 in production
  service_name: "graphiti-admin"
```

---

## 5. Acceptance Criteria

- [ ] `POST /v1/graphiti/admin/tenants` → tenant created, `GET /v1/graphiti/admin/tenants` trả về tenant trong list.
- [ ] `DELETE /v1/graphiti/admin/tenants/tenant-a` → all data for group-id "tenant-a" cleared.
- [ ] After `POST /v1/graphiti/episodes/bulk` với 100 episodes: `POST /v1/graphiti/admin/communities` → community detection runs, `CommunityNode` objects created.
- [ ] `GET /v1/graphiti/admin/token-usage` → trả về `{by_prompt: {extract_nodes: {total_tokens: N}}}`.
- [ ] `estimated_cost_usd` trong token usage report tính đúng với pricing config.
- [ ] OTel traces visible trong Jaeger UI: `ingestion.pipeline` span có đầy đủ child spans.
- [ ] `graphiti_knowledge_llm_calls_total` Prometheus metric tăng mỗi khi LLM được gọi.
- [ ] `GRAPHITI_TELEMETRY_DISABLED=true` → không có HTTP calls tới PostHog.
- [ ] gRPC health check `graphiti-knowledge:9003` → trả về `SERVING`.
- [ ] `POST /v1/graphiti/admin/indices` → Neo4j vector index được tạo (verify via Neo4j Browser).
