# ADR-012 — Bifrost làm LLM Multi-Provider Router

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-02 |
| **Deciders** | Platform Team |
| **Feature** | F05 (Memobase YOLO), F12 (Consolidation), tất cả LLM-using features |

---

## Context

VNP Memory cần gọi LLM cho nhiều tasks: extraction, classification, summarization, consolidation. Vấn đề:
- Phụ thuộc 1 provider (e.g., OpenAI) = single point of failure
- Giá OpenAI cao → cần option dùng cheaper providers cho non-critical tasks
- Muốn support self-hosted models (Ollama) cho privacy-sensitive tenants
- Provider rate limits → cần fallback

---

## Decision

**Route tất cả LLM calls qua Bifrost — multi-provider LLM gateway.**

```go
// All LLM calls via Bifrost HTTP API (OpenAI compatible)
type BifrostClient struct {
    baseURL string  // http://bifrost:8090
    apiKey  string
}

func (c *BifrostClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    // Bifrost routes based on model name + fallback config
    // bifrost.yaml config:
    // providers:
    //   - openai (primary)
    //   - anthropic (fallback)
    //   - google (fallback)
    //   - ollama (self-hosted, privacy mode)
    ...
}
```

**Provider selection strategy:**
```yaml
# Per-task model routing (bifrost.yaml)
tasks:
  extraction:     gpt-4o-mini      # cheap, fast
  classification: gpt-4o-mini      # cheap, fast
  summarization:  claude-3-haiku   # good quality/cost ratio
  consolidation:  gpt-4o           # high quality needed
  embeddings:     text-embedding-3-small  # openai embeddings
```

---

## Consequences

**Positive:**
- **No vendor lock-in:** Switch provider bằng config change, không thay code
- **Fallback:** Bifrost retry với provider khác nếu primary fails
- **Cost optimization:** Per-task model selection (cheap cho simple tasks)
- **Self-hosted option:** Ollama cho privacy-sensitive enterprise tenants
- **Rate limit management:** Bifrost handles provider rate limits + retry

**Negative:**
- Thêm 1 hop latency (Bifrost proxy ~5-10ms)
- Bifrost là external dependency (cần self-host hoặc managed)
- Model capabilities differ across providers (prompt engineering per provider)

**Mitigations:**
- Bifrost runs as sidecar trong Docker Compose
- `VNP_MEMORY_BIFROST_URL` configurable
- Circuit breaker: fallback trực tiếp đến provider nếu Bifrost down

---

## Alternatives Considered

### A1 — Direct OpenAI SDK
- **Rejected:** Vendor lock-in; không flexible; separate retry logic cần implement

### A2 — LangChain/LiteLLM
- **Rejected:** Python dependency trong Go codebase; complexity; LiteLLM cũng HTTP proxy như Bifrost nhưng less Go-native

### A3 — Build custom router
- **Rejected:** Duplicates Bifrost functionality; maintenance burden; Bifrost already production-tested
