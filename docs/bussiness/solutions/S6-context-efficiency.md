# S6 — Smart Context Assembly & Token Efficiency

> **Giải quyết Pain Points:** PP-P1-06, PP-P1-09, PP-P5-04
> **Actor chính:** P1 (AI Agent Developer), P5 (IDE Plugin User)
> **Features:** F05, F06, F12, F13

---

## Vấn đề cần giải quyết

Naive approach: nhồi toàn bộ conversation history vào prompt → context window "ô nhiễm" bởi irrelevant info → token cost tăng phi tuyến → LLM distracted → quality giảm. Với 100K token context = $0.30-1.50 mỗi call.

---

## Giải pháp: 3 lớp Context Optimization

### Lớp 1 — Tiered Context (OpenViking, F06)

Thay vì load full content, chia content thành 3 tầng, chỉ load khi cần:

```
L0 — Abstract (~100 tokens)
    "File này là authentication middleware, xử lý JWT validation"
    → AI đọc để orient, hiểu file là gì
    → Load mặc định cho mọi file

L1 — Overview (~2,000 tokens)
    Core functions, interfaces, public API
    → Load khi AI cần viết code tương tác với file

L2 — Full Detail (toàn bộ)
    Mọi dòng code
    → Chỉ load khi AI cần edit file
```

**Kết quả:** Thay vì load 50 files × 500 lines = 25K tokens → Chỉ load L0 cho 50 files = 5K tokens, L1 cho 5 files relevant = 10K tokens. **Tiết kiệm 60-80% token**.

```http
POST /v1/ov/search
{
  "query": "authentication logic",
  "tier": "L1",           // chỉ lấy L1
  "max_results": 5
}
```

---

### Lớp 2 — Context Injection với Token Budget (F13)

Context Injection là middleware tự động inject **chỉ relevant memory** vào LLM call:

```
Before LLM call:
    Query relevant memory
        ├── Memobase profile: top-5 relevant preferences
        ├── OpenViking L0: files liên quan đến task
        └── Supermemory: relevant domain facts
    
    Token Budget Management:
        budget = 2000 tokens
        priority 1: user profile (must-have)  → 200 tokens
        priority 2: task-specific facts        → 800 tokens
        priority 3: related context            → 1000 tokens
        Total: 2000 tokens ← không vượt budget
    
    Inject vào system prompt:
        [MEMORY CONTEXT]
        User: Senior TypeScript dev, prefer functional programming
        Task: Working on auth middleware
        Relevant: JWT library docs cached
        [/MEMORY CONTEXT]
```

**Cấu hình:**
```http
POST /v1/sm/context-inject
{
  "query": "current task",
  "user_id": "user-123",
  "token_budget": 2000,
  "sources": ["memobase", "openviking", "supermemory"],
  "scope": "project"
}
```

**Agent Scoping:**
- `isolated`: Chỉ memory của session này
- `shared`: Memory share giữa agents cùng tenant
- `project`: Memory scoped theo project namespace

---

### Lớp 3 — Consolidation Pipeline (F12)

Raw observations từ agent sessions tích lũy → 70-90% compression qua 4 tầng:

```
Raw observations (100% size)
        │
        ▼ Tier 1: LLM Compression (every N observations)
        │  Group by 5-minute windows → compress thành condensed text
        │  Circuit breaker: LLM fail → keep raw
        │
        ▼ Tier 2: Session Summary (khi session kết thúc)
        │  LLM summarize: {attempted, succeeded, failed, decisions, entities}
        │  → session_summaries table (~500 tokens)
        │
        ▼ Tier 3: Procedural Memory Extraction (daily cron)
        │  Extract procedures từ nhiều session summaries
        │  "Cách debug timeout: [5 bước]"
        │
        ▼ Tier 4: Lessons & Insights (weekly)
           Cross-agent patterns, best practices
           → Highest durability, shared across agents
```

**Storage reduction:**
| Tầng | Kích thước | Reduction |
|---|---|---|
| Raw observations | 100% | baseline |
| After Tier 1 | 15-30% | 70-85% |
| After Tier 2 | 2-5% | 95-98% |
| Tier 3 Procedures | < 1% | >99% |

**Recall hiệu quả:** Thay vì scan 10K raw observations → Recall từ session summary (500 tokens) → Nhanh hơn 20x.

---

## So sánh Token Cost

| Approach | Token per call | Cost (GPT-4, $10/1M) | Latency p95 |
|---|---|---|---|
| Naive: full history | 50,000 | $0.50 | 3-5s |
| RAG vector only | 5,000 | $0.05 | 800ms |
| **VNP Memory** | **2,000** | **$0.02** | **< 200ms** |

**Savings: 80% token cost reduction vs naive approach.**
