# VNP Memory — Research Insights & Design Principles

> Tài liệu này tổng hợp insights từ **neuroscience research** và **market research** thành
> các nguyên lý thiết kế của VNP Memory — lý giải "tại sao" đằng sau từng quyết định kiến trúc.
>
> **Nguồn:**
> - Neuroscience: [`docs/research/*.md`](../../research/)
> - Market: [`docs/research/market/`](../../research/market/)

---

## Phần 1 — Neuroscience Foundations

### 1.1 Não người có 2 hệ thống memory — VNP Memory làm theo

**Nguồn:** [`sleep.md`](../../research/sleep.md), [`sensor.md`](../../research/sensor.md)

**Complementary Learning Systems Theory:**
```
Hippocampus = RAM (fast, temporary, high-capacity)
    → Ghi mọi thứ ngay lập tức trong ngày
    → Có thể bị overwritten nhanh

Neocortex = Hard Drive (slow, permanent, structured)
    → Chỉ nhận thông tin đã được "validated" qua sleep
    → Học chậm nhưng bền
```

**Implication cho VNP Memory:**
```
AgentMemory Observe (F08) = Hippocampus
    → Capture MỌI event ngay lập tức
    → Raw observations — high volume, temporary

Consolidation Pipeline (F12) = Sleep + Neocortex
    → Tier 1-4: compress → summarize → extract procedures → insights
    → "Offline processing" khi agent không active
    → Permanent memory với higher durability

Kết quả: Không mất raw data (capture all) nhưng storage không bùng nổ (consolidate)
```

---

### 1.2 Não không lưu facts — lưu **prediction errors**

**Nguồn:** [`predictive-processing.md`](../../research/predictive-processing.md)

**Predictive Processing Framework:**
```
Não KHÔNG làm:  World → Sense → Store
Não THỰC SỰ:   Predict → Compare → Store (prediction error)
```

Não liên tục tạo **world model**, dự đoán điều sẽ xảy ra.
Khi thực tế khác dự đoán → prediction error → UPDATE world model.
Chỉ những điều **bất ngờ** (high prediction error) mới được ghi nhớ mạnh.

**Implication cho VNP Memory:**

| Não | VNP Memory |
|---|---|
| World model | Memory Graph (6 engines) |
| Prediction error | Contradiction detection → `isLatest=false` |
| Update world model | Memory versioning (Supermemory/F07, F09) |
| Replay (consolidation) | Consolidation Pipeline Tier 1-4 (F12) |
| Hippocampus fast-write | Agent Observe instant capture (F08) |
| Neocortex slow-write | Procedural Memory extraction (F12 Tier 3) |

> **Design principle:** Memory không phải "lưu tất cả" mà là "lưu những gì khác với kỳ vọng".
> Contradiction resolution (F07, F09) là tính năng quan trọng nhất — không phải storage.

---

### 1.3 Memory là một **mạng lưới quan hệ**, không phải list

**Nguồn:** [`personal-memory.md`](../../research/personal-memory.md), [`synapse.md`](../../research/synapse.md)

**Schema Theory:**
> "Não ghi nhớ tốt những thông tin có thể gắn vào một mạng lưới kiến thức đã tồn tại."

Khi học thông tin mới:
- Người có schema phong phú → gắn được nhiều connections → nhớ nhanh và lâu
- Người không có schema → isolated node → quên nhanh

**Hebbian Learning:** "Neurons that fire together, wire together"
- Synapse mạnh hơn khi 2 neurons kích hoạt cùng nhau nhiều lần
- Synaptic strength = "weight" của connection

**Implication cho VNP Memory:**

```
Knowledge Graph (Cognee F03, Graphiti F02, Zep F04):
  → Mỗi memory không phải isolated node mà có edges đến entities liên quan
  → Recall qua graph traversal: "nhớ" thông qua relationships
  → Hybrid Search (F10): graph path + vector similarity giống schema activation

Memory Lifecycle (F09):
  → Salience score = "synaptic strength"
  → Memories được recalled nhiều → score tăng → khó bị evict
  → Memories không được access → score giảm → evict (giống synaptic pruning)

Jaccard-based Versioning (F09):
  → Khi thông tin mới tương tự (Jaccard > threshold) → merge thay vì create new
  → Giống schema assimilation (Piaget): "fit new info into existing schema"
```

---

### 1.4 Ngủ = Offline Consolidation — Agent cần "giờ offline"

**Nguồn:** [`sleep.md`](../../research/sleep.md)

**9 chức năng của sleep trong memory:**
1. Hippocampus replay experiences → neocortex consolidation
2. Strengthen useful memories (important signals)
3. Remove unnecessary information (synaptic pruning)
4. Reorganize knowledge (extract general patterns)
5. Remove noise from weak connections
6. Integrate multiple memories across sessions
7. Support **forgetting** as well as remembering
8. Save energy (offline processing)
9. Different stages do different jobs (NREM vs REM)

**Implication → Consolidation Pipeline (F12):**

