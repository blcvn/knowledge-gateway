# P1 — AI Agent Developer

> **Vai trò:** Xây dựng AI Agent có persistent memory cho sản phẩm thực tế (chatbot, coding assistant, research agent, support bot).
> **Kỹ năng:** Python / TypeScript / Go, async programming, LLM APIs, REST.
> **Tần suất sử dụng VNP Memory:** Hàng ngày.

---

## Bức tranh công việc hàng ngày

Một AI Agent Developer đang xây chatbot hỗ trợ khách hàng. Họ phải:
1. Gọi LLM với prompt + context
2. Lưu lại conversation vào đâu đó
3. Recall context liên quan cho câu hỏi tiếp theo
4. Quản lý user profile để cá nhân hóa

Nghe đơn giản — nhưng thực tế là một mớ hỗn độn.

---

## Pain Points

### PP-P1-01 — Agent mất toàn bộ ngữ cảnh sau mỗi phiên

**Mô tả:**
Mỗi conversation session là một tờ giấy trắng. Agent không biết user đã nói gì hôm qua, tuần trước, hay trong project khác. User phải giải thích lại từ đầu mỗi lần.

**Hậu quả thực tế:**
- User frustration: "Tôi đã nói điều này 10 lần rồi!"
- Agent đưa ra lời khuyên mâu thuẫn vì không nhớ context trước
- Churn rate tăng vì AI "không thông minh"

**Ví dụ cụ thể:**
```
Session 1: User nói "Tôi đang dùng PostgreSQL 15, không muốn upgrade"
Session 2: Agent suggest "Hãy upgrade lên PostgreSQL 16 để có tính năng X"
→ User: "Tôi vừa nói không muốn upgrade mà!"
```

**Features giải quyết:**
- [F04] Conversational Memory (Zep) — session memory persist cross-session
- [F05] Profile Memory (Memobase) — user facts/preferences được lưu vĩnh viễn
- [F01] Unified Memory API — `POST /v1/memory/store` lưu context, `POST /v1/memory/recall` truy xuất

---

### PP-P1-02 — Memory fragmented ở nhiều hệ thống, không ai nói chuyện được với nhau

**Mô tả:**
Developer phải tự maintain nhiều storage backends cùng lúc: vector DB cho semantic search, Redis cho session, PostgreSQL cho user data, Neo4j cho graph. Khi cần recall, họ phải query từng nơi rồi tự merge kết quả.

**Hậu quả thực tế:**
- Code complexity tăng gấp 5x
- Mỗi engine có API khác nhau, schema khác nhau
- Không có unified view về "AI đang biết gì về user này"
- Bug khi data inconsistent giữa các storage

**Ví dụ cụ thể:**
```python
# Developer phải tự viết code này mỗi project
results = []
results += vector_db.search(query)       # Semantic
results += neo4j.query(cypher)           # Graph
results += redis.get(session_key)        # Session
results += postgres.query(user_profile)  # Profile
merged = custom_merge_logic(results)     # Tự viết merge
```

**Features giải quyết:**
- [F01] Unified Memory API — 1 endpoint duy nhất, gateway tự route
- [F10] Hybrid Search Engine — BM25 + Vector + RRF tự động merge
- `POST /v1/memory/recall` — parallel fan-out + merge từ tất cả engines

---

### PP-P1-03 — RAG truyền thống không hiểu thời gian và quan hệ

**Mô tả:**
Vector similarity search không biết sự kiện A xảy ra trước hay sau sự kiện B. Không biết "fact X đúng vào tháng 3 nhưng đã bị thay thế vào tháng 6". Không hiểu quan hệ nhân quả giữa các sự kiện.

**Hậu quả thực tế:**
- Agent recall thông tin đã lỗi thời và trình bày như thật
- Không thể trả lời câu hỏi "lúc đó bạn nói gì?" hay "khi nào thì X thay đổi?"
- Mâu thuẫn khi retrieval ra 2 facts về cùng 1 entity ở 2 thời điểm khác nhau

**Ví dụ cụ thể:**
```
User: "Ngân sách project của tôi là bao nhiêu?"
[Vector search trả về fact từ tháng 1: "Budget: $50,000"]
[Nhưng fact tháng 6 đã thay đổi: "Budget revised to $80,000"]
Agent: "Budget của bạn là $50,000" ← SAI
```

**Features giải quyết:**
- [F02] Episodic Memory (Graphiti) — temporal facts với `valid_at`/`invalid_at`
- [F04] Conversational Memory (Zep) — graph-based retrieval với temporal context
- [F09] Agent Memory Lifecycle — `isLatest` flag, version chain tránh stale data

---

### PP-P1-04 — Knowledge cũ không tự update khi có thông tin mới

**Mô tả:**
User nói "Tôi đã chuyển từ React sang Vue", nhưng agent vẫn suggest React solutions vì memory cũ chưa được update. Không có cơ chế "contradiction resolution" — thông tin mới và cũ cùng tồn tại gây nhầm lẫn.

**Hậu quả thực tế:**
- Agent liên tục đưa ra advice không phù hợp với tình huống hiện tại
- Developer phải tự implement "memory invalidation" logic — phức tạp
- Users không tin tưởng AI vì hay sai

