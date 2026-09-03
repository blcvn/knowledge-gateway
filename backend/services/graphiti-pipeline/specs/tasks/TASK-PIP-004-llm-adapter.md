---
id: TASK-PIP-004
title: Implement LLM Adapter
feature: FEAT-PIP-004
status: Done
---

## Objective
Thực thi implement LLM adapter layer (Bifrost client + prompt registry) dựa trên FEAT-PIP-004.

## Tasks
1. Tạo file `internal/adapter/llm/bifrost_client.go`
   - Implement HTTP client for `LLMClient` port.
   - Tích hợp Circuit breaker (open sau 5 failures).
   - Tích hợp Bulkhead (giới hạn `LLM_MAX_CONCURRENT`).
   - Tích hợp Retry logic (3x exponential backoff cho 429, 503).
   - Track token usage via Prometheus metrics.

2. Tạo file `internal/adapter/llm/prompt_registry.go`
   - Implement template loading và variable interpolation.
   - Load các templates: `extract_entities`, `resolve_entities`, `extract_edges`, `resolve_edges`, `summarize_community`.

3. Tạo file `internal/adapter/llm/response_parser.go`
   - Implement JSON extraction từ LLM responses.
   - Handle malformed JSON gracefully.

4. Tạo file `internal/adapter/embedder/bifrost_embedder.go`
   - Implement Embedding generation qua Bifrost.

5. Unit và Integration Tests
   - Viết unit tests cho prompt rendering, JSON parsing, circuit breaker state, retry logic.
   - Viết integration tests cho Bifrost HTTP roundtrip với mock server.
   - Đảm bảo coverage >= 80%.
