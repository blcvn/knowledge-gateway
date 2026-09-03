# P8 — Product Manager

> **Vai trò:** Quản lý sản phẩm AI có tương tác người dùng, cần data về user behavior.
> **Tần suất:** Hàng tuần.

---

## Pain Points

### PP-P8-01 — Không có structured data về user preferences từ conversations

**Mô tả:**
Conversations là goldmine của insight — user nói về pain points, preferences, goals. Nhưng tất cả chỉ là raw text không structured. PM không có cách nào analyze "50% users quan tâm đến performance, 30% quan tâm đến ease-of-use".

**Features giải quyết:**
- [F05] Memobase Structured Profiles:
  - `GET /v1/memobase/users/{uid}/profiles` → `{key, value, category, score}`
  - Categories: preference / fact / goal / habit
  - Aggregated across all users → product insights

---

### PP-P8-02 — Không biết AI memory đang hoạt động tốt hay không

**Mô tả:**
"Users có hài lòng với AI memory không?" — PM không có metrics. Không biết recall accuracy, không biết token cost, không biết latency distribution.

**Features giải quyết:**
- [F25] Observability: `GET /v1/console/observability/costs` — LLM cost tracking
- [F15] Console Dashboard: memory heatmap, throughput, error rates
- [F23] Pipeline Monitor: pipeline success/failure rates

---

### PP-P8-03 — Không track được feature usage

**Mô tả:**
"Có bao nhiêu users đang dùng Graphiti temporal memory vs Zep conversational?" — không có answer. Không biết feature nào đang được dùng để prioritize roadmap.

**Features giải quyết:**
- [F25] Observability: metrics breakdown per engine
- [F27] Organization SDK Manager: API usage per tenant, per endpoint
- Prometheus metrics: request count per route

---

## Summary

| Pain | Giải pháp |
|---|---|
| No structured user insights | Memobase profiles (category/score) |
| No AI memory quality metrics | Observability + Dashboard |
| No feature usage tracking | Per-engine metrics + API usage |
