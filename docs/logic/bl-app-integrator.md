# Business Logic — App Integrator

> **Actor**: App Integrator  
> **Vai trò**: Developer tích hợp KG Service vào ứng dụng — ghi dữ liệu, đọc theo pattern, tìm kiếm ngữ nghĩa  
> **Phạm vi quyền**: Theo API key — chỉ thấy và ghi vào domain trong effective ontology

---

## Nghiệp vụ 1: Ghi Dữ liệu (Write)

### BL-AI-01: Ghi Node

**Mô tả**: Ứng dụng ghi một node (một đơn vị tri thức) vào domain.

**Điều kiện tiên quyết**:
- API key của app phải có quyền `write` trên domain
- `domain_id` phải thuộc effective ontology của app
- `node_type` phải đã được khai báo trong domain

**Business Rules**:
- **Required props** phải có đủ — thiếu bất kỳ field nào → 422 với danh sách lỗi cụ thể
- **Validation rules** của node type phải pass — ví dụ `end_date > start_date`
- **Cross-domain rel rules**: nếu domain khai báo rule bắt buộc, phải gửi kèm `bridge_{rel}_ids`
- Identity được lấy từ API key — **không truyền `tenant_id`/`app_id` trong body**
- Write trả về **202 Accepted** — data chưa chắc đã xuất hiện trong graph/vector ngay

**Luồng**:
```
Input: { domain_id, node_type, properties, bridge_*_ids? }
  → Validate: domain_id trong effective ontology?   → 422 nếu không
  → Validate: node_type tồn tại?                    → 422 nếu không
  → Validate: required_props đủ + type đúng?        → 422 nếu không
  → Validate: validation_rules pass?                → 422 nếu không
  → Validate: cross_domain_rules satisfied?         → 422 nếu không
  → INSERT vào PostgreSQL (source of truth)
  → Tạo outbox event → trigger async sync
Output: { node_id, status: "processing", sync_eta_ms: 800 }
```

**Sau write**:
- `status: "processing"` → data đang được sync sang graph/vector
- Dự kiến < 2s xuất hiện trong graph/vector
- Dùng `?mode=realtime` khi cần đọc lại ngay (xem BL-AI-07)

---

### BL-AI-02: Cập nhật Node

**Business Rules**:
- Chỉ update `properties` — không thể thay đổi `node_type` hoặc `domain_id`
- Chỉ có thể update node do app này sở hữu (hoặc được grant write)
- Partial update: chỉ gửi các fields cần thay đổi
- Update cũng trigger async sync → graph/vector cập nhật sau < 2s

---

### BL-AI-03: Xóa Node (Soft Delete)

**Business Rules**:
- Node **không bị xóa vật lý** — chỉ mark `is_deleted = true`
- Node đã xóa không xuất hiện trong kết quả read/search
- Node đã xóa vẫn có thể restore (bằng cách update `is_deleted = false` — admin operation)
- Chỉ owner app mới xóa được node của mình

---

### BL-AI-04: Ghi Relationship

**Business Rules**:
- `rel_type` phải đã khai báo trong domain ontology
- `from_node_id` và `to_node_id` phải tồn tại và không bị xóa
- Caller phải có quyền write trên domain của relationship
- Cross-domain relationship: cả hai đầu phải accessible

---

### BL-AI-05: Ingest Document

**Mô tả**: Gửi một file/document để hệ thống tự parse thành nodes.

**Business Rules**:
- Là async job — trả về `job_id` ngay lập tức
- Theo dõi trạng thái qua `GET /v1/kg/write/ingest/jobs/{job_id}`
- Domain config quyết định cách parse document thành nodes
- Job có thể partial success: một số nodes tạo thành công, một số fail

**Trạng thái job**:
```
queued → processing → completed
                    ↘ failed (partial or full)
```

---

## Nghiệp vụ 2: Đọc Dữ liệu (Read)

### BL-AI-06: Đọc theo Named Template

**Mô tả**: App đọc dữ liệu bằng cách gọi template đã được Tenant Admin đăng ký — không tự soạn query.

**Business Rules**:
- Template phải có `status = active` → 404 UNKNOWN_TEMPLATE nếu không
- Template phải thuộc domain trong effective ontology của app
- Params phải đúng với `param_schema` của template
- Kết quả tự động filter theo ACL — chỉ thấy data mà app có quyền xem
- Nếu domain có status config: kết quả tự filter bỏ nodes invalid status

**Luồng**:
```
Input: POST /v1/kg/read/template/{domain_id}/{template_name}
       { params: { field: value, ... } }
  → Load template từ domain_query_templates
  → Compile DSL → Cypher (với ACL tokens injected)
  → Execute trên Graph DB (timeout 3000ms)
  → Return filtered results
Output: { results: [...], query_time_ms: 42 }
```

