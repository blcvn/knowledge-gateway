# User Requirements Document (URD)

## Graphiti — Temporal Context Graph Engine

| Field | Value |
|-------|-------|
| **Product** | Graphiti (graphiti-core v0.28.2) |
| **Owner** | Zep Software, Inc. |
| **Last Updated** | 2026-05-07 |

---

## 1. Mục đích tài liệu

Mô tả yêu cầu người dùng đối với hệ thống Graphiti: personas, mô hình tương tác, SOPs, và acceptance criteria.

---

## 2. User Personas

### 2.1 AI Agent Developer
- **Vai trò:** Phát triển AI agents cần bộ nhớ ngữ cảnh dài hạn
- **Kỹ năng:** Python, async programming, LLM APIs
- **Mục tiêu:** Tích hợp Graphiti làm persistent memory layer
- **Use Cases:** Customer support bots, research assistants, AI companions

### 2.2 Platform Engineer
- **Vai trò:** Triển khai và vận hành Graphiti ở quy mô sản xuất
- **Kỹ năng:** DevOps, Docker, Graph DBs, monitoring
- **Mục tiêu:** Hệ thống ổn định, scalable, observable

### 2.3 Data Scientist / Knowledge Engineer
- **Vai trò:** Phân tích và tối ưu knowledge graph
- **Kỹ năng:** Graph theory, NLP, data analysis
- **Mục tiêu:** Khai phá insights, tối ưu search quality

### 2.4 Enterprise Integrator
- **Vai trò:** Tích hợp Graphiti vào hệ sinh thái doanh nghiệp
- **Kỹ năng:** REST APIs, system integration
- **Mục tiêu:** Kết nối với existing data pipelines

---

## 3. Mô hình tương tác

### 3.1 Python SDK (Direct Integration)

```python
from graphiti_core import Graphiti
from graphiti_core.llm_client import OpenAIClient, LLMConfig
from graphiti_core.embedder import OpenAIEmbedder, EmbedderConfig

graphiti = Graphiti(
    uri="bolt://localhost:7687", user="neo4j", password="password",
    llm_client=OpenAIClient(LLMConfig(api_key="sk-...")),
    embedder=OpenAIEmbedder(EmbedderConfig(api_key="sk-...")),
)
await graphiti.build_indices_and_constraints()

result = await graphiti.add_episode(
    name="meeting_notes", episode_body="Alice joined engineering in March.",
    source=EpisodeType.text, source_description="Meeting notes",
    reference_time=datetime.now(timezone.utc), group_id="project-alpha",
)

results = await graphiti.search(query="Who joined engineering?", group_ids=["project-alpha"])
```

**Lifecycle:** `Initialize → Build Indices → Add Episodes → Search → Close`

### 3.2 REST API Server

| Endpoint | Method | Mô tả |
|----------|--------|-------|
| `/v1/episodes` | POST | Ingest new episode |
| `/v1/episodes/bulk` | POST | Bulk ingestion |
| `/v1/search` | POST | Hybrid search |
| `/v1/entities/{uuid}` | GET | Retrieve entity |
| `/v1/edges/{uuid}` | GET | Retrieve edge/fact |
| `/v1/episodes/{uuid}` | DELETE | Remove episode |
| `/healthz` | GET | Health check |

### 3.3 MCP Server (AI Assistant Integration)

| Tool | Mô tả |
|------|-------|
| `add_memory` | Add episode to graph |
| `search_memory` | Search knowledge graph |
| `delete_entity_edge` | Delete edge |
| `delete_entity_node` | Delete entity |
| `delete_episode` | Delete episode |
| `get_entity_edge` | Retrieve edge details |
| `get_episodes` | List recent episodes |
| `clear_graph` | Clear entire graph |
| `get_status` | Server status |

---

## 4. Standard Operating Procedures (SOPs)

### SOP-001: Initial Setup

1. Khởi tạo graph database (Neo4j/FalkorDB/Kuzu)
2. Cài đặt: `pip install graphiti-core[neo4j]`
3. Cấu hình env vars (`NEO4J_URI`, `OPENAI_API_KEY`, etc.)
4. Build indices: `await graphiti.build_indices_and_constraints()`
5. Test với sample episode + search

### SOP-002: Episode Ingestion Pipeline

**Input Types:**
- `EpisodeType.text` — Documents, notes
- `EpisodeType.json` — Structured data
- `EpisodeType.message` — Chat messages
- `EpisodeType.fact_triple` — Pre-structured (S, P, O) triples

**Internal Pipeline:**
```
Raw Episode → Extract Nodes (LLM) → Extract Edges (LLM) →
Resolve Nodes (dedup) → Resolve Edges (conflict/invalidation) →
Build Episodic Edges → Generate Embeddings → Community Detection →
Graph Updated ✓
```

### SOP-003: Hybrid Search & Retrieval

**Search Methods:** Cosine Similarity, BM25, BFS Graph Traversal

**Reranking:** RRF, MMR, Cross-Encoder, Node Distance, Episode Mentions

**Pre-built Recipes:**
- `NODE_HYBRID_SEARCH_RRF` / `EDGE_HYBRID_SEARCH_RRF`
- `NODE_HYBRID_SEARCH_MMR` / `EDGE_HYBRID_SEARCH_MMR`
- `NODE_HYBRID_SEARCH_CROSS_ENCODER` / `EDGE_HYBRID_SEARCH_CROSS_ENCODER`

**Temporal Filtering:** Hỗ trợ `created_at_start/end`, `valid_at`, `invalid_at`

### SOP-004: Saga Management

- Nhóm episodes liên quan qua `saga_id`
- Hệ thống tự động liên kết theo thứ tự thời gian
- Saga summary tự động cập nhật

### SOP-005: Custom Ontology

```python
class PersonNode(BaseModel):
    name: str = Field(description="Full name")
    role: str = Field(default="")

result = await graphiti.add_episode(
    ..., entity_types=[PersonNode], edge_types=[WorksAtEdge],
)
```

### SOP-006: Multi-Backend Deployment

| Backend | Config |
|---------|--------|
| Neo4j | `Graphiti(uri="bolt://...", user="neo4j", password="...")` |
| FalkorDB | `FalkorDBDriver(host, port, graph_name)` |
| Kuzu | `KuzuDriver(db_path="./kuzu_data")` |
| Neptune | `NeptuneDriver(endpoint, region)` |

---

## 5. User Experience Requirements

| ID | Requirement |
|----|-------------|
| UX-001 | API phải intuitive, Pythonic, async/await nhất quán |
| UX-002 | Error messages rõ ràng, actionable |
| UX-003 | Type hints đầy đủ cho IDE support |
| UX-004 | Default configs hoạt động tốt cho hầu hết use cases |
| OX-001 | Health check trả về trạng thái chi tiết |
| OX-002 | OpenTelemetry tracing cho mọi operation |
| OX-003 | Token usage tracking cho cost management |
| SX-001 | Search latency < 1 giây |
| SX-002 | Kết quả bao gồm provenance |
| SX-003 | Temporal filtering chính xác |

---

## 6. Acceptance Criteria

| ID | Criteria |
|----|----------|
| AC-001 | Ingest 1 episode → verify entities/edges created |
| AC-002 | Search trả về relevant results |
| AC-003 | Temporal invalidation khi ingest contradicting facts |
| AC-004 | Bulk ingestion ≥100 episodes không lỗi |
| AC-005 | Multi-backend switch chỉ cần thay driver config |
| AC-006 | MCP Server tools hoạt động từ Claude Desktop |
| AC-007 | Custom ontology constraints enforced |
| AC-008 | Community detection tạo meaningful clusters |
