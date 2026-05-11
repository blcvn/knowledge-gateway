---
id: TASK-KNW-007
title: Implement Prompt Registry
feature: FEAT-KNW-007
status: Done
---

## Objective
Thực thi implement prompt template registry cho 7 LLM extraction/resolution tasks dựa trên FEAT-KNW-007.

## Tasks
1. Tạo file `internal/adapter/llm/prompt_registry.go`:
   - Định nghĩa struct/method implement interface `PromptRegistry` (`Render`, `GetModel`, `List`).

2. Tạo 7 templates sử dụng Go `text/template` syntax:
   - `extract_entities` (gpt-4o, vars: Content, PreviousEpisodes, EntityTypes)
   - `resolve_entities` (gpt-4o-mini, vars: Extracted, Existing)
   - `extract_edges` (gpt-4o, vars: Content, Entities, PreviousEpisodes)
   - `resolve_edges` (gpt-4o-mini, vars: NewEdge, ExistingEdges)
   - `summarize_community` (gpt-4o-mini, vars: Members, Edges)
   - `classify_entity` (gpt-4o-mini, vars: Entity, Context)
   - `expand_summary` (gpt-4o-mini, vars: Entity, NewContext)

3. Error handling:
   - Trả về `ErrPromptNotFound` nếu template bị thiếu.

4. Unit Tests:
   - Viết render-only tests. Kiểm tra output string có nội suy biến đúng cấu trúc JSON không.
   - Đảm bảo coverage >= 90%.
