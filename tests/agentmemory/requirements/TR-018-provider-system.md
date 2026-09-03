# TR-018: Provider System Test Requirements

**Module:** LLM & Embedding Providers  
**Nguồn:** SRS §4.2, TDD §8.1-8.3, Architecture §5.2, URD §3.7  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-018-PRV-001 — Provider auto-detection: priority order
🔴 P0 | `[UNIT]` | **TDD §8.1**

**Given:** Env vars có `OPENAI_API_KEY` và `ANTHROPIC_API_KEY` đều set  
**When:** `detectProvider(env)` chạy  
**Then:** OpenAI được chọn (higher priority)

**Priority order:**
1. OpenAI (`OPENAI_API_KEY`)
2. MiniMax (`MINIMAX_API_KEY`)
3. Anthropic (`ANTHROPIC_API_KEY`)
4. Gemini (`GEMINI_API_KEY` / `GOOGLE_API_KEY`)
5. OpenRouter (`OPENROUTER_API_KEY`)
6. noop (default, no LLM)

**Traceability:** TDD §8.1, UR-028

---

## TR-018-PRV-002 — Provider fallback: noop khi không có API key
🔴 P0 | `[UNIT]` | **UR-029**

**Given:** Không có API key nào set  
**When:** Provider detection  
**Then:** `{provider: "noop", model: "noop"}` được trả về

**Traceability:** UR-029, TDD §8.1

---

## TR-018-PRV-003 — Noop provider: compress trả về empty/synthetic
🔴 P0 | `[UNIT]`

**Given:** noop provider được active  
**When:** `provider.compress(systemPrompt, userPrompt)` được gọi  
**Then:** Trả về empty string hoặc minimal response, không throw

**Traceability:** TDD §8.1

---

## TR-018-PRV-004 — Fallback chain: try primary, then fallbacks
🔴 P0 | `[UNIT]` | **TDD §8.1**

**Given:** Primary provider throws error, fallback1 throws, fallback2 succeeds  
**When:** `FallbackProvider.compress()` được gọi  
**Then:**
- Primary tried → fails
- fallback1 tried → fails
- fallback2 tried → succeeds
- Result từ fallback2 được trả về

**Traceability:** TDD §8.1

---

## TR-018-PRV-005 — Fallback chain: all fail → throw last error
🟠 P1 | `[UNIT]`

**Given:** Tất cả providers fail  
**When:** `FallbackProvider.compress()`  
**Then:** Last error được thrown

**Traceability:** TDD §8.1

---

## TR-018-PRV-006 — Circuit breaker: open sau threshold failures
🔴 P0 | `[UNIT]` | **FR-DIAG-003**

**Given:** CircuitBreaker với threshold=3  
**When:** 3 consecutive failures  
**Then:** `state = "open"` — subsequent calls throw immediately

**Traceability:** FR-DIAG-003, TDD §8.3

---

## TR-018-PRV-007 — Circuit breaker: cooldown period
🔴 P0 | `[UNIT]`

**Given:** Circuit is "open", `cooldownMs = 30000`  
**When:** Call attempted < 30s after opening  
**Then:** Immediate throw "Circuit breaker is open"

**Traceability:** TDD §8.3

---

## TR-018-PRV-008 — Circuit breaker: half-open probe
🟠 P1 | `[UNIT]`

**Given:** Circuit "open", cooldown elapsed  
**When:** Call attempted  
**Then:**
- State → "half-open"
- Single probe call executed
- IF success → state = "closed", failures = 0
- IF fail → state = "open" again

**Traceability:** TDD §8.3, Architecture §11.3

---

## TR-018-PRV-009 — Embedding provider: auto selection
🟠 P1 | `[UNIT]`

**Given:** `EMBEDDING_PROVIDER=auto`  
**When:** `createEmbeddingProvider()` chạy  
**Then:** Provider được select dựa trên available API keys (Gemini > OpenAI > Voyage > Cohere > local)

**Traceability:** TDD §8.2, Architecture §5.2

---

## TR-018-PRV-010 — Local embedding: 384 dimensions
🔴 P0 | `[INT]`

**Given:** `EMBEDDING_PROVIDER=local`  
**When:** `localProvider.embed("text")`  
**Then:** Float32Array với length=384 trả về

**Traceability:** TDD §8.2

---

## TR-018-PRV-011 — Local embedding: lazy model load
🟠 P1 | `[INT]`

**Given:** LocalEmbeddingProvider mới khởi tạo  
**When:** `embed()` được gọi lần đầu  
**Then:**
- ONNX model được load (all-MiniLM-L6-v2)
- Subsequent calls reuse loaded pipeline (không reload)

**Traceability:** TDD §8.2

---

## TR-018-PRV-012 — Embedding: normalize output
🟠 P1 | `[UNIT]`

**Given:** Any embedding provider  
**When:** `embed("text")` được gọi  
**Then:** Output vector normalized: L2 norm ≈ 1.0

**Traceability:** TDD §8.2

---

## TR-018-PRV-013 — Provider: Anthropic API call format
🟡 P2 | `[UNIT]`

**Given:** `ANTHROPIC_API_KEY=sk-ant-xxx`  
**When:** `provider.compress(system, user)` được gọi  
**Then:** Request format đúng Anthropic Messages API

**Traceability:** TDD §8.1

---

## TR-018-PRV-014 — Provider detection: AGENTMEMORY_ALLOW_AGENT_SDK
🟡 P2 | `[UNIT]`

**Given:** `AGENTMEMORY_ALLOW_AGENT_SDK=true`  
**When:** Không có API key nào khác  
**Then:** `{provider: "agent-sdk"}` được chọn

**Traceability:** TDD §8.1
