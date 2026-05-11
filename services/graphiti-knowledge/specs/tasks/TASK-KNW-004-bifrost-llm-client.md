---
id: TASK-KNW-004
title: Implement Bifrost LLM Client Adapter
feature: FEAT-KNW-004
status: Done
---

## Objective
Thực thi implement Bifrost HTTP client adapter dựa trên FEAT-KNW-004.

## Tasks
1. Tạo file `internal/adapter/llm/bifrost_client.go`:
   - Implement `LLMClient` port.
   - Gửi POST request tới Bifrost `/v1/chat/completions`.
   - Track token usage (prompt_tokens, completion_tokens, model).

2. Implement Resilience Patterns:
   - Circuit breaker: dùng `gobreaker` (mở sau 5 lỗi liên tiếp).
   - Bulkhead: dùng semaphore channel (giới hạn LLM_MAX_CONCURRENT).
   - Retry: exponential backoff 3x trên lỗi 429/503.

3. Tạo file `internal/adapter/llm/response_parser.go`:
   - Implement parser để extract JSON từ markdown responses (markdown code fences).

4. Unit Tests:
   - Dùng mock HTTP server.
   - Test circuit breaker, parser.
   - Đảm bảo coverage >= 80%.
