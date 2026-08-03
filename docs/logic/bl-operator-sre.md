# Business Logic — Operator / SRE

> **Actor**: Operator / SRE  
> **Vai trò**: Triển khai, vận hành, giám sát, và xử lý sự cố cho KG Service  
> **Phạm vi quyền**: Infrastructure level — không trực tiếp thao tác data nghiệp vụ

---

## Nghiệp vụ 1: Triển khai Service

### BL-OPS-01: Chọn Runtime Profile

**Mô tả**: KG Service hỗ trợ nhiều backend combination — Operator chọn profile phù hợp với môi trường.

**Business Rules**:
- Profile được set qua `KG_RUNTIME_PROFILE` environment variable
- Profile phải khớp với backend thực tế đang chạy — mismatch gây startup error
- **Không có silent fallback**: nếu profile yêu cầu Qdrant nhưng Qdrant không chạy → service không tự chuyển về memory
- Memory profile chỉ cho dev/test — data mất khi restart

**Profiles**:

| Profile | Graph Backend | Vector Backend | Use case |
|:---|:---:|:---:|:---|
| (không set) | Memory | Memory | Local dev, unit test |
| `pgvector-memgraph` | Memgraph | pgvector | Lightweight staging |
| `qdrant-memgraph` | Memgraph | Qdrant | Production (standard) |
| `qdrant-nebula` | Nebula | Qdrant | High-scale graph |

**Operator checklist trước khi chọn profile**:
1. Backend services đã chạy và healthy?
2. Environment variables phù hợp đã set?
3. PostgreSQL migrations đã chạy?

---

### BL-OPS-02: Chạy Database Migrations

**Business Rules**:
- Migration phải chạy **trước** khi start API server lần đầu
- Migration là idempotent — chạy lại không gây lỗi nếu đã apply
- Thứ tự migration có thể phụ thuộc nhau — không skip
- Rollback (`.down.sql`) phải test trước khi dùng trong production

**15 migrations hiện tại**:
```
000001: extensions (uuid-ossp, etc.)
000002: identity & access tables
000003: ontology tables
000004: KG data tables + RLS
000005: audit log partitioning
000006: test fixtures
000007: FTS vector columns
000008: vector documents table
000009: projection versions
000010–011: UUID backfill + FK optimization
000012: graph versioning
000013: graph scope leases
000014–015: re-applied optimizations
```

---

### BL-OPS-03: Startup Ordering

**Business Rules**:
- PostgreSQL phải healthy trước khi API server start — không auto-retry vô hạn
- Redis phải reachable (dùng cho auth cache và rate limit)
- Graph/Vector backend phải reachable theo profile đã chọn
- Nếu backend không reachable → startup fail với error rõ ràng (không silent)

**Docker Compose startup order**:
```
postgres (health check: pg_isready)
   ↓
migrate (one-shot job, exit 0 khi xong)
   ↓
redis + graph_backend + vector_backend (parallel)
   ↓
kg-service (depends_on: migrate, redis, graph, vector)
```

---

## Nghiệp vụ 2: Validation và Smoke Testing

### BL-OPS-04: Chạy Integration Validation

**Mô tả**: Sau khi deploy, Operator chạy validation script để xác nhận service hoạt động đúng end-to-end.

**Business Rules**:
- Validation script phải tương ứng với deployment profile đang dùng
- Script chạy deterministic — cùng input luôn cho cùng kết quả
- Script exit 0 = pass, exit non-zero = fail (CI/CD compatible)
- Validation không xóa data production — dùng isolated test fixtures

**Checklist validation đầy đủ**:
1. ✅ Health check (`GET /healthz`) → status: ok
2. ✅ Identity resolution (`GET /v1/access/resolve`) → tenant/app đúng
3. ✅ Write node → 202 Accepted
4. ✅ Wait for projection (< 2s)
5. ✅ Read template → kết quả đúng
6. ✅ Semantic search → kết quả khớp với data đã write
7. ✅ ACL isolation → tenant A không thấy tenant B's data
8. ✅ Reconciliation check → drift = 0

---

### BL-OPS-05: Health Check

**Mô tả**: Kiểm tra nhanh service và dependencies.

**Business Rules**:
- `GET /healthz` là **public endpoint** — không cần authentication
- Trả về status của: service, PostgreSQL, Redis
- Không trả về data nghiệp vụ
- Load balancer dùng endpoint này để routing decisions

**Response**:
```json
{
  "status": "ok",
  "components": {
    "postgres": "ok",
    "redis": "ok"
  },
  "version": "1.0.0"
}
```

---

## Nghiệp vụ 3: Giám sát Consistency

### BL-OPS-06: Kiểm tra Integrity / Drift

**Mô tả**: Operator kiểm tra xem 3 stores (PostgreSQL, Graph DB, Vector DB) có đồng bộ không.

**Business Rules**:
- Drift < 0.1% là chấp nhận được — alert khi vượt ngưỡng
- Reconciliation job chạy tự động mỗi giờ
- Operator có thể trigger manual check qua API

**Types of drift**:

