# VNP Memory — Admin Console UX/UI Specification

## 1. Product Positioning

### Product Name
**VNP Memory Console**

### Tagline
> Enterprise Cognitive Infrastructure Control Plane

### Primary Goal
Cung cấp giao diện quản trị tập trung cho toàn bộ hệ sinh thái VNP Memory:
- Quản trị tenant/app/API keys
- Quan sát memory flow realtime
- Điều phối 6 memory engines (Cognee, Graphiti, Zep, OpenViking, Memobase, Supermemory)
- Governance / ontology / policies
- User profile management & personalization
- Adaptive memory & external connectors
- Context debugging cho AI agents
- Monitoring & observability
- Audit & compliance

---

# 2. Design Principles

| Principle | Description |
|---|---|
| Cognitive-first UX | Thiết kế giống "Operating System for AI Cognition" thay vì CRUD dashboard thông thường |
| Graph-native | Mọi entity, memory, workflow đều có thể trace thành graph |
| Explainable Memory | AI nhớ gì, tại sao nhớ, nguồn từ đâu phải nhìn được |
| Multi-tenant by default | Tenant isolation hiển thị rõ trong toàn bộ UI |
| Fast observability | Realtime state, timeline, memory flows |
| Layer abstraction | User không cần biết backend engine nào xử lý |
| Developer-centric | UX giống Datadog + Neo4j Bloom + OpenAI Console + Temporal UI |
| Profile-aware | User profiles và adaptive memory là first-class citizens trong UI |

---

# 3. User Roles

| Role | Permissions |
|---|---|
| Super Admin | Full platform access |
| Organization Admin | Manage org tenants, quotas, policies |
| AI Engineer | Manage agents, memory pipelines, ontologies |
| ML Engineer | Search tuning, evaluation, embeddings |
| Security Officer | Audit logs, GDPR, policy review |
| Developer | API usage, sessions, debugging |
| Product Manager | User profiles analytics, usage metrics |
| Readonly Analyst | Monitoring only |

---

# 4. Information Architecture

```text
VNP Memory Console
│
├── Overview
│   ├── Global Health
│   ├── Active Agents
│   ├── Memory Throughput
│   ├── Token Savings
│   └── Alerts
│
├── Memory Explorer
│   ├── Episodic Memory (Graphiti)
│   ├── Semantic Memory (Cognee)
│   ├── Conversational Memory (Zep)
│   ├── Procedural Memory (OpenViking)
│   ├── Profile Memory (Memobase)
│   ├── Adaptive Memory (Supermemory)
│   └── Unified Search
│
├── Graph Studio
│   ├── Knowledge Graph
│   ├── Timeline View
│   ├── Entity Explorer
│   ├── Ontology Designer
│   └── Relationship Inspector
│
├── User Profiles
│   ├── Profile Explorer
│   ├── Profile Config (Schema)
│   ├── Buffer Zone Monitor
│   ├── Event Timeline
│   └── Context Assembly Preview
│
├── Adaptive Memory
│   ├── Memory Versions
│   ├── Auto-Forget Rules
│   ├── External Connectors
│   ├── Contradiction Resolution
│   └── Memory Graph (Supermemory)
│
├── Agent Context Debugger
│   ├── Context Compiler
│   ├── Recall Trace
│   ├── Prompt Assembly
│   ├── Salience Ranking
│   └── Token Analyzer
│
├── Sessions
│   ├── Users
│   ├── Conversations
│   ├── Live Sessions
│   └── Replay
│
├── Governance
│   ├── Policies (OPA)
│   ├── Tenants
│   ├── Namespace Isolation
│   ├── Retention Rules
│   ├── GDPR Forget
│   └── Audit Logs
│
├── Pipelines
│   ├── Ingestion Jobs
│   ├── Extraction Pipelines
│   ├── Embedding Queue
│   ├── Profile Extraction Pipeline
│   ├── Rules Engine
│   └── Background Workers
│
├── Infrastructure
│   ├── Services (6 engines + KGS)
│   ├── Queues
│   ├── Databases
│   ├── Vector Indexes
│   └── Cache Layers
│
├── Observability
│   ├── Metrics
│   ├── Traces
│   ├── Logs
│   ├── Errors
│   └── Cost Analytics
│
├── API & SDK
│   ├── API Keys
│   ├── Rate Limits
│   ├── SDK Usage
│   ├── MCP Connections
│   └── Webhooks
│
└── Settings
    ├── Organization
    ├── Members
    ├── Billing
    ├── Integrations
    └── Preferences
```

