# Design: CodeGraph Ontology Bootstrap

## Overview

Change này sinh ra và thực thi một bootstrap script để freeze contract mà mọi bridge/tooling sẽ phụ thuộc vào sau đó.

## Approach: Script-based bootstrap

Thay vì gọi API trực tiếp từng bước, toàn bộ quá trình bootstrap được thực hiện qua 3 pha:

### Phase 1 — Sinh script

Script là một shell script (hoặc tương đương) chứa tuần tự các `curl` call (hoặc SDK call) theo đúng thứ tự dependency:

1. Create domain `code-graph`
2. Register node type schemas (`Function`, `Method`, `Struct`, `Interface`, `Package`, `File`)
3. Register relationship type schemas (`CALLS`, `IMPLEMENTS`, `CONTAINS`, `REFERENCES`, `IMPORTS`)
4. Register query strategy `code-graph-traversal`
5. Upsert search profile cho domain `code-graph`
6. Create và activate query templates (`code_callers`, `code_callees`, `code_impact`, `code_implements`)

Script phải **idempotent**: chạy lại không gây lỗi nếu entity đã tồn tại (dùng upsert hoặc check-before-create).

### Phase 2 — Chạy script

Thực thi script trên môi trường target. Output được capture để làm evidence.

### Phase 3 — Verify kết quả

Sau khi script chạy xong, đọc lại từng entity qua read API và đối chiếu với spec:

- Domain `code-graph` tồn tại
- Tất cả node/relationship type schemas đúng
- Search profile có đúng fields
- Query templates ở trạng thái active

## Domain Model

### Node types

- `Function`
- `Method`
- `Struct`
- `Interface`
- `Package`
- `File`

### Relationship types

- `CALLS`
- `IMPLEMENTS`
- `CONTAINS`
- `REFERENCES`
- `IMPORTS`

## Key Decisions

### 1. Script là artifact chính

Script được commit vào repo (e.g. `examples/codegraph/bootstrap-codegraph-ontology.sh`) để có thể tái dùng khi reset môi trường.

### 2. SearchProfile chỉ dùng field có schema support

Baseline field set phải được xác thực với node type schemas trước khi upsert profile. Script chạy register schemas trước khi upsert search profile.

### 3. Traversal baseline dùng query templates

Persistent callers/callees/impact/implements đi qua:

- `POST /v1/kg/read/template/{domain_id}/{template_name}`

### 4. Query strategy được tách khỏi template definitions

`code-graph-traversal` là strategy key dùng lại cho nhiều template.