**Features giải quyết:**
- [F07] Adaptive Memory (Supermemory) — `forgetAfter`, `isLatest`, relation types: updates/extends/derives
- [F09] Agent Memory Lifecycle — Jaccard-based versioning, auto-update khi tương tự > threshold
- Khi user nói thông tin mới → memory cũ tự động mark `isLatest=false`

---

### PP-P1-05 — Không có user profile có cấu trúc từ conversations

**Mô tả:**
Sau 100 conversation sessions, developer không có cách nào biết "user này thích gì, dùng gì, mục tiêu là gì" — tất cả chỉ là raw text trong database. Không có structured profile để cá nhân hóa response.

**Hậu quả thực tế:**
- Không cá nhân hóa được AI response
- Không có analytics về user behavior từ conversations
- Product Manager không biết users care về gì

**Features giải quyết:**
- [F05] Profile Memory (Memobase YOLO Engine):
  - Tự động extract từ conversation: `preference` / `fact` / `goal` / `habit`
  - 3 LLM calls fixed per flush: extract → merge → events
  - `GET /v1/memobase/users/{uid}/profiles` → structured `{key, value, category, score}`

---

### PP-P1-06 — Context assembly chậm và tốn token

**Mô tả:**
Mỗi LLM call phải fetch toàn bộ conversation history → stuffed vào prompt → context window bị "ô nhiễm" bởi thông tin không liên quan → token cost cao + quality thấp.

**Hậu quả thực tế:**
- API cost tăng phi tuyến với độ dài conversation
- Latency p95 > 2s vì phải load/process lượng lớn context
- LLM bị "distracted" bởi irrelevant context → quality giảm

**Ví dụ:**
```
Thay vì: Nhồi 50K token lịch sử vào prompt
VNP Memory: Chỉ lấy top-10 relevant memories (< 2K tokens)
→ Tiết kiệm 80% token cost
```

**Features giải quyết:**
- [F05] Memobase context: `GET /v1/memobase/users/{uid}/context` — prompt-ready < 100ms
- [F06] OpenViking Tiered Context: L0 (~100 tok) → L1 (~2K tok) → L2 (full) — load on demand
- [F13] Context Injection — tự động inject chỉ relevant context với configurable token budget
- [F12] Consolidation Pipeline — nén raw observations thành summaries (70-90% size reduction)

---

### PP-P1-07 — Không track được agent đã làm gì trong session

**Mô tả:**
Khi agent gặp lỗi, developer không biết chính xác: agent đã call tool gì, LLM đã nhận prompt gì, decision được đưa ra trên cơ sở nào. Debugging là "mò kim đáy bể".

**Hậu quả thực tế:**
- Debug thời gian trung bình: 2-4 giờ cho 1 issue
- Không reproduce được lỗi vì không có full trace
- Khó tối ưu agent behavior vì không biết agent đang làm gì

**Features giải quyết:**
- [F08] Agent Observe & Hook Capture:
  - 12 hook types: tool_call, llm_prompt, llm_response, error, decision, v.v.
  - 14-step pipeline: capture → validate → dedup → redact → store → index → stream SSE
  - Session timeline với full event history
- [F26] Session Replay — xem lại agent session như video playback, scrub timeline
- [F20] Agent Context Debugger — trace context assembly pipeline

---

### PP-P1-08 — Multi-agent race conditions khi share resources

**Mô tả:**
Khi nhiều agent chạy song song cùng truy cập shared memory, xảy ra race conditions: Agent A và Agent B cùng update cùng memory → last-write-wins → data corruption.

**Hậu quả thực tế:**
- Data corruption trong shared memory space
- Nondeterministic behavior — khó debug
- Agent workflows thất bại khi scale > 2 agents

**Features giải quyết:**
- [F11] Multi-Agent Orchestration:
  - Distributed Leases: Agent phải acquire lease trước khi modify shared resource
  - Inter-agent Signals: Async handoff với guaranteed delivery (NATS)
  - Action DAG: Dependencies giữa actions, chỉ start khi prerequisites completed
  - Sentinels: Background watchers trigger action khi condition met

---

### PP-P1-09 — Observation và log storage bùng nổ không kiểm soát

**Mô tả:**
Mỗi session agent emit hàng nghìn events (tool calls, LLM calls, decisions...). Lưu raw hết → database bùng nổ. Không lưu → mất context cho debug và learning.

**Features giải quyết:**
- [F12] Consolidation Pipeline:
  - Tier 1: Compress raw observations (70-90% reduction)
  - Tier 2: Session summary khi session end
  - Tier 3: Procedural memory extraction (periodic)
  - Tier 4: Lessons & Insights cross-agent

---

## Summary — Tại sao P1 phải dùng VNP Memory

| Không có VNP Memory | Có VNP Memory |
|---|---|
| Tự xây memory infra (6-12 tháng) | Start trong < 5 phút (`make dev`) |
| 5-6 storage backends riêng lẻ | 1 API thống nhất |
| Tự viết context assembly | `GET /context` < 100ms |
| Debug agent = mò kim đáy bể | Session Replay + Hook Capture |
| Memory không tự cập nhật | Auto-versioning + auto-forget |
| Context window tốn kém | Tiết kiệm 80% token cost |
| Multi-agent race conditions | Distributed leases |
