# Proposal: Compose CodeGraph Runtime Stack

## Problem

`kg-service` đã có Compose manifests, runtime profiles, và bridge tooling cho CodeGraph, nhưng chưa có
một local stack chuẩn để:

- chạy `kg-service` với đúng backend `Postgres + Memgraph + Qdrant`
- cấp cấu hình embedding HTTP để semantic search cho domain `code-graph`
- chạy trọn một workflow bootstrap/sync/verify mà không phải ghép biến môi trường và lệnh thủ công từ
  nhiều tài liệu cũ

Hiện trạng còn lệch ở vài điểm:

- Compose mặc định vẫn boot `Redis` và profile tổng quát, chưa chốt một path rõ ràng cho `qdrant-memgraph`
- docs Compose vẫn mô tả bootstrap runtime cũ thay vì một flow end-to-end cho CodeGraph
- thông tin embedding trong `tests/llm/embedding-vnp.txt` chưa được chuyển thành contract cấu hình vận hành an
  toàn cho local test
- thiếu một script repo-owned để:
  - khởi động stack
  - tạo tenant/app dành cho CodeGraph nếu chưa có
  - bootstrap domain/ontology `code-graph`
  - upsert dữ liệu CodeGraph vào graph/vector backends của `kg-service`
  - verify lại bootstrap, get/list, search, và template/index behavior

## Proposed Solution

Chuẩn hóa một Compose validation path và một script orchestration dành cho CodeGraph:

1. **Compose stack chuẩn** — một entrypoint/manifest repo-owned khởi động `kg-service` với Postgres
   tương thích migration `pgvector`, Memgraph, và Qdrant cho runtime profile `qdrant-memgraph`.
2. **Embedding HTTP config** — tài liệu hóa và pass-through bộ biến môi trường embedding HTTP dựa trên
   contract trong `tests/llm/embedding-vnp.txt`, nhưng không commit API key thực.
3. **Full CodeGraph bootstrap script** — thêm một script repo-owned chạy đủ các bước:
   - boot Compose stack
   - bootstrap tenant, app, domain, và ontology `code-graph`
   - verify bootstrap result
   - upsert CodeGraph KG data vào `kg-service`
   - verify get/list, search, và template-backed query
4. **Safe rerun behavior** — script cho phép skip hoặc detect các bước đã init ở lần chạy đầu để những
   lần rerun tiếp theo không fail chỉ vì resource đã tồn tại.

## Scope

### In scope

- Docker Compose path cho stack `Postgres + Memgraph + Qdrant + kg-service`
- env contract cho HTTP embedding provider dùng trong CodeGraph validation
- script orchestration end-to-end cho local CodeGraph runtime test
- semantics rerun/idempotent cho bootstrap và sync flow

### Out of scope

- thay đổi API surface của `kg-service`
- thay nhà cung cấp embedding hoặc secret management production
- redesign bridge mapping hoặc ontology schema `code-graph`

## Success Criteria

- Có một Compose path duy nhất, repeatable, boot được stack phục vụ CodeGraph validation
- Operator biết chính xác cần set biến nào để bật HTTP embedding provider
- Có thể chạy một script duy nhất để boot, bootstrap, upsert, và verify domain `code-graph`
- Script có thể rerun an toàn sau lần chạy đầu bằng cách skip hoặc reuse các bước init đã hoàn tất
