# Product Requirements Document (PRD)

## Graphiti — Temporal Context Graph Engine

| Field              | Value                                          |
|--------------------|------------------------------------------------|
| **Product**        | Graphiti (graphiti-core v0.28.2)                |
| **Owner**          | Zep Software, Inc.                             |
| **License**        | Apache License 2.0                             |
| **Repository**     | https://github.com/getzep/graphiti             |
| **Last Updated**   | 2026-05-07                                     |

---

## 1. Executive Summary

Graphiti là một framework mã nguồn mở để xây dựng và truy vấn **Temporal Context Graph** (đồ thị ngữ cảnh theo thời gian) cho các AI agent. Khác với knowledge graph truyền thống tĩnh, Graphiti theo dõi sự thay đổi của các sự kiện theo thời gian, duy trì provenance (nguồn gốc) đến dữ liệu gốc, và hỗ trợ cả ontology được định nghĩa trước (prescribed) lẫn tự học (learned) — được thiết kế đặc biệt cho agent hoạt động trên dữ liệu thực tế, liên tục thay đổi.

### Tầm nhìn sản phẩm

Cung cấp bộ nhớ ngữ cảnh cấp sản xuất cho AI agent, cho phép chúng hiểu không chỉ "điều gì đúng bây giờ" mà còn "điều gì đã đúng trước đây", thông qua một đồ thị tri thức temporal có khả năng mở rộng, truy vấn lai (hybrid retrieval), và tích hợp liên tục dữ liệu mới.

---

## 2. Problem Statement

### 2.1 Hạn chế của RAG truyền thống

| Vấn đề | Mô tả |
|---------|-------|
| **Batch processing** | Các hệ thống RAG truyền thống phụ thuộc vào xử lý hàng loạt và tóm tắt dữ liệu tĩnh |
| **Thiếu Temporal Tracking** | Không theo dõi được sự thay đổi thông tin theo thời gian |
| **Không có Provenance** | Không thể truy nguyên nguồn gốc dữ liệu |
| **Latency cao** | Tổng hợp LLM tuần tự mất từ vài giây đến hàng chục giây |
| **Flat context** | Trả về "document chunks" phẳng thay vì ngữ cảnh có cấu trúc |

### 2.2 Hạn chế của GraphRAG

| Vấn đề | Mô tả |
|---------|-------|
| **Tĩnh** | Chủ yếu dành cho tóm tắt tài liệu tĩnh |
| **Batch-oriented** | Yêu cầu xử lý lại toàn bộ graph khi có dữ liệu mới |
| **Khả năng thích ứng thấp** | Không hỗ trợ cập nhật gia tăng (incremental) |
| **Temporal cơ bản** | Chỉ theo dõi timestamp đơn giản, không invalidation tự động |

---

## 3. Product Value Proposition

Graphiti giải quyết các thách thức trên bằng cách cung cấp:

1. **Temporal Fact Management** — Mỗi fact có validity window; khi thông tin thay đổi, fact cũ bị invalidated chứ không bị xóa
2. **Episodes & Provenance** — Mỗi entity và relationship đều truy ngược được đến raw data (episodes) đã tạo ra nó
3. **Prescribed & Learned Ontology** — Định nghĩa entity/edge types trước qua Pydantic models, hoặc để cấu trúc tự xuất hiện từ dữ liệu
4. **Incremental Graph Construction** — Dữ liệu mới tích hợp ngay lập tức mà không cần recomputation
5. **Hybrid Retrieval** — Kết hợp semantic embeddings, keyword (BM25), và graph traversal để truy vấn chính xác, low-latency
6. **Multi-backend Scalability** — Hỗ trợ Neo4j, FalkorDB, Kuzu, Amazon Neptune

---

## 4. Target Users

| Persona | Mô tả |
|---------|-------|
| **AI Agent Developer** | Xây dựng agent cần bộ nhớ ngữ cảnh theo thời gian |
| **Platform Engineer** | Triển khai và vận hành hệ thống knowledge graph ở quy mô sản xuất |
| **Data Scientist** | Phân tích đồ thị tri thức, phát hiện mẫu và xu hướng |
| **Enterprise Integrator** | Tích hợp Graphiti vào pipeline dữ liệu doanh nghiệp |

---

## 5. Core Feature Set

### 5.1 Context Graph Engine (graphiti_core)

| Feature | Mô tả | Priority |
|---------|-------|----------|
| **Episode Ingestion** | Ingest dữ liệu từ nhiều nguồn (text, JSON, message, fact_triple) | P0 |
| **Entity Extraction** | Tự động trích xuất entities từ episodes thông qua LLM | P0 |
| **Edge/Fact Extraction** | Trích xuất relationships/facts với temporal validity windows | P0 |
| **Node Deduplication** | Tự động phát hiện và merge entities trùng lặp | P0 |
| **Edge Resolution** | Giải quyết xung đột edges, invalidation tự động khi thông tin thay đổi | P0 |
| **Bulk Episode Processing** | Xử lý nhiều episodes cùng lúc với parallel processing | P1 |
| **Community Detection** | Clustering algorithm để phát hiện communities of nodes | P1 |
| **Saga Management** | Liên kết và theo dõi chuỗi episodes liên quan qua Saga nodes | P1 |
| **Custom Ontology** | Định nghĩa entity types và edge types qua Pydantic models | P1 |
| **Triplet API** | Thêm trực tiếp (source, edge, target) triplets vào graph | P2 |

