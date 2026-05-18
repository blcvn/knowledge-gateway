# VNP Memory — Admin Console UX/UI Specification

## 1. Product Positioning

### Product Name

**VNP Memory Console**

### Tagline

> Enterprise Cognitive Infrastructure Control Plane

### Primary Goal

Cung cấp giao diện quản trị tập trung cho toàn bộ hệ sinh thái VNP Memory:

* Quản trị tenant/app/API keys
* Quan sát memory flow realtime
* Điều phối memory engines
* Governance / ontology / policies
* Context debugging cho AI agents
* Monitoring & observability
* Audit & compliance

---

# 2. Design Principles

| Principle               | Description                                                                            |
| ----------------------- | -------------------------------------------------------------------------------------- |
| Cognitive-first UX      | Thiết kế giống “Operating System for AI Cognition” thay vì CRUD dashboard thông thường |
| Graph-native            | Mọi entity, memory, workflow đều có thể trace thành graph                              |
| Explainable Memory      | AI nhớ gì, tại sao nhớ, nguồn từ đâu phải nhìn được                                    |
| Multi-tenant by default | Tenant isolation hiển thị rõ trong toàn bộ UI                                          |
| Fast observability      | Realtime state, timeline, memory flows                                                 |
| Layer abstraction       | User không cần biết backend engine nào xử lý                                           |
| Developer-centric       | UX giống Datadog + Neo4j Bloom + OpenAI Console + Temporal UI                          |

---

# 3. User Roles

| Role               | Permissions                                 |
| ------------------ | ------------------------------------------- |
| Super Admin        | Full platform access                        |
| Organization Admin | Manage org tenants, quotas, policies        |
| AI Engineer        | Manage agents, memory pipelines, ontologies |
| ML Engineer        | Search tuning, evaluation, embeddings       |
| Security Officer   | Audit logs, GDPR, policy review             |
| Developer          | API usage, sessions, debugging              |
| Readonly Analyst   | Monitoring only                             |

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
│   ├── Episodic Memory
│   ├── Semantic Memory
│   ├── Conversational Memory
│   ├── Procedural Memory
│   └── Unified Search
│
├── Graph Studio
│   ├── Knowledge Graph
│   ├── Timeline View
│   ├── Entity Explorer
│   ├── Ontology Designer
│   └── Relationship Inspector
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
│   ├── Rules Engine
│   └── Background Workers
│
├── Infrastructure
│   ├── Services
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

* Overview
* Memory Explorer
* Graph Studio
* Sessions
* Governance
* Pipelines
* Infrastructure
* Observability
* API & SDK
* Settings

### Top Navigation

Contains:

* Tenant selector
* Environment switcher (Dev/Staging/Prod)
* Global search
* Notifications
* AI Assistant shortcut
* User profile

### Main Workspace

Adaptive content area.

### Right Context Panel

Dynamic contextual inspector:

* Entity details
* Trace metadata
* Timeline
* Related memories
* Policies applied

---

# 6. Core Screens

# 6.1 Dashboard / Overview

## Purpose

Platform-wide operational awareness.

## Layout

### Top KPI Cards

| Widget          | Description                |
| --------------- | -------------------------- |
| Active Agents   | Number of connected agents |
| Recall Latency  | p50/p95 memory recall      |
| Context Savings | Token reduction percentage |
| Graph Growth    | Nodes/edges growth         |
| Error Rate      | Failed retrievals          |
| Active Sessions | Live conversations         |

### Memory Flow Visualization

Animated pipeline:

```text
Agent → Gateway → Engine → KGS → Storage
```

Realtime indicators:

* ingest/sec
* embeddings/sec
* recall/sec
* queue backlog

### Realtime Engine Health Grid

| Engine     | Status  | Latency | Queue |
| ---------- | ------- | ------- | ----- |
| Graphiti   | Healthy | 412ms   | 12    |
| Cognee     | Healthy | 823ms   | 41    |
| Zep        | Healthy | 82ms    | 3     |
| OpenViking | Warning | 1.2s    | 84    |
| KGS        | Healthy | 92ms    | 0     |

### AI Memory Heatmap

Visualization:

* memory density
* retrieval frequency
* stale memories
* forgotten memories

---

# 6.2 Memory Explorer

## Purpose

Unified interface để search và inspect mọi loại memory.

## UX Style

Mix giữa:

* ElasticSearch Kibana
* OpenSearch Dashboards
* Graph explorer

## Layout

### Search Header

Inputs:

* semantic search
* hybrid search
* graph query
* timeline query

