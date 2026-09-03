# TC-018: Provider System — Test Cases

**Test Design tham chiếu:** [TD-018](../designs/TD-018-provider-system.md)  
**Requirements tham chiếu:** [TR-018](../requirements/TR-018-provider-system.md)  
**Module:** Embedding Providers (none, xenova, openai), LLM Provider (anthropic)  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-018-001: EMBEDDING_PROVIDER=none → zero vector, không crash

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-001 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-018-PRV-001 |

**Điều kiện tiên quyết:** `EMBEDDING_PROVIDER = none`

**Dữ liệu đầu vào:** `embed("some text to embed")`

**Các bước thực hiện:**
1. Set `EMBEDDING_PROVIDER = none`
2. Gọi embedding provider `embed("some text")`
3. Kiểm tra return value

**Kết quả mong đợi:**
- Không throw exception
- Return type là `Float32Array`
- Tất cả values = `0`
- `length = 384` (default dimension for none provider)

---

## TC-018-002: AGENTMEMORY_AUTO_COMPRESS=false → không gọi LLM

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-002 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-018-PRV-009 |

**Điều kiện tiên quyết:** `AGENTMEMORY_AUTO_COMPRESS = false`

**Các bước thực hiện:**
1. Setup HTTP request spy/interceptor
2. Observe 5 hooks (các hooks có đủ content để LLM compress nếu enabled)
3. Đếm HTTP requests đến Anthropic API endpoint

**Kết quả mong đợi:**
- Số requests đến Anthropic API = 0
- Observations vẫn được ghi (với synthetic compression)

---

## TC-018-003: Embedding output là Float32Array với đúng dimension

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-003 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-018-PRV-005 |

**Điều kiện tiên quyết:** Provider đã được khởi tạo (bất kỳ provider nào)

**Dữ liệu đầu vào:** `embed("test text")`

**Kết quả mong đợi:**
- Return type: `Float32Array` (kiểm tra bằng `instanceof Float32Array`)
- `output.length = 384` (hoặc theo configured dimension của provider)

---

## TC-018-004: Embedding output được L2-normalized (unit length)

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-004 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-018-PRV-006 |

**Điều kiện tiên quyết:** Provider trả về non-zero embedding

**Tính toán:**
- L2 norm = `sqrt(sum(v[i]^2))`
- Expected: `|L2_norm - 1.0| < 1e-5`

**Kết quả mong đợi:** Vector được normalize về đơn vị (unit length ≈ 1.0)

---

## TC-018-005: Embedding fail (503) → fallback gracefully

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-018-PRV-004 |

**Điều kiện tiên quyết:**
- `EMBEDDING_PROVIDER = openai` (hoặc bất kỳ network provider)
- Mock HTTP server được setup để trả về HTTP 503

**Các bước thực hiện:**
1. Configure mock server: `/embeddings` endpoint → 503
2. Gọi `embed("test text")`
3. Kiểm tra response

**Kết quả mong đợi:**
- Không throw unhandled exception
- Trả về zero Float32Array (fallback) HOẶC raise Error rõ ràng
- Warning được log (không fail silently)

---

## TC-018-006: Unknown EMBEDDING_PROVIDER → informative error khi startup

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-006 |
| **Loại** | Unit |
| **Ưu tiên** | 🟡 P2 |
| **Requirement** | TR-018-PRV-008 |

**Điều kiện tiên quyết:** `EMBEDDING_PROVIDER = completely_unknown_xyz`

**Các bước thực hiện:**
1. Khởi tạo provider với `EMBEDDING_PROVIDER = "completely_unknown_xyz"`
2. Kiểm tra error

**Kết quả mong đợi:**
- Throw `Error` với message đề cập đến provider name
- Error message có thể chứa danh sách providers hợp lệ

---

## TC-018-007: Embedding deterministic — cùng input → cùng vector

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-007 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-018-PRV-007 |

**Điều kiện tiên quyết:** Provider có deterministic output (xenova, none)

**Dữ liệu đầu vào:** `embed("auth middleware implementation")`

**Các bước thực hiện:**
1. `v1 = embed("auth middleware implementation")`
2. `v2 = embed("auth middleware implementation")` (cùng input)
3. So sánh v1 và v2 element-by-element

**Kết quả mong đợi:** `v1[i] === v2[i]` cho mọi i

---

## TC-018-008: AGENTMEMORY_AUTO_COMPRESS=true → LLM được gọi

| Trường | Giá trị |
|---|---|
| **ID** | TC-018-008 |
| **Loại** | Integration (mock) |
| **Ưu tiên** | 🟡 P2 |
| **Requirement** | TR-018-PRV-010 |

**Điều kiện tiên quyết:**
- `AGENTMEMORY_AUTO_COMPRESS = true`
- Mock Anthropic API server trả về valid response

**Kết quả mong đợi:**
- Sau observe: POST request đến Anthropic `/messages` endpoint được ghi nhận

---

## Tổng kết TC-018

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-018-001 | none provider → zero vector | 🔴 P0 | Unit |
| TC-018-002 | AUTO_COMPRESS=false → 0 LLM calls | 🔴 P0 | Unit |
| TC-018-003 | Output Float32Array đúng dim | 🔴 P0 | Unit |
| TC-018-004 | Output L2-normalized | 🟠 P1 | Unit |
| TC-018-005 | 503 → fallback | 🟠 P1 | Integration |
| TC-018-006 | Unknown provider → error | 🟡 P2 | Unit |
| TC-018-007 | Deterministic output | 🟠 P1 | Unit |
| TC-018-008 | AUTO_COMPRESS=true → LLM called | 🟡 P2 | Integration |
