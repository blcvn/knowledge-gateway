# SOL-003 — Migration Map: Service Cũ → Service Mới

## Quick Reference Table

| Service Cũ | Lines | Trạng Thái | → Service Mới | Ghi Chú |
|------------|-------|------------|----------------|---------|
| **VNP Core** | | | | |
| vnp-admin | 1,925 | ✅ Archived | → vnp-platform | Admin + Tenant CRUD |
| vnp-event | 985 | ✅ Archived | → vnp-platform | NATS publisher, GDPR |
| vnp-platform | 1,455 | ✅ Expanded | → vnp-platform | **Kept & expanded** |
| vnp-dashboard | 228 | ✅ Archived | → vnp-platform | Dashboard metrics |
| vnp-infra | 203 | ✅ Archived | → obs-service | Infra topology |
| vnp-observability | 234 | ✅ Archived | → obs-service | Metrics/traces |
| vnp-pipelines | 205 | ✅ Archived | → pipeline-service | Pipeline mgmt API |
| vnp-search-hub | 1,120 | ✅ Expanded | → search-service | **Kept & expanded** |
| **Cognee** | | | | |
| cognee-ingestion | 4,588 | ✅ Archived | → kg-service | Cognee dataset API |
| cognee-cognify | 2,488 | ✅ Archived | → kg-service | Cognification pipeline |
| cognee-pipeline | 655 | ✅ Archived | → kg-service | Cognee batch |
| cognee-search | 1,192 | ✅ Archived | → kg-service | Cognee search + ontology |
| **Graphiti** | | | | |
| graphiti-ingestion | 1,785 | ✅ Archived | → kg-service | Episode ingest |
| graphiti-knowledge | 2,296 | ✅ Archived | → kg-service | Knowledge extract |
| graphiti-pipeline | 1,706 | ✅ Archived | → kg-service | Graphiti batch |
| graphiti-search | 3,005 | ✅ Archived | → kg-service | Graph search |
| graphiti-store | 3,934 | ✅ Archived | → kg-service | Node/Edge CRUD + subgraph |
| **Memobase** | | | | |
| memobase-context | 1,173 | ✅ Archived | → memory-service | Context retrieval, profiles |
| memobase-engine | 958 | ✅ Archived | → memory-service | Scoring engine |
| memobase-ingestion | 799 | ✅ Archived | → memory-service | Blob insertion, buffers |
| memobase-pipeline | 1,051 | ✅ Archived | → memory-service | Memobase batch |
| **OpenViking** | | | | |
| ov-admin | 965 | ✅ Archived | → vnp-platform | Account/Agent mgmt |
| ov-crypto | 926 | ✅ Archived | → storage-service | Encryption ops |
| ov-fs | 1,562 | ✅ Archived | → storage-service | File operations |
| ov-resource | 1,406 | ✅ Archived | → storage-service | Resource ingest |
| ov-search | 1,259 | ✅ Archived | → search-service | OV semantic search |
| ov-session | 1,683 | ✅ Archived | → storage-service | Chat sessions (working memory) |
| ov-storage | 1,084 | ✅ Archived | → storage-service | **Base structure** |
| **Supermemory** | | | | |
| sm-analytics | 127 | ✅ Archived | → vnp-platform | Analytics agg |
| sm-auth | 1,415 | ✅ Archived | → vnp-platform | JWT + Google SSO |
| sm-connector | 196 | ✅ Archived | → search-service | External connectors |
| sm-document | 144 | ✅ Archived | → memory-service | Document CRUD |
| sm-engine | 410 | ✅ Archived | → obs-service | Engine metrics/analytics |
| sm-mcp | 142 | ✅ Archived | → search-service | MCP tools |
| sm-memory | 207 | ✅ Archived | → memory-service / search-service | SM memory entries (see note below) |
| sm-profile | 137 | ✅ Archived | → memory-service | User profiles |
| sm-project | 361 | ✅ Archived | → vnp-platform | Space management |
| sm-search | 127 | ✅ Archived | → search-service | SM search + RAG |
| **Zep** | | | | |
| zep-admin | 308 | ✅ Archived | → vnp-platform | Project mgmt |
| zep-core | 850 | ✅ Archived | → memory-service | Core Zep adapter (sessions) |
| zep-go | 14,749 | **External SDK** | → (library dep) | Go client SDK — stays as dependency |
| zep-graph | 310 | ✅ Archived | → memory-service | Graph facts |
| zep-memory | 187 | ✅ Archived | → memory-service | Memory put/get |
| zep-search | 616 | ✅ Archived | → memory-service | Zep search |
| zep-thread | 179 | ✅ Archived | → memory-service | Thread/session |
| zep-user | 179 | ✅ Archived | → memory-service | User management |
| **BA Knowledge** | | | | |
| ba-knowledge-service | 2,848 | ✅ Archived | → pipeline-service | BA knowledge API |
| ba-knowledge-worker | 1,061 | ✅ Archived | → pipeline-service | Redis queue worker |