**Khám phá template**:
```
GET /v1/kg/read/templates?domain_id={d}
→ Danh sách templates active với param_schema
```

---

### BL-AI-07: Đọc Node Trực tiếp (Single Node)

**Business Rules**:
- Trả về node theo ID + relationships
- `mode=realtime`: nếu graph projection chưa cập nhật → fallback đọc từ PostgreSQL (source of truth)
- `mode` mặc định (non-realtime): đọc từ graph projection
- Trả về 403 NO_READ_ACCESS nếu node không trong visible scope của app

**Khi nào dùng `mode=realtime`**:
- Vừa write xong và cần đọc lại ngay (< 2s sau write)
- Cần data consistent tuyệt đối (trả lời yêu cầu audit)

---

### BL-AI-08: Kiểm tra Projection Status

**Mô tả**: App muốn biết node đã được sync sang graph/vector chưa.

**Projection States**:
```
SYNCED       → graph version == pg version → đọc từ graph là đúng
IN_FLIGHT    → đang được sync, sẽ xong sớm
LAGGING      → bị trễ nhưng chưa stuck
STUCK        → không sync được → cần operator check
```

---

## Nghiệp vụ 3: Tìm kiếm (Search)

### BL-AI-09: Semantic Search

**Mô tả**: Tìm kiếm theo ngữ nghĩa — không cần exact match, dùng vector similarity.

**Business Rules**:
- Query text được chuyển thành vector → so sánh với vector trong Vector DB
- Kết quả tự động filter theo ACL — không thấy data không có quyền
- Nếu domain có status config: filter bỏ nodes invalid status
- `domain_ids` để giới hạn search trong domain cụ thể (tùy chọn)
- `top_k` kiểm soát số kết quả (mặc định 10)

**Luồng**:
```
Input: { query: "payment timeout errors", domain_ids: ["payment"], top_k: 5 }
  → embed(query) → vector
  → filter = { acl_visible_to: any_of(acl_tokens), is_deleted: false, domain_id: any_of(domain_ids) }
  → vector search với filter
  → [optional] rerank theo authority_score nếu domain khai báo
Output: { results: [{ node_id, score, content, domain_id, ... }], search_time_ms: 78 }
```

---

### BL-AI-10: Full-Text Search

**Business Rules**:
- Search theo text exact/prefix trong các fields được khai báo trong search profile
- Nhanh hơn semantic search, phù hợp lookup theo code/ID
- Filter ACL giống semantic search

---

### BL-AI-11: Hybrid Search

**Mô tả**: Kết hợp semantic search và full-text search với trọng số.

**Business Rules**:
- `semantic_weight` (0-1) kiểm soát tỷ lệ: ví dụ 0.7 = 70% semantic + 30% FTS
- Domain phải khai báo `search_profile` với `hybrid_search_enabled: true`
- Kết quả là merge và rerank từ cả hai nguồn

---

### BL-AI-12: RAG Search

**Mô tả**: Search để cung cấp context cho LLM — trả về structured context, citations, conflict notes.

**Business Rules**:
- Tự động kết hợp semantic search + graph traversal
- Trả về `answer_context` với `structured_data`, `citations`, `disclaimer`
- Luôn kèm disclaimer về tính tham khảo (không thay thế chuyên gia)

---

## Nghiệp vụ 4: Xác minh Identity & Access

### BL-AI-13: Kiểm tra Identity

**Mô tả**: App tự xác nhận mình đang được nhận diện đúng và thấy những domain nào.

```
GET /v1/access/resolve
→ { tenant_id, app_id, visible_domains: [...] }
```

**Khi nào cần dùng**:
- Sau khi tích hợp lần đầu — xác nhận key hoạt động đúng
- Debug: tại sao search không thấy data dù biết data có trong hệ thống
- Sau khi grant thay đổi — verify visibility đã cập nhật

---

## Tóm tắt Business Rules — App Integrator

| ID | Rule |
|:---:|:---|
| **BR-AI-01** | App chỉ write vào domain trong effective ontology (own + granted write) |
| **BR-AI-02** | Required props phải đủ — thiếu → 422 với chi tiết field nào thiếu |
| **BR-AI-03** | Cross-domain rel rules phải satisfied khi write — nếu domain khai báo |
| **BR-AI-04** | Write returns 202 — data chưa chắc visible ngay trong graph/vector |
| **BR-AI-05** | Read/search chỉ qua named template — không raw query |
| **BR-AI-06** | ACL filter tự động — không thể xem data ngoài effective ontology |
| **BR-AI-07** | `mode=realtime` khi cần đọc ngay sau write — fallback sang PostgreSQL |
| **BR-AI-08** | Soft delete — node không bị xóa vật lý |
| **BR-AI-09** | Identity từ API key — không truyền tenant_id/app_id trong body |
| **BR-AI-10** | 422 trả về danh sách lỗi cụ thể (field + issue) — không generic |
