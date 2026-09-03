# VNP Memory — Data Models Index

> **Workspace**: `vnp-memory`  
> **Generated**: 2026-06-18  
> **Source**: `services/*/internal/domain/**/*.go`

This directory contains extracted domain model specifications for each service in the VNP Memory platform.

---

## Service Groups

### 🏛️ Platform & Admin

| File | Service(s) | Description |
|------|-----------|-------------|
| [vnp-platform.md](./vnp-platform.md) | `vnp-platform` | Auth, tenant management, project spaces, analytics, event timeline (absorbs sm-auth, sm-project, sm-analytics, vnp-admin*, ov-admin*, zep-admin*) |
| [vnp-admin.md](./vnp-admin.md) | `vnp-admin` | Multi-tenant admin: Tenant, User, APIKey, Policy (OPA), AuditLog |
| [vnp-event.md](./vnp-event.md) | `vnp-event` | Bi-temporal cross-engine event store: UserEvent, EventGist, Timeline |

### 🧠 Memory

| File | Service(s) | Description |
|------|-----------|-------------|
| [memory-service.md](./memory-service.md) | `memory-service` | Unified memory facade: AgentMemory, MemorySlot, consolidation entities, Zep/Memobase/SM integration |
| [sm-engine.md](./sm-engine.md) | `sm-engine` | Supermemory adaptive engine: Ebbinghaus Memory, Relation, Document, Chunk, adaptive Profile |

### 🗂️ Knowledge Graph

| File | Service(s) | Description |
|------|-----------|-------------|
| [kg-service.md](./kg-service.md) | `kg-service` | KG facade: Cognee (Dataset, DataItem, CognifyJob, MemifyJob, DataPoint) + Graphiti (Episode, Node, Edge, Fact, Ontology) |
| [cognee-cognify.md](./cognee-cognify.md) | `cognee-cognify`, `cognee-*` | Cognee pipeline engine: GraphNode, Entity, Relationship, CognifyJob, PipelineMetrics, Ontology, GraphDiff |
| [graphiti.md](./graphiti.md) | `graphiti-*` | Graphiti engine: ExtractedEntity, ExtractedEdge, CommunityNode, Resolution, Tenant, TenantStats |

### 🔍 Search

| File | Service(s) | Description |
|------|-----------|-------------|
| [search-service.md](./search-service.md) | `search-service` | Unified search: SearchResult (hybrid BM25+vector), ContextBlock, Observation, AgentMemory |
| [ov-search.md](./ov-search.md) | `ov-search` | OpenViking search: EmbeddingVector, UpsertPayload, SearchResult, HotnessScore, DecayConfig |

### 💾 Storage

| File | Service(s) | Description |
|------|-----------|-------------|
| [storage-service.md](./storage-service.md) | `storage-service` | Unified storage: File, TreeNode, Resource, IngestJob, Session, WorkingMemory, Message, CandidateMemory |
| [ov-fs.md](./ov-fs.md) | `ov-fs` | OpenViking virtual filesystem: FSNode, TreeNode, FileRelation |
| [ov-resource.md](./ov-resource.md) | `ov-resource` | File ingestion: Resource, Chunk (AST-aware), WatchTask, WatchEvent |
| [ov-crypto.md](./ov-crypto.md) | `ov-crypto` | Encryption: AccountKey, KeyRotationLog, OVE1 Envelope, KMSProvider |

### 🔄 Pipeline

| File | Service(s) | Description |
|------|-----------|-------------|
| [pipeline-service.md](./pipeline-service.md) | `pipeline-service`, `ba-knowledge-*`, `vnp-pipelines` | Pipeline orchestration: Pipeline, Job, Queue, Worker, PipelineTemplate |

### 📊 Observability

| File | Service(s) | Description |
|------|-----------|-------------|
| [obs-service.md](./obs-service.md) | `obs-service`, `vnp-observability`, `vnp-infra` | Metrics, Traces, Spans, ErrorEntry, CostEntry, ServiceInfo, TopologyGraph |
| [observe-service.md](./observe-service.md) | `observe-service` | Agent observation capture: Session, RawObservation, CompressedObservation |

### 🤖 Engine-Specific Admin

| File | Service(s) | Description |
|------|-----------|-------------|
| [ov-admin.md](./ov-admin.md) | `ov-admin` | OpenViking admin: Account, User, Agent, APIKey, NamespaceURI |
| [memobase.md](./memobase.md) | `memobase-*` | Memobase engine: Project, ProjectProfileConfig, Blob, Profile, UserEvent, ContextResult |
| [zep.md](./zep.md) | `zep-*` | Zep engine: User, Thread, Session, Message, ContextAssembly, Fact |

---

## Architecture Notes

### Merge Map (Consolidated Services)

The platform uses a **consolidation pattern** where multiple smaller services are absorbed into larger umbrella services:

```
vnp-platform ← sm-auth, sm-project, sm-analytics, vnp-event*, admin services
memory-service ← sm-memory, sm-document, sm-profile, memobase-*, zep-*
kg-service ← cognee-*, graphiti-*
storage-service ← ov-fs, ov-crypto, ov-resource, ov-session
pipeline-service ← vnp-pipelines, ba-knowledge-service, ba-knowledge-worker
obs-service ← vnp-observability, vnp-infra
```

### Common ID Patterns

| Pattern | Description |
|---------|-------------|
| `uuid.UUID` | Standard identifiers for most entities |
| `string` (OpenViking) | ov-* services use string IDs for flexibility |
| `TenantID` | Present on all multi-tenant entities |
| `viking://{account}/{user}/{agent}/` | NamespaceURI for OpenViking resources |

### Common Enums

| Enum | Values |
|------|--------|
| Engine aliases | `cognee`, `graphiti`, `memobase`, `openviking`, `zep`, `supermemory` |
| Memory types | `pattern`, `preference`, `architecture`, `bug`, `workflow`, `fact` |
| Job status | `pending`, `running`, `completed`, `failed` |
| Session status | `active`, `committed/completed`, `archived/abandoned` |
| User roles | `owner/admin/editor/viewer` (varies by service) |
| Subscription tiers | `free`, `starter/pro`, `enterprise` |