---

## Kết Quả (Thực Tế — Đã Hoàn Thành 2026-06-11)

```
TRƯỚC                    SAU
━━━━━━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
48 go modules           8 go modules
47 service containers   7 backend containers + 1 worker
48 Dockerfiles          8 Dockerfiles
48 go.mod files         8 go.mod files

Services thật (có logic): 3        Services thật: 8 (100%) — tất cả build ✅
Services stub (no routes): 44      Stubs: 0
Services skeleton (TODO): 1        Archived: 46 → services/archived/
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Reduction: -83.3% services    Quality: stubs → real implementations
Build verification: 2026-06-11 — go build ✅ PASS (all 8 services)
```


## Service → Port Mapping

| Service | gRPC Port | Health Port | Vai Trò |
|---------|-----------|-------------|---------|
| vnp-gateway | 8080 (REST), 8082 (MCP) | 11080 | Entry point |
| vnp-platform | 9010 | 9110 | Auth, Admin, Tenant, Events |
| kg-service | 9020 | 9120 | Knowledge Graph |
| memory-service | 9030 | 9130 | Working Memory |
| storage-service | 9040 | 9140 | Files & Resources |
| search-service | 9050 | 9150 | Cross-engine Search |
| pipeline-service | 9060 | 9160 | Async Processing |
| obs-service | 9070 | 9170 | Observability |

---

## Gateway Service Name Mapping (cho `defaultServiceAddresses()`)

Đây là mapping từ các service name trong `gateway/adapter/handler/services.go` và `console.go`
sang service name mới trong `config.go`:

