# P5 — IDE Plugin User (AI Coding Assistant)

> **Vai trò:** Developer dùng AI coding assistant (Claude Code, Copilot, Codex) hàng ngày trong IDE.
> **Tần suất:** Hàng ngày — nhiều giờ/ngày.

---

## Pain Points

### PP-P5-01 — AI assistant quên toàn bộ project context sau mỗi session

**Mô tả:**
Mỗi lần mở IDE session mới, phải giải thích lại cho AI: tech stack, naming conventions, project structure, current task, constraints. Mất 5-10 phút "warm up" mỗi sáng.

**Ví dụ cụ thể:**
```
Thứ 2: Nói với AI "Chúng ta dùng snake_case cho Python, camelCase cho JS"
Thứ 3: AI lại suggest PascalCase → phải nhắc lại
Thứ 4: AI forget lại → phải nhắc lần 3
```

**Features giải quyết:**
- [F06] OpenViking Procedural Memory (VikingFS):
  - `ov_write_file`: Lưu conventions vào virtual filesystem
  - `ov_read_file`: AI đọc lại khi cần
  - Tiered Context L0/L1/L2: load project summary nhanh
- [F13] MCP Tools: `memory_store` với `type=procedural` — persist coding preferences
- [F05] Profile Memory: `type=preference` — lưu developer preferences vĩnh viễn

---

### PP-P5-02 — AI không tìm được code đã viết trước đây

**Mô tả:**
"Chúng ta đã có pattern này chưa?" — AI không biết. Developer phải tự grep, tự tìm. Khi codebase lớn, mất nhiều thời gian tìm patterns, utilities đã implement.

**Features giải quyết:**
- [F06] OpenViking Search:
  - `ov_grep`: grep với semantic understanding, không chỉ string matching
  - `ov_search`: semantic search trong indexed codebase
  - `ov_tree`: navigate directory structure
- [F13] MCP: `ov_list_dir`, `ov_grep`, `ov_tree` — AI tự search thay cho developer

---

### PP-P5-03 — Không có working memory cho tasks phức tạp

**Mô tả:**
Khi implement feature phức tạp spanning nhiều files, nhiều sessions, AI mất track của "đã làm gì, còn làm gì, dependency là gì". Mỗi session AI start từ zero.

**Features giải quyết:**
- [F06] OpenViking Session Management:
  - Working Memory: structured document `{title, state, goals, facts, errors}`
  - 2-phase commit: draft (sketch) → commit (permanent)
  - `ov_session_commit`: crystallize working session thành permanent knowledge
- [F11] Orchestration: Sketches (ephemeral drafts) → Crystals (permanent knowledge)

---

### PP-P5-04 — AI phải re-read toàn bộ files mỗi lần

**Mô tả:**
Mỗi context window, AI phải đọc lại files từ đầu. Với files lớn (500+ lines), tốn nhiều context window space. AI không có "memory về file này" từ session trước.

**Features giải quyết:**
- [F06] Tiered Context:
  - L0 (~100 tokens): "File này là service authentication middleware"
  - L1 (~2K tokens): core functions + interfaces
  - L2 (full): chi tiết khi cần
  - AI chỉ load L0/L1 để orientate, L2 chỉ khi cần edit

---

## Summary

| Pain | Giải pháp |
|---|---|
| Quên conventions mỗi session | VikingFS + MCP memory_store |
| Không tìm được code cũ | ov_grep + ov_search |
| Mất track task phức tạp | Working Memory + Session commit |
| Re-read files tốn context | Tiered Context L0/L1/L2 |
