# Zep — Product Requirements Document (PRD)

> **Product**: Zep — End-to-End Context Engineering Platform  
> **Version**: 1.0  
> **Generated**: 2026-05-07  
> **Status**: Active (Cloud) / Deprecated (Community Edition)

---

## 1. Product Vision

Zep is an **end-to-end context engineering platform** that delivers the right information to AI agents at the right time with **sub-200ms latency**. It solves the critical *agent context problem* — the challenge of assembling comprehensive, relationship-aware context from heterogeneous data sources — enabling AI agents to perform accurately and consistently in production environments.

---

## 2. Problem Statement

### 2.1 Core Problem

Modern AI agents suffer from **context fragmentation**: relevant information is scattered across chat histories, business data, documents, and application events. Without intelligent context assembly, agents produce inaccurate, hallucinated, or inconsistent responses.

### 2.2 Existing Limitations

| Challenge | Impact |
|---|---|
| **Stateless LLMs** | No persistent memory across conversations |
| **Context window limits** | Cannot fit all relevant data into a single prompt |
| **Temporal blindness** | Standard RAG cannot track how facts evolve over time |
| **Relationship unawareness** | Vector search misses interconnected facts and entities |
| **Latency constraints** | Production agents require sub-second response times |

---

## 3. Target Users

### 3.1 Primary Personas

| Persona | Description | Key Needs |
|---|---|---|
| **AI Application Developer** | Builds conversational AI, copilots, and agent systems | SDK integration, fast retrieval, memory persistence |
| **ML/AI Engineer** | Optimizes agent accuracy and context quality | Ontology customization, evaluation tools, graph inspection |
| **Platform Engineer** | Deploys and operates AI agent infrastructure | Docker deployment, health checks, observability, scalability |

### 3.2 Secondary Personas

| Persona | Description | Key Needs |
|---|---|---|
| **AI Framework Author** | Builds/maintains agent frameworks (LangChain, AutoGen, CrewAI) | Integration SDK, standardized memory interface |
| **Research Engineer** | Benchmarks and evaluates memory systems | Eval harness, reproducible experiments, metric analysis |

---

## 4. Value Proposition

### 4.1 Core Value

Zep transforms raw conversational and business data into **pre-formatted, relationship-aware context blocks** optimized for LLM consumption, delivered with enterprise-grade latency and reliability.

### 4.2 Differentiation

| Capability | Zep | Traditional RAG | Manual Context |
|---|---|---|---|
| Temporal reasoning | ✅ `valid_at`/`invalid_at` timestamps | ❌ | ❌ |
| Relationship awareness | ✅ Knowledge graph edges | ❌ Vector similarity only | ❌ |
| Multi-source ingestion | ✅ Chat + data + docs + events | ⚠️ Documents only | ❌ |
| Latency SLA | ✅ < 200ms | ⚠️ Variable | ✅ |
| Automatic extraction | ✅ LLM-powered graph building | ❌ Manual indexing | ❌ |
| Context assembly | ✅ Pre-formatted blocks | ❌ Raw chunks | ❌ |

---

## 5. Product Architecture

### 5.1 Three-Step Workflow

```
1. ADD CONTEXT    →    2. GRAPH RAG    →    3. RETRIEVE & ASSEMBLE
   (Messages,            (Automatic            (Pre-formatted,
    Business Data,        Relationship           relationship-aware
    Documents,            Extraction,            context blocks
    App Events)           Temporal KG)           for LLM)
```

### 5.2 System Components