| Sleep Stage | Chức năng | VNP Memory Tier |
|---|---|---|
| NREM Stage 1-2 | Light compression, immediate replay | Tier 1: LLM Compression |
| NREM Stage 3-4 | Deep replay, memory transfer to cortex | Tier 2: Session Summary |
| REM | Emotional processing, pattern extraction | Tier 3: Procedural Memory |
| Multi-night integration | Cross-session learning, insight | Tier 4: Lessons & Insights |

> **Insight quan trọng:** Não cần `forgetAfter` cũng như cần remember.
> **F07 `forgetAfter`** và **F09 eviction** là feature quan trọng, không phải bug.

---

### 1.5 Tiered Context = Neocortex Hierarchy

**Nguồn:** [`neocortex.md`](../../research/neocortex.md), [`predictive-processing.md`](../../research/predictive-processing.md)

Neocortex có 6 lớp, mỗi lớp xử lý ở độ abstraction khác nhau:
- Layer 1-2: Abstract, high-level concepts
- Layer 4: Detail-level sensory input
- Layer 6: Feedback, prediction

**Implication → OpenViking Tiered Context (F06):**

| Neocortex Layer | Abstraction | OpenViking Tier |
|---|---|---|
| Layer 1-2 (Abstract) | Concept-level | L0 (~100 tokens): "File này là auth middleware" |
| Layer 3-4 (Intermediate) | Function-level | L1 (~2K tokens): Core functions + interfaces |
| Layer 5-6 (Detail) | Implementation-level | L2 (Full): Mọi dòng code |

> **Design principle:** Load L0 mặc định, L1 khi cần tương tác, L2 chỉ khi cần edit.
> **Mirrors** how the brain handles abstraction hierarchies.

---

## Phần 2 — Market Research Insights

### 2.1 Ai đang giải quyết gì?

| Engine | Core Innovation | Limitation |
|---|---|---|
| **Cognee** | 7-step cognify pipeline; 15+ search types; multi-dataset | No user profiling; no session memory; no agent lifecycle |
| **Graphiti** | Temporal KG với `valid_at`/`invalid_at`; bi-directional edges; episode types | No user profiles; no filesystem memory; no consolidation |
| **Memobase** | YOLO Engine (3 fixed LLM calls); profile categories (pref/fact/goal/habit); `<100ms` context | No graph; no temporal reasoning; no agent orchestration |
| **Supermemory** | Living KG; auto-forget; external connectors (Drive/Gmail/Notion/GitHub); version chain | No session management; no hook capture; no multi-agent |
| **Zep** | Sub-200ms latency; context engineering; temporal extraction; custom ontology | No user profiling; no agent orchestration; no consolidation |

### 2.2 Pain points từ competitor research

**Từ Zep URD (mirror với P1-P4):**
- AI Agent Developer: "Managing conversation state, assembling relevant context, handling temporal data changes"
- ML/AI Engineer: "Generic ontologies miss domain entities, difficulty measuring retrieval quality"
- Platform Engineer: "Managing multi-service dependencies, monitoring graph processing latency"

**Từ Memobase PRD:**
- "Mem0: Chi phí LLM cao, thiếu cấu trúc profile"
- "Context window limitation"

**Từ Graphiti — Limitations of traditional RAG:**
- Vector search không hiểu temporal changes
- GraphRAG tốt hơn nhưng vẫn thiếu user profile layer

### 2.3 VNP Memory — Competitive Advantage

VNP Memory là hệ thống **duy nhất** trong thị trường:
1. **Unify 6 memory engines** dưới 1 API thống nhất
2. **AgentMemory Layer** (Observe + Lifecycle + Orchestration + Consolidation)
3. **Neuroscience-inspired design** (sleep-like consolidation, schema-based versioning)
4. **Enterprise governance** (GDPR cascading, OPA policies, audit trail)
5. **MCP + REST dual interface** (IDE + Framework integration)

---

## Phần 3 — Design Principles (từ Research)

Tổng hợp các nguyên lý thiết kế của VNP Memory, derived từ neuroscience:

| Principle | Nguồn neuroscience | Implementation |
|---|---|---|
| **Capture everything, store smart** | Hippocampus vs Neocortex | F08 (capture all) + F12 (consolidate) |
| **Memory = relationships, not facts** | Schema theory, Hebbian | Knowledge Graphs (F02, F03, F04) |
| **Surprise drives learning** | Predictive Processing | Contradiction detection (F07, F09) |
| **Forgetting is a feature** | Sleep pruning | forgetAfter + eviction (F07, F09) |
| **Offline consolidation needed** | Sleep stages → memory transfer | Consolidation Pipeline (F12) |
| **Tiered abstraction** | Neocortex 6 layers | L0/L1/L2 context (F06) |
| **Context = active reconstruction** | Memory as reconstruction | Context assembly (F05, F13) |
| **Temporal reasoning essential** | Memory has timestamps | valid_at/invalid_at (F02, F09) |

---

*Tài liệu này là cơ sở lý luận cho: [Pain Points](../painpoints/README.md) | [Solutions](../solutions/README.md) | [PRD](../../product/v2/PRD.md)*
