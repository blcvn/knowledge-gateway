# Proposal: CodeGraph Ontology Bootstrap

## Problem

`kg-service` chưa có domain ontology cho source-code symbols. Không có schema rõ ràng cho node
types, relationship types, search profile, hay query templates để bridge/sync dùng ổn định.

## Proposed Solution

Sinh ra một script bootstrap, chạy script đó, và verify kết quả:

1. **Sinh script** — tạo file script chứa toàn bộ API calls để khởi tạo domain `code-graph`:
   - domain metadata
   - node type schemas
   - relationship type schemas
   - search profile
   - query strategy
   - query templates cho code traversal

2. **Chạy script** — thực thi script trên môi trường target và capture output.

3. **Verify kết quả** — đọc lại từng entity vừa tạo và xác nhận đúng với spec.

## Scope

### In scope

- Script sinh ra tất cả `code-graph` ontology entities
- Verify domain, schemas, search profile, query templates sau khi chạy script

### Out of scope

- CodeGraph local bootstrap
- sync bridge implementation
- MCP tools
- platform extension endpoints

## Success Criteria

- Script sinh ra đúng và đầy đủ (không lỗi cú pháp, có thể chạy lại idempotent)
- Script chạy thành công, mọi API call trả về status hợp lệ
- Verify xác nhận domain `code-graph` tồn tại với schemas, search profile, và query templates khớp spec