---

# 5. Layout System

## Global Layout

### Left Sidebar
Persistent navigation.

Sections:
- Overview
- Memory Explorer
- Graph Studio
- User Profiles
- Adaptive Memory
- Sessions
- Governance
- Pipelines
- Infrastructure
- Observability
- API & SDK
- Settings

### Top Navigation
Contains:
- Tenant selector
- Environment switcher (Dev/Staging/Prod)
- Global search
- Notifications
- AI Assistant shortcut
- User profile

### Main Workspace
Adaptive content area.

### Right Context Panel
Dynamic contextual inspector:
- Entity details
- Trace metadata
- Timeline
- Related memories
- User profile summary
- Policies applied

---

# 6. Core Screens

# 6.1 Dashboard / Overview

## Purpose
Platform-wide operational awareness across all 6 engines.

## Layout

### Top KPI Cards
| Widget | Description |
|---|---|
| Active Agents | Number of connected agents |
| Recall Latency | p50/p95 memory recall |
| Context Savings | Token reduction percentage |
| Graph Growth | Nodes/edges growth |
| Error Rate | Failed retrievals |
| Active Sessions | Live conversations |
| Active Profiles | Memobase user profiles |
| Memory Versions | Supermemory active memories |

### Memory Flow Visualization
Animated pipeline:

```text
Agent → Gateway → Engine Router → [Cognee|Graphiti|Zep|OpenViking|Memobase|Supermemory] → KGS → Storage
```

Realtime indicators:
- ingest/sec
- embeddings/sec
- recall/sec
- profile extractions/sec
- queue backlog

### Realtime Engine Health Grid
| Engine | Role | Status | Latency | Queue |
|---|---|---|---|---|
| Graphiti | Episodic Memory | Healthy | 412ms | 12 |
| Cognee | Semantic Memory | Healthy | 823ms | 41 |
| Zep | Conversational Memory | Healthy | 82ms | 3 |
| OpenViking | Procedural Memory | Warning | 1.2s | 84 |
| Memobase | Profile Memory | Healthy | 45ms | 7 |
| Supermemory | Adaptive Memory | Healthy | 38ms | 15 |
| KGS | Governance | Healthy | 92ms | 0 |

### AI Memory Heatmap
Visualization:
- memory density per engine
- retrieval frequency
- stale memories
- forgotten memories (Supermemory auto-forget)
- profile freshness (Memobase)

---

# 6.2 Memory Explorer

## Purpose
Unified interface để search và inspect mọi loại memory across 6 engines.

## UX Style
Mix giữa:
- ElasticSearch Kibana
- OpenSearch Dashboards
- Graph explorer

## Layout

### Search Header
Inputs:
- semantic search
- hybrid search
- graph query
- timeline query
- profile search (Memobase)

Advanced filters:
- tenant
- user
- memory type (episodic/semantic/conversational/procedural/profile/adaptive)
- source engine
- time range
- confidence score
- ontology class
- memory version (Supermemory)

### Result Tabs
- All
- Episodic (Graphiti)
- Semantic (Cognee)
- Conversational (Zep)
- Procedural (OpenViking)
- Profile (Memobase)
- Adaptive (Supermemory)

### Result Card Structure
```text
[Memory Type Badge] [Engine Badge]
Title
Summary
Entities detected
Related sessions
Temporal validity
Source engine
Policy tags
Version chain (Supermemory: parent → root)
```

### Side Inspector
Shows:
- provenance
- vector similarity
- graph neighbors
- timeline
- raw payload
- embeddings metadata
- version history (for adaptive memories)
- profile associations

---

# 6.3 Graph Studio

## Purpose
Visual knowledge graph exploration.

## Main Features

### Interactive Graph Canvas
Capabilities:
- zoom/pan
- clustering
- edge grouping
- temporal playback
- drag-and-connect
- expand neighbors

### Entity Inspector
Shows:
- entity type
- ontology schema
- related facts
- confidence
- source memories
- tenant namespace

### Timeline Slider
Temporal replay.

Example:
```text
2026-01-01 ────────────●──────────── 2026-05-01
```

Replay changes over time:
- entity evolution
- invalidated facts
- relationship changes
- memory version transitions (Supermemory)

### Ontology Designer
Visual schema builder.

Features:
- define node types
- define edge rules
- validation constraints
- inheritance
- policy attachment

### Query Playground
Modes:
- Cypher visual builder
- Natural language → Cypher
- Saved queries

