# Platform Services

---

# VNP Event Service

> **Service**: `vnp-event` | **gRPC Port**: 9041

## 1. Responsibility

Cross-domain event timeline: stores temporal events from all engines (Memobase events, Graphiti episodes, Cognee pipeline events). Semantic + temporal search.

## 2. gRPC API

```protobuf
service VnpEventService {
  rpc CreateEvent(CreateEventRequest) returns (Event);
  rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
  rpc SearchGists(SearchGistsRequest) returns (SearchGistsResponse);
  rpc GetTimeline(GetTimelineRequest) returns (TimelineResponse);
  rpc FilterByTags(FilterByTagsRequest) returns (FilterResponse);
}
```

## 3. Event Model

```go
type UserEvent struct {
    ID        uuid.UUID
    UserID    string
    TenantID  string
    Source    EventSource  // MEMOBASE, GRAPHITI, COGNEE
    Content   string
    Tags      []string
    Embedding []float64
    CreatedAt time.Time
    ValidAt   *time.Time
    InvalidAt *time.Time
}
```

## 4. Storage

- **PostgreSQL + pgvector**: Event storage + vector search
- **Redis**: Recent events cache

---

# VNP Search Hub Service

> **Service**: `vnp-search-hub` | **gRPC Port**: 9042

## 1. Responsibility

Cross-engine search orchestration. Fan-out queries to all engine search services, merge + rerank results into unified response. This is the `memory.recall()` backend.

## 2. gRPC API

```protobuf
service VnpSearchHubService {
  rpc Recall(RecallRequest) returns (RecallResponse);
  rpc MultiSearch(MultiSearchRequest) returns (MultiSearchResponse);
}
```

## 3. Recall Pipeline

```
RecallRequest{query, tenant_id, scope, max_results}
  │
  ├── Parallel fan-out (errgroup):
  │   ├── cognee-search.Search(query)           → []DocumentChunk
  │   ├── graphiti-search.HybridSearch(query)    → []GraphFact
  │   ├── memobase-context.GetContext(user_id)   → ContextString
  │   └── vnp-event.SearchEvents(query)          → []UserEvent
  │
  ├── Merge + Dedup (by content hash)
  ├── Rerank (RRF / MMR / Cross-Encoder)
  ├── Token budget truncation
  │
  ▼
RecallResponse {
  profiles:   []ProfileSection
  facts:      []GraphFact
  events:     []TemporalEvent
  documents:  []DocumentChunk
  context:    string  // pre-formatted for LLM prompt
  metadata:   RecallMetadata{latency_ms, engines_used[], total_results}
}
```

## 4. Reranking Strategies

| Strategy | Use Case |
|----------|----------|
| RRF (Reciprocal Rank Fusion) | Default, fast |
| MMR (Maximal Marginal Relevance) | Diversity-focused |
| Cross-Encoder | Highest quality (via graphiti-knowledge) |

## 5. SLA Targets

| Metric | Target |
|--------|--------|
| Recall latency (p95) | < 500ms |
| Profile retrieval (p95) | < 100ms |
| Fan-out timeout | 2s max per engine |

---

# VNP Admin Service

> **Service**: `vnp-admin` | **gRPC Port**: 9050

## 1. Responsibility

Tenant management, API key lifecycle, user CRUD, health aggregation, billing, config management.

## 2. gRPC API

```protobuf
service VnpAdminService {
  // Tenant management
  rpc CreateTenant(CreateTenantRequest) returns (Tenant);
  rpc GetTenant(GetTenantRequest) returns (Tenant);
  rpc UpdateTenant(UpdateTenantRequest) returns (Tenant);
  rpc DeleteTenant(DeleteTenantRequest) returns (Empty);
  // API Keys
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (APIKey);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (Empty);
  rpc RotateAPIKey(RotateAPIKeyRequest) returns (APIKey);
  // Users
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc DeleteUser(DeleteUserRequest) returns (Empty);
  // Health
  rpc AggregatedHealth(Empty) returns (HealthResponse);
  // Config
  rpc GetConfig(GetConfigRequest) returns (ConfigResponse);
  rpc UpdateConfig(UpdateConfigRequest) returns (ConfigResponse);
}
```

## 3. NATS Events

| Event | Subscriber |
|-------|------------|
| `admin.tenant.created` | All services (init schema/cache) |
| `admin.tenant.deleted` | All services (cascade delete) |
| `admin.user.deleted` | memobase-*, vnp-event (cascade) |

## 4. Storage

- **PostgreSQL**: Tenants, API keys (hashed), users, billing, config
