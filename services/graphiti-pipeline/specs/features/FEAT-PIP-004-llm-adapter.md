---
id: FEAT-PIP-004
title: LLM Adapter — Bifrost Client + Prompt Registry
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement LLM adapter layer gồm Bifrost HTTP client, prompt template registry, và LLM response parser cho entity/edge extraction, resolution, và community summarization.

## Scope

### In Scope
- `internal/adapter/llm/bifrost_client.go` — HTTP client implementing LLMClient port
- `internal/adapter/llm/prompt_registry.go` — Template loading and variable interpolation
- `internal/adapter/llm/response_parser.go` — JSON extraction from LLM responses
- `internal/adapter/embedder/bifrost_embedder.go` — Embedding generation via Bifrost
- Prompt templates: extract_entities, resolve_entities, extract_edges, resolve_edges, summarize_community
- Circuit breaker + retry + bulkhead for LLM calls
- Token usage tracking

### Out of Scope
- Direct OpenAI/Anthropic SDK calls (all via Bifrost)

## Acceptance Criteria

- [ ] AC-1: LLM completion via Bifrost returns parsed JSON entities
- [ ] AC-2: Circuit breaker opens after 5 consecutive LLM failures
- [ ] AC-3: Bulkhead limits concurrent LLM requests to `LLM_MAX_CONCURRENT`
- [ ] AC-4: Retry 3x with exponential backoff on transient errors (429, 503)
- [ ] AC-5: Token usage tracked per model and reported via Prometheus metric
- [ ] AC-6: Prompt templates support variable interpolation ({{.Content}}, {{.Entities}})
- [ ] AC-7: LLM response parser handles malformed JSON gracefully (fallback re-extraction)

## Test Requirements
- **Unit tests**: Prompt rendering, JSON parsing, circuit breaker state, retry logic
- **Integration tests**: Bifrost HTTP roundtrip with mock server
- **Minimum coverage**: 80%