Advanced filters:

* tenant
* user
* memory type
* source engine
* time range
* confidence score
* ontology class

### Result Tabs

* All
* Episodic
* Semantic
* Conversational
* Procedural

### Result Card Structure

```text
[Memory Type Badge]
Title
Summary
Entities detected
Related sessions
Temporal validity
Source engine
Policy tags
```

### Side Inspector

Shows:

* provenance
* vector similarity
* graph neighbors
* timeline
* raw payload
* embeddings metadata

---

# 6.3 Graph Studio

## Purpose

Visual knowledge graph exploration.

## Main Features

### Interactive Graph Canvas

Capabilities:

* zoom/pan
* clustering
* edge grouping
* temporal playback
* drag-and-connect
* expand neighbors

### Entity Inspector

Shows:

* entity type
* ontology schema
* related facts
* confidence
* source memories
* tenant namespace

### Timeline Slider

Temporal replay.

Example:

```text
2026-01-01 ────────────●──────────── 2026-05-01
```

Replay changes over time:

* entity evolution
* invalidated facts
* relationship changes

### Ontology Designer

Visual schema builder.

Features:

* define node types
* define edge rules
* validation constraints
* inheritance
* policy attachment

### Query Playground

Modes:

* Cypher visual builder
* Natural language → Cypher
* Saved queries

---

# 6.4 Agent Context Debugger

## Purpose

Debug exactly how context was assembled for an AI request.

## Core Value

This is the signature differentiator.

## Flow

```text
Prompt Input
    ↓
Memory Recall
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

* user prompt
* metadata
* session
* tenant
* selected model

### Center Panel — Context Pipeline

Step-by-step assembly.

Example:

```text
1. Query Rewrite
2. Semantic Recall
3. Graph Traversal
4. Salience Ranking
5. Policy Filter
6. Compression
7. Final Context
```

### Right Panel — Token Analysis

Charts:

* token allocation
* memory categories
* compression savings
* duplicated context

### Bottom Panel — Final Prompt

Display:

* full prompt
* injected memories
* citations
* retrieval rationale

---

# 6.5 Sessions & Conversations

## Features

### Session Timeline

Replay full AI interaction.

### Live Conversation Viewer

Realtime messages streaming.

### User Memory Summary

Generated profile:

* preferences
* goals
* recurring entities
* long-term memory clusters

### Session Diff

Compare memory before/after conversation.

---

# 6.6 Governance Center

## Purpose

Enterprise-grade compliance & governance.

## Sections

### Tenant Management

Features:

* quota limits
* namespaces
* API usage
* billing usage
* active policies

### OPA Policy Editor

Visual + code mode.

```rego
allow {
  input.user.role == "admin"
}
```

### GDPR Forget Center

Actions:

* erase user
* cascade deletion preview
* dry-run mode
* deletion audit

### Retention Policies

Configure:

* TTL by memory type
* archive schedules
* deletion workflows

### Audit Explorer

Searchable audit logs.

Fields:

* actor
* action
* entity
* tenant
* timestamp
* policy result

---

# 6.7 Pipelines Console

## Purpose

Observe ingestion & processing pipelines.

## Features

### DAG View

Like Airflow/Temporal.

Stages:

```text
Ingest → Parse → Chunk → Embed → Extract → Validate → Store
```

### Queue Monitoring

Metrics:

* queue depth
* retries
* failures
* throughput

### Worker Status

Realtime worker state.

### Pipeline Templates

Templates:

* PDF ingestion
* Git repo ingestion
* Web crawler
* Slack sync
* CRM sync

---

# 6.8 Infrastructure View

## Purpose

DevOps + SRE operational control.

## Components

### Service Map

Topology graph.

### Database Health

Metrics:

* Neo4j cluster state
* PostgreSQL replication
* Qdrant index status
* Redis memory usage

### Resource Monitoring

Charts:

* CPU
* RAM
* GPU
* disk
* network

### Deployment Timeline

Track:

* deployments
* rollbacks
* incidents

---

# 6.9 Observability

## Metrics Dashboard

### Categories

* API latency
* retrieval latency
* graph query latency
* embedding throughput
* token usage
* LLM costs

### Trace Viewer

Distributed tracing:

```text
Gateway
 → Graphiti
 → KGS Planner
 → Qdrant
 → OpenAI
