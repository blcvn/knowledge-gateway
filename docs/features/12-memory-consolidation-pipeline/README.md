# Feature 12 — Memory Consolidation Pipeline

> **Loại:** AgentMemory | **Priority:** High | **Status:** Implemented (CR-AM-006)

## Mô tả

Memory Consolidation Pipeline là hệ thống nén và tổng hợp memory theo 4 tầng — chuyển đổi raw observations (nhiều, chi tiết) thành compressed knowledge (ít, cô đọng). Pipeline bao gồm session summarization, procedural memory extraction, và lessons/insights system.

Triết lý: AI không thể nhớ mọi thứ mãi mãi. Consolidation giúp "nén" experience thành knowledge durable hơn.

---

## Business Logic

### 4-Tier Consolidation

| Tier | Input | Output | Trigger |
|------|-------|--------|---------|
| **Tier 1 — Compression** | Raw observations (batch) | Compressed observations | Every N observations |
| **Tier 2 — Summarization** | Session messages + compressed obs | Session summary | Session end |
| **Tier 3 — Extraction** | Session summaries (multi-session) | Procedural memories | Periodic (daily) |
| **Tier 4 — Insights** | Procedural memories (cross-agent) | Lessons & Insights | Weekly / on demand |

### Tier 1: LLM Compression

- Nhóm raw observations theo time window (e.g., 5 phút).
- Một LLM call compress cả nhóm thành condensed form.
- Giảm storage 70-90%, giữ semantic meaning.
- Circuit breaker: nếu LLM fail → skip compression (giữ raw).

### Tier 2: Session Summarization

Khi session kết thúc, summarizer tạo session summary:
- **What was attempted**: Các task agent đã thực hiện.
- **What succeeded/failed**: Kết quả.
- **Key decisions made**: Các decision quan trọng.
- **Entities encountered**: Entities/files/tools được dùng.

Summary được lưu vào `session_summaries` table, có thể recall nhanh.

### Tier 3: Procedural Memory Extraction

Từ nhiều session summaries, pipeline extract **procedural memories** — generic procedures không tied to specific session:
- "Cách debug network timeout" (dựa trên 5 sessions gặp vấn đề tương tự)
- "Deploy process cho service X" (được verify qua nhiều lần deploy)

Procedural memories có higher durability — không decay nhanh như regular memories.

### Tier 4: Lessons & Insights

Cross-session, cross-agent analysis:
- **Lessons**: Specific lessons từ failed actions ("Don't call API X without rate limit").
- **Insights**: Higher-level patterns ("Most production bugs come from missing input validation").

Insights là knowledge cấp cao nhất, được share across agents trong same tenant.

---

## Dataflow

### Consolidation Pipeline Trigger

```
Session End Event (NATS: consolidation.trigger)
        │
        ▼
consolidation pipeline (trong memory-service hoặc observe-service)
        │
        ├── TIER 1: Compression
        │         ├── Load uncompressed raw observations for session
        │         ├── Group by time window (5 min)
        │         ├── For each group: LLM call → compressed text
        │         │         (Circuit Breaker: if LLM fail → skip, keep raw)
        │         └── Store → agent_compressed_observations
        │
        ├── TIER 2: Session Summary
        │         ├── Load all compressed observations + session messages
        │         ├── LLM call: "Summarize what happened in this session"
        │         │         Output: {attempted, succeeded, failed, decisions, entities}
        │         └── Store → session_summaries table
        │
        └── (async, periodic)
        │
        ▼
Background Job (daily)
        │
        ├── TIER 3: Procedural Extraction
        │         ├── Load session summaries (last 7 days)
        │         ├── Cluster similar summaries (same task type)
        │         ├── LLM call: "Extract generic procedure from these sessions"
        │         └── Store → procedural_memories table
        │
        ▼
Background Job (weekly)
        │
        └── TIER 4: Lessons & Insights
                  ├── Analyze: which procedures led to success vs failure?
                  ├── Cross-session pattern detection
                  ├── LLM call: "What lessons can be learned?"
                  ├── Store lessons → lessons table
                  └── Store insights → insights table
```

### LLM Circuit Breaker

```
LLM Call Attempt
        │
        ├── Circuit state: CLOSED (normal)
        │         → Call LLM → Success → reset failure count
        │                    → Failure → increment failure count
        │                                if failure_count > threshold → OPEN circuit
        │
        ├── Circuit state: OPEN
        │         → Skip LLM call
        │         → Fallback: keep raw data (Tier 1) / skip summary (Tier 2)
        │         → After timeout (30s) → HALF-OPEN
        │
        └── Circuit state: HALF-OPEN
                  → Try 1 LLM call
                  → Success → CLOSE circuit
                  → Failure → OPEN circuit again
```

---

## Database Tables

| Table | Nội dung |
|-------|---------|
| `agent_compressed_observations` | Tier 1: compressed observation batches |
| `session_summaries` | Tier 2: per-session summaries |
| `procedural_memories` | Tier 3: extracted procedures |
| `lessons` | Tier 4: specific lessons |
| `insights` | Tier 4: high-level patterns |

---

## Services

| Service | Vai trò |
|---------|---------|
| `memory-service` (extended) | Consolidation pipeline, circuit breaker |
| `pipeline-service` | Job scheduling, queue management |
| `vnp-pipelines` | Platform-level pipeline monitoring |