---

# 6.4 User Profiles (NEW — Memobase)

## Purpose
Manage và inspect structured user profiles extracted from conversations.

## Core Value
Transforms raw conversations into structured, actionable user knowledge.

## Sections

### Profile Explorer
- Browse all user profiles by tenant
- Search by topic/sub_topic
- Filter by profile schema

### Profile Detail View
```text
User: user_123
├── Preferences
│   ├── coding_style: "functional, minimal comments"
│   ├── language: "TypeScript"
│   └── theme: "dark mode"
├── Projects
│   ├── vnp-memory: "blockchain infrastructure"
│   └── openledger: "AI platform"
└── Goals
    ├── short_term: "ship v1.0"
    └── long_term: "enterprise adoption"
```

### Profile Config Editor
- Define profile schema (topic/sub_topic/content)
- Strict mode toggle: chỉ collect theo schema
- Buffer zone settings:
  - Token threshold (default: 1024)
  - Idle timeout (default: 1h)

### Buffer Zone Monitor
Realtime view:
- Active buffers per user
- Token accumulation progress
- Flush history
- LLM call count (fixed 3 per flush: extract → merge → events)

### Event Timeline
- Chronological event list per user
- Event gist search
- Tag-based filtering
- Event embedding visualization

### Context Assembly Preview
- Preview prompt-ready context string
- Token budget configuration
- Profile section + Event section assembly
- Latency metric (target: < 100ms)

---

# 6.5 Adaptive Memory (NEW — Supermemory)

## Purpose
Manage living knowledge graph with auto-forgetting and external data connectors.

## Core Value
Memory that evolves — automatically forgets outdated info, resolves contradictions.

## Sections

### Memory Version Explorer
- Version chain visualization: parent → root
- `isLatest` flag indicators
- Relation types: updates, extends, derives
- Diff view between versions

### Auto-Forget Rules
- Configure `forgetAfter` duration per memory type
- Static vs Dynamic memory classification
- Noise filtering rules
- Contradiction resolution history

### External Connectors
| Connector | Status | Last Sync | Docs |
|---|---|---|---|
| Google Drive | Connected | 2h ago | 1,234 |
| Gmail | Connected | 4h ago | 892 |
| Notion | Disconnected | — | — |
| OneDrive | Connected | 4h ago | 456 |
| GitHub | Connected | 30m ago | 2,891 |

Settings per connector:
- OAuth key management
- Sync frequency (cron + webhooks)
- Document limit
- Sync history & error logs

### Memory Graph Visualization
- Interactive graph of Supermemory's adaptive KG
- Node coloring by memory type (static/dynamic)
- Edge coloring by relation type (updates/extends/derives)
- Time-decay opacity for aging memories

### Analytics
- Memory creation/deletion rate
- Contradiction resolution frequency
- Connector sync metrics
- Storage usage per project

---

# 6.6 Agent Context Debugger

## Purpose
Debug exactly how context was assembled for an AI request.

## Core Value
This is the signature differentiator.

## Flow

```text
Prompt Input
    ↓
Memory Recall (6 engines)
    ↓
Profile Injection (Memobase)
    ↓
Hybrid Ranking
    ↓
Policy Filtering
    ↓
Compression
    ↓
Final Context Package
```

## UI Sections

### Left Panel — Agent Request
Shows:
- user prompt
- metadata
- session
- tenant
- selected model
- user profile summary

### Center Panel — Context Pipeline
Step-by-step assembly.

Example:

```text
1. Query Rewrite
2. Semantic Recall (Cognee)
3. Graph Traversal (Graphiti)
4. Profile Lookup (Memobase)
5. Adaptive Memory Check (Supermemory)
6. Tiered Context Load (OpenViking)
7. Salience Ranking
8. Policy Filter
9. Compression
10. Final Context
```

### Right Panel — Token Analysis
Charts:
- token allocation per engine
- memory categories breakdown
- compression savings
- duplicated context
- profile token usage

### Bottom Panel — Final Prompt
Display:
- full prompt
- injected memories (with engine source badges)
- injected profile sections
- citations
- retrieval rationale

---

# 6.7 Sessions & Conversations

## Features

### Session Timeline
Replay full AI interaction.

### Live Conversation Viewer
Realtime messages streaming.

### User Memory Summary
Generated profile (Memobase-powered):
- preferences
- goals
- recurring entities
- long-term memory clusters
- profile freshness indicator

