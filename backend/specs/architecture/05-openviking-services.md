# OpenViking Services

---

# OV-FS Service (Filesystem)

> **Service**: `ov-fs` | **gRPC Port**: 9051

## 1. Responsibility

File CRUD, directory tree, grep, glob, relations. Go-native VikingFS replacing RAGFS Rust. Integrates with ov-crypto for transparent envelope encryption.

## 2. gRPC API

```protobuf
service OvFsService {
  rpc ReadFile(ReadFileRequest) returns (ReadFileResponse);
  rpc WriteFile(WriteFileRequest) returns (WriteFileResponse);
  rpc DeleteFile(DeleteFileRequest) returns (Empty);
  rpc MkDir(MkDirRequest) returns (Empty);
  rpc ListDir(ListDirRequest) returns (ListDirResponse);
  rpc Tree(TreeRequest) returns (TreeResponse);
  rpc Grep(GrepRequest) returns (GrepResponse);
  rpc Glob(GlobRequest) returns (GlobResponse);
  rpc Move(MoveRequest) returns (Empty);
  rpc GetRelations(GetRelationsRequest) returns (RelationsResponse);
}
```

## 3. Key Features

- **VikingFS**: Go-native filesystem with `viking://` URI scheme
- **PathLock**: point/subtree/mv locking for concurrent access
- **Tiered context**: L0 (abstract ~100 tokens), L1 (overview ~2K), L2 (full content)
- **Transparent encryption**: delegates to ov-crypto for envelope encrypt/decrypt

## 4. NATS Events

| Event | Subscriber |
|-------|------------|
| `ov.content.written` | ov-search (embed + upsert) |
| `ov.content.deleted` | ov-search (remove index) |

---

# OV-Search Service

> **Service**: `ov-search` | **gRPC Port**: 9052

## 1. Responsibility

Hierarchical retrieval with score propagation, reranking, hotness scoring, convergence detection.

## 2. gRPC API

```protobuf
service OvSearchService {
  rpc HierarchicalSearch(SearchRequest) returns (SearchResponse);
  rpc RetrieveContext(ContextRequest) returns (ContextResponse);
  rpc GetHotness(HotnessRequest) returns (HotnessResponse);
  rpc UpsertEmbedding(UpsertRequest) returns (Empty);
  rpc DeleteEmbedding(DeleteRequest) returns (Empty);
}
```

## 3. Search Pipeline

```
1. Query → embedding (via Bifrost)
2. Vector search (dense + sparse hybrid)
3. Score propagation (child → parent directory)
4. Hotness boost (recently accessed/modified)
5. Reranking (cross-encoder or RRF/MMR)
6. Convergence detection (stop when quality plateaus)
7. Tiered loading: L0 → L1 → L2 on demand
```

## 4. Hotness Scoring

- Session commit → boost referenced files
- Recent writes → higher score
- Decay over time (configurable half-life)

---

# OV-Session Service

> **Service**: `ov-session` | **gRPC Port**: 9053

## 1. Responsibility

Session lifecycle, 2-phase commit (archive + extract), Working Memory v2, memory extraction.

## 2. gRPC API

```protobuf
service OvSessionService {
  rpc CreateSession(CreateSessionRequest) returns (Session);
  rpc AddMessage(AddMessageRequest) returns (Empty);
  rpc GetMessages(GetMessagesRequest) returns (MessagesResponse);
  rpc CommitSession(CommitSessionRequest) returns (CommitResponse);
  rpc GetWorkingMemory(GetWMRequest) returns (WorkingMemory);
  rpc UpdateWorkingMemory(UpdateWMRequest) returns (WorkingMemory);
}
```

## 3. 2-Phase Commit

```
Phase 1 (Archive): compress conversation → write to ov-fs
Phase 2 (Extract): LLM extract memories → write to ov-fs
```

## 4. Working Memory v2

Structured document: `{title, state, goals, facts, errors, context}`

---

# OV-Resource Service

> **Service**: `ov-resource` | **gRPC Port**: 9054

## 1. Responsibility

Resource ingestion pipeline: parse files (tree-sitter for code, markdown, PDF/DOCX), generate embeddings, watch for changes.

## 2. gRPC API

```protobuf
service OvResourceService {
  rpc Ingest(IngestRequest) returns (IngestResponse);
  rpc Parse(ParseRequest) returns (ParseResponse);
  rpc Watch(WatchRequest) returns (stream WatchEvent);
  rpc Refresh(RefreshRequest) returns (RefreshResponse);
}
```

## 3. Parse Engine

| Format | Parser | Chunking |
|--------|--------|----------|
| Code (Go/Py/JS/TS) | tree-sitter | AST-aware |
| Markdown | Custom | Section-based |
| PDF/DOCX | Go libraries | Page-based |
| Plain text | — | Paragraph |

## 4. NATS: `ov.resource.ingested` → ov-search

---

# OV-Crypto Service

> **Service**: `ov-crypto` | **gRPC Port**: 9055

## 1. Responsibility

Envelope encryption (AES-256-GCM per file), KMS adapter abstraction, key rotation.

## 2. gRPC API

```protobuf
service OvCryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  rpc RotateKey(RotateKeyRequest) returns (RotateKeyResponse);
  rpc GetKeyStatus(KeyStatusRequest) returns (KeyStatus);
}
```

## 3. KMS Backends

| Backend | Use Case |
|---------|----------|
| Local (file-based) | Dev/testing |
| HashiCorp Vault | Production |
| AWS KMS / GCP KMS | Cloud deployments |

---

# OV-Admin Service

> **Service**: `ov-admin` | **gRPC Port**: 9056

## 1. Responsibility

Account/User/Agent CRUD, API key management, health aggregation, maintenance tasks.

## 2. gRPC API

```protobuf
service OvAdminService {
  rpc CreateAccount(CreateAccountRequest) returns (Account);
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (APIKey);
  rpc RevokeAPIKey(RevokeRequest) returns (Empty);
  rpc GetHealth(Empty) returns (HealthResponse);
}
```

## 3. RBAC Model

```
Account → User → Agent (namespace isolation)
Roles: OWNER, ADMIN, USER, AGENT
```
