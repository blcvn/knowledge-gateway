---
id: FEAT-KNW-004
title: Bifrost LLM Client Adapter
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement Bifrost HTTP client adapter implementing LLMClient port. Handles completion requests, response parsing, circuit breaker, retry, bulkhead, và token usage tracking.

## Scope

- `internal/adapter/llm/bifrost_client.go` — HTTP → Bifrost /v1/chat/completions
- `internal/adapter/llm/response_parser.go` — Extract JSON from LLM markdown responses
- Circuit breaker (gobreaker), retry 3x, bulkhead (semaphore for max concurrent)
- Token usage accumulation per model

### LLM Client Implementation

```go
type BifrostClient struct {
    httpClient *http.Client
    baseURL    string
    apiKey     string
    cb         *gobreaker.CircuitBreaker
    semaphore  chan struct{} // bulkhead
}

func (c *BifrostClient) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    // Acquire bulkhead semaphore
    // Execute with circuit breaker
    // POST to Bifrost /v1/chat/completions
    // Parse response, extract content
    // Track token usage
}
```

## Acceptance Criteria

- [ ] AC-1: LLM completion returns parsed response
- [ ] AC-2: Circuit breaker opens after 5 consecutive failures
- [ ] AC-3: Bulkhead limits concurrent requests to LLM_MAX_CONCURRENT
- [ ] AC-4: Retry on 429/503 with exponential backoff
- [ ] AC-5: Token usage tracked per (model, prompt_tokens, completion_tokens)
- [ ] AC-6: Response parser handles JSON in markdown code fences

## Test Requirements
- **Unit tests**: HTTP mock server, circuit breaker, parser
- **Minimum coverage**: 80%
