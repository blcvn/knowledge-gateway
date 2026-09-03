# P6 — AI Framework Integrator

> **Vai trò:** Xây dựng integration giữa AI frameworks (LangChain, CrewAI, AutoGen, Mastra) và memory systems.
> **Tần suất:** Theo dự án (1-2 lần/tháng).

---

## Pain Points

### PP-P6-01 — Mỗi memory system có API khác nhau — không có standard

**Mô tả:**
LangChain Memory interface, CrewAI memory, AutoGen memory — mỗi framework implement memory theo cách riêng. Khi muốn đổi memory backend, phải rewrite toàn bộ integration code.

**Features giải quyết:**
- [F01] Unified Memory API: standard REST interface (`/v1/memory/*`)
- [F13] MCP Server: Model Context Protocol — standard được Anthropic define, ngày càng nhiều frameworks adopt
- JSON-RPC 2.0: phổ biến, có library cho mọi ngôn ngữ

---

### PP-P6-02 — Không có SDK — phải tự wrap HTTP calls

**Mô tả:**
Không có official SDK → phải tự viết wrapper code: retry logic, error handling, rate limiting, type safety. Mỗi integrator viết lại từ đầu.

**Features giải quyết:**
- [F27] Organization & SDK Manager — chuẩn bị SDK foundation
- REST API với clear OpenAPI schema → auto-generate client SDKs
- MCP protocol → Claude Code, AutoGen đã có native MCP support

---

### PP-P6-03 — Context injection phức tạp, phải tự implement

**Mô tả:**
Trước mỗi LLM call, phải: query memory → filter relevant → rank → truncate to token budget → format → inject vào prompt. Đây là boilerplate code mọi project phải viết.

**Features giải quyết:**
- [F13] Context Injection:
  - Pre-call hook tự động inject relevant memory
  - Configurable token budget
  - Sources: Memobase profile + OpenViking tiered + Supermemory KG
  - Agent scoping: isolated / shared / project

---

## Summary

| Pain | Giải pháp |
|---|---|
| No standard API | Unified REST API + MCP protocol |
| No SDK | OpenAPI schema + SDK Manager |
| Manual context injection | Auto context injection với token budget |
