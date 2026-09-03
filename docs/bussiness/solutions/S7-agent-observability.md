# S7 — Agent Observability & Debugging

> **Giải quyết Pain Points:** PP-P1-07, PP-P3-03
> **Actor chính:** P1 (AI Agent Developer), P3 (ML/AI Engineer)
> **Features:** F08 (Agent Observe), F20 (Context Debugger), F26 (Session Replay)

---

## Vấn đề cần giải quyết

Khi agent gặp lỗi hoặc trả lời sai, developer không biết: agent đã call tool gì, LLM nhận prompt gì, context nào được inject, decision dựa trên cơ sở gì. Debug mất 2-4 giờ mỗi issue.

---

## Giải pháp: 3-Layer Agent Observability

### Layer 1 — Hook Capture Pipeline (F08)

Observe Service thu thập MỌI hoạt động của agent qua 12 hook types:

```
Agent hoạt động
        │
        ▼
12 Hook Points:
  session_start    → "Agent bắt đầu session lúc 09:00"
  llm_prompt       → "Prompt gửi đến GPT-4: [full text]"
  llm_response     → "Response nhận được: [full text]"
  tool_call        → "Agent gọi: read_file('/src/auth.ts')"
  tool_response    → "File content: [content]"
  memory_read      → "Agent đọc memory: 'user prefers TypeScript'"
  memory_write     → "Agent lưu: 'Fixed bug in auth.ts'"
  decision         → "Decision: 'Use JWT instead of session cookies'"
  error            → "Error: connection timeout to DB"
  observation      → "Observed: test suite passing"
  checkpoint       → "Checkpoint: auth module completed"
  session_end      → "Session ended, 47 events captured"
```

**14-Step Processing Pipeline:**
```
Event received
  → Validate schema
  → Authenticate session
  → Deduplicate (30s TTL DedupMap)      ← Loại duplicate events
  → Privacy Redact                       ← Xóa API keys, PII
  → Parse Hook Type
  → Enrich with session metadata
  → Classify (raw vs compressed)
  → Store → agent_raw_observations
  → Index (BM25)
  → Embed (vector)
  → Publish NATS
  → Update Session State
  → Stream SSE                           ← Real-time tới Console
```

**Privacy Redaction tự động:**
- API keys: `sk-*`, `Bearer *`, `AKIA*` → `[REDACTED]`
- JWT tokens → `[JWT_REDACTED]`
- Email/phone/credit card → `[PII_REDACTED]`
- Database URLs → `[DB_URL_REDACTED]`

---

### Layer 2 — Agent Context Debugger (F20)

Trace toàn bộ context assembly — biết chính xác AI "nhìn thấy" gì trước mỗi LLM call:

```http
POST /v1/console/debugger/trace
{
  "session_id": "sess-001",
  "event_id": "evt-042"   // llm_prompt event
}

Response:
{
  "context_sources": [
    {"source": "memobase", "content": "User: Senior TS dev", "tokens": 45},
    {"source": "openviking_L1", "content": "auth.ts interface", "tokens": 230},
    {"source": "supermemory", "content": "JWT best practices", "tokens": 180}
  ],
  "total_tokens": 455,
  "retrieval_latency_ms": 87,
  "llm_prompt_preview": "You are a TypeScript expert. Context: ..."
}
```

**Session Timeline View:**
```http
GET /v1/console/sessions/{id}/timeline

[
  {ts: "09:00:01", hook: "session_start", summary: "New coding session"},
  {ts: "09:00:05", hook: "memory_read", summary: "Read 3 profile facts"},
  {ts: "09:00:06", hook: "llm_prompt", summary: "Prompt: 455 tokens"},
  {ts: "09:00:09", hook: "llm_response", summary: "Response: 'Use JWT...'"},
  {ts: "09:00:10", hook: "tool_call", summary: "read_file('auth.ts')"},
  {ts: "09:00:11", hook: "tool_response", summary: "File: 320 lines"},
  {ts: "09:00:15", hook: "decision", summary: "Use RS256 signing"},
  ...
]
```

---

### Layer 3 — Session Replay (F26)

Replay session như "video playback" — scrub timeline, filter events, control speed:

```
Timeline: 09:00 ──────────────── 09:45
           |    |    |   |    |    |
           S   LLM  Tool Err  Dec   E
           t   call call      ion   nd
           a
           r
           t

User scrubs đến ts=09:23 (Error event)
→ Xem full context: prompt, context, tool output dẫn đến error
→ Biết chính xác nguyên nhân
```

**JSONL Import — Replay sessions từ Claude Code:**
```http
POST /v1/observe/sessions/import
Content-Type: multipart/form-data

[Upload: transcript.jsonl]
→ Parse Claude Code format
→ Create virtual session
→ Replay available ngay
```

**Filter by hook type:**
```
Chỉ xem: memory_read + memory_write events
→ Hiểu AI đã "học" gì và "nhớ" gì trong session
→ Debug incorrect memory recalls
```

---

## Debug Workflow — từ 4 giờ xuống 20 phút

```
Báo cáo: "AI trả lời sai về budget của project"

Trước VNP Memory (4 giờ):
  1. Grep logs → không tìm thấy relevant
  2. Add debug prints → redeploy
  3. Reproduce lỗi → không reproduce được
  4. Guess và fix

Sau VNP Memory (20 phút):
  1. Mở Console → Sessions Explorer
  2. Tìm session đó → Timeline
  3. Click vào "memory_read" event
  4. Thấy ngay: "Agent recall fact '$50K' (isLatest=false)" ← Đây là lỗi
  5. Fix: Update memory versioning rule
```

---

## Kết quả

| Metric | Trước | Sau |
|---|---|---|
| Debug time per issue | 2-4 giờ | 20-30 phút |
| Root cause visibility | Không có | Full trace |
| Reproduce bugs | Khó | Session replay |
| Privacy in logs | Manual redaction | Auto-redact (step 5) |
| Real-time monitoring | Không | SSE stream |
| MTTR (Mean Time to Resolution) | 4 giờ | 30 phút |
