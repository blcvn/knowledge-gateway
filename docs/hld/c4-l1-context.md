# C4 Level 1 — System Context Diagram

> **C4 Level**: 1 — System Context  
> **Câu hỏi**: KG Service là gì? Ai sử dụng? Kết nối với hệ thống nào bên ngoài?  
> **Audience**: Everyone — từ business stakeholder đến developer

---

## System Context Diagram

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                         SYSTEM CONTEXT                                      ║
╚══════════════════════════════════════════════════════════════════════════════╝

  [ Platform Admin ]          [ Tenant Admin ]          [ App Integrator ]
  Quản lý toàn bộ            Thiết kế ontology          Tích hợp API vào
  tenant/app trong           domain, cấp quyền          ứng dụng nội bộ
  hệ thống                   cross-tenant
        │                          │                          │
        │  REST API (:8082)        │  REST API (:8082)        │  REST API (:8082)
        │                          │                          │
        ▼                          ▼                          ▼
  ╔═══════════════════════════════════════════════════════════════════════╗
  ║                                                                       ║
  ║                    [[ KG SERVICE ]]                                   ║
  ║                                                                       ║
  ║    Knowledge Graph Platform — Multi-tenant, Domain-agnostic           ║
  ║    Lưu trữ, truy vấn, và tìm kiếm tri thức có cấu trúc               ║
  ║    Expose REST API + MCP protocol                                     ║
  ║    Go 1.25 · Port 8082                                                ║
  ║                                                                       ║
  ╚═══════════════════════════════════════════════════════════════════════╝
        ▲                          ▲                          ▲
        │  REST API                │  MCP / SSE               │  REST API
        │                          │                          │
  [ Ingestion Pipeline ]    [ AI Agent Service ]      [ Operator / SRE ]
  Upload tài liệu,          Thực hiện RAG, tool        Monitor health,
  bulk ingest nodes,        calling qua MCP,           deploy, reconcile,
  sync bridge               semantic search            incident response

─────────────────────────────────────────────────────────────────────────────

  External Systems KG Service depends on:

  [[ PostgreSQL ]]      [[ Graph DB ]]         [[ Vector DB ]]
  Source of truth       Traversal queries       Semantic search
  RLS-enforced          Neo4j / Memgraph        Qdrant / Milvus
  Schema migrations     Nebula / Memory         pgvector / Memory

  [[ Redis ]]           [[ Embedding Provider ]]
  Auth cache            HTTP embedding API
  ACL cache             (VNPay AI Gateway /
  Rate limiting         any OpenAI-compatible)
```

---

## Actors (People)

| Actor | Role | Primary Interaction |
|:---|:---|:---|
| **Platform Admin** | Quản lý toàn bộ hệ thống. Tạo/xóa tenant, quản lý tier. | REST API với platform admin key |
| **Tenant Admin** | Thiết kế ontology cho domain riêng. Cấp quyền cross-tenant. | REST API với tenant admin key |
| **App Integrator** | Developer tích hợp kg-service vào ứng dụng. | REST API với app API key |
| **AI Agent Service** | Hệ thống AI tự động. Gọi search/read qua MCP tools. | MCP protocol (SSE transport) |
| **Ingestion Pipeline** | Hệ thống ETL. Bulk ingest nodes, sync từ external source. | REST Write API + sync bridge |
| **Operator / SRE** | Vận hành. Deploy, monitor, xử lý incident. | REST Integrity/Health API + CLI tools |

---

## External Systems

| System | Type | Relationship với KG Service |
|:---|:---|:---|
| **PostgreSQL 15** | Relational database | Source of truth cho mọi data. RLS enforcement. Migration target. |
| **Graph DB** (Neo4j / Memgraph / Nebula / Memory) | Graph database | Read replica cho graph traversal queries. Không nhận write trực tiếp. |
| **Vector DB** (Qdrant / Milvus / pgvector / Memory) | Vector database | Read replica cho semantic search. Không nhận write trực tiếp. |
| **Redis** | In-memory store | API key cache (30s TTL), ACL cache (60s), rate limit counters. |
| **Embedding Provider** | HTTP API | Convert text → embedding vector. Pluggable: `EMBEDDING_PROVIDER=http\|deterministic`. |
| **Agent Service** | Software system | Upstream consumer. Gọi KG Service để retrieve context cho LLM. |
| **Ingestion Pipeline** | Software system | Upstream producer. Gọi KG Service Write API để load knowledge. |

---

## Key Design Principles (Invariants)

Các nguyên tắc không đổi qua mọi phase triển khai:

| # | Principle | Implication |
|:---:|:---|:---|
| P1 | **Unified graph** — một graph duy nhất, domain/tenant là property | Không tạo DB riêng per tenant |
| P2 | **CQRS 3-mode** — Write→PG, Read→Graph, Search→Vector | Ba store tách biệt, async sync |
| P3 | **Deny-by-default** — chỉ thấy data của mình + platform | Cross-tenant cần explicit grant |
| P4 | **No raw query** — không Cypher thô, không raw filter từ client | Query Pattern DSL + compiler |
| P5 | **Identity from token** — không trust body values | Middleware strips tenant_id/app_id |
| P6 | **Platform ontology immutable** — tenant không sửa platform schema | Separate RBAC |
| P7 | **Cross-domain rules are generic** — domain tự khai báo | Service chỉ execute, không hardcode |
| P8 | **Lifecycle is config** — status/authority là domain_status_field_configs | No-op nếu không khai báo |

---

## Scope Boundaries

### Trong phạm vi KG Service

- Lưu trữ và truy vấn Knowledge Graph (nodes + relationships)
- Quản lý đa tenant (tenant, app, access grant)
- Quản lý ontology theo domain (node type, rel type, query template, lifecycle)
- Expose API qua REST (`:8082`) và MCP (SSE transport)
- Đồng bộ 3 store (PostgreSQL ↔ Graph DB ↔ Vector DB)
- Authentication, ACL, audit trail
- Integrity check và self-healing (reconciliation)

### Ngoài phạm vi KG Service

- OCR và document parsing → `pipeline-service`
- LLM inference và prompt engineering → `agent-service`
- UI admin portal → external consumer gọi KG Service API
- Billing, payment processing → external systems
- Kafka / message broker → phase D option nếu volume tăng

---

## NFR Summary

| Dimension | Target |
|:---|:---|
| Read API latency | P95 < 200ms |
| Search API latency | P95 < 300ms |
| GraphRAG full pipeline | P95 < 5s |
| Write → sync latency | P95 < 2s (PG commit → Graph/Vector visible) |
| AccessGrant revoke → enforcement | < 5s end-to-end |
| Data consistency drift | < 0.1% (hourly reconciliation) |
| Availability | 99.5% |
