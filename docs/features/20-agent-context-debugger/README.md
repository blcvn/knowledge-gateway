# Feature 20 — Agent Context Debugger

> **Loại:** Console UI | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

Agent Context Debugger là công cụ trace và debug quá trình context assembly — cho ML Engineer thấy chính xác AI "lấy" context từ đâu, qua những bước nào, và tại sao một số memories được chọn trong khi những cái khác bị bỏ qua.

---

## Business Logic

### Context Assembly Trace

Khi AI Agent gọi `memory_recall` hoặc `GET context`, Debugger có thể capture full trace:
- **Query** được gửi đi
- **Engines queried**: Danh sách engines và kết quả từ mỗi engine
- **Scoring**: Score của mỗi candidate memory
- **Token Budget**: Bao nhiêu tokens được allocate per engine
- **Final Selection**: Memories nào được chọn vào context
- **Latency**: Time per stage

### Trace History

List recent traces:
- Filter theo: agent_id, user_id, time range
- Sort theo latency (debug slow queries)
- Export trace as JSON

### Side-by-Side Comparison

Compare 2 traces để thấy:
- Tại sao context khác nhau giữa 2 queries
- Which engine contributed different results
- Score differences

---

## Dataflow

```
POST /v1/console/debugger/trace
        │
        ├── Input: {query: "...", agent_id, user_id, engines: ["all"]}
        │
        ▼
DebuggerHandler
        │
        ├── Execute context assembly với tracing mode enabled
        │         ├── Fan-out to all engines
        │         └── Capture: {engine, results, scores, latency} per engine
        │
        ├── Capture token budget allocation
        ├── Capture final selection
        └── Store trace → Return trace_id + full detail


GET /v1/console/debugger/traces/{id}
        └── Full trace detail: per-engine breakdown, scoring, selection

GET /v1/console/debugger/traces
        └── List recent traces (filterable)
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/console/debugger/trace` | Create debug trace |
| `GET` | `/v1/console/debugger/traces/{id}` | Get trace detail |
| `GET` | `/v1/console/debugger/traces` | List traces |