```
┌──────────────────────────────────────────────────────────────────┐
│                         Client Layer                             │
│  Python SDK │ TypeScript SDK │ Go SDK │ REST API │ MCP Server    │
└──────────────┬───────────────────────────────────────────────────┘
               │
┌──────────────▼───────────────────────────────────────────────────┐
│                       Zep Cloud API                              │
│           (gRPC/REST gateway, Auth, Rate-limiting)               │
└──────────────┬───────────────────────────────────────────────────┘
               │
      ┌────────┴────────┐
      ▼                 ▼
┌──────────┐    ┌──────────────────┐
│ Postgres │    │    Graphiti      │
│ (Users,  │    │  (Graph RAG,     │
│ Sessions,│    │   Fact Store,    │
│ Messages)│    │   Extraction)    │
└──────────┘    └────────┬─────────┘
                         │
                   ┌─────▼──────┐
                   │   Neo4j    │
                   │  (Temporal │
                   │   KG Store)│
                   └────────────┘
```

---

## 6. Feature Requirements

### 6.1 Core Features (P0)

#### F1: User & Session Management
- **F1.1**: CRUD operations for users with metadata (JSONB)
- **F1.2**: Session/Thread lifecycle management with soft-delete support
- **F1.3**: Session-to-user association for personalized context
- **F1.4**: Multi-tenant isolation via `project_uuid`

#### F2: Memory Management
- **F2.1**: Message ingestion with role-typed messages (`user`, `assistant`, `system`, `function`, `tool`)
- **F2.2**: Memory retrieval with configurable `lastN` message history
- **F2.3**: Automatic Graphiti graph ingestion on message creation
- **F2.4**: Combined memory response (messages + relevant facts)

#### F3: Temporal Knowledge Graph (Graph RAG)
- **F3.1**: Automatic relationship extraction from ingested data
- **F3.2**: Temporal fact management with `valid_at`/`invalid_at` semantics
- **F3.3**: Typed entity nodes (User, Assistant, Preference, Organization, Event, Location, Document, Topic, Object)
- **F3.4**: Typed edge relationships (LOCATED_AT, OCCURRED_AT, custom)

#### F4: Search & Retrieval
- **F4.1**: Semantic graph search with scope filtering (edges, nodes, episodes)
- **F4.2**: Multiple reranking strategies: RRF, MMR, node_distance, episode_mentions, cross_encoder
- **F4.3**: Label and type filtering for precision control
- **F4.4**: Context assembly with custom template support

### 6.2 Integration Features (P1)

#### F5: Multi-SDK Support
- **F5.1**: Python SDK (`zep-cloud >= 3.0.0`)
- **F5.2**: TypeScript/JavaScript SDK (`@getzep/zep-cloud`)
- **F5.3**: Go SDK (`github.com/getzep/zep-go/v3`)

#### F6: Framework Integrations
- **F6.1**: Microsoft AutoGen integration (`zep-autogen`) — `autogen_core.memory.Memory` interface
- **F6.2**: CrewAI integration (`zep-crewai`) — `ZepUserStorage` + `ZepGraphStorage`
- **F6.3**: Google ADK integration (`zep-adk`)
- **F6.4**: LiveKit integration (`zep-livekit`)

#### F7: MCP Server
- **F7.1**: 13 read-only tools for AI assistant access
- **F7.2**: Dual transport: stdio (Claude Desktop/Cline) + HTTP Streamable (Claude Code)
- **F7.3**: Docker deployment support

### 6.3 Evaluation & Benchmarking Features (P2)

#### F8: Eval Harness
- **F8.1**: End-to-end QA pipeline evaluation (Search → Context Evaluation → Generate → Grade)
- **F8.2**: Multi-user support with independent test execution
- **F8.3**: Decoupled user/document ingestion with combinatorial evaluation
- **F8.4**: Reproducible runs with config snapshotting

#### F9: Benchmarking
- **F9.1**: LoCoMo long-context conversation benchmark
- **F9.2**: LongMemEval extended memory benchmark
- **F9.3**: Configurable graph retrieval parameters (edge/node limits, reranker selection)

### 6.4 Ontology & Customization (P1)

#### F10: Knowledge Graph Ontology
- **F10.1**: Default 9-node, 2-edge ontology with priority-based classification
- **F10.2**: Custom entity/edge type definition via Pydantic models
- **F10.3**: Custom instructions for domain-specific graph extraction
- **F10.4**: User summary instructions for node-level summarization

