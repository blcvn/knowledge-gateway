# Supermemory Services

> **Engine**: Adaptive Knowledge Graph Memory | **Forgetting Curve** | **Multi-source Connectors**

---

# SM-Document Service

> **Service**: `sm-document` | **gRPC Port**: 9071

## 1. Responsibility

Document CRUD, chunking, ingestion pipeline (PDF/HTML/text/image), content extraction.

## 2. gRPC API

```protobuf
service SmDocumentService {
  rpc CreateDocument(CreateDocumentRequest) returns (Document);
  rpc GetDocument(GetDocumentRequest) returns (Document);
  rpc DeleteDocument(DeleteDocumentRequest) returns (Empty);
  rpc ListDocuments(ListDocumentsRequest) returns (ListDocumentsResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

## 3. NATS Events

- `sm.document.created` → sm-memory (extract facts), sm-search (index)
- `sm.document.deleted` → sm-memory (cleanup), sm-search (deindex)

---

# SM-Memory Service

> **Service**: `sm-memory` | **gRPC Port**: 9072

## 1. Responsibility

Memory engine: fact extraction from documents, knowledge graph construction, forgetting curve decay.

## 2. gRPC API

```protobuf
service SmMemoryService {
  rpc CreateMemory(CreateMemoryRequest) returns (Memory);
  rpc GetMemory(GetMemoryRequest) returns (Memory);
  rpc ForgetMemory(ForgetMemoryRequest) returns (Empty);
  rpc ListMemories(ListMemoriesRequest) returns (ListMemoriesResponse);
  rpc CreateRelation(CreateRelationRequest) returns (Relation);
}
```

## 3. Forgetting Curve

```go
// Ebbinghaus-inspired decay
func (m *Memory) ShouldForget(now time.Time) bool {
    age := now.Sub(m.LastAccessedAt)
    strength := m.AccessCount * m.RelevanceScore
    return age > decayThreshold(strength)
}
```

## 4. NATS: `sm.memory.created` → sm-search, sm-profile

---

# SM-Search Service

> **Service**: `sm-search` | **gRPC Port**: 9073

## 1. Responsibility

Hybrid search (vector + fulltext), RAG completion, reranking.

## 2. gRPC API

```protobuf
service SmSearchService {
  rpc HybridSearch(SearchRequest) returns (SearchResponse);
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
}
```

## 3. Pipeline: embedding → pgvector + fulltext → RRF merge → rerank → return

---

# SM-Profile Service

> **Service**: `sm-profile` | **gRPC Port**: 9074

## 1. Responsibility

User profiles (static preferences + dynamic learned traits). Updated from memory events.

## 2. gRPC API

```protobuf
service SmProfileService {
  rpc GetProfile(GetProfileRequest) returns (Profile);
  rpc UpdateProfile(UpdateProfileRequest) returns (Profile);
  rpc GetDynamicTraits(GetTraitsRequest) returns (TraitsResponse);
}
```

---

# SM-Connector Service

> **Service**: `sm-connector` | **gRPC Port**: 9075

## 1. Responsibility

External data source sync: Google Drive, Notion, OneDrive, GitHub. OAuth2 flow, incremental sync.

## 2. gRPC API

```protobuf
service SmConnectorService {
  rpc CreateConnection(CreateConnectionRequest) returns (Connection);
  rpc SyncConnection(SyncConnectionRequest) returns (SyncResponse);
  rpc GetSyncStatus(GetStatusRequest) returns (SyncStatus);
  rpc DeleteConnection(DeleteConnectionRequest) returns (Empty);
}
```

## 3. NATS: `sm.connection.synced` → sm-document (batch ingest)

---

# SM-MCP Service

> **Service**: `sm-mcp` | **gRPC Port**: 9076

## 1. Responsibility

MCP server for AI agent integration. SSE/JSON-RPC transport.

## 2. MCP Tools

| Tool | Target |
|------|--------|
| `add_memory` | sm-memory |
| `search_memory` | sm-search |
| `get_profile` | sm-profile |
| `list_documents` | sm-document |
| `rag_query` | sm-search |

---

# SM-Auth Service

> **Service**: `sm-auth` | **gRPC Port**: 9077

## 1. Responsibility

Authentication, API key management, RBAC, organization management.

## 2. Key Features

- JWT RS256 + API Keys (`sm_` prefix)
- Organization-level isolation (`org_id`)
- Role-based access: Owner, Admin, Member

---

# SM-Analytics Service

> **Service**: `sm-analytics` | **gRPC Port**: 9078

## 1. Responsibility

Usage tracking, token economics, reporting dashboards.

## 2. Tracks: API requests, token usage, storage, active users per org

---

# SM-Project Service

> **Service**: `sm-project` | **gRPC Port**: 9079

## 1. Responsibility

Spaces (containers), container tags, membership management.

## 2. gRPC API

```protobuf
service SmProjectService {
  rpc CreateSpace(CreateSpaceRequest) returns (Space);
  rpc AddToSpace(AddToSpaceRequest) returns (Empty);
  rpc ListSpaces(ListSpacesRequest) returns (ListSpacesResponse);
  rpc ManageTags(ManageTagsRequest) returns (TagsResponse);
}
```