### Session Diff
Compare memory before/after conversation.

### Working Memory Inspector
- Structured document view (title, state, goals, facts, errors)
- 2-phase commit status: archive → extract
- Long-term memory extraction progress

---

# 6.8 Governance Center

## Purpose
Enterprise-grade compliance & governance.

## Sections

### Tenant Management
Features:
- quota limits (max_nodes, max_requests)
- namespaces
- API usage per engine
- billing usage
- active policies

### OPA Policy Editor
Visual + code mode.

```rego
allow {
  input.user.role == "admin"
}
```

### GDPR Forget Center
Actions:
- erase user (cascading across all 6 engines)
- cascade deletion preview
- dry-run mode
- deletion audit
- Supermemory version chain cleanup

### Retention Policies
Configure:
- TTL by memory type
- archive schedules
- deletion workflows
- Memobase profile expiration
- Supermemory forgetAfter rules

### Audit Explorer
Searchable audit logs.

Fields:
- actor
- action
- entity
- tenant
- engine
- timestamp
- policy result

---

# 6.9 Pipelines Console

## Purpose
Observe ingestion & processing pipelines across all engines.

## Features

### DAG View
Like Airflow/Temporal.

Stages per engine:
```text
Cognee:      Ingest → Parse → Chunk → Embed → Extract → Validate → Store
Graphiti:    Episode → Extract → Deduplicate → Graph → Community
Zep:         Message → Graph Ingestion → Context Assembly
OpenViking:  Data → Parse → Chunk → Embed → L0/L1/L2 Context
Memobase:    Blob → Buffer → Extract → Merge → Profile → Cache
Supermemory: Document → Memory → Version → Search Index
```

### Queue Monitoring
Metrics:
- queue depth per engine
- retries
- failures
- throughput

### Worker Status
Realtime worker state per engine.

### Pipeline Templates
Templates:
- PDF ingestion (Cognee)
- Git repo ingestion (OpenViking/Supermemory)
- Web crawler (Cognee)
- Slack sync (Supermemory connector)
- CRM sync (Supermemory connector)
- Chat profile extraction (Memobase)

---

# 6.10 Infrastructure View

## Purpose
DevOps + SRE operational control.

## Components

### Service Map
Topology graph showing all 6 engine apps + KGS + shared infra.

```text
┌─────────────────────────────────────────────┐
│              Memory API Gateway             │
├──────┬──────┬──────┬──────┬──────┬──────────┤
│Cognee│Graph-│ Zep  │Open- │Memo- │Super-    │
│:8080 │iti   │:8080 │Viking│base  │memory    │
│      │:8080 │      │:8080 │:8080 │:8080     │
├──────┴──────┴──────┴──────┴──────┴──────────┤
│        Shared Infrastructure                 │
│  PostgreSQL │ Neo4j │ Qdrant │ Redis │ NATS  │
└──────────────────────────────────────────────┘
```

### Database Health
Metrics:
- Neo4j cluster state
- PostgreSQL replication (+ pgvector status)
- Qdrant index status
- Redis memory usage
- NATS JetStream state

### Resource Monitoring
Charts:
- CPU / RAM / GPU / disk / network per engine app

### Deployment Timeline
Track:
- deployments
- rollbacks
- incidents

---

# 6.11 Observability

## Metrics Dashboard

### Categories
- API latency (per engine)
- retrieval latency
- graph query latency
- embedding throughput
- token usage
- LLM costs (including Memobase fixed 3-call budget)
- profile extraction throughput

### Trace Viewer
Distributed tracing:

```text
Gateway
 → Cognee (Ingestion → Cognify → Search)
 → Graphiti (Store → Knowledge → Search)
 → Zep (User → Thread → Memory → Graph)
 → OpenViking (FS → Resource → Search)
 → Memobase (Ingestion → Engine → Context)
 → Supermemory (Document → Memory → Search)
 → KGS Planner
 → Qdrant / Neo4j / PostgreSQL
```

### Error Explorer
Features:
- stack traces
- failed memory retrievals (per engine)
- policy violations
- timeout analysis
- LLM call failures

---

# 7. UX Patterns

## 7.1 Unified Memory Badge System

| Type | Color | Icon | Engine |
|---|---|---|---|
| Episodic | Purple | Clock | Graphiti |
| Semantic | Blue | Network | Cognee |
| Conversational | Green | Message | Zep |
| Procedural | Orange | Folder | OpenViking |
| Profile | Teal | User | Memobase |
| Adaptive | Amber | Sparkle | Supermemory |
| Governance | Red | Shield | KGS |

