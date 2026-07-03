# Design: Compose CodeGraph Runtime Stack

## Overview

Change này đóng gói một local validation path dành riêng cho CodeGraph trên `kg-service`. Mục tiêu là
biến bộ ba backend `Postgres + Memgraph + Qdrant` và HTTP embedding config thành một contract triển
khai rõ ràng, có thể lặp lại, và đủ để operator chạy từ khởi động stack đến verify dữ liệu đã sync.

## Current Constraints

### 1. Migrations vẫn cần Postgres có `vector` extension

`kg-service` hiện có migration `000008_kg_vector_documents.up.sql` với `CREATE EXTENSION IF NOT EXISTS vector`.
Vì vậy Compose stack cho CodeGraph validation vẫn phải dùng image Postgres tương thích `pgvector`,
dù runtime vector adapter được chốt là `qdrant`.

### 2. Runtime profile phù hợp đã tồn tại nhưng chưa có path rõ ràng

`scripts/runtime-profile.sh` đã hỗ trợ `KG_RUNTIME_PROFILE=qdrant-memgraph` với:

- `GRAPH_ADAPTER=memgraph`
- `KG_GRAPH_ENDPOINT=bolt://memgraph:7687`
- `VECTOR_ADAPTER=qdrant`
- `KG_VECTOR_ENDPOINT=http://qdrant:6333`
- `FTS_ADAPTER=postgres`

Change này tận dụng profile sẵn có thay vì tạo profile mới.

### 3. Embedding config phải bám contract HTTP hiện tại

`internal/config/load.go` đã nhận:

- `EMBEDDING_PROVIDER`
- `EMBEDDING_URL`
- `EMBEDDING_MODEL`
- `EMBEDDING_API_KEY`

Tài liệu trong `tests/llm/embedding-vnp.txt` cung cấp giá trị tham chiếu cho URL/model của VNPAY
embedding gateway. API key trong file test phải được xem là local secret input, không được copy nguyên
vào manifest, docs, hay file mẫu commit vào repo.

## Proposed Shape

### Compose runtime path

Compose validation path phải boot các service sau:

- Postgres tương thích `pgvector` để chạy toàn bộ migrations
- Memgraph cho graph adapter
- Qdrant cho vector adapter
- `kg-service`
- one-shot migration/init containers khi cần cho schema và vector collection

`kg-service` phải nhận trực tiếp:

- `KG_RUNTIME_PROFILE=qdrant-memgraph`
- `GRAPH_ADAPTER=memgraph`
- `KG_GRAPH_ENDPOINT=bolt://memgraph:7687`
- `VECTOR_ADAPTER=qdrant`
- `KG_VECTOR_ENDPOINT=http://qdrant:6333`
- `FTS_ADAPTER=postgres`
- `EMBEDDING_PROVIDER=http`
- `EMBEDDING_URL`, `EMBEDDING_MODEL`, `EMBEDDING_API_KEY` từ local env hoặc env file ngoài repo

### Operator config surface

Repo nên cung cấp một config pattern rõ ràng cho CodeGraph validation:

- file mẫu hoặc docs liệt kê đủ biến bắt buộc
- placeholder cho `EMBEDDING_API_KEY`
- ghi rõ `tests/llm/embedding-vnp.txt` là nguồn tham chiếu cho endpoint/model của local test, không phải
  nơi để lấy secret commit vào repo

### Validation flow

Luồng test CodeGraph cần đi theo thứ tự cố định trong một script orchestration:

1. boot Compose stack
2. bootstrap tenant và app dành cho CodeGraph nếu chưa có
3. bootstrap domain/ontology `code-graph` nếu chưa có
4. verify bootstrap result qua health/read/list/template checks
5. chạy sync bridge để upsert CodeGraph KG data vào `kg-service`
6. xác nhận get/list, semantic search, và template query trả kết quả trên domain `code-graph`

Validation script/docs nên fail sớm nếu thiếu `EMBEDDING_URL`, `EMBEDDING_MODEL`, hoặc
`EMBEDDING_API_KEY` khi bật `EMBEDDING_PROVIDER=http`.

### Rerun and skip behavior

Vì workflow này sẽ được chạy nhiều lần trên cùng local stack, script cần có semantics rõ ràng cho rerun:

- bước Compose start chỉ cần đảm bảo stack đang chạy, không phụ thuộc việc đó là lần đầu hay rerun
- bước tạo tenant/app có thể skip nếu resource đã tồn tại hoặc nếu operator đã cung cấp sẵn API key để dùng lại
- bước bootstrap ontology phải theo hướng create-or-verify thay vì fail cứng khi domain/schema/template đã tồn tại
- bước sync phải là upsert để dữ liệu CodeGraph được refresh trên graph/vector backends thay vì giả định chỉ insert một lần
- bước verify phải chạy được sau cả first run lẫn rerun

Script có thể hiện thực semantics này bằng cờ explicit như `--skip-compose`, `--skip-bootstrap`, `--skip-sync`,
hoặc bằng auto-detect resource tồn tại rồi reuse. Điều quan trọng là operator không phải xóa stack hay reset dữ
liệu chỉ để chạy lại flow.

## Key Decisions

### 1. Reuse `qdrant-memgraph` instead of inventing a new runtime profile

Điều này giữ implementation bám config surface hiện tại và tránh nhân đôi matrix backend.

### 2. Keep Postgres `pgvector`-compatible even when Qdrant is active

Mục tiêu là tương thích migration và bootstrap hiện có; Qdrant là vector runtime backend, không thay thế
nhu cầu extension ở migration layer.

### 3. Treat embedding credentials as operator-supplied secrets

Repo chỉ nên commit placeholder/example an toàn. Dữ liệu nhạy cảm trong `tests/llm/embedding-vnp.txt`
phải được chuyển thành hướng dẫn set env cục bộ, không sao chép verbatim vào artifact triển khai.

### 4. Use upsert semantics for CodeGraph sync

Vì local CodeGraph index có thể được sync nhiều lần trong quá trình phát triển, flow này phải mô tả bước cập nhật
KG data là upsert vào `kg-service` thay vì insert-only. Điều này áp dụng cho cả dữ liệu phục vụ graph traversal và
semantic/vector search verification.
