# 12 — Unified Data Models & Protobuf

---

## 1. Protobuf Organization

```
api/proto/
├── buf.yaml
├── common/v1/
│   ├── pagination.proto        # CursorRequest, CursorResponse
│   ├── temporal.proto          # Timestamp, TimeRange, BiTemporal
│   ├── health.proto            # HealthStatus, ServiceHealth
│   └── errors.proto            # ErrorDetail, ErrorCode
├── graph/v1/
│   ├── node.proto              # EntityNode, EpisodicNode, CommunityNode
│   └── edge.proto              # EntityEdge, EpisodicEdge
├── cognee/
│   ├── ingestion/v1/
│   │   └── ingestion.proto     # AddData, ListDatasets, DeleteDataset
│   ├── cognify/v1/
│   │   └── cognify.proto       # StartCognify, GetStatus
│   ├── search/v1/
│   │   └── search.proto        # Search, ExploreGraph
│   └── memory/v1/
│       └── memory.proto        # Remember, Recall, Forget
├── graphiti/
│   ├── ingestion/v1/
│   │   └── ingestion.proto     # IngestEpisode, ListEpisodes
│   ├── search/v1/
│   │   └── search.proto        # HybridSearch, EntitySearch
│   ├── knowledge/v1/
│   │   └── knowledge.proto     # ExtractEntities, ResolveEntities
│   └── store/v1/
│       └── store.proto         # CRUD for nodes, edges, episodes
└── admin/v1/
    └── admin.proto             # Tenants, Users, APIKeys, Health
```

---

## 2. Common Proto Definitions

```protobuf
// common/v1/pagination.proto
message CursorRequest {
  int32 page_size = 1;
  string cursor = 2;          // opaque cursor for next page
}

message CursorResponse {
  string next_cursor = 1;
  bool has_more = 2;
  int32 total_count = 3;
}

// common/v1/temporal.proto
message BiTemporal {
  google.protobuf.Timestamp created_at = 1;
  google.protobuf.Timestamp updated_at = 2;
  google.protobuf.Timestamp valid_from = 3;
  google.protobuf.Timestamp valid_to = 4;    // nil = still valid
  google.protobuf.Timestamp invalid_at = 5;  // nil = not invalidated
}
```

---

## 3. Cognee Service Protos (Examples)

```protobuf
// cognee/ingestion/v1/ingestion.proto
service IngestionService {
  rpc AddData(AddDataRequest) returns (AddDataResponse);
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc DeleteDataset(DeleteDatasetRequest) returns (DeleteDatasetResponse);
  rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);
}

// cognee/search/v1/search.proto
service SearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc ExploreGraph(ExploreGraphRequest) returns (ExploreGraphResponse);
}

message SearchRequest {
  string query = 1;
  repeated string strategies = 2;  // SIMILARITY, HYBRID, etc.
  int32 top_k = 3;
  string dataset_id = 4;
  bool rerank = 5;
}

// cognee/memory/v1/memory.proto
service MemoryService {
  rpc Remember(RememberRequest) returns (RememberResponse);
  rpc Recall(RecallRequest) returns (RecallResponse);
  rpc Forget(ForgetRequest) returns (ForgetResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc PersistSession(PersistSessionRequest) returns (PersistSessionResponse);
}
```

---

## 4. Graphiti Service Protos (Examples)

```protobuf
// graphiti/store/v1/store.proto
service StoreService {
  rpc CreateEntity(CreateEntityRequest) returns (CreateEntityResponse);
  rpc GetEntity(GetEntityRequest) returns (GetEntityResponse);
  rpc UpdateEntity(UpdateEntityRequest) returns (UpdateEntityResponse);
  rpc DeleteEntity(DeleteEntityRequest) returns (DeleteEntityResponse);
  rpc CreateEdge(CreateEdgeRequest) returns (CreateEdgeResponse);
  rpc GetEdges(GetEdgesRequest) returns (GetEdgesResponse);
  rpc InvalidateEdge(InvalidateEdgeRequest) returns (InvalidateEdgeResponse);
  rpc SaveEpisode(SaveEpisodeRequest) returns (SaveEpisodeResponse);
}

// graphiti/knowledge/v1/knowledge.proto
service KnowledgeService {
  rpc ExtractEntities(ExtractEntitiesRequest) returns (ExtractEntitiesResponse);
  rpc ResolveEntities(ResolveEntitiesRequest) returns (ResolveEntitiesResponse);
  rpc InvalidateEdges(InvalidateEdgesRequest) returns (InvalidateEdgesResponse);
  rpc BuildCommunities(BuildCommunitiesRequest) returns (BuildCommunitiesResponse);
  rpc GenerateEmbeddings(GenerateEmbeddingsRequest) returns (GenerateEmbeddingsResponse);
  rpc Rerank(RerankRequest) returns (RerankResponse);
}
```

---

## 5. PostgreSQL Schema (Admin + Cognee Metadata)

```sql
-- Multi-tenant base
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    key_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    scopes TEXT[],
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Row-Level Security for Cognee
ALTER TABLE datasets ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON datasets
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

-- Cognee metadata
CREATE TABLE datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE data_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID NOT NULL REFERENCES datasets(id),
    tenant_id UUID NOT NULL,
    content_type VARCHAR(50),
    storage_path TEXT,
    file_hash VARCHAR(64),
    status VARCHAR(20) DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE pipeline_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID NOT NULL REFERENCES datasets(id),
    tenant_id UUID NOT NULL,
    stage VARCHAR(50),
    status VARCHAR(20) DEFAULT 'RUNNING',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error TEXT
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    state VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active TIMESTAMPTZ
);

CREATE TABLE session_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id),
    role VARCHAR(20),
    content TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```