---

# 7.2 Timeline Everywhere

Temporal interaction là core UX primitive.

Every entity should support:
- current state
- historical replay
- future invalidation
- version chain (Supermemory)

---

# 7.3 Explainability UX

Every memory retrieval must answer:

```text
Why was this memory retrieved?
Which engine provided it?
```

UI includes:
- similarity score
- graph path
- policy result
- source engine badge
- retrieval step
- version info (if adaptive)

---

# 7.4 Profile-Aware Context (NEW)

Every context assembly should show:
- which profile sections were injected
- token budget allocation (profile vs memories)
- profile freshness indicator
- buffer zone status

---

# 8. Design System

## Typography
| Usage | Font |
|---|---|
| UI | Inter |
| Code | JetBrains Mono |
| Graph Labels | IBM Plex Sans |

## Visual Style
| Element | Style |
|---|---|
| Background | Deep dark graphite |
| Panels | Frosted glass / elevated cards |
| Graphs | Neon edge highlights |
| Status | Minimalistic system indicators |
| Charts | Dense observability-first |
| Engine badges | Color-coded with subtle glow |

## Spacing
8pt grid system.

## Border Radius
12px cards.

---

# 9. Frontend Architecture

## Recommended Stack

| Layer | Technology |
|---|---|
| Framework | Vite + React 19 |
| Language | TypeScript |
| UI | shadcn/ui |
| State | Zustand + React Query |
| Charts | Recharts + Apache ECharts |
| Graph Visualization | React Flow + Cytoscape.js |
| Realtime | WebSocket + SSE |
| Auth | JWT + API Key |
| Styling | TailwindCSS |
| Router | React Router v7 |

---

# 10. Suggested Route Structure

```text
/app
  /overview
  /memory
  /memory/[id]
  /graph
  /graph/timeline
  /graph/ontology
  /profiles
  /profiles/[user_id]
  /profiles/config
  /profiles/buffers
  /profiles/events
  /adaptive
  /adaptive/versions
  /adaptive/connectors
  /adaptive/forget-rules
  /sessions
  /sessions/[id]
  /governance
  /governance/tenants
  /governance/policies
  /governance/gdpr
  /governance/audit
  /pipelines
  /pipelines/[engine]
  /infra
  /infra/services
  /observability
  /observability/traces
  /observability/errors
  /api-keys
  /settings
```

---

# 11. API Requirements for UI

## Dashboard APIs

```http
GET /v1/admin/metrics
GET /v1/admin/health
GET /v1/admin/throughput
```

## Memory Explorer APIs

```http
GET /v1/memory/search
GET /v1/memory/{id}
GET /v1/memory/{id}/neighbors
```

## Graph APIs

```http
GET /v1/graph/subgraph
GET /v1/graph/timeline
POST /v1/graph/query
```

## Profile APIs (Memobase)

```http
POST /api/v1/users
GET  /api/v1/users/{user_id}
POST /api/v1/blobs/insert/{user_id}
GET  /api/v1/users/profile/{user_id}
POST /api/v1/users/profile/{user_id}
GET  /api/v1/users/context/{user_id}
GET  /api/v1/users/event/{user_id}
GET  /api/v1/users/event/search/{user_id}
POST /api/v1/users/buffer/{user_id}/{buffer_type}
GET  /api/v1/users/buffer/capacity/{user_id}/{buffer_type}
GET  /api/v1/project/profile_config
POST /api/v1/project/profile_config
GET  /api/v1/project/billing
GET  /api/v1/project/usage
```

## Adaptive Memory APIs (Supermemory)

```http
POST /api/v1/documents
GET  /api/v1/memories
GET  /api/v1/memories/{id}/versions
GET  /api/v1/search
GET  /api/v1/profiles
GET  /api/v1/connectors
POST /api/v1/connectors
GET  /api/v1/analytics
GET  /api/v1/projects
```

## Cognee APIs

```http
POST /api/v1/cognee/add
GET  /api/v1/cognee/datasets
POST /api/v1/cognee/cognify
GET  /api/v1/cognee/cognify/{id}/status
POST /api/v1/cognee/search
GET  /api/v1/cognee/search/explore
POST /api/v1/cognee/search/rag
```

## Zep APIs

```http
POST /api/v1/users
GET  /api/v1/threads
POST /api/v1/memories
GET  /api/v1/graph
GET  /api/v1/search
```

