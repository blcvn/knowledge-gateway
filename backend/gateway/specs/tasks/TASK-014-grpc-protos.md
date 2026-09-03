---
id: TASK-014
title: gRPC Proto Definitions — Typed Service Contracts
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-10
updated: 2026-05-10
completed: 2026-05-10
linked_sol: SOL-001
depends_on: [TASK-007]
estimate: 4h
actual: 3h
---

## Mục Tiêu

Define `.proto` files cho 8 gRPC service APIs covering all 6 cognitive engines + unified memory + admin. 52 RPCs total.

## Phạm Vi

### Files đã tạo

| File | Package | RPCs | Messages | Lines |
|------|---------|------|----------|-------|
| `proto/memory/v1/memory.proto` | vnp.gateway.memory.v1 | 4 | 10 | 92 |
| `proto/cognee/v1/cognee.proto` | vnp.gateway.cognee.v1 | 4 | 9 | 72 |
| `proto/graphiti/v1/graphiti.proto` | vnp.gateway.graphiti.v1 | 4 | 9 | 88 |
| `proto/memobase/v1/memobase.proto` | vnp.gateway.memobase.v1 | 7 | 16 | 114 |
| `proto/openviking/v1/openviking.proto` | vnp.gateway.openviking.v1 | 11 | 25 | 174 |
| `proto/zep/v1/zep.proto` | vnp.gateway.zep.v1 | 9 | 18 | 153 |
| `proto/supermemory/v1/supermemory.proto` | vnp.gateway.supermemory.v1 | 9 | 19 | 166 |
| `proto/admin/v1/admin.proto` | vnp.gateway.admin.v1 | 4 | 8 | 56 |
| **Total** | | **52** | **114** | **915** |

### Additional config files
- `proto/buf.yaml` — Buf module config (STANDARD lint, FILE breaking)
- `proto/buf.gen.yaml` — Go codegen (protoc-gen-go + protoc-gen-go-grpc)

### Chi tiết triển khai

#### Service → Proto mapping (52 RPCs = 50 REST routes + 2 admin)

| REST Namespace | Proto Service | RPCs |
|----------------|---------------|------|
| `/v1/memory/*` | `MemoryService` | Store, Recall, Forget, Timeline |
| `/v1/cognee/*` | `CogneeIngestionService`, `CogneeSearchService` | CreateDataset, UploadData, Cognify, Search |
| `/v1/graphiti/*` | `GraphitiIngestionService`, `GraphitiSearchService`, `GraphitiStoreService` | IngestEpisode, Search, GetNode, GetEdge |
| `/v1/memobase/*` | `MemobaseIngestionService`, `MemobaseContextService` | InsertBlob, FlushBuffer, GetBufferStatus, DeleteBlob, GetContext, GetProfiles, GetEvents |
| `/v1/ov/*` | `OVFileService`, `OVSearchService`, `OVSessionService`, `OVResourceService` | ReadFile, WriteFile, DeleteFile, ListDir, Tree, Grep, Search, CreateSession, AddMessage, CommitSession, Ingest |
| `/v1/zep/*` | `ZepUserService`, `ZepMemoryService`, `ZepSearchService`, `ZepGraphService` | CreateUser, GetUser, UpdateUser, PutMemory, GetMemory, SessionSearch, GraphSearch, AddFact, SetOntology |
| `/v1/sm/*` | `SMDocumentService`, `SMMemoryService`, `SMSearchService`, `SMProfileService`, `SMConnectorService`, `SMProjectService` | CreateDocument, GetDocument, CreateMemory, Search, RAG, GetProfile, CreateConnection, SyncConnection, CreateSpace |
| `/v1/admin/*` | `AdminService` | CreateTenant, IssueAPIKey, Health, Metrics |

#### Proto design patterns
- All RPCs use request/response message pairs (no streaming in v1)
- Common fields: `map<string, string> metadata`, `google.protobuf.Timestamp`
- `google.protobuf.Struct` for flexible JSON-like payloads (Zep ontology, user metadata)
- Consistent naming: `*Request` / `*Response` suffix
- Each engine is its own package for independent evolution

> **Thay đổi so với spec**: Created 8 proto files (not 7) — added `admin.proto` for completeness. Total 52 RPCs covering all 50 REST routes plus 2 extra admin RPCs. Code generation (`buf generate`) deferred until `buf` CLI is installed.

## Acceptance Criteria

- [x] AC-1: Proto files define all RPCs matching 50+ REST routes ✅ (52 RPCs)
- [x] AC-2: `buf generate` config ready in `buf.gen.yaml` ✅ (awaiting `buf` CLI)
- [x] AC-3: Proto message types match domain entities ✅
- [x] AC-4: Backward-compatible: raw `[]byte` forwarding still works ✅ (existing code unchanged)
- [x] AC-5: Proto files include standard well-known types (Timestamp, Struct, Empty) ✅

## Verification

```bash
# Proto files created:
find proto/ -name '*.proto' | wc -l  # → 8
grep -r 'rpc ' proto/ | wc -l       # → 52

# Go build unaffected:
go build ./internal/...    # ✅ PASS
go test ./internal/... ./tests/...  # ✅ 28 tests PASS
```