### 5.2 Hybrid Search System

| Feature | Mô tả | Priority |
|---------|-------|----------|
| **Semantic Search** | Vector similarity search qua embeddings | P0 |
| **BM25 Full-text Search** | Keyword-based fulltext search | P0 |
| **BFS Graph Traversal** | Breadth-first search trên graph | P1 |
| **Multi-layer Reranking** | RRF, MMR, Cross-encoder, Node Distance, Episode Mentions | P0 |
| **Search Filters** | Temporal filters (valid_at, invalid_at, created_at, expired_at), node labels, edge types | P1 |
| **Search Recipes** | Pre-built search configurations cho các use cases phổ biến | P1 |

### 5.3 Multi-Backend Graph Database Support

| Backend | Status | Mô tả |
|---------|--------|-------|
| **Neo4j** | ✅ Primary | Mặc định, full-featured |
| **FalkorDB** | ✅ Supported | In-memory graph DB, Docker-ready |
| **Kuzu** | ✅ Supported | Embedded graph DB |
| **Amazon Neptune** | ✅ Supported | Managed cloud graph DB + OpenSearch |

### 5.4 Multi-LLM Provider Support

| Provider | Status | Components |
|----------|--------|------------|
| **OpenAI** | ✅ Default | LLM + Embedder + Reranker |
| **Azure OpenAI** | ✅ Supported | LLM + Embedder |
| **Google Gemini** | ✅ Supported | LLM + Embedder + Reranker |
| **Anthropic** | ✅ Supported | LLM |
| **Groq** | ✅ Supported | LLM |
| **Ollama** | ✅ Supported | LLM + Embedder (via OpenAI-compatible) |
| **Voyage AI** | ✅ Supported | Embedder |

### 5.5 Server & API Layer

| Feature | Mô tả | Priority |
|---------|-------|----------|
| **FastAPI REST Server** | REST API cho ingest và retrieve operations | P0 |
| **MCP Server** | Model Context Protocol server cho AI assistants (Claude, Cursor) | P1 |
| **Health Check** | Endpoint kiểm tra trạng thái hệ thống | P0 |
| **Configurable Deployment** | YAML-based configuration, CLI args, Docker support | P1 |

### 5.6 Observability

| Feature | Mô tả | Priority |
|---------|-------|----------|
| **OpenTelemetry Tracing** | Distributed tracing cho mọi operation | P1 |
| **Token Usage Tracking** | Theo dõi token sử dụng theo prompt type | P1 |
| **Telemetry** | Anonymous usage statistics qua PostHog (opt-out) | P2 |

---

## 6. Data Model

### 6.1 Graph Components