## Governance APIs

```http
GET  /v1/admin/tenants
POST /v1/admin/policies
GET  /v1/admin/audit
```

---

# 12. MVP Scope

## Phase 1 — Admin MVP

### Include
- Dashboard overview (6 engines health)
- Unified memory explorer (6 memory types)
- Session explorer
- Tenant management
- API keys
- Basic observability
- Graph visualization
- Audit logs
- User profile explorer (Memobase)
- Basic adaptive memory view (Supermemory)

### Exclude
- Advanced ontology designer
- Context compiler visualization
- AI-assisted query builder
- Memory optimization automation
- External connector management (Supermemory)
- Advanced profile analytics

---

# 13. Future UX Opportunities

| Feature | Description |
|---|---|
| AI Copilot for Admin | Ask system questions in natural language |
| Auto-detect memory anomalies | AI-generated incident detection |
| Memory quality scoring | Evaluate memory usefulness |
| Replay agent cognition | Step-by-step reasoning replay |
| Collaborative graph editing | Multi-user graph studio |
| Voice observability | Speak queries to graph/search |
| Profile analytics dashboard | Cross-user pattern discovery |
| Connector marketplace | Community-built external connectors |
| Memory cost optimizer | Automated token/LLM cost reduction |

---

# 14. Competitive Benchmarking

| Product | Inspiration |
|---|---|
| Datadog | Observability UX |
| Temporal UI | Workflow visibility |
| Neo4j Bloom | Graph exploration |
| Kibana | Search & logs |
| OpenAI Platform | API console simplicity |
| Retool | Internal tool ergonomics |
| Vercel Dashboard | Developer experience |
| Segment CDP | User profile management |
| Zapier | External connector UX |

---

# 15. Recommended Initial UI Modules

## Highest Priority

### Module 1 — Global Dashboard
Business + infrastructure overview across all 6 engines.

### Module 2 — Memory Explorer
Most-used feature. Unified search across all memory types.

### Module 3 — Context Debugger
Core differentiation. Shows how context is assembled from 6 engines.

### Module 4 — Governance Center
Enterprise requirement. Multi-tenant, GDPR, audit.

### Module 5 — Graph Studio
Strategic visual layer.

### Module 6 — User Profiles (NEW)
Memobase-powered profile management and context assembly.

### Module 7 — Adaptive Memory (NEW)
Supermemory-powered memory versioning and auto-forget visualization.

---

# 16. Suggested UI Development Roadmap

## Sprint 1
- Design system
- Authentication
- App shell
- Navigation

## Sprint 2
- Dashboard
- Health monitoring (6 engines)
- Metrics

## Sprint 3
- Memory explorer
- Unified search (6 memory types)
- Memory inspector

## Sprint 4
- Graph studio
- Timeline replay
- Entity explorer

## Sprint 5
- User profiles (Memobase)
- Profile config editor
- Buffer zone monitor
- Context assembly preview

## Sprint 6
- Adaptive memory (Supermemory)
- Memory versions
- External connectors
- Auto-forget rules

## Sprint 7
- Governance center
- Audit logs
- Policies

## Sprint 8
- Context debugger
- Prompt assembly visualization
- Token analytics (6 engines breakdown)

---

# 17. Engine-to-Screen Mapping

| Engine App | Memory Type | Primary Screens | Port |
|---|---|---|---|
| `apps/cognee` | Semantic | Memory Explorer, Pipelines, Graph Studio | 8080 |
| `apps/graphiti` | Episodic | Memory Explorer, Graph Studio, Timeline | 8080 |
| `apps/zep` | Conversational | Sessions, Memory Explorer | 8080 |
| `apps/OpenViking` | Procedural | Memory Explorer, Pipelines | 8080 |
| `apps/memobase` | Profile | User Profiles, Context Preview | 8080 |
| `apps/supermemory` | Adaptive | Adaptive Memory, Connectors | 8080 |

Each app exposes a unified gateway via embedded monolith (Supervisor pattern), with internal gRPC services on localhost and MCP on port 8082 (where applicable).

---

# 18. Final Positioning

VNP Memory Console không chỉ là dashboard quản trị.

Nó là:

> "Cognitive Control Plane for Enterprise AI Systems"

Một hệ điều hành quan sát, kiểm soát, và tối ưu hóa trí nhớ của AI agents ở quy mô enterprise — bao gồm 6 loại memory chuyên biệt: Episodic, Semantic, Conversational, Procedural, Profile, và Adaptive.
