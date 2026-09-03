# TD-018: Provider System Test Design

**Liên kết Requirements:** [TR-018-provider-system.md](../requirements/TR-018-provider-system.md)  
**Source:** `references/agentmemory/src/providers/`  
**Test file:** `tests/agentmemory/specs/provider-system.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Provider system quản lý embedding backends và LLM providers.

**Providers:**
- **Embedding:** `xenova` (local, ONNX), `openai`, `voyage`, `none`
- **LLM:** `anthropic` (Claude), `none`

---

## 2. Chiến lược kiểm thử

- **None provider:** Test path không có embedding/LLM — hầu hết unit tests dùng path này
- **Real providers:** Test với mock HTTP server để tránh real API calls và costs
- **Provider selection:** Verify env var → provider mapping

---

## 3. Test Cases

### Group A: Embedding Provider Selection

#### TC-001 — `EMBEDDING_PROVIDER=none`: noop provider, embed trả về zero vector
**Requirement:** TR-018-PRV-001 | **Type:** unit | 🔴 P0

**Given:** `EMBEDDING_PROVIDER=none`  
**When:** `embed("some text")` gọi  
**Then:** Trả về zero Float32Array (384 dims), không throw

---

#### TC-002 — `EMBEDDING_PROVIDER=xenova`: load local model
**Requirement:** TR-018-PRV-002 | **Type:** integration | 🟡 P2

**Given:** `EMBEDDING_PROVIDER=xenova`, model files available locally  
**When:** Provider được initialized  
**Then:** Provider loaded thành công (không throw)

*Skip nếu model files không có trong CI*

---

#### TC-003 — `EMBEDDING_PROVIDER=openai`: call OpenAI API
**Type:** integration (với mock HTTP) | 🟠 P1

**Given:** Mock HTTP server tại `OPENAI_API_BASE`, trả về valid embedding response  
**When:** `embed("auth middleware")` gọi  
**Then:**
- HTTP POST đến `/embeddings` endpoint
- Trả về Float32Array từ response

---

#### TC-004 — Provider fallback: OpenAI fail → noop
**Requirement:** TR-018-PRV-004 | **Type:** integration | 🟠 P1

**Given:** `EMBEDDING_PROVIDER=openai`, mock server trả về 503  
**When:** `embed()` gọi  
**Then:**
- Không crash
- Trả về zero vector (fallback)
- Warning được log

---

#### TC-005 — Unknown provider → error sớm khi khởi tạo
**Type:** unit | 🟡 P2

**Given:** `EMBEDDING_PROVIDER=unknown_xyz`  
**When:** Provider được khởi tạo  
**Then:** Throw Error với message rõ ràng về provider không được hỗ trợ

---

### Group B: Embedding Output

#### TC-006 — Output là Float32Array với đúng dimension
**Requirement:** TR-018-PRV-005 | **Type:** unit | 🔴 P0

**Given:** Provider trả về valid embedding  
**When:** `embed("text")` gọi  
**Then:**
- Type là `Float32Array`
- Length = provider's configured dimension (ví dụ 384 cho all-MiniLM-L6)

---

#### TC-007 — Output vector được L2-normalized (unit length)
**Requirement:** TR-018-PRV-006 | **Type:** unit | 🟠 P1

**Given:** Provider trả về embedding  
**When:** `embed("text")` gọi  
**Then:** `|v| ≈ 1.0` (L2 norm, tolerance 1e-5)

---

#### TC-008 — Deterministic: cùng text → cùng vector
**Requirement:** TR-018-PRV-007 | **Type:** unit | 🟠 P1

**Given:** Deterministic test provider  
**When:** `embed("auth middleware")` gọi 2 lần  
**Then:** Cả 2 lần trả về identical vector

---

### Group C: LLM Provider

#### TC-009 — `AGENTMEMORY_AUTO_COMPRESS=false`: LLM không được gọi
**Requirement:** TR-018-PRV-009 | **Type:** unit | 🔴 P0

**Given:** `AGENTMEMORY_AUTO_COMPRESS=false`  
**When:** Observation được compressed  
**Then:** Không có HTTP request nào đến Anthropic API (spy verify 0 calls)

---

#### TC-010 — `AGENTMEMORY_AUTO_COMPRESS=true`: LLM được gọi
**Type:** integration (mock) | 🟡 P2

**Given:** Mock Anthropic API, `AGENTMEMORY_AUTO_COMPRESS=true`  
**When:** Observation được compressed  
**Then:** POST request đến `/messages` endpoint với correct model

---

## 4. Coverage Notes

- Real provider tests cần environment variables mà CI không có → mark as `@skip-ci`
- Focus coverage vào provider selection logic và fallback paths
- Noop provider: 100% coverage expected (simple code)
