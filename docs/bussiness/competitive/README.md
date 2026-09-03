# VNP Memory — Competitive Landscape

> Phân tích vị trí cạnh tranh của VNP Memory trong thị trường AI Memory.
> Dựa trên market research từ: [`docs/research/market/`](../../research/market/)

---

## Thị trường AI Memory — Hiện trạng 2026

Thị trường đang trong giai đoạn **fragmentation → consolidation**:

```
2022-2023: Vector DB wars (Pinecone vs Weaviate vs Qdrant)
2024:      Specialized memory (Zep, Mem0, Graphiti)
2025:      User-centric memory (Memobase, Supermemory)
2026:      Unified platforms (→ VNP Memory)
```

**Transition:** RAG → GraphRAG → Persistent Memory → **Cognitive Infrastructure**

---

## 5 Competitors — Điểm mạnh & Gap

### Cognee

**Vị trí:** Knowledge Graph construction engine

**Điểm mạnh:**
- **7-step Cognify pipeline**: classify → validate → chunk → extract entities → detect relationships → build graph → summarize
- **15+ search strategies**: GRAPH_COMPLETION, RAG_COMPLETION, CHUNKS, SUMMARIES, ENTITY_SEARCH, v.v.
- **Multi-dataset**: Isolate knowledge per project/tenant
- **Multi-modal**: PDF, text, audio, image, URL, CSV
- **Memify (non-destructive enrichment)**: Add graph layer lên existing data

**Gap (không có):**
- ❌ Session/conversational memory
- ❌ User profile extraction
- ❌ Agent lifecycle hooks
- ❌ Memory versioning / auto-forget
- ❌ Multi-agent coordination

**VNP Memory lấy gì từ Cognee:** F03 (Semantic Memory) dùng Cognee engine cho knowledge extraction.

---

### Graphiti

**Vị trí:** Temporal knowledge graph cho AI agents

**Điểm mạnh:**
- **Temporal facts**: `valid_at` / `invalid_at` / `expired_at` → không recall stale facts
- **Bi-directional edges**: Relationship từ A→B VÀ B→A
- **Episode types**: text / JSON / fact_triple — nhiều nguồn input
- **Custom ontology**: Domain-specific entity types
- **Sub-200ms search**: Optimized for production latency

**Gap (không có):**
- ❌ User profile memory (no YOLO engine)
- ❌ Filesystem/procedural memory
- ❌ Memory consolidation
- ❌ Agent hook capture / session replay
- ❌ Multi-agent coordination

**VNP Memory lấy gì từ Graphiti:** F02 (Episodic Memory) dùng Graphiti engine.

---

### Memobase

**Vị trí:** User profile memory system cho LLM apps

**Điểm mạnh:**
- **YOLO Engine (3 fixed LLM calls)**: Predictable cost — extract → merge → events
- **4 profile categories**: preference / fact / goal / habit
- **Context API < 100ms**: Pre-computed, prompt-ready string
- **Blob buffering**: Auto-flush tại FlushThreshold (configurable)
- **Profile score**: float64 confidence per attribute

**Gap (không có):**
- ❌ Graph memory / temporal reasoning
- ❌ Filesystem/procedural memory
- ❌ Agent lifecycle hooks
- ❌ Multi-agent coordination
- ❌ External data connectors

**VNP Memory lấy gì từ Memobase:** F05 (Profile Memory) dùng Memobase YOLO Engine.

---

### Supermemory

**Vị trí:** Adaptive knowledge graph + connector platform

**Điểm mạnh:**
- **Living KG**: Memory tự update, resolve contradictions
- **forgetAfter**: Configurable TTL per memory
- **Version chain**: parent → root, isLatest tracking
- **External connectors**: Google Drive, Gmail, Notion, OneDrive, GitHub
- **Static vs Dynamic**: Memory classification

**Gap (không có):**
- ❌ Session/conversational memory (no Zep-style)
- ❌ User profile extraction (no YOLO engine)
- ❌ Agent hook capture / session replay
- ❌ Multi-agent coordination (no leases)
- ❌ Memory consolidation pipeline

**VNP Memory lấy gì từ Supermemory:** F07 (Adaptive Memory) + connectors.

---

### Zep

**Vị trí:** End-to-end context engineering platform

**Điểm mạnh:**
- **Sub-200ms latency SLA**: Optimized for production
- **Context engineering**: Pre-formatted blocks cho LLM
- **Temporal extraction**: Auto-detect temporal facts từ conversations
- **Custom ontology**: Domain-specific entity types qua API
- **Multi-source**: Chat + business data + documents + events
- **Python/TypeScript/Go SDK**: First-class SDK support

**Gap (không có):**
- ❌ User profile memory (no YOLO engine)
- ❌ Filesystem/procedural memory
- ❌ Agent hook capture / session replay
- ❌ Multi-agent coordination
- ❌ Memory consolidation pipeline

**VNP Memory lấy gì từ Zep:** F04 (Conversational Memory) dùng Zep engine.

---

## Competitive Matrix

| Capability | Cognee | Graphiti | Memobase | Supermemory | Zep | **VNP Memory** |
|---|---|---|---|---|---|---|
| Knowledge graph | ✅ | ✅ | ❌ | ✅ (adaptive) | ✅ | ✅ (5 engines) |
| Temporal reasoning | ⚠️ | ✅ | ❌ | ✅ (isLatest) | ✅ | ✅ |
| User profile | ❌ | ❌ | ✅ | ⚠️ | ❌ | ✅ |
| Filesystem memory | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (OpenViking) |
| Auto-forget | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| Agent hooks | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (12 hooks) |
| Session replay | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Multi-agent coord. | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (leases) |
| Memory consolidation | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (4-tier) |
| GDPR cascading | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| MCP server | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ (37+ tools) |
| External connectors | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| Custom ontology | ❌ | ✅ | ❌ | ❌ | ✅ | ✅ |
| Multi-modal ingestion | ✅ | ⚠️ | ❌ | ✅ | ✅ | ✅ |
| Context < 100ms | ⚠️ | ✅ | ✅ | ⚠️ | ✅ | ✅ |
| Enterprise governance | ⚠️ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Unified API | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |

**Kết luận:** VNP Memory là hệ thống duy nhất có **full coverage** (✅ trong tất cả capabilities).

---

## Positioning Statement

> **VNP Memory** không cạnh tranh trực tiếp với Cognee, Graphiti, Memobase, Supermemory, hay Zep.
> VNP Memory **orchestrates** chúng — tích hợp 5 engines này dưới một Unified API,
> thêm AgentMemory Layer không ai có, và enterprise governance.
>
> **VNP Memory = Cognee + Graphiti + Memobase + Supermemory + Zep + AgentMemory Layer + Enterprise**

---

*Tham chiếu: [Research Insights](../research/README.md) | [Pain Points](../painpoints/README.md) | [Solutions](../solutions/README.md)*