| Service Name Cũ (trong ForwardToService calls) | Service Name Mới | Ghi Chú |
|------------------------------------------------|-----------------|---------|
| `"cognee-ingestion"` | `"kg-service"` | CogneeHandler.CreateDataset, UploadData |
| `"cognee-cognify"` | `"kg-service"` | CogneeHandler.Cognify |
| `"cognee-search"` | `"kg-service"` | CogneeHandler.Search, GraphHandler.Ontology |
| `"graphiti-ingestion"` | `"kg-service"` | GraphitiHandler.IngestEpisode |
| `"graphiti-search"` | `"kg-service"` | GraphitiHandler.Search |
| `"graphiti-knowledge"` | `"kg-service"` | Không dùng trong handler hiện tại |
| `"graphiti-store"` | `"kg-service"` | GraphitiHandler.GetNode/Edge, GraphHandler.* |
| `"memobase-ingestion"` | `"memory-service"` | MemobaseHandler.InsertBlob/Flush, ProfileHandler.GetBuffers |
| `"memobase-engine"` | `"memory-service"` | Không dùng trong handler hiện tại |
| `"memobase-context"` | `"memory-service"` | MemobaseHandler.GetContext/Profiles, ProfileHandler.* |
| `"vnp-event"` | `"vnp-platform"` | MemobaseHandler.GetEvents, ProfileHandler.GetEvents, GovernanceHandler.GDPR*, DebuggerHandler.GetTrace/ListTraces |
| `"vnp-search-hub"` | `"search-service"` | ExplorerHandler.Search/GetMemory/GetNeighbors, DebuggerHandler.CreateTrace |
| `"vnp-admin"` | `"vnp-platform"` | AdminHandler.*, GovernanceHandler.* |
| `"vnp-dashboard"` | `"vnp-platform"` | DashboardHandler.* |
| `"vnp-pipelines"` | `"pipeline-service"` | PipelineHandler.* |
| `"vnp-infra"` | `"obs-service"` | InfraHandler.* |
| `"vnp-observability"` | `"obs-service"` | ObservabilityHandler.* |
| `"ov-fs"` | `"storage-service"` | OpenVikingHandler.ReadFile/WriteFile/DeleteFile/Tree/Grep |
| `"ov-search"` | `"search-service"` | OpenVikingHandler.Search |
| `"ov-session"` | `"storage-service"` | OpenVikingHandler.CreateSession/AddMessage/CommitSession, SessionHandler.GetWorkingMemory |
| `"ov-resource"` | `"storage-service"` | OpenVikingHandler.Ingest |
| `"ov-crypto"` | `"storage-service"` | Không dùng trong handler hiện tại |
| `"ov-admin"` | `"vnp-platform"` | Không dùng trong handler hiện tại |
| `"zep-user"` | `"memory-service"` | ZepHandler.CreateUser/GetUser/UpdateUser |
| `"zep-thread"` | `"memory-service"` | Không dùng trong handler hiện tại |
| `"zep-memory"` | `"memory-service"` | ZepHandler.PutMemory/GetMemory |
| `"zep-graph"` | `"memory-service"` | ZepHandler.AddFact/SetOntology |
| `"zep-search"` | `"memory-service"` | ZepHandler.GraphSearch/SessionSearch |
| `"zep-admin"` | `"vnp-platform"` | Không dùng trong handler hiện tại |
| `"zep-core"` | `"memory-service"` | SessionHandler.ListSessions/GetSession/GetTimeline/ListLiveSessions |
| `"sm-document"` | `"memory-service"` | SMHandler.CreateDocument/GetDocument |
| `"sm-memory"` (SMHandler) | `"memory-service"` | ⚠️ **SMHandler.CreateMemory** — services.go:207 |
| `"sm-memory"` (console) | `"search-service"` | ⚠️ **ExplorerHandler.GetVersions, AdaptiveHandler.ListMemories/GetVersions** — console.go |
| `"sm-search"` | `"search-service"` | SMHandler.Search, SMHandler.RAG |
| `"sm-profile"` | `"memory-service"` | SMHandler.GetProfile |
| `"sm-connector"` | `"search-service"` | SMHandler.CreateConnection/SyncConnection, AdaptiveHandler.* |
| `"sm-mcp"` | `"search-service"` | Không dùng trong handler hiện tại |
| `"sm-auth"` | `"vnp-platform"` | Không dùng trong handler hiện tại |
| `"sm-analytics"` | `"vnp-platform"` | Không dùng trong handler hiện tại |
| `"sm-project"` | `"vnp-platform"` | SMHandler.CreateSpace |
| `"sm-engine"` | `"obs-service"` | AdaptiveHandler.GetAnalytics/GetForgetRules/UpdateForgetRules |
| `"ba-knowledge-service"` | `"pipeline-service"` | Không dùng trong handler hiện tại |
| `"ba-knowledge-worker"` | `"pipeline-service"` | Không dùng trong handler hiện tại |

> ⚠️ **`"sm-memory"` là đặc biệt**: Cùng string nhưng dùng ở 2 context — `SMHandler.CreateMemory` (services.go) → `memory-service`; `ExplorerHandler`/`AdaptiveHandler` (console.go) → `search-service`. Xem diff chi tiết trong MERGE-P4-T3.
