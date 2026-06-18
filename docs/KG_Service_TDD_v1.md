# KG Service — Technical Design Document (TDD)

> Phiên bản 1.1 · 17/06/2026  
> Tài liệu kỹ thuật hợp nhất: High-Level Architecture, Low-Level Design, API Specification  
> Tài liệu liên quan: `KG_Ontology_v4.md` (ví dụ ontology lịch sử, không phải baseline mặc định của repo), `KG_Service_MultiTenant_Design.md` (thiết kế gốc multi-tenant)

---

## Mục lục

1. [Giới thiệu](#1-giới-thiệu)
2. [High-Level Architecture](#2-high-level-architecture)
3. [Low-Level Design](#3-low-level-design)
4. [API Specification](#4-api-specification)
5. [Non-Functional Requirements](#5-non-functional-requirements)
6. [Security Design](#6-security-design)
7. [Phân kỳ triển khai](#7-phân-kỳ-triển-khai)
8. [Phụ lục](#8-phụ-lục)

---

## 1. Giới thiệu

### 1.1 Mục đích

Tài liệu này là đặc tả kỹ thuật đầy đủ cho **KG Service** — một nền tảng Knowledge Graph đa tenant **domain-agnostic**: kiến trúc service không hardcode bất kỳ lĩnh vực nghiệp vụ cụ thể nào. Mọi node type, relationship type, quy tắc lifecycle/hiệu lực, và mẫu truy vấn đều là cấu hình do Ontology Plane cung cấp theo từng domain — service chỉ thực thi engine generic, không "biết" trước nội dung domain. Các ví dụ domain-specific trong tài liệu, bao gồm ví dụ pháp luật từ `KG_Ontology_v4.md`, chỉ nhằm minh hoạ khả năng cấu hình chứ không đại diện cho phạm vi mặc định của core service. Tài liệu hợp nhất ba lớp đặc tả:

| Lớp | Trả lời câu hỏi |
|---|---|
| High-Level Architecture | Hệ thống có những thành phần nào, chúng tương tác ra sao |
| Low-Level Design | Mỗi thành phần hoạt động bên trong thế nào, data model chi tiết, thuật toán |
| API Specification | Client gọi API như thế nào, request/response, mã lỗi |

### 1.2 Phạm vi

KG Service chịu trách nhiệm:
- Lưu trữ và truy vấn Knowledge Graph **tổng quát** — không gắn cứng với bất kỳ domain nghiệp vụ cụ thể. Node type, relationship type, quy tắc lifecycle/hiệu lực, và mẫu truy vấn (query template) đều là cấu hình do Ontology Plane cung cấp; service chỉ là engine thực thi, không chứa business logic của domain nào.
- Quản lý đa tenant: tổ chức khách hàng (Tenant), ứng dụng thuộc tenant (App), chia sẻ/phân quyền giữa chúng — cơ chế này cũng hoàn toàn domain-agnostic.
- Quản lý ontology theo tenant — cho phép tenant tự mở rộng schema riêng, đăng ký query template riêng, khai báo status/lifecycle field riêng, bên cạnh ontology nền tảng dùng chung.
- Cung cấp dữ liệu qua 3 mode tách biệt: Write (PostgreSQL), Read (Graph DB), Search (Vector DB) — hành vi của cả 3 mode do domain config quyết định tại runtime.
- Expose API qua REST và MCP cho Agent Service và các hệ thống tiêu thụ khác.

> **Lưu ý kiến trúc quan trọng:** Các ví dụ pháp luật trong `KG_Ontology_v4.md` chỉ là một bộ ontology minh hoạ/historical reference, không phải một phần kiến trúc bắt buộc và không phải baseline bootstrap của repo hiện tại. Một tenant khác hoàn toàn có thể triển khai catalog sản phẩm, quy trình nội bộ, knowledge base y tế, hay bất kỳ domain nào khác trên cùng service mà không cần sửa code service.

**Không thuộc phạm vi:** OCR/ingestion pipeline chi tiết (xem `pipeline-service`), LLM prompt engineering cho Agent Service, UI admin portal (chỉ đặc tả API mà portal gọi).

### 1.3 Tài liệu tham chiếu

| Tài liệu | Nội dung |
|---|---|
| `KG_Ontology_v4.md` | Ontology ví dụ/historical reference cho một cấu hình domain-specific |
| `KG_Ontology_HKD.md` | Ontology domain HKD chi tiết (lịch sử v3, đã merge vào v4) |
| `KG_Service_HLA_Ontology.md` | Thiết kế Ontology Plane / Data Plane ban đầu (chưa có multi-tenant) |
| `KG_Service_MultiTenant_Design.md` | Thiết kế gốc Tenant/App/AccessGrant (tài liệu này mở rộng và chính thức hoá) |
| `SRS_LegalAI_Advisor_v1_0.docx` | Yêu cầu hệ thống tổng thể, NFR gốc |

### 1.4 Định nghĩa thuật ngữ

| Thuật ngữ | Định nghĩa |
|---|---|
| **Tenant** | Tổ chức khách hàng, đơn vị cô lập dữ liệu cao nhất |
| **App** | Ứng dụng cụ thể thuộc một tenant, đơn vị sở hữu (owner) dữ liệu KG |
| **Platform** | Tenant đặc biệt (sentinel), sở hữu ontology nền tảng dùng chung (nếu có) |
| **Domain** | Đơn vị phân loại ontology (vd: `sample_policy`), có owner_tenant_id — service không biết trước nội dung domain |
| **AccessGrant** | Bản ghi cấp quyền chia sẻ giữa (tenant, app) nguồn và đích |
| **acl_visible_to** | Field denormalized trên node/vector, danh sách `{tenant}:{app}` được phép xem |
| **Effective ontology** | Tập hợp domain mà một app cụ thể được phép dùng (sở hữu + platform + được share) |
| **Query Pattern DSL** | Đặc tả JSON generic mô tả một mẫu truy vấn graph (start node, hops, return fields), domain owner đăng ký qua Ontology API; service biên dịch thành Cypher, không nhận Cypher thô |
| **status_value / status_field_config** | Cơ chế generic cho khái niệm "trạng thái/lifecycle" mà từng domain tự khai báo field nào đóng vai trò này (domain không khai báo = không có khái niệm này) |
| **external_ref** | Key generic dùng để đồng bộ một node giữa PostgreSQL ↔ Graph DB ↔ Qdrant, convention do domain tự định nghĩa |
| **Khoan / QUY_DINH_BOI** | *Ví dụ cụ thể của domain pháp luật* (`KG_Ontology_v4.md`) — Khoan là đơn vị văn bản nhỏ nhất, QUY_DINH_BOI là tên rel_type domain này dùng cho cross-domain rule. Đây **không phải** khái niệm có trong core service — domain khác có thể không có node/rel tương đương |

---

## 2. High-Level Architecture

### 2.1 System Context

```
┌────────────────────────────────────────────────────────────────────┐
│                         Bên ngoài KG Service                       │
│                                                                    │
│   Agent Service        Admin Portal         Ingestion Pipeline    │
│   (port 8081)          (web UI)             (pipeline-service)    │
│   - GraphRAG caller    - Tenant/App mgmt    - Document upload     │
│   - Gọi query template - Grant management   - Chunking theo       │
│     đã đăng ký theo      - Ontology editor      domain config      │
│     domain (vd: domain                                            │
│     pháp luật có 5 mẫu)                                           │
└──────────────┬─────────────────┬─────────────────┬────────────────┘
               │  REST / MCP      │ REST              │ REST
               ▼                 ▼                   ▼
┌────────────────────────────────────────────────────────────────────┐
│                          KG SERVICE (port 8082)                     │
│  Mọi request mang API key → resolve (tenant_id, app_id)             │
└────────────────────────────────────────────────────────────────────┘
               │
               ▼
   PostgreSQL · Graph DB (AGE/Memgraph) · Qdrant · Redis
```

### 2.2 Nguyên tắc thiết kế (invariants)

Các nguyên tắc này **không thay đổi** qua các phase triển khai — mọi quyết định thiết kế chi tiết phải tuân theo:

| # | Nguyên tắc |
|---|---|
| P1 | KG là **một graph thống nhất** — domain và tenant là property/label phân loại trên node, không phải graph/database riêng |
| P2 | **CQRS 3-mode**: Write luôn qua PostgreSQL trước; Graph DB và Vector DB là read replica có index, không nhận write trực tiếp |
| P3 | **Deny-by-default**: app chỉ thấy dữ liệu của chính nó + platform public; mọi truy cập khác cần `AccessGrant` tường minh |
| P4 | **Không raw query từ client** — Read API chỉ chạy named template tham số hoá; Search API chỉ nhận filter đã được service kiểm soát |
| P5 | Identity (tenant_id/app_id) **luôn resolve từ token xác thực**, không bao giờ tin giá trị do client/LLM cung cấp trong request body |
| P6 | Ontology nền tảng (platform-owned) **không thể bị tenant sửa**; tenant chỉ mở rộng domain riêng |
| P7 | Cơ chế *cross-domain rule bắt buộc* là generic — mỗi domain tự khai báo `CrossDomainRelRule` cần áp dụng trước khi publish node; service chỉ thực thi rule, không biết trước nội dung rule (ví dụ domain pháp luật tự định nghĩa rule `QUY_DINH_BOI → :Khoan`, đây là cấu hình của domain đó, không phải invariant của service) |
| P8 | **Lifecycle/status field** (vd: khái niệm "hiệu lực" trong domain pháp luật), **authority/ranking score** dùng cho rerank, và **named query pattern** đều là cấu hình theo domain (`domain_status_field_configs`, `domain_query_templates`) — không bao giờ hardcode trong service code; domain không khai báo thì engine bỏ qua bước đó (no-op), không giả định giá trị mặc định |

### 2.3 Kiến trúc Component — 3 Plane

```
┌──────────────────────────────────────────────────────────────────────┐
│  KG Service                                                          │
│                                                                       │
│  ┌────────────────────┐ ┌────────────────────┐ ┌───────────────────┐ │
│  │ Identity & Access  │ │ Ontology Plane     │ │ Data Plane         │ │
│  │ Plane              │ │                    │ │                    │ │
│  │                    │ │ DomainRegistry     │ │ WriteService       │ │
│  │ TenantRegistry     │ │ NodeTypeRegistry   │ │  → PostgreSQL      │ │
│  │ AppRegistry        │ │ RelTypeRegistry    │ │ ReadService         │ │
│  │ AccessGrantStore   │◀┼─OntologyResolver──▶│ │ (QueryTemplate-     │ │
│  │ AccessResolver     │ │ CrossDomainRules   │ │  Compiler)          │ │
│  │ AuditLogger        │ │ OntologyVersioning │ │  → Graph DB         │ │
│  │                    │ │ QueryTemplateReg.  │ │ SearchService       │ │
│  │                    │ │ StatusFieldConfig  │ │  → Vector DB        │ │
│  └────────────────────┘ └────────────────────┘ └───────────────────┘ │
│                                       │                                │
│                          ┌────────────┴────────────┐                  │
│                          │  Sync & Consistency      │                  │
│                          │  OutboxPublisher          │                  │
│                          │  GraphSyncWorker           │                  │
│                          │  VectorSyncWorker          │                  │
│                          │  AccessSyncWorker          │                  │
│                          │  StatusGate (generic)       │                  │
│                          │  GraphRAGPipeline (generic)  │                  │
│                          └──────────────────────────┘                  │
└────────────────────────────────────────────────────────────────────────┘
```

| Plane | Trách nhiệm | Component chính |
|---|---|---|
| Identity & Access | Xác thực, xác định tenant/app, resolve quyền truy cập | TenantRegistry, AppRegistry, AccessResolver |
| Ontology Plane | Khai báo & validate schema theo domain/tenant, **kể cả query template và status field config** | DomainRegistry, OntologyResolver, QueryTemplateRegistry, StatusFieldConfigRegistry |
| Data Plane | Vận hành 3 mode dữ liệu — hành vi hoàn toàn do domain config quyết định, không có business logic cố định | WriteService, ReadService (qua QueryTemplateCompiler), SearchService |
| Sync & Consistency | Đảm bảo 3 store đồng bộ, đúng quyền | Sync Workers, StatusGate (generic, điều khiển bởi `domain_status_field_configs`) |

### 2.4 Technology Stack

| Layer | Phase 1 (MVP) | Phase 2 (Production) | Lý do |
|---|---|---|---|
| API Server | Go (net/http + chi router) | Go | Hiệu năng, concurrency tốt cho I/O-bound |
| Relational DB | PostgreSQL 15 + RLS | PostgreSQL 15 | Source of truth, transaction mạnh, RLS native |
| Graph DB | Apache AGE (extension PG) | Memgraph Community | AGE đơn giản cho seed nhỏ; Memgraph khi traversal phức tạp tăng |
| Vector DB | Qdrant | Qdrant | Payload filtering mạnh, phù hợp multitenancy |
| Cache | Redis | Redis Cluster | Cache AccessResolver, rate limit counter |
| Queue (outbox) | Redis Streams | Kafka (nếu volume tăng) | Đơn giản trước, scale sau |
| MCP transport | HTTP+SSE | HTTP+SSE | Chuẩn MCP hiện hành |

**Quyết định không dùng Neo4j Enterprise:** licensing cao (~$50k–200k+/năm), Community Edition thiếu HA/RLS-equivalent — không khả thi cho v1.0.

### 2.5 Deployment view

```
Phase 1 (MVP, single region):
  1× API server (stateless, scale-out theo CPU)
  1× PostgreSQL primary + 1 replica (read-only cho Read API fallback nếu cần)
  1× AGE (chung instance với PostgreSQL — cùng cluster)
  1× Qdrant single node
  1× Redis single node

Phase 2 (Production):
  3× API server behind load balancer
  PostgreSQL primary + 2 replica, automated failover
  Memgraph cluster (nếu cần HA)
  Qdrant 2 shard, replication factor 2
  Redis Sentinel (3 node)
```

---

## 3. Low-Level Design

### 3.1 Data Model đầy đủ (PostgreSQL DDL)

#### 3.1.1 Identity & Access tables

```sql
CREATE TABLE tenants (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                    TEXT NOT NULL UNIQUE,
    name                    TEXT NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active','suspended','trial')),
    tier                    TEXT NOT NULL DEFAULT 'free'
                              CHECK (tier IN ('free','pro','enterprise')),
    default_sharing_policy  TEXT NOT NULL DEFAULT 'deny_all'
                              CHECK (default_sharing_policy IN ('deny_all','share_within_tenant_read')),
    settings                JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO tenants (id, slug, name, status, tier) VALUES
  ('00000000-0000-0000-0000-000000000000', 'platform', 'Aevlex Platform', 'active', 'enterprise');

CREATE TABLE apps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL
                      CHECK (type IN ('agent_consumer','ingestion_producer','admin_tool','hybrid')),
    api_key_hash    TEXT NOT NULL,
    api_key_prefix  TEXT NOT NULL,            -- 8 ký tự đầu, hiển thị cho user nhận diện key
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','revoked')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at      TIMESTAMPTZ,
    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_apps_api_key_hash ON apps(api_key_hash) WHERE status = 'active';

CREATE TABLE access_grants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grantor_tenant_id   UUID NOT NULL REFERENCES tenants(id),
    grantor_app_id      UUID REFERENCES apps(id),
    grantee_tenant_id   UUID NOT NULL REFERENCES tenants(id),
    grantee_app_id      UUID REFERENCES apps(id),
    scope_type          TEXT NOT NULL CHECK (scope_type IN ('domain','node_type','dataset_tag','all')),
    scope_value         TEXT,
    permission          TEXT NOT NULL CHECK (permission IN ('read','search','write','admin')),
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','revoked','expired')),
    expires_at          TIMESTAMPTZ,
    created_by          UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ,
    CONSTRAINT chk_scope_value CHECK (
      (scope_type = 'all' AND scope_value IS NULL) OR
      (scope_type <> 'all' AND scope_value IS NOT NULL)
    )
);

CREATE INDEX idx_grant_grantee ON access_grants(grantee_tenant_id, grantee_app_id, status);
CREATE INDEX idx_grant_grantor ON access_grants(grantor_tenant_id, grantor_app_id, status);
CREATE INDEX idx_grant_scope   ON access_grants(scope_type, scope_value) WHERE status = 'active';

CREATE TABLE access_audit_log (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_tenant_id       UUID NOT NULL,
    requester_app_id          UUID NOT NULL,
    action                    TEXT NOT NULL CHECK (action IN ('read','search','write','grant_created','grant_revoked')),
    resource_domain_id        TEXT,
    resource_owner_tenant_id  UUID,
    allowed                   BOOLEAN NOT NULL,
    reason                    TEXT NOT NULL,
    request_id                UUID,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);
-- Partition theo tháng, retain 12 tháng, archive sau đó
```

#### 3.1.2 Ontology tables

```sql
CREATE TABLE domains (
    id                  TEXT PRIMARY KEY,              -- "sample_policy", "noi_bo_hop_dong"
    name                TEXT NOT NULL,
    description         TEXT,
    owner_tenant_id     UUID NOT NULL REFERENCES tenants(id),
    parent_domain_id    TEXT REFERENCES domains(id),
    status              TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','active','deprecated')),
    version             INT NOT NULL DEFAULT 1,
    visibility          TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('public','tenant_shared','private')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ontology_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id           TEXT NOT NULL REFERENCES domains(id),
    version             INT NOT NULL,
    changes             JSONB NOT NULL,                -- [{type, detail}, ...]
    breaking_change     BOOLEAN NOT NULL DEFAULT false,
    published_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_by        UUID,
    UNIQUE(domain_id, version)
);

CREATE TABLE node_type_schemas (
    id                  TEXT PRIMARY KEY,              -- "{domain_id}.{node_type_name}"
    domain_id           TEXT NOT NULL REFERENCES domains(id),
    node_type_name      TEXT NOT NULL,
    graph_label         TEXT NOT NULL,
    required_props      JSONB NOT NULL DEFAULT '[]',   -- [{name, type, description}]
    optional_props      JSONB NOT NULL DEFAULT '[]',
    validation_rules    JSONB NOT NULL DEFAULT '[]',   -- ["nguong_max > nguong_min OR nguong_max IS NULL"]
    version             INT NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(domain_id, node_type_name)
);

CREATE TABLE rel_type_schemas (
    id                  TEXT PRIMARY KEY,
    domain_id           TEXT NOT NULL REFERENCES domains(id),
    rel_type_name       TEXT NOT NULL,
    from_node_type      TEXT NOT NULL,
    to_node_type        TEXT NOT NULL,
    same_domain         BOOLEAN NOT NULL DEFAULT true,
    required_props      JSONB NOT NULL DEFAULT '[]',
    optional_props      JSONB NOT NULL DEFAULT '[]',
    UNIQUE(domain_id, rel_type_name, from_node_type, to_node_type)
);

CREATE TABLE cross_domain_rel_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rel_type_name       TEXT NOT NULL,
    from_domain_id      TEXT NOT NULL REFERENCES domains(id),
    to_domain_id        TEXT NOT NULL REFERENCES domains(id),
    from_node_types     TEXT[] NOT NULL DEFAULT ARRAY['*'],
    to_node_types       TEXT[] NOT NULL,
    required             BOOLEAN NOT NULL DEFAULT true,
    exception_types      TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    allowed_loai_values  TEXT[] DEFAULT ARRAY['can_cu_chinh','tham_khao']
);
```

**Hai bảng dưới đây là cơ chế then chốt để giữ service domain-agnostic** — chúng cho phép mỗi domain "cắm" hành vi đọc dữ liệu và quy tắc lifecycle riêng mà không cần sửa code service.

```sql
-- Query Pattern DSL — KHÔNG lưu Cypher thô (giữ nguyên invariant P4: không raw query,
-- áp dụng cả với domain owner, không riêng end-client)
CREATE TABLE domain_query_templates (
    id                  TEXT PRIMARY KEY,              -- "{domain_id}.{template_name}"
    domain_id           TEXT NOT NULL REFERENCES domains(id),
    template_name       TEXT NOT NULL,
    pattern_spec        JSONB NOT NULL,                -- xem cấu trúc DSL ở §3.4.5
    param_schema        JSONB NOT NULL DEFAULT '[]',   -- [{name, type, required}]
    return_fields        TEXT[] NOT NULL,
    description          TEXT,
    status                TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','active','deprecated')),
    created_by            UUID,
    version               INT NOT NULL DEFAULT 1,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(domain_id, template_name)
);

-- Khai báo lifecycle/status field và authority score riêng theo domain — generic,
-- domain nào không có khái niệm "hiệu lực" thì không insert row, StatusGate tự bỏ qua.
CREATE TABLE domain_status_field_configs (
    domain_id              TEXT PRIMARY KEY REFERENCES domains(id),
    status_field_name      TEXT,                       -- vd: "tinh_trang" (domain pháp luật); NULL = không có
    valid_status_values     TEXT[],                      -- vd: ["con_hieu_luc"]
    warning_status_values   TEXT[],                      -- vd: ["bi_sua_doi"] — vẫn trả về nhưng kèm cảnh báo
    cascade_rules            JSONB NOT NULL DEFAULT '[]', -- mô tả cascade status giữa node type qua rel nào
    authority_field_name     TEXT,                        -- vd: "loai_van_ban"; NULL = domain không cần rerank theo authority
    authority_values_map     JSONB                        -- vd: {"Luat": 4, "NghiDinh": 3, "ThongTu": 2, "CongVan": 1}
);
```

#### 3.1.3 KG Data tables (mode 1 — Write)

```sql
CREATE TABLE kg_nodes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type           TEXT NOT NULL,
    domain_id           TEXT NOT NULL REFERENCES domains(id),
    owner_tenant_id     UUID NOT NULL REFERENCES tenants(id),
    owner_app_id        UUID REFERENCES apps(id),
    visibility          TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('public','tenant_shared','private')),
    properties          JSONB NOT NULL,
    domain_version      INT NOT NULL,
    external_ref        TEXT UNIQUE,                  -- key đồng bộ Graph DB/Qdrant, domain tự định nghĩa convention
                                                         -- (vd: domain pháp luật dùng "chunk_id" làm external_ref cho node Khoan)
    status_value        TEXT,                          -- generic, map từ field domain khai báo trong
                                                         -- domain_status_field_configs.status_field_name; NULL nếu domain
                                                         -- không có khái niệm lifecycle
    is_deleted          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_kg_nodes_domain   ON kg_nodes(domain_id) WHERE NOT is_deleted;
CREATE INDEX idx_kg_nodes_owner    ON kg_nodes(owner_tenant_id, owner_app_id);
CREATE INDEX idx_kg_nodes_type     ON kg_nodes(node_type, domain_id);
CREATE INDEX idx_kg_nodes_ref      ON kg_nodes(external_ref) WHERE external_ref IS NOT NULL;

ALTER TABLE kg_nodes ENABLE ROW LEVEL SECURITY;

CREATE POLICY kg_nodes_isolation ON kg_nodes
  USING (
    owner_tenant_id = current_setting('app.tenant_id')::uuid
    OR owner_tenant_id = '00000000-0000-0000-0000-000000000000'
    OR EXISTS (
        SELECT 1 FROM access_grants g
        WHERE g.grantee_tenant_id = current_setting('app.tenant_id')::uuid
          AND (g.grantee_app_id = current_setting('app.app_id')::uuid OR g.grantee_app_id IS NULL)
          AND g.status = 'active'
          AND (g.expires_at IS NULL OR g.expires_at > now())
          AND (g.scope_type = 'all'
               OR (g.scope_type = 'domain' AND g.scope_value = kg_nodes.domain_id)
               OR (g.scope_type = 'node_type' AND g.scope_value = kg_nodes.node_type))
    )
  );

CREATE TABLE kg_relationships (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rel_type            TEXT NOT NULL,
    from_node_id        UUID NOT NULL REFERENCES kg_nodes(id),
    to_node_id          UUID NOT NULL REFERENCES kg_nodes(id),
    domain_id           TEXT NOT NULL REFERENCES domains(id),
    owner_tenant_id     UUID NOT NULL REFERENCES tenants(id),
    owner_app_id        UUID REFERENCES apps(id),
    properties          JSONB NOT NULL DEFAULT '{}',
    is_deleted          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE kg_relationships ENABLE ROW LEVEL SECURITY;
-- Policy tương tự kg_nodes_isolation, áp dụng theo domain_id/owner_tenant_id của relationship

CREATE TABLE kg_outbox_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type  TEXT NOT NULL,           -- "kg_node" | "kg_relationship" | "access_grant"
    aggregate_id    UUID NOT NULL,
    event_type      TEXT NOT NULL,           -- "NODE_UPSERTED" | "STATUS_VALUE_CHANGED" | "ACCESS_GRANT_CHANGED" | ...
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','PROCESSING','DONE','FAILED')),
    retry_count     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at    TIMESTAMPTZ
);

CREATE INDEX idx_outbox_pending ON kg_outbox_events(status, created_at) WHERE status = 'PENDING';
```

### 3.2 Graph DB Schema

Kế thừa toàn bộ label/constraint từ `KG_Ontology_v4.md §12`, bổ sung field ACL trên mọi node:

```cypher
-- Field bổ sung trên MỌI node (không riêng loại nào)
-- owner_tenant_id    String
-- owner_app_id       String?
-- domain_id          String
-- acl_visible_to     String[]     ★ denormalized, dùng để filter nhanh

CREATE INDEX node_acl IF NOT EXISTS
  FOR (n:Khoan) ON (n.acl_visible_to);
-- Lặp lại cho mọi label cần filter theo ACL (NhomDoanhThu, TyLeThue, ...)

CREATE INDEX node_owner IF NOT EXISTS
  FOR (n:Khoan) ON (n.owner_tenant_id, n.owner_app_id);
```

> Constraint/index chi tiết cho từng node type (VanBanLuat, Dieu, Khoan, NhomDoanhThu...) — xem `KG_Ontology_v4.md §12.1–12.2`, không lặp lại ở đây.

### 3.3 Vector DB Schema (Qdrant)

**Nguyên tắc:** collection chỉ có một bộ field **core generic** (bắt buộc với mọi domain — phục vụ ACL và sync), phần còn lại là **domain-specific payload** mở (schema-less, do domain tự định nghĩa qua Ontology Plane, lưu trong sub-object `domain_props`). Service không biết và không cần biết domain pháp luật có field `tinh_trang` hay `loai_van_ban` — đó là chi tiết của domain cụ thể.

```json
{
  "collection": "kg_vectors",
  "vector_size": 1024,
  "distance": "Cosine",
  "payload_schema": {
    "node_id":           { "type": "keyword" },
    "node_type":         { "type": "keyword" },
    "domain_id":         { "type": "keyword" },
    "owner_tenant_id":   { "type": "keyword" },
    "owner_app_id":      { "type": "keyword" },
    "acl_visible_to":    { "type": "keyword[]", "index": true },
    "is_deleted":        { "type": "bool" },
    "status_value":      { "type": "keyword" },
    "authority_score":   { "type": "integer" },
    "domain_props":      { "type": "object" }
  },
  "hnsw_config": { "m": 16, "ef_construct": 200 }
}
```

| Field generic | Nguồn gốc |
|---|---|
| `status_value` | Map từ `domain_status_field_configs.status_field_name` của domain đó (vd: domain pháp luật map từ `tinh_trang`); domain không khai báo status field thì để `null`, `StatusGate` bỏ qua |
| `authority_score` | Map từ `domain_status_field_configs.authority_field_name` (vd: domain pháp luật map từ `loai_van_ban` qua bảng tra `authority_values_map`); domain không khai báo thì `null`, rerank bỏ trọng số này |
| `domain_props` | Toàn bộ field còn lại đặc thù domain (vd: `chunk_id`, `loai_van_ban`, `so_hieu`, `ngay_hieu_luc`, `linh_vuc_ids` của domain pháp luật) — service không filter trực tiếp trên các field này trừ khi domain khai báo qua query template |

> **Một collection duy nhất `kg_vectors` cho mọi domain** — không tạo collection riêng `legal_clauses` hay `domain_knowledge` trong core schema; việc tách collection theo domain/tenant (nếu cần ở scale lớn) là quyết định vận hành ở Phase D, không phải ràng buộc schema.

`acl_visible_to` được payload-index hoá để filter không làm chậm HNSW search (Qdrant filterable index, xem khuyến nghị multitenancy chính thức của Qdrant).

### 3.4 Component Design

#### 3.4.1 IdentityResolver

**Trách nhiệm:** Map API key/MCP connection token → `(tenant_id, app_id)`.

```
function resolve_identity(api_key: string) -> (tenant_id, app_id):
    key_hash = sha256(api_key)
    app = SELECT * FROM apps WHERE api_key_hash = key_hash AND status = 'active'
    if app is None:
        raise AuthError("invalid or revoked api key")
    return (app.tenant_id, app.id)
```

Cache kết quả trong Redis với key `apikey:{hash}`, TTL 30s (đủ ngắn để revoke có hiệu lực nhanh — yêu cầu S9 trong security checklist).

#### 3.4.2 AccessResolver

**Trách nhiệm:** Tính tập "visible owners" cho một `(tenant_id, app_id)`.

```
function resolve_visible_owners(tenant_id, app_id) -> Set[VisibleOwner]:
    cache_key = f"acl:{tenant_id}:{app_id}"
    if cached := redis.get(cache_key):
        return deserialize(cached)

    visible = set()
    visible.add(Owner(tenant_id, app_id))                     # chính mình
    visible.add(Owner(tenant_id, None))                       # tenant-wide owned

    tenant = db.get_tenant(tenant_id)
    if tenant.default_sharing_policy == "share_within_tenant_read":
        visible.add(Owner(tenant_id, "*"))

    visible.add(Owner("platform", "*"))                       # luôn public

    grants = db.query("""
        SELECT * FROM access_grants
        WHERE grantee_tenant_id = %s
          AND (grantee_app_id = %s OR grantee_app_id IS NULL)
          AND status = 'active'
          AND (expires_at IS NULL OR expires_at > now())
    """, tenant_id, app_id)

    for g in grants:
        visible.add(GrantedOwner(g.grantor_tenant_id, g.grantor_app_id or "*",
                                  g.scope_type, g.scope_value, g.permission))

    redis.set(cache_key, serialize(visible), ttl=60)
    return visible
```

**Invalidation:** mỗi khi `AccessGrant` thay đổi (tạo/revoke), publish event `ACCESS_GRANT_CHANGED` → tất cả cache key liên quan đến `grantee_tenant_id`/`grantee_app_id` của grant đó bị xoá ngay (không chờ TTL).

#### 3.4.3 OntologyResolver / NodeValidator

```
function get_effective_ontology(tenant_id, app_id) -> List[Domain]:
    domains = db.get_domains(owner_tenant_id="platform")
    domains += db.get_domains(owner_tenant_id=tenant_id)

    visible = resolve_visible_owners(tenant_id, app_id)
    shared_domain_ids = [v.scope_value for v in visible
                          if v.scope_type == "domain" and v.permission in ("read","write")]
    domains += db.get_domains(id_in=shared_domain_ids)
    return dedupe(domains)

function validate_node(tenant_id, app_id, domain_id, node_type, properties) -> ValidationResult:
    effective = get_effective_ontology(tenant_id, app_id)
    if domain_id not in [d.id for d in effective]:
        return Error("domain not in effective ontology")

    schema = db.get_node_type_schema(domain_id, node_type)
    if schema is None:
        return Error(f"node_type {node_type} not declared in domain {domain_id}")

    for prop in schema.required_props:
        if prop.name not in properties:
            return Error(f"missing required prop: {prop.name}")
        if not type_matches(properties[prop.name], prop.type):
            return Error(f"type mismatch on {prop.name}")

    for rule in schema.validation_rules:
        if not evaluate_rule(rule, properties):
            return Error(f"validation rule failed: {rule}")

    # Cross-domain rule: generic — lặp qua MỌI CrossDomainRelRule khai báo cho domain này,
    # service không biết trước tên rel_type cụ thể (vd: domain pháp luật tự đăng ký rule
    # tên "QUY_DINH_BOI"; domain khác có thể đăng ký rel_type khác hoàn toàn, hoặc không có rule nào)
    cross_rules = db.get_cross_domain_rules(from_domain=domain_id)
    for rule in cross_rules:
        if not rule.required or node_type in rule.exception_types:
            continue
        bridge_key = f"bridge_{rule.rel_type_name.lower()}_ids"   # convention chung, domain tự đặt giá trị
        if bridge_key not in properties or len(properties[bridge_key]) == 0:
            return Error(f"{rule.rel_type_name} required: must provide {bridge_key}")

    return OK()
```

#### 3.4.4 WriteService (Mode 1)

```
function write_node(tenant_id, app_id, domain_id, node_type, properties) -> NodeId:
    # 1. AuthZ: app có quyền write trên domain này?
    if not has_write_permission(tenant_id, app_id, domain_id):
        raise ForbiddenError()

    # 2. Validate ontology
    result = validate_node(tenant_id, app_id, domain_id, node_type, properties)
    if not result.ok:
        raise ValidationError(result.errors)

    # 3. Transaction: set RLS context + insert + outbox, atomic
    with db.transaction() as tx:
        tx.execute("SET LOCAL app.tenant_id = %s", tenant_id)
        tx.execute("SET LOCAL app.app_id = %s", app_id)

        node_id = tx.insert("kg_nodes", {
            "node_type": node_type, "domain_id": domain_id,
            "owner_tenant_id": tenant_id, "owner_app_id": app_id,
            "properties": properties,
            "domain_version": get_current_domain_version(domain_id)
        })

        # Tạo relationship cho mọi cross-domain rule có bridge_key tương ứng trong properties
        # — generic, không hardcode tên rel_type cụ thể nào
        for rule in db.get_cross_domain_rules(from_domain=domain_id):
            bridge_key = f"bridge_{rule.rel_type_name.lower()}_ids"
            if bridge_key in properties:
                for target_id in properties[bridge_key]:
                    tx.insert("kg_relationships", {
                        "rel_type": rule.rel_type_name, "from_node_id": node_id, "to_node_id": target_id,
                        "domain_id": domain_id, "owner_tenant_id": tenant_id, "owner_app_id": app_id,
                        "properties": {}
                    })

        tx.insert("kg_outbox_events", {
            "aggregate_type": "kg_node", "aggregate_id": node_id,
            "event_type": "NODE_UPSERTED", "payload": {...}
        })

    return node_id
```

#### 3.4.5 ReadService (Mode 2) — Query Pattern DSL + Compiler

**Vấn đề với named Cypher template tĩnh:** nếu service tự viết sẵn Cypher cho từng pattern (như "calculator", "tax_routing"...), service code sẽ chứa tên label/relationship của một domain cụ thể — vi phạm tính domain-agnostic. Ngược lại, nếu cho domain owner tự nộp Cypher thô để đăng ký template, lại vi phạm invariant P4 (không raw query) vì Cypher thô có thể bỏ qua ACL filter hoặc chạy traversal không kiểm soát.

**Giải pháp:** domain owner đăng ký template dưới dạng **Query Pattern DSL** (JSON, không phải Cypher) qua Ontology API (`POST /v1/tenants/{id}/ontology/domains/{id}/query-templates`, xem §4.4). Service có một `QueryTemplateCompiler` **generic duy nhất**, biên dịch DSL này thành Cypher tại runtime, **luôn tự inject ACL filter ở mọi hop** — domain owner không thể bỏ qua bước này dù có cố ý.

**Ví dụ pattern_spec lưu trong `domain_query_templates` cho domain `sample_policy`, template `action-guide`** (đây là dữ liệu cấu hình, không phải code service):

```json
{
  "start": { "node_type": "Topic", "match": { "topic_key": "$topic_key" } },
  "hops": [
    { "rel_type": "ROUTES_TO", "to_node_type": "ActionGuide" },
    { "rel_type": "REQUIRES", "to_node_type": "Obligation", "filter_status": "valid_only" },
    { "rel_type": "SCHEDULED_BY", "direction": "in", "to_node_type": "Record" }
  ],
  "return_fields": ["Topic.title", "ActionGuide.title", "Obligation.summary", "Record.record_key"]
}
```

`filter_status: "valid_only"` không hardcode tên field `tinh_trang` — compiler tự tra `domain_status_field_configs` của domain để biết field nào và giá trị hợp lệ nào cần lọc (§3.4.8). Domain không khai báo status config thì `filter_status` bị bỏ qua.

```
function compile_pattern_to_cypher(pattern_spec, domain_id) -> CypherQuery:
    status_cfg = db.get_status_field_config(domain_id)   # có thể là None
    clauses = []
    bind = f"n0:{pattern_spec.start.node_type} {dict_to_cypher_map(pattern_spec.start.match)}"
    clauses.append(f"MATCH ({bind})")
    clauses.append(f"WHERE ANY(tok IN n0.acl_visible_to WHERE tok IN $acl_tokens)")   # ★ luôn inject, không thể tắt

    prev_alias = "n0"
    for i, hop in enumerate(pattern_spec.hops):
        alias = f"n{i+1}"
        arrow = "<-" if hop.direction == "in" else "-"
        clauses.append(f"MATCH ({prev_alias}){arrow}[:{hop.rel_type}]{'-' if hop.direction=='in' else '->'}({alias}:{hop.to_node_type})")
        clauses.append(f"WHERE ANY(tok IN {alias}.acl_visible_to WHERE tok IN $acl_tokens)")   # ★ mọi hop đều bị filter
        if hop.filter:
            clauses.append(render_filter(alias, hop.filter))
        if hop.filter_status == "valid_only" and status_cfg and status_cfg.status_field_name:
            clauses.append(f"AND {alias}.{status_cfg.status_field_name} IN {status_cfg.valid_status_values}")
        prev_alias = alias

    clauses.append(f"RETURN {', '.join(pattern_spec.return_fields)}")
    return "\n".join(clauses)

function execute_read(tenant_id, app_id, domain_id, template_name, params) -> List[Record]:
    template = db.get_query_template(domain_id, template_name)
    if template is None or template.status != "active":
        raise BadRequestError("unknown or inactive template")

    visible = resolve_visible_owners(tenant_id, app_id)
    acl_tokens = [f"{o.tenant_id}:{o.app_id}" for o in visible]

    cypher = compile_pattern_to_cypher(template.pattern_spec, domain_id)   # compile mỗi lần, cache theo template.version
    bound_params = {**params, "acl_tokens": acl_tokens}

    records = graph_db.run(cypher, bound_params, timeout_ms=3000, max_rows=1000)
    audit_log(tenant_id, app_id, "read", f"{domain_id}.{template_name}", allowed=True)
    return records
```

Lợi ích của thiết kế này so với named-Cypher-dict cũ:

| Tiêu chí | Cypher dict hardcode (cũ) | Query Pattern DSL (mới) |
|---|---|---|
| Thêm domain mới | Phải sửa code service, deploy lại | Chỉ cần gọi Ontology API, không deploy |
| Domain owner có thể bỏ qua ACL? | Có thể (nếu được phép viết Cypher) | Không thể — compiler luôn chèn ACL ở mọi hop |
| Service có biết tên label cụ thể? | Có (vi phạm domain-agnostic) | Không — service chỉ thấy chuỗi string từ JSON |
| Kiểm soát độ sâu/độ phức tạp | Khó, phụ thuộc người viết Cypher | Dễ — giới hạn số hop tối đa trong DSL (vd: ≤ 5) tại tầng validate khi đăng ký template |

#### 3.4.6 SearchService (Mode 3)

```
function semantic_search(tenant_id, app_id, query_text, domain_ids=None, top_k=10) -> List[SearchResult]:
    visible = resolve_visible_owners(tenant_id, app_id)
    acl_tokens = [f"{o.tenant_id}:{o.app_id}" for o in visible]

    qdrant_filter = {
        "must": [
            {"key": "acl_visible_to", "match": {"any": acl_tokens}},
            {"key": "is_deleted", "match": {"value": False}},
        ]
    }
    if domain_ids:
        qdrant_filter["must"].append({"key": "domain_id", "match": {"any": domain_ids}})
        # status_value chỉ filter nếu (TẤT CẢ) domain trong domain_ids có khai báo status field
        # — generic, không giả định mọi domain đều có khái niệm "hiệu lực"
        status_configs = [db.get_status_field_config(d) for d in domain_ids]
        if all(c and c.status_field_name for c in status_configs):
            valid_values = union(c.valid_status_values for c in status_configs)
            qdrant_filter["must"].append({"key": "status_value", "match": {"any": valid_values}})

    vector = embedding_model.embed(query_text)
    results = qdrant.search("kg_vectors", query_vector=vector,
                             query_filter=qdrant_filter, limit=top_k)
    audit_log(tenant_id, app_id, "search", query_text[:50], allowed=True)
    return results
```

#### 3.4.7 Sync Workers

```
GraphSyncWorker.on(event):
    match event.event_type:
        case "NODE_UPSERTED":
            node = fetch_node(event.aggregate_id)
            acl = compute_acl_visible_to(node)
            graph_db.merge_node(node, acl_visible_to=acl)
        case "STATUS_VALUE_CHANGED":
            # Generic cascade — đọc cascade_rules từ domain_status_field_configs của domain đó,
            # KHÔNG hardcode tên label/relationship cụ thể nào (vd: domain pháp luật tự khai báo
            # cascade "VanBanLuat → Dieu → Khoan qua BAO_GOM/CO_KHOAN", lưu trong cascade_rules JSON)
            cfg = db.get_status_field_config(event.payload.domain_id)
            for rule in cfg.cascade_rules:
                graph_db.run(build_cascade_cypher(rule), root_id=event.payload.node_id,
                              new_status=event.payload.new_status)
        case "ACCESS_GRANT_CHANGED":
            affected_nodes = find_nodes_by_scope(event.payload.scope_type, event.payload.scope_value,
                                                  owner_tenant_id=event.payload.grantor_tenant_id)
            for n in affected_nodes:
                graph_db.update_property(n.id, "acl_visible_to", compute_acl_visible_to(n))

VectorSyncWorker.on(event):
    match event.event_type:
        case "NODE_UPSERTED":
            node = fetch_node(event.aggregate_id)
            vector = embedding_model.embed(build_embedding_text(node))   # domain quyết định field nào → text
            cfg = db.get_status_field_config(node.domain_id)
            status_value = node.properties.get(cfg.status_field_name) if cfg else None
            authority_score = map_authority(node, cfg) if cfg and cfg.authority_field_name else None
            qdrant.upsert("kg_vectors", id=node.external_ref or node.id, vector=vector,
                           payload=build_payload(node, status_value, authority_score))
        case "ACCESS_GRANT_CHANGED":
            qdrant.update_payload_bulk(filter=scope_filter, payload={"acl_visible_to": new_acl})

AccessSyncWorker:
    # Chạy mỗi khi access_grants thay đổi — điều phối cả Graph và Vector worker
    # đồng thời invalidate Redis cache liên quan (§3.4.2)
```

`compute_acl_visible_to(node)`:
```
function compute_acl_visible_to(node) -> List[str]:
    tokens = {f"{node.owner_tenant_id}:{node.owner_app_id}"}
    if node.visibility == "public":
        tokens.add("*:*")
    if node.visibility == "tenant_shared":
        tokens.add(f"{node.owner_tenant_id}:*")

    grants = db.query("""
        SELECT * FROM access_grants
        WHERE grantor_tenant_id = %s
          AND (grantor_app_id = %s OR grantor_app_id IS NULL)
          AND status = 'active'
          AND (scope_type = 'all' OR (scope_type = 'domain' AND scope_value = %s))
    """, node.owner_tenant_id, node.owner_app_id, node.domain_id)

    for g in grants:
        tokens.add(f"{g.grantee_tenant_id}:{g.grantee_app_id or '*'}")

    return list(tokens)
```

#### 3.4.8 StatusGate (generic — trước đây gọi "ComplianceGate", đổi tên vì không riêng pháp luật)

`StatusGate` là **no-op nếu domain không khai báo `domain_status_field_configs`** — không giả định mọi domain đều có khái niệm "hiệu lực"/"trạng thái". Với domain có khai báo, gate hoạt động generic dựa trên config, không biết trước domain đó là gì:

```
function status_gate(candidate_nodes: List[Node], domain_id) -> List[Node]:
    cfg = db.get_status_field_config(domain_id)
    if cfg is None or cfg.status_field_name is None:
        return candidate_nodes   # domain không có khái niệm lifecycle — pass-through toàn bộ

    passed = []
    for node in candidate_nodes:
        status = node.properties.get(cfg.status_field_name)
        if status in cfg.valid_status_values:
            passed.append(node)
        elif status in cfg.warning_status_values:    # domain tự khai báo, vd: "bi_sua_doi"
            node.warning = build_warning_from_cascade(node, cfg)
            passed.append(node)
        # các giá trị status khác (vd: domain pháp luật: "het_hieu_luc", "bi_bai_bo") → loại bỏ hoàn toàn
    return passed
```

#### 3.4.9 GraphRAGPipeline

8 bước, chi tiết xem `KG_Service_HLA_Ontology.md §6.2`. Điểm khác biệt khi multi-tenant: **bước 4 (Vector Search)** và **bước 3 (Graph Expansion)** đều nhận `acl_tokens` làm tham số bắt buộc, không có version "bỏ ACL" cho bất kỳ luồng nào — kể cả pipeline nội bộ.

### 3.5 Sequence Diagrams (textual)

#### 3.5.1 Write flow

```
App ──POST /v1/kg/write/nodes──▶ API Gateway
  │  Header: Authorization: Bearer {api_key}
  ▼
IdentityResolver.resolve(api_key) → (tenant_id, app_id)
  ▼
AuthZ: has_write_permission(tenant_id, app_id, domain_id)?
  │  No → 403 Forbidden
  ▼ Yes
NodeValidator.validate(domain_id, node_type, properties)
  │  Fail → 422 Unprocessable Entity (kèm chi tiết lỗi)
  ▼ OK
PostgreSQL transaction:
  SET LOCAL app.tenant_id, app.app_id
  INSERT kg_nodes (RLS tự enforce thêm 1 lớp bảo vệ)
  INSERT kg_outbox_events (NODE_UPSERTED)
  COMMIT
  ▼
202 Accepted { node_id, status: "processing" }
  ▼ (async, không block response)
GraphSyncWorker + VectorSyncWorker consume outbox → đồng bộ sang Graph DB / Qdrant
```

#### 3.5.2 Read flow (ví dụ: template `action-guide` của domain `sample-policy`)

```
Agent ──POST /v1/kg/read/template/sample-policy/action-guide──▶ API Gateway
  { topic_key: "returns" }
  ▼
IdentityResolver → (tenant_id, app_id)
  ▼
AccessResolver.resolve_visible_owners(tenant_id, app_id)
  → cache hit (60s) hoặc compute mới
  ▼
ReadService.execute_read(domain_id, "action-guide", params)
  → load pattern_spec từ domain_query_templates
  → QueryTemplateCompiler biên dịch DSL → Cypher, tự chèn ACL ở mọi hop
  ▼
Graph DB chạy query, timeout 3000ms
  ▼
AuditLogger.log(action="read", allowed=true)
  ▼
200 OK { results: [...] }
```

> Endpoint trên là **một ví dụ cụ thể** của domain pháp luật, không phải route cố định trong service. Bất kỳ domain nào đăng ký template qua Ontology API đều tự động có endpoint dạng `/v1/kg/read/template/{domain_id}/{template_name}` — xem §4.6.

#### 3.5.3 Cross-tenant access flow (tạo grant rồi search)

```
Admin TenantA ──POST /v1/access/grants──▶
  { grantee_tenant_id: B, scope_type: "domain", scope_value: "noi_bo_hop_dong",
    permission: "search", expires_at: "2027-01-01" }
  ▼
AccessGrant tạo trong PostgreSQL
  ▼
Publish event ACCESS_GRANT_CHANGED
  ├──▶ Invalidate Redis cache key acl:B:*
  └──▶ AccessSyncWorker: recompute acl_visible_to cho mọi node domain "noi_bo_hop_dong" của TenantA
        → Graph DB update + Qdrant payload update (bulk)

[Sau khi sync hoàn tất — thường < 5s]

TenantB/App1 ──POST /v1/kg/search/semantic──▶ { query: "..." , domain_ids: ["noi_bo_hop_dong"] }
  ▼
AccessResolver(B, App1) → bao gồm GrantedOwner(A, *, domain, noi_bo_hop_dong, search)
  ▼
Qdrant filter: acl_visible_to CONTAINS "B:App1" (đã được sync worker thêm vào payload)
  ▼
200 OK { results: [...] }   ← thấy được dữ liệu TenantA đã chia sẻ
```

### 3.6 Caching Strategy

| Cache key | TTL | Invalidate khi |
|---|---|---|
| `apikey:{hash}` → (tenant_id, app_id, status) | 30s | App bị revoke (xoá ngay, không chờ TTL) |
| `acl:{tenant_id}:{app_id}` → visible owners | 60s | AccessGrant tạo/revoke liên quan đến tenant/app đó |
| `ontology:effective:{tenant_id}:{app_id}` | 300s | Domain mới được tạo/share, ontology version bump |
| `domain_schema:{domain_id}:{version}` | Vô hạn (immutable theo version) | — |

### 3.7 Consistency & Idempotency

- **Outbox pattern** đảm bảo at-least-once delivery; mọi sync worker phải idempotent (dùng `MERGE` trong Cypher, `upsert` trong Qdrant theo `external_ref`/`node_id` cố định).
- **Reconciliation job** (chạy mỗi giờ): so sánh checksum giữa PostgreSQL (`kg_nodes` count theo domain) và Graph DB/Qdrant, cảnh báo nếu lệch > 0.1%.
- **Write amplification từ AccessGrant thay đổi:** nếu một domain có N node và bị share/unshare liên tục, mỗi lần đổi grant trigger recompute N node. Giới hạn: rate-limit số lần thay đổi grant trên 1 domain (vd: tối đa 10 lần/giờ) để tránh thrashing.

### 3.8 Error Handling

| Tình huống | Xử lý |
|---|---|
| Graph DB timeout/down | Read API fallback: trả lỗi `503` kèm `retry_after`; KHÔNG fallback sang vector-only cho query cần độ chính xác cao (Calculator) — chỉ fallback cho GraphRAG retrieval bước 3 |
| Qdrant timeout | Search API trả `503`; GraphRAG pipeline tiếp tục với kết quả graph traversal thuần (giảm recall, không sập toàn pipeline) |
| Validation lỗi ontology | `422` kèm danh sách lỗi cụ thể từng field |
| AccessGrant hết hạn giữa lúc xử lý request | Request đang chạy vẫn hoàn thành (đã resolve ACL ở đầu); request tiếp theo sẽ bị từ chối |
| Outbox event xử lý thất bại > 5 lần | Chuyển sang `status = FAILED`, đẩy vào dead-letter queue, alert on-call |

---

## 4. API Specification

### 4.0 Conventions chung

```
Base URL:        https://kg.legalai.internal/v1
Authentication:  Authorization: Bearer {api_key}
Content-Type:    application/json
```

**Error envelope chuẩn:**

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Missing required property: nguong_min",
    "details": [{ "field": "nguong_min", "issue": "required" }],
    "request_id": "req_8f3a2b1c"
  }
}
```

**Pagination** (cursor-based, áp dụng cho mọi list endpoint):

```
GET /v1/...?limit=20&cursor=eyJpZCI6...
→ { "data": [...], "next_cursor": "eyJpZCI6...", "has_more": true }
```

**Rate limit theo tenant tier:**

| Tier | Requests/phút | Burst |
|---|---|---|
| free | 60 | 10 |
| pro | 600 | 100 |
| enterprise | 6000 | 1000 |

Header trả về: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.

### 4.1 Tenant Management API

#### `POST /v1/tenants`

```
Auth: platform admin only
Request:
{
  "slug": "acme-accounting",
  "name": "ACME Accounting Co.",
  "tier": "pro"
}

Response 201:
{
  "id": "tenant_uuid",
  "slug": "acme-accounting",
  "name": "ACME Accounting Co.",
  "status": "active",
  "tier": "pro",
  "default_sharing_policy": "deny_all",
  "created_at": "2026-06-17T10:00:00Z"
}
```

#### `GET /v1/tenants/{tenant_id}`

```
Response 200: { ...tenant object... }
Response 404: { "error": { "code": "TENANT_NOT_FOUND" } }
```

#### `PUT /v1/tenants/{tenant_id}`

```
Request: { "tier": "enterprise", "default_sharing_policy": "share_within_tenant_read" }
Response 200: { ...updated tenant... }
```

#### `DELETE /v1/tenants/{tenant_id}`

```
Soft-suspend. Response 200: { "id": "...", "status": "suspended" }
```

### 4.2 App Management API

#### `POST /v1/tenants/{tenant_id}/apps`

```
Request:
{ "slug": "chatbot-web", "name": "Chatbot Web Widget", "type": "agent_consumer" }

Response 201:
{
  "id": "app_uuid",
  "tenant_id": "tenant_uuid",
  "slug": "chatbot-web",
  "type": "agent_consumer",
  "status": "active",
  "api_key": "kgsk_live_a1b2c3...",   ← CHỈ trả về MỘT LẦN lúc tạo, không lưu plaintext
  "created_at": "2026-06-17T10:05:00Z"
}
```

#### `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key`

```
Response 200: { "api_key": "kgsk_live_new...", "rotated_at": "..." }
```

#### `GET /v1/tenants/{tenant_id}/apps`

```
Response 200:
{ "data": [{ "id": "...", "slug": "chatbot-web", "type": "agent_consumer", "status": "active" }, ...] }
```

#### `DELETE /v1/tenants/{tenant_id}/apps/{app_id}`

```
Response 200: { "id": "...", "status": "revoked", "revoked_at": "..." }
```

### 4.3 Access & Sharing API

#### `POST /v1/access/grants`

```
Request:
{
  "grantor_app_id": "app_compliance_uuid",       (optional, null = cả tenant)
  "grantee_tenant_id": "tenant_b_uuid",
  "grantee_app_id": null,
  "scope_type": "domain",
  "scope_value": "noi_bo_hop_dong",
  "permission": "search",
  "expires_at": "2027-01-01T00:00:00Z"
}

Response 201:
{
  "id": "grant_uuid",
  "grantor_tenant_id": "tenant_a_uuid",
  "grantor_app_id": "app_compliance_uuid",
  "grantee_tenant_id": "tenant_b_uuid",
  "grantee_app_id": null,
  "scope_type": "domain", "scope_value": "noi_bo_hop_dong",
  "permission": "search",
  "status": "active",
  "expires_at": "2027-01-01T00:00:00Z",
  "created_at": "2026-06-17T10:10:00Z"
}

Response 400 (cross-tenant write grant không có expires_at):
{
  "error": { "code": "CROSS_TENANT_GRANT_REQUIRES_EXPIRY",
             "message": "Cross-tenant grant với permission write/admin yêu cầu expires_at" }
}
```

#### `GET /v1/access/grants?grantor_tenant_id=...&grantee_tenant_id=...`

```
Response 200: { "data": [...grant objects...], "next_cursor": null, "has_more": false }
```

#### `DELETE /v1/access/grants/{id}`

```
Response 200: { "id": "...", "status": "revoked", "revoked_at": "..." }
→ Trigger ngay AccessSyncWorker + invalidate cache (không async delay, xem S8)
```

#### `GET /v1/access/resolve`

```
Auth: app hiện tại (lấy từ token)
Response 200:
{
  "tenant_id": "tenant_b_uuid",
  "app_id": "app_x_uuid",
  "visible_domains": [
    { "domain_id": "luat_thue_hkd", "owner": "platform", "permission": "read" },
    { "domain_id": "noi_bo_hop_dong", "owner": "tenant_a_uuid", "permission": "search" }
  ]
}
```

#### `GET /v1/access/audit?resource_owner_tenant_id=...`

```
Response 200:
{
  "data": [
    { "requester_tenant_id": "tenant_b", "requester_app_id": "app_x",
      "action": "search", "resource_domain_id": "noi_bo_hop_dong",
      "allowed": true, "reason": "grant:grant_uuid", "created_at": "..." }
  ]
}
```

### 4.4 Ontology API

#### `POST /v1/tenants/{tenant_id}/ontology/domains`

```
Request: { "id": "noi_bo_hop_dong", "name": "Hợp đồng mẫu nội bộ", "status": "draft" }
Response 201: { ...domain object với owner_tenant_id = tenant_id... }
```

#### `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types`

```
Request:
{
  "node_type_name": "HopDongMau",
  "required_props": [
    { "name": "ten", "type": "string" },
    { "name": "loai_hop_dong", "type": "string" }
  ],
  "optional_props": [{ "name": "ghi_chu", "type": "string" }],
  "validation_rules": []
}
Response 201: { "id": "noi_bo_hop_dong.HopDongMau", "version": 1, ... }
```

#### `GET /v1/tenants/{tenant_id}/ontology/effective`

```
Response 200:
{
  "domains": [
    { "id": "van_ban_phap_luat", "owner": "platform" },
    { "id": "luat_thue_hkd", "owner": "platform" },
    { "id": "noi_bo_hop_dong", "owner": "tenant_a_uuid" }
  ]
}
```

#### `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates`

```
Request (Query Pattern DSL — không phải Cypher thô, xem §3.4.5):
{
  "template_name": "calculator",
  "pattern_spec": {
    "start": { "node_type": "NhomDoanhThu", "match": { "ma_nhom": "$ma_nhom" } },
    "hops": [
      { "rel_type": "CO_TY_LE", "to_node_type": "TyLeThue", "filter": { "nganh_code": "$nganh_code" } },
      { "rel_type": "QUY_DINH_BOI", "to_node_type": "Khoan" },
      { "rel_type": "CO_KHOAN", "direction": "in", "to_node_type": "Dieu" },
      { "rel_type": "BAO_GOM", "direction": "in", "to_node_type": "VanBanLuat", "filter_status": "valid_only" }
    ]
  },
  "param_schema": [{ "name": "ma_nhom", "type": "string" }, { "name": "nganh_code", "type": "string" }],
  "return_fields": ["TyLeThue.loai_thue", "TyLeThue.ty_le_pct", "VanBanLuat.so_hieu", "Dieu.so_dieu", "Khoan.so_khoan", "Khoan.noi_dung"]
}

Response 201: { "id": "luat_thue_hkd.calculator", "status": "draft", "version": 1 }

Response 422 (vượt giới hạn độ sâu):
{ "error": { "code": "TEMPLATE_TOO_DEEP", "message": "pattern_spec có 7 hop, vượt giới hạn 5" } }
```

#### `PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate`

```
Response 200: { "id": "luat_thue_hkd.calculator", "status": "active" }
```

#### `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config`

```
Request (tuỳ chọn — domain không có khái niệm lifecycle thì không cần gọi endpoint này):
{
  "status_field_name": "tinh_trang",
  "valid_status_values": ["con_hieu_luc"],
  "warning_status_values": ["bi_sua_doi"],
  "authority_field_name": "loai_van_ban",
  "authority_values_map": { "Luat": 4, "NghiDinh": 3, "ThongTu": 2, "CongVan": 1 },
  "cascade_rules": [
    { "from_node_type": "VanBanLuat", "via_rel": "BAO_GOM>CO_KHOAN", "to_node_type": "Khoan" }
  ]
}
Response 201: { "domain_id": "luat_thue_hkd", "status_field_name": "tinh_trang" }
```

#### `GET /v1/ontology/domains/{domain_id}`

```
Response 200: { ...domain + node_types[] + rel_types[] + query_templates[] + status_field_config }
Response 403: { "error": { "code": "NO_READ_ACCESS_TO_DOMAIN" } }   (nếu domain private và không có grant)
```

### 4.5 Data API — WRITE

#### `POST /v1/kg/write/nodes`

```
Request:
{
  "domain_id": "luat_thue_hkd",
  "node_type": "TyLeThue",
  "properties": {
    "id": "GTGT_thuong_mai", "loai_thue": "GTGT", "nganh_code": "thuong_mai",
    "ty_le_pct": 1.0, "ap_dung_nhom": ["N2"], "hieu_luc_tu": "2026-01-01"
  },
  "bridge_khoan_ids": ["khoan_uuid_dieu3_khoan1"]
}

Response 202:
{ "node_id": "node_uuid", "status": "processing", "sync_eta_ms": 800 }

Response 422:
{
  "error": {
    "code": "VALIDATION_FAILED",
    "details": [{ "field": "bridge_khoan_ids", "issue": "QUY_DINH_BOI required for this node type" }]
  }
}
```

#### `PUT /v1/kg/write/nodes/{id}`

```
Request: { "properties": { "ty_le_pct": 1.5 } }
Response 200: { "node_id": "...", "domain_version": 4, "status": "processing" }
```

#### `DELETE /v1/kg/write/nodes/{id}`

```
Response 200: { "node_id": "...", "is_deleted": true }
```

#### `POST /v1/kg/write/relationships`

```
Request:
{ "rel_type": "CO_TY_LE", "from_node_id": "nhom_n2_uuid", "to_node_id": "ty_le_uuid",
  "domain_id": "luat_thue_hkd", "properties": {} }
Response 201: { "relationship_id": "...", "status": "processing" }
```

#### `POST /v1/kg/write/ingest/document`

```
Request: { "file_url": "s3://...", "loai_van_ban": "NghiDinh", "domain_id": "luat_thue_hkd" }
Response 202: { "job_id": "job_uuid", "status": "queued" }
```

#### `GET /v1/kg/write/ingest/jobs/{job_id}`

```
Response 200: { "job_id": "...", "status": "completed", "nodes_created": 142, "errors": [] }
```

### 4.6 Data API — READ (Graph DB)

**Nguyên tắc:** không có route REST cố định riêng cho domain nào. Mọi template đã `status = "active"` trong `domain_query_templates` tự động có một route generic theo pattern `/v1/kg/read/template/{domain_id}/{template_name}` — route được resolve dynamic, không khai báo cứng trong router.

#### `POST /v1/kg/read/template/{domain_id}/{template_name}`

```
Request: { "params": { "ma_nhom": "N2", "nganh_code": "thuong_mai" } }

Response 200:
{
  "results": [
    { "loai_thue": "GTGT", "ty_le_pct": 1.0, "so_hieu": "117/2025/NĐ-CP",
      "so_dieu": 3, "so_khoan": "1", "noi_dung": "..." },
    { "loai_thue": "TNCN", "ty_le_pct": 0.5, "so_hieu": "117/2025/NĐ-CP",
      "so_dieu": 3, "so_khoan": "2", "noi_dung": "..." }
  ],
  "query_time_ms": 42
}

Response 404: { "error": { "code": "UNKNOWN_TEMPLATE" } }   (domain_id/template_name không tồn tại hoặc chưa active)
```

> **Ví dụ minh hoạ với domain pháp luật** (KHÔNG phải endpoint built-in của service — chỉ là kết quả của việc domain `luat_thue_hkd` đã đăng ký các template này qua Ontology API):
>
> ```
> POST /v1/kg/read/template/luat_thue_hkd/tax-routing
>   { "params": { "doanh_thu": 800000000 } }
>   → { "results": [{ "ma_nhom": "N2", "phuong_phap_tinh": "ty_le_tren_doanh_thu", "can_ke_khai": true }] }
>
> POST /v1/kg/read/template/luat_thue_hkd/calculator
>   { "params": { "ma_nhom": "N2", "nganh_code": "thuong_mai", "doanh_thu": 800000000 } }
>   → { "results": [
>         { "loai_thue": "GTGT", "ty_le_pct": 1.0, "thue_tinh": 8000000 },
>         { "loai_thue": "TNCN", "ty_le_pct": 0.5, "thue_tinh": 4000000 }
>       ], "tong_thue": 12000000 }
>
> POST /v1/kg/read/template/luat_thue_hkd/citation-check
>   { "params": { "chunk_ref": "vb_nd117_2025_ndcp_dieu3_khoan1" } }
>   → { "results": [{ "so_hieu": "117/2025/NĐ-CP", "status_value": "con_hieu_luc", "noi_dung": "..." }] }
> ```
>
> Một domain khác (vd: catalog sản phẩm) đăng ký template tên `find-substitutes` sẽ tự có route `/v1/kg/read/template/product_catalog/find-substitutes` mà không cần thay đổi gì trong service.

#### `GET /v1/kg/read/templates?domain_id=...`

```
Response 200:
{ "data": [{ "template_name": "calculator", "status": "active", "param_schema": [...] }, ...] }
```

#### `GET /v1/kg/read/nodes/{id}`

```
Response 200: { "id": "...", "node_type": "TyLeThue", "properties": {...}, "relationships": [...] }
Response 403: { "error": { "code": "NO_READ_ACCESS" } }
```

### 4.7 Data API — SEARCH (Vector DB)

#### `POST /v1/kg/search/semantic`

```
Request:
{ "query": "Hộ kinh doanh bán hàng online đóng thuế như thế nào?",
  "domain_ids": ["luat_thue_hkd"], "top_k": 10 }

Response 200:
{
  "results": [
    {
      "chunk_id": "vb_nd117_2025_ndcp_dieu3_khoan1",
      "score": 0.89,
      "content": "Hộ kinh doanh thực hiện hoạt động thương mại điện tử...",
      "so_hieu": "117/2025/NĐ-CP", "so_dieu": 3, "so_khoan": "1",
      "tinh_trang": "con_hieu_luc"
    }
  ],
  "search_time_ms": 78
}
```

#### `POST /v1/kg/search/rag`

```
Request: { "query": "HKD doanh thu 800 triệu ngành bán lẻ đóng thuế bao nhiêu?" }

Response 200:
{
  "answer_context": {
    "structured_data": { "ma_nhom": "N2", "tong_thue": 12000000, "items": [...] },
    "citations": [{ "so_hieu": "117/2025/NĐ-CP", "so_dieu": 3, "so_khoan": "1" }],
    "conflict_notes": [],
    "disclaimer": "Thông tin mang tính tham khảo, không thay thế ý kiến chuyên gia thuế."
  }
}
```

### 4.8 Admin/Integrity API

#### `GET /v1/kg/integrity/tenant/{tenant_id}`

```
Response 200:
{
  "checks": [
    { "id": "IC-04", "name": "TyLeThue thiếu bridge", "result": 0, "status": "pass" },
    { "id": "IC-12", "name": "Node thiếu domain_id", "result": 0, "status": "pass" }
  ],
  "overall": "pass"
}
```

#### `GET /v1/kg/integrity/missing-bridges?tenant_id=...`

```
Response 200: { "data": [{ "node_id": "...", "node_type": "TyLeThue", "domain_id": "..." }] }
```

### 4.9 MCP Server Spec

Đã đặc tả 6 tool gốc trong `KG_Service_MultiTenant_Design.md §8.2`; bổ sung `kg_list_templates` để agent tự phát hiện template khả dụng theo domain (cần thiết sau khi tổng quát hoá `kg_read_pattern`, không còn enum cố định). Tóm tắt input/output schema:

| Tool | Input | Output |
|---|---|---|
| `kg_search` | `{query, domain_ids?, top_k?}` | `{results: [{node_ref, score, content, domain_id}]}` |
| `kg_read_pattern` | `{domain_id, template_name, params}` ★ không có enum cố định — `template_name` hợp lệ tuỳ theo domain đã đăng ký qua Ontology API | Theo schema `return_fields` của template đó (xem §3.4.5, §4.6) |
| `kg_list_domains` | `{}` | `{domains: [{domain_id, owner, permission}]}` |
| `kg_list_templates` | `{domain_id}` | `{templates: [{template_name, param_schema}]}` — agent gọi tool này trước để biết template nào khả dụng, không cần biết trước tên |
| `kg_get_node` | `{node_type?, id}` | `{node, relationships: [...]}` |
| `kg_write_node` | `{domain_id, node_type, properties}` | `{node_id, status}` |
| `kg_check_access` | `{}` | `{visible_domains, visible_owners}` |

MCP server xác thực qua token tại thời điểm **kết nối** (connection-level), không qua tool call params — xem §6.1.

### 4.10 Error Code Reference

| Code | HTTP Status | Ý nghĩa |
|---|---|---|
| `INVALID_API_KEY` | 401 | API key không hợp lệ hoặc đã revoke |
| `FORBIDDEN` | 403 | Không có quyền thực hiện action |
| `NO_READ_ACCESS` | 403 | Không có quyền đọc resource cụ thể |
| `NO_READ_ACCESS_TO_DOMAIN` | 403 | Domain private, không có grant |
| `TENANT_NOT_FOUND` | 404 | Tenant không tồn tại |
| `VALIDATION_FAILED` | 422 | Dữ liệu không khớp ontology schema |
| `UNKNOWN_TEMPLATE` | 404 | Read template không tồn tại hoặc chưa active |
| `TEMPLATE_TOO_DEEP` | 422 | Query Pattern DSL vượt giới hạn số hop cho phép |
| `CROSS_TENANT_GRANT_REQUIRES_EXPIRY` | 400 | Grant cross-tenant write/admin thiếu expires_at |
| `DOMAIN_NOT_IN_EFFECTIVE_ONTOLOGY` | 422 | domain_id không thuộc effective ontology của app |
| `RATE_LIMIT_EXCEEDED` | 429 | Vượt rate limit theo tier |
| `GRAPH_DB_TIMEOUT` | 503 | Graph DB không phản hồi kịp |
| `INTERNAL_ERROR` | 500 | Lỗi không mong đợi, đã log request_id |

---

## 5. Non-Functional Requirements

| NFR | Mục tiêu | Đo bằng |
|---|---|---|
| Latency Read API | P95 < 200ms | APM theo template_name |
| Latency Search API | P95 < 300ms | APM |
| Latency GraphRAG full pipeline | P95 < 5s | Theo SRS §4.1, 500 concurrent |
| Availability | 99.5% | Uptime monitor |
| Write → sync latency | P95 < 2s (PostgreSQL commit → Graph DB/Qdrant visible) | Outbox lag metric |
| AccessGrant revoke → enforcement | < 5s end-to-end | Cache invalidation latency |
| Data consistency | Reconciliation lệch < 0.1% | Hourly job |

---

## 6. Security Design

Tóm tắt từ `KG_Service_MultiTenant_Design.md §9`, bổ sung chi tiết triển khai:

| # | Yêu cầu | Triển khai cụ thể |
|---|---|---|
| S1 | Identity không tin client | `IdentityResolver` chỉ đọc từ `Authorization` header, mọi field `tenant_id`/`app_id` trong request body (nếu có) bị **ignore** |
| S2 | Không raw query — áp dụng cả với domain owner | `ReadService` chỉ chấp nhận **Query Pattern DSL** (JSON) qua Ontology API, không nhận Cypher thô từ bất kỳ ai kể cả tenant admin; `QueryTemplateCompiler` luôn tự chèn ACL filter ở mọi hop khi biên dịch, domain owner không thể bỏ qua bước này; `SearchService` không nhận raw Qdrant filter từ client |
| S3 | RLS tại DB | Policy `kg_nodes_isolation` set qua `SET LOCAL`, transaction-scoped, không leak giữa request |
| S4 | ACL đồng bộ | Reconciliation job hourly + `AccessSyncWorker` đồng bộ tức thời khi grant đổi |
| S5 | Audit trail | `access_audit_log` ghi mọi access (allow + deny), partition theo tháng |
| S6 | Platform ontology bất biến với tenant | Route `/v1/platform/...` tách RBAC riêng, không chung middleware với `/v1/tenants/.../ontology` |
| S7 | Grant có hạn | API trả lỗi `400` nếu cross-tenant grant permission `write`/`admin` thiếu `expires_at` |
| S8 | Revoke lan toả nhanh | Cache invalidate đồng bộ (không qua queue) ngay trong handler `DELETE /v1/access/grants/{id}` |
| S9 | App revoke có hiệu lực ngay | Cache TTL `apikey:{hash}` chỉ 30s, đồng thời invalidate active ngay khi `DELETE /v1/tenants/.../apps/{id}` được gọi |

---

## 7. Phân kỳ triển khai

| Phase | Nội dung | Acceptance |
|---|---|---|
| A — Foundation | Tenant/App/AccessGrant schema + RLS, platform sentinel, AccessResolver, WriteService, QueryTemplateCompiler (engine generic) | Integration test: app chỉ thấy data của chính nó |
| B — Read/Search + ACL + Sample Ontology | acl_visible_to trên Graph DB/Qdrant, Sync Workers, ReadService + SearchService với ACL injection, **seed sample ontology + đăng ký 5 query template qua Ontology API** (không phải code service) | TenantA không thấy TenantB khi chưa grant; 5 template hoạt động đúng qua route generic |
| C — Sharing + MCP | Access & Sharing API đầy đủ, MCP Server 7 tools (kể cả `kg_list_templates`), Audit log API | Tạo grant → search thấy dữ liệu chia sẻ trong < 5s |
| D — Production hardening | Reconciliation job, rate limit theo tier, Qdrant sharding nếu cần, pentest, **thử triển khai một domain phi-pháp-luật để verify domain-agnostic** | NFR §5 đạt mục tiêu; pentest không tìm thấy cross-tenant escalation |

---

## 8. Phụ lục

### 8.1 Glossary

Xem §1.4.

### 8.2 Tài liệu tham chiếu

Xem §1.3.

### 8.3 Lịch sử thay đổi

| Phiên bản | Ngày | Nội dung |
|---|---|---|
| 1.0 | 17/06/2026 | Hợp nhất HLA + LLD + API Spec thành Technical Design Document đầy đủ, dựa trên Ontology v4 và Multi-Tenant Design v1 |
| 1.1 | 17/06/2026 | **Domain-agnostic refactor**: thay `READ_QUERY_TEMPLATES` hardcode bằng Query Pattern DSL + `QueryTemplateCompiler` generic; đổi `ComplianceGate` → `StatusGate` điều khiển bởi `domain_status_field_configs`; tổng quát hoá `kg_nodes.chunk_id/tinh_trang` → `external_ref/status_value`; gộp 2 Qdrant collection thành `kg_vectors` với `domain_props` mở; bỏ 5 route REST cứng `/agent/{name}` thay bằng `/read/template/{domain_id}/{template_name}` generic; sửa invariant P7, thêm P8; cập nhật glossary phân biệt rõ khái niệm core service vs ví dụ domain pháp luật |