```
┌─────────────────────────────────────────────────────────┐
│                    CONTEXT GRAPH                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐      RELATES_TO     ┌──────────────┐ │
│  │  EntityNode  │─────────────────────│  EntityNode  │ │
│  │  (Entity)    │   [EntityEdge]      │  (Entity)    │ │
│  └──────┬───────┘   - fact            └──────┬───────┘ │
│         │           - valid_at               │         │
│      MENTIONS       - invalid_at          MENTIONS     │
│         │           - fact_embedding         │         │
│  ┌──────┴───────┐                     ┌──────┴───────┐ │
│  │ EpisodicNode │     NEXT_EPISODE    │ EpisodicNode │ │
│  │  (Episode)   │────────────────────>│  (Episode)   │ │
│  └──────┬───────┘                     └──────────────┘ │
│         │                                               │
│      HAS_EPISODE                                        │
│         │                                               │
│  ┌──────┴───────┐                                       │
│  │   SagaNode   │  (groups episodes into sequences)     │
│  └──────────────┘                                       │
│                                                         │
│  ┌──────────────┐      HAS_MEMBER    ┌──────────────┐  │
│  │CommunityNode │───────────────────>│  EntityNode  │  │
│  │ (Community)  │  [CommunityEdge]   │              │  │
│  └──────────────┘                    └──────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 6.2 Node Types

| Node Type | Properties | Mô tả |
|-----------|-----------|-------|
| **EntityNode** | uuid, name, summary, labels[], attributes{}, name_embedding, group_id | Entities trong graph (people, products, concepts) |
| **EpisodicNode** | uuid, name, content, source, source_description, valid_at, entity_edges[], episode_metadata{} | Raw data đã ingest — ground truth |
| **CommunityNode** | uuid, name, summary, name_embedding, group_id | Cluster summary of related entities |
| **SagaNode** | uuid, name, group_id, summary, first_episode_uuid, last_episode_uuid, last_summarized_at | Groups related episodes into sequences |

### 6.3 Edge Types

| Edge Type | Mô tả |
|-----------|-------|
| **EntityEdge** (RELATES_TO) | Fact giữa 2 entities, có temporal validity (valid_at, invalid_at, expired_at) |
| **EpisodicEdge** (MENTIONS) | Liên kết Episode → Entity |
| **CommunityEdge** (HAS_MEMBER) | Liên kết Community → Entity |
| **HasEpisodeEdge** (HAS_EPISODE) | Liên kết Saga → Episode |
| **NextEpisodeEdge** (NEXT_EPISODE) | Liên kết Episode → Episode (sequence order) |

---

## 7. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     APPLICATION LAYER                       │
│  ┌───────────────┐  ┌─────────────────┐  ┌──────────────┐  │
│  │ FastAPI Server│  │   MCP Server    │  │ Python SDK   │  │
│  │  (REST API)   │  │ (AI Assistants) │  │ (graphiti_   │  │
│  │               │  │                 │  │     core)    │  │
│  └───────┬───────┘  └────────┬────────┘  └──────┬───────┘  │
│          │                   │                   │          │
│          └───────────┬───────┴───────────────────┘          │
│                      │                                      │
│  ┌───────────────────▼──────────────────────────────────┐   │
│  │              Graphiti Core Engine                     │   │
│  │  ┌──────────────────────────────────────────────┐    │   │
│  │  │  Episode Ingestion Pipeline                  │    │   │
│  │  │  - Extract Nodes (LLM)                       │    │   │
│  │  │  - Extract Edges (LLM)                       │    │   │
│  │  │  - Deduplicate (LLM)                         │    │   │
│  │  │  - Resolve Conflicts (LLM)                   │    │   │
│  │  │  - Build Episodic Edges                      │    │   │
│  │  │  - Generate Embeddings                       │    │   │
│  │  │  - Community Detection                       │    │   │
│  │  └──────────────────────────────────────────────┘    │   │
│  │  ┌──────────────────────────────────────────────┐    │   │
│  │  │  Hybrid Search Engine                        │    │   │
│  │  │  - Vector Similarity + BM25 + BFS            │    │   │
│  │  │  - Multi-strategy Reranking                  │    │   │
│  │  │  - Temporal & Property Filters               │    │   │
│  │  └──────────────────────────────────────────────┘    │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │            Pluggable Infrastructure                 │    │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌─────────┐  │    │
│  │  │  LLM    │ │Embedder │ │  Cross   │ │ Graph   │  │    │
│  │  │ Client  │ │ Client  │ │ Encoder  │ │ Driver  │  │    │
│  │  └────┬────┘ └────┬────┘ └────┬─────┘ └────┬────┘  │    │
│  │       │           │           │             │       │    │
│  │  OpenAI     OpenAI      OpenAI       Neo4j        │    │
│  │  Azure      Azure       Gemini       FalkorDB     │    │
│  │  Anthropic  Gemini      BGE          Kuzu         │    │
│  │  Gemini     Voyage                   Neptune      │    │
│  │  Groq       Ollama                                │    │
│  │  Ollama                                           │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## 8. Key Performance Requirements

| Metric | Target |
|--------|--------|
| **Search Latency** | Sub-second (<1000ms) cho hybrid search |
| **Ingestion Throughput** | Configurable via SEMAPHORE_LIMIT (10-50 concurrent) |
| **Embedding Dimension** | Configurable per provider (default 1536 for OpenAI) |
| **Max Episode Content** | Configurable, chunking support cho large documents |
| **Concurrency Control** | Semaphore-based, environment-configurable |

---

## 9. Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| **Python Version** | ≥ 3.10, < 4 |
| **Type Safety** | Full type hints, Pyright basic mode |
| **Code Quality** | Ruff linting (E, F, UP, B, SIM, I), single-quote style |
| **Testing** | pytest + pytest-asyncio, parallel via pytest-xdist |
| **Packaging** | Hatchling build system, optional extras per backend/provider |
| **Deployment** | Docker + Docker Compose, configurable via env vars |

---

## 10. Success Metrics

| Metric | Mô tả |
|--------|-------|
| **Graph Construction Accuracy** | Entity/edge extraction precision vs. ground truth |
| **Temporal Consistency** | Fact invalidation correctness when contradictions appear |
| **Search Relevance** | Hybrid retrieval quality measured by MRR/NDCG |
| **Adoption** | Community engagement (stars, forks, contributors) |
| **State of Art Memory** | Benchmark performance vs. competing agent memory solutions |

---

## 11. Roadmap Considerations

| Phase | Capability |
|-------|-----------|
| **Current** | Multi-backend support (Neo4j, FalkorDB, Kuzu, Neptune), MCP server, Saga management |
| **Next** | Enhanced content chunking strategies, improved community algorithms |
| **Future** | Streaming ingestion, real-time subscriptions, federated graph queries |