| Loại drift | Nguyên nhân thường gặp | Severity |
|:---|:---|:---|
| `graph_mismatch` | Node tồn tại trong PG nhưng không có trong Graph DB | High |
| `vector_mismatch` | Node tồn tại trong PG nhưng không có trong Vector DB | High |
| `orphan_graph_node` | Node trong Graph DB nhưng đã deleted trong PG | Medium |
| `orphan_vector_doc` | Document trong Vector DB nhưng đã deleted trong PG | Medium |
| `stale_projection_version` | Graph/Vector version lag so với PG source | Low → Medium |

**Kiểm tra**:
```
GET /v1/kg/integrity/tenant/{tenant_id}
→ {
    checks: [
      { id: "IC-01", name: "graph_sync_drift", result: 0, status: "pass" },
      { id: "IC-02", name: "vector_sync_drift", result: 0, status: "pass" },
      ...
    ],
    overall: "pass"
  }
```

---

### BL-OPS-07: Reconciliation — Tự động vs Thủ công

**Business Rules**:
- **Tự động**: ReconciliationWorker chạy mỗi giờ
- **Thủ công**: Operator trigger khi cần ngay (sau incident, sau schema migration)
- Reconciliation không lock service — chạy read-only comparison trước, repair sau
- Repair operations là idempotent — có thể re-run

**Repair actions**:

| Action | Khi nào dùng |
|:---|:---|
| `rebuild` | Sync lại toàn bộ từ PG → Graph + Vector |
| `purge-orphans` | Xóa nodes orphan trong Graph/Vector không có trong PG |
| `recompute-acl` | Tính lại acl_visible_to cho tất cả nodes |

---

## Nghiệp vụ 4: Xử lý Sự cố

### BL-OPS-08: Backend Down

**Mô tả**: Graph DB hoặc Vector DB không phản hồi.

**Business Rules**:

| Backend down | Impact | Action |
|:---|:---|:---|
| **Graph DB down** | Read API → 503 + retry_after. Write vẫn work (data vào PG, outbox pending). | Check graph container, restart, outbox sẽ resume auto |
| **Vector DB down** | Search API → 503. Write vẫn work. | Check vector container, restart, outbox sẽ resume auto |
| **PostgreSQL down** | Write API down. Read có thể serve từ graph (mode=non-realtime). | **Critical** — PG là source of truth, restore priority |
| **Redis down** | Auth cache miss → mọi request hit PG. Rate limit không hoạt động. | Degrade gracefully, restore Redis |

---

### BL-OPS-09: Outbox Dead Letter

**Mô tả**: Sync worker thất bại > 5 lần → event chuyển sang `DEAD_LETTER` — data tồn tại trong PG nhưng không sync sang graph/vector.

**Business Rules**:
- Dead letter event được alert ngay (monitoring hook)
- Data không mất — chỉ chưa sync
- Operator phải investigate root cause trước khi re-queue
- Re-queue: update status = PENDING → worker sẽ retry

**Investigation checklist**:
1. Graph/Vector backend có reachable không?
2. Node data có valid không (ontology đã thay đổi sau khi write)?
3. Embedding provider có reachable không (với VectorSync events)?

---

### BL-OPS-10: Grant Rate Limit Thrashing

**Business Rules**:
- Thay đổi AccessGrant trigger recompute `acl_visible_to` cho N nodes trong domain
- Giới hạn: tối đa 10 grant changes/domain/giờ — vượt → bị throttle
- Nếu cần thay đổi nhiều grants → batch hoặc schedule off-peak

---

## Nghiệp vụ 5: Environment & Configuration

### BL-OPS-11: Environment Variables

**Business Rules**:
- Tất cả environment variables phải documented — không có hidden config
- Config theo runtime profile — không hardcode connection string trong code
- Sensitive values (passwords, API keys) phải inject qua secret management
- Operator phải verify `KG_RUNTIME_PROFILE` khớp với backend đang chạy

**Critical env vars**:

| Variable | Required | Ý nghĩa |
|:---|:---:|:---|
| `KG_RUNTIME_PROFILE` | Yes | Chọn backend combination |
| `KG_DATABASE_URL` | Yes | PostgreSQL connection string |
| `KG_REDIS_URL` | Yes | Redis connection string |
| `GRAPH_ADAPTER` | By profile | Loại graph backend |
| `VECTOR_ADAPTER` | By profile | Loại vector backend |
| `EMBEDDING_PROVIDER` | Yes | `http` hoặc `deterministic` |
| `EMBEDDING_URL` | If http | Embedding API endpoint |

---

## Tóm tắt Business Rules — Operator / SRE

| ID | Rule |
|:---:|:---|
| **BR-OPS-01** | Profile phải khớp với backend đang chạy — không silent fallback |
| **BR-OPS-02** | Migration phải chạy trước API server — không skip |
| **BR-OPS-03** | Health check là điều kiện tiên quyết trước khi đưa vào traffic |
| **BR-OPS-04** | Validation script phải tương ứng với deployment profile |
| **BR-OPS-05** | Reconciliation drift > 0.1% → alert |
| **BR-OPS-06** | Dead letter events → investigate trước khi re-queue |
| **BR-OPS-07** | Repair operations là idempotent — an toàn để re-run |
| **BR-OPS-08** | PostgreSQL down là critical — write path bị ảnh hưởng |
| **BR-OPS-09** | Grant change throttle: max 10/domain/giờ |
| **BR-OPS-10** | Mọi environment variables phải documented — không hidden config |