```

### Error Explorer

Features:

* stack traces
* failed memory retrievals
* policy violations
* timeout analysis

---

# 7. UX Patterns

## 7.1 Unified Memory Badge System

| Type           | Color  | Icon    |
| -------------- | ------ | ------- |
| Episodic       | Purple | Clock   |
| Semantic       | Blue   | Network |
| Conversational | Green  | Message |
| Procedural     | Orange | Folder  |
| Governance     | Red    | Shield  |

---

# 7.2 Timeline Everywhere

Temporal interaction là core UX primitive.

Every entity should support:

* current state
* historical replay
* future invalidation

---

# 7.3 Explainability UX

Every memory retrieval must answer:

```text
Why was this memory retrieved?
```

UI includes:

* similarity score
* graph path
* policy result
* source engine
* retrieval step

---

# 8. Design System

## Typography

| Usage        | Font           |
| ------------ | -------------- |
| UI           | Inter          |
| Code         | JetBrains Mono |
| Graph Labels | IBM Plex Sans  |

## Visual Style

| Element    | Style                          |
| ---------- | ------------------------------ |
| Background | Deep dark graphite             |
| Panels     | Frosted glass / elevated cards |
| Graphs     | Neon edge highlights           |
| Status     | Minimalistic system indicators |
| Charts     | Dense observability-first      |

## Spacing

8pt grid system.

## Border Radius

12px cards.

---

# 9. Frontend Architecture

## Recommended Stack

| Layer               | Technology                |
| ------------------- | ------------------------- |
| Framework           | Next.js 15                |
| Language            | TypeScript                |
| UI                  | shadcn/ui                 |
| State               | Zustand + React Query     |
| Charts              | Recharts + Apache ECharts |
| Graph Visualization | React Flow + Cytoscape.js |
| Realtime            | WebSocket + SSE           |
| Auth                | Clerk/Auth.js             |
| Styling             | TailwindCSS               |
| Monorepo            | Turborepo                 |

---

# 10. Suggested Route Structure

```text
/app
  /overview
  /memory
  /memory/[id]
  /graph
  /graph/timeline
  /sessions
  /governance
  /pipelines
  /infra
  /observability
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

## Governance APIs

```http
GET /v1/admin/tenants
POST /v1/admin/policies
GET /v1/admin/audit
```

---

# 12. MVP Scope

## Phase 1 — Admin MVP

### Include

* Dashboard overview
* Unified memory explorer
* Session explorer
* Tenant management
* API keys
* Basic observability
* Graph visualization
* Audit logs

### Exclude

* Advanced ontology designer
* Context compiler visualization
* AI-assisted query builder
* Memory optimization automation

---

# 13. Future UX Opportunities

| Feature                      | Description                              |
| ---------------------------- | ---------------------------------------- |
| AI Copilot for Admin         | Ask system questions in natural language |
| Auto-detect memory anomalies | AI-generated incident detection          |
| Memory quality scoring       | Evaluate memory usefulness               |
| Replay agent cognition       | Step-by-step reasoning replay            |
| Collaborative graph editing  | Multi-user graph studio                  |
| Voice observability          | Speak queries to graph/search            |

---

# 14. Competitive Benchmarking

| Product          | Inspiration              |
| ---------------- | ------------------------ |
| Datadog          | Observability UX         |
| Temporal UI      | Workflow visibility      |
| Neo4j Bloom      | Graph exploration        |
| Kibana           | Search & logs            |
| OpenAI Platform  | API console simplicity   |
| Retool           | Internal tool ergonomics |
| Vercel Dashboard | Developer experience     |

---

# 15. Recommended Initial UI Modules

## Highest Priority

### Module 1 — Global Dashboard

Business + infrastructure overview.

### Module 2 — Memory Explorer

Most-used feature.

### Module 3 — Context Debugger

Core differentiation.

### Module 4 — Governance Center

Enterprise requirement.

### Module 5 — Graph Studio

Strategic visual layer.

---

# 16. Suggested UI Development Roadmap

## Sprint 1

* Design system
* Authentication
* App shell
* Navigation

## Sprint 2

* Dashboard
* Health monitoring
* Metrics

## Sprint 3

* Memory explorer
* Unified search
* Memory inspector

## Sprint 4

* Graph studio
* Timeline replay
* Entity explorer

## Sprint 5

* Governance center
* Audit logs
* Policies

## Sprint 6

* Context debugger
* Prompt assembly visualization
* Token analytics

---

# 17. Final Positioning

VNP Memory Console không chỉ là dashboard quản trị.

Nó là:

> “Cognitive Control Plane for Enterprise AI Systems”

Một hệ điều hành quan sát, kiểm soát, và tối ưu hóa trí nhớ của AI agents ở quy mô enterprise.