---

## 7. Non-Functional Requirements

### 7.1 Performance

| Metric | Target |
|---|---|
| Context retrieval latency | < 200ms (Cloud SLA) |
| API request timeout | 30 seconds |
| Max request payload | 5MB |
| Concurrent metadata updates | Handled via PostgreSQL advisory locks |

### 7.2 Security & Compliance

| Requirement | Implementation |
|---|---|
| Authentication | API Key (Cloud) / Shared Secret (CE) |
| MCP access control | Read-only by design |
| Data deletion | Soft deletes with `deleted_at` timestamps |
| Multi-tenancy | `project_uuid` isolation on all entities |
| Compliance (Cloud) | SOC2 Type 2, HIPAA |

### 7.3 Observability

| Signal | Implementation |
|---|---|
| Logging | Structured logging (`log/slog`, JSON/console) |
| Tracing | OpenTelemetry via `otelchi` middleware |
| Health checks | `GET /healthz` |
| Telemetry | Anonymous usage tracking (opt-out available) |

---

## 8. Technology Stack

### 8.1 Cloud Platform
- Managed service with enterprise SLAs
- SDKs: Python, TypeScript/JavaScript, Go

### 8.2 Community Edition (Legacy/Deprecated)
| Layer | Technology |
|---|---|
| Language | Go 1.21+ |
| HTTP | `go-chi/chi` v5 |
| ORM | `uptrace/bun` (PostgreSQL) |
| Database | PostgreSQL + pgvector |
| Graph DB | Neo4j 5.22+ |
| Graph Engine | Graphiti (separate service) |
| Observability | OpenTelemetry |

### 8.3 MCP Server
| Layer | Technology |
|---|---|
| Language | Go 1.21+ |
| MCP SDK | `modelcontextprotocol/go-sdk` |
| Transport | stdio / HTTP Streamable |

### 8.4 Integrations
| Layer | Technology |
|---|---|
| Language | Python 3.10+ |
| Package Manager | UV workspace |
| Quality | ruff, mypy, pytest |
| CI/CD | GitHub Actions |

---

## 9. Success Metrics

| Metric | Target | Measurement |
|---|---|---|
| Retrieval latency (p95) | < 200ms | Cloud monitoring |
| Context completeness rate | > 90% | Eval harness |
| Answer accuracy rate | > 80% | Eval harness |
| SDK adoption | 3+ framework integrations | PyPI downloads |
| Integration coverage | AutoGen, CrewAI, ADK, LiveKit | Framework compatibility matrix |

---

## 10. Roadmap

### Phase 1 — Foundation (Complete)
- ✅ Core data model (User, Session, Message, Fact)
- ✅ Graph RAG with temporal knowledge graph
- ✅ REST API with full CRUD
- ✅ Python/TypeScript/Go SDKs

### Phase 2 — Integration Ecosystem (Current)
- ✅ MCP Server with 13 read-only tools
- ✅ AutoGen, CrewAI, ADK, LiveKit integrations
- ✅ Eval harness with reproducible runs
- 🔲 LangChain/LlamaIndex integrations

### Phase 3 — Enterprise Hardening (Planned)
- 🔲 MCP write operations (gated)
- 🔲 Custom reranker training
- 🔲 Advanced ontology builder UI
- 🔲 Real-time streaming context updates

---

## 11. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Graph ingestion latency (10-20s async) | Stale context on rapid exchanges | Document async nature; polling utilities in eval harness |
| CE deprecation disruption | Users on self-hosted must migrate | Migration guide; Cloud free tier |
| Ontology complexity | Misconfigured graphs reduce accuracy | Default ontology with priority-based classification |
| MCP read-only limitation | AI assistants cannot write data | Intentional safety design; write via SDK only |

---

*Document generated from source analysis of `github.com/getzep/zep` repository.*
