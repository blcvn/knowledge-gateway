# Zep Services

> **Engine**: Context Engineering for AI Agents | **Sub-200ms latency** | **Temporal KG via Graphiti**

---

# Zep User Service

> **Service**: `zep-user` | **gRPC Port**: 9061

## 1. Responsibility

User CRUD, metadata management (JSONB merge-patch), project-level isolation via `project_uuid`.

## 2. gRPC API

```protobuf
service ZepUserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc UpdateUser(UpdateUserRequest) returns (User);
  rpc DeleteUser(DeleteUserRequest) returns (Empty);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
```

## 3. Key Features

- JSONB metadata merge-patch with advisory locks
- Soft deletes (`deleted_at`)
- Project-scoped isolation

---

# Zep Thread Service

> **Service**: `zep-thread` | **gRPC Port**: 9062

## 1. Responsibility

Thread/session lifecycle management. Tracks `ended_at` for session boundary detection.

## 2. gRPC API

```protobuf
service ZepThreadService {
  rpc CreateThread(CreateThreadRequest) returns (Thread);
  rpc GetThread(GetThreadRequest) returns (Thread);
  rpc UpdateThread(UpdateThreadRequest) returns (Thread);
  rpc EndThread(EndThreadRequest) returns (Thread);
  rpc ListThreads(ListThreadsRequest) returns (ListThreadsResponse);
}
```

## 3. NATS: `zep.thread.session.ended` → zep-memory

---

# Zep Memory Service

> **Service**: `zep-memory` | **gRPC Port**: 9063

## 1. Responsibility

Message ingestion (PutMemory) and context assembly (GetMemory). Core synchronous path — sub-200ms target.

## 2. gRPC API

```protobuf
service ZepMemoryService {
  rpc PutMemory(PutMemoryRequest) returns (PutMemoryResponse);
  rpc GetMemory(GetMemoryRequest) returns (GetMemoryResponse);
  rpc DeleteMemory(DeleteMemoryRequest) returns (Empty);
}
```

## 3. PutMemory Flow

```
1. gRPC → zep-thread: UpsertSession
2. Check session.EndedAt
3. INSERT messages → PostgreSQL
4. NATS Publish: zep.memory.messages.ingested (async → zep-graph)
5. Return immediately (sub-200ms)
```

## 4. GetMemory Flow

```
1. Fetch messages from PostgreSQL
2. gRPC → zep-search: get relevant facts
3. Overlay facts onto message context
4. Return context-enriched memory
```

---

# Zep Graph Service

> **Service**: `zep-graph` | **gRPC Port**: 9064

## 1. Responsibility

Knowledge graph extraction via Graphiti. LLM-heavy async processing (10-20s). Temporal reasoning with `valid_at`/`invalid_at`.

## 2. gRPC API

```protobuf
service ZepGraphService {
  rpc AddFact(AddFactRequest) returns (Fact);
  rpc DeleteFact(DeleteFactRequest) returns (Empty);
  rpc GetEpisodes(GetEpisodesRequest) returns (EpisodesResponse);
  rpc SetOntology(SetOntologyRequest) returns (Empty);
  rpc GetOntology(GetOntologyRequest) returns (Ontology);
}
```

## 3. Extraction Pipeline (NATS subscriber)

```
NATS: zep.memory.messages.ingested
  → Graphiti PutMemory(sessionID, msgs)  [session scope]
  → Graphiti PutMemory(userID, msgs)     [user scope]
  → LLM entity extraction
  → Temporal annotation (valid_at/invalid_at)
  → Neo4j upsert (nodes + edges)
  → NATS Publish: zep.graph.extraction.completed
```

## 4. Storage: Neo4j 5.x (temporal KG), PostgreSQL (fact metadata)

---

# Zep Search Service

> **Service**: `zep-search` | **gRPC Port**: 9065

## 1. Responsibility

Semantic search across graph + sessions. 5 reranking strategies. Sub-200ms target for search.

## 2. gRPC API

```protobuf
service ZepSearchService {
  rpc GraphSearch(GraphSearchRequest) returns (SearchResponse);
  rpc SessionSearch(SessionSearchRequest) returns (SearchResponse);
}
```

## 3. Search Modes

| Mode | Scope | Source |
|------|-------|--------|
| `graph` | Temporal KG facts | Neo4j traversal + vector |
| `session` | Conversation messages | PostgreSQL + pgvector |

## 4. Reranking: RRF, MMR, Cross-Encoder, Temporal Decay, Node Priority

---

# Zep Admin Service

> **Service**: `zep-admin` | **gRPC Port**: 9066

## 1. Responsibility

Health aggregation, project/tenant management, API key lifecycle.

## 2. NATS Events

- `zep.admin.project.created` → All (init schema)
- `zep.admin.project.deleted` → All (cascade delete)
