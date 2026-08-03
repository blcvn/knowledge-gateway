# Business Solutions — Knowledge Graph Service

> **Project**: knowledge-graph-service  
> **Mục đích**: Giải pháp cho từng pain point theo actor — ánh xạ rõ giải pháp đã có vs. đề xuất mới  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Phương pháp phân tích

Mỗi solution file được tổ chức theo actor, cho từng pain point:

| Ký hiệu | Ý nghĩa |
|:---:|:---|
| ✅ **Đã có** | Sản phẩm hiện tại đã hỗ trợ — cần tận dụng và document tốt hơn |
| 🔧 **Cần bổ sung** | Skeleton/infrastructure có sẵn, cần hoàn thiện logic |
| 🆕 **Đề xuất mới** | Tính năng chưa có, cần phát triển |

---

## Files theo Actor

| File | Actor | Pain Points giải quyết |
|:---|:---|:---|
| [platform-admin.md](./platform-admin.md) | Platform Admin | PP-PA-01 → PP-PA-05 (5 PPs) |
| [tenant-admin.md](./tenant-admin.md) | Tenant Admin | PP-TA-01 → PP-TA-05 (5 PPs) |
| [app-integrator.md](./app-integrator.md) | App Integrator | PP-AI-01 → PP-AI-06 (6 PPs) |
| [agent-runtime-client.md](./agent-runtime-client.md) | AI Agent / MCP Client | PP-ARC-01 → PP-ARC-05 (5 PPs) |
| [operator-sre.md](./operator-sre.md) | Operator / SRE | PP-OPS-01 → PP-OPS-05 (5 PPs) |

> Pain Points từ KG Digitization (PP-KGD-01 → PP-KGD-08) xem tại [kg-digitization.md](../painpoints/kg-digitization.md) — solutions đang được phát triển trong PIKB AI-Assisted Digitization.

---

## Master Solution Map

### ✅ Giải pháp đã có — Tận dụng ngay

| Pain Point | Giải pháp hiện có | API / Docs |
|:---|:---|:---|
| PP-PA-01: Không có admin portal | Full REST API cho tenant/app lifecycle | `POST /v1/tenants`, `POST /v1/tenants/{t}/apps` |
| PP-PA-02: Key rotation | rotate-key + delete app endpoints | `POST /v1/tenants/{t}/apps/{a}/rotate-key` |
| PP-PA-03: Cross-tenant grants | Full grants CRUD | `POST /v1/access/grants`, `GET /v1/access/audit` |
| PP-PA-04: Health monitoring | /healthz + metrics endpoint | `GET /healthz`, `GET /v1/kg/metrics` |
| PP-TA-01: Ontology design | Full ontology CRUD API | `POST /v1/tenants/{t}/ontology/domains` |
| PP-TA-02: Template lifecycle | Register + activate flow | `POST` + `PUT .../query-templates/{name}/activate` |
| PP-TA-03: Effective access visibility | access/resolve + grants list | `GET /v1/access/resolve`, `GET /v1/access/grants` |
| PP-TA-04: Lifecycle rules | status-field-config API | `POST .../status-field-config` |
| PP-TA-05: Domain documentation | Full domain spec endpoint | `GET /v1/ontology/domains/{id}` |
| PP-AI-01: Schema discovery | Domain spec với node type attributes | `GET /v1/ontology/domains/{id}` |
| PP-AI-02: Projection lag signal | realtime mode + 202 Accepted | `?mode=realtime`, integrity endpoints |
| PP-AI-03: Auth model clarity | access/resolve + middleware strip | `GET /v1/access/resolve` |
| PP-AI-04: Template discovery | List templates + domain spec | `GET /v1/kg/read/templates` |
| PP-AI-05: Offline dev mode | GRAPH_ADAPTER=memory + compose-integration | `make deploy-compose-integration` |
| PP-AI-06: Error messages | Error envelope format + troubleshooting | Structured error + troubleshooting guide |
| PP-ARC-01: MCP tool set | 9 MCP tools + tools/list discovery | `GET /v1/mcp/connect`, tools/list |
| PP-ARC-02: Semantic search | semantic/rag/hybrid search endpoints | `POST /v1/kg/search/semantic` |
| PP-ARC-03: Result scoping | top_k + domain_ids filtering | `top_k`, `domain_ids` params |
| PP-ARC-04: Shared identity | Same API key REST và MCP | MCP connect với same Bearer token |
| PP-ARC-05: Embedding cache | EMBEDDING_CACHE_TTL_S | env var config |
| PP-OPS-01: Startup ordering | Compose migration ordering + /healthz | `make deploy-compose-integration` |
| PP-OPS-02: Validation scripts | make integration-test + validate-runtime-profile | Exit code 0/1, CI/CD compatible |
| PP-OPS-03: Reconciliation | integrity endpoints + repair + runbooks | `GET /v1/kg/integrity/tenant/{t}`, repair endpoints |
| PP-OPS-04: Env config | Profile-based config + per-env .env | KG_RUNTIME_PROFILE |
| PP-OPS-05: Structured logging | node_id, tenant_id trong logs | Existing log format |

---

### 🆕 Đề xuất mới — Cần phát triển

Nhóm theo **priority** và **effort**:

#### 🔴 P0 — Quick wins (giải quyết adoption blocker ngay)

| PP | Đề xuất | Effort |
|:---|:---|:---:|
| PP-PA-01 | `kg-admin` CLI wrapper cho REST API | S |
| PP-TA-01 | Ontology starter templates (`GET /v1/ontology/starter-templates`) | S |
| PP-TA-02 | Template preview mode (`POST /v1/ontology/templates/preview`) | M |
| PP-TA-03 | Tenant access summary (`GET /v1/tenants/{t}/access/summary`) | S |
| PP-AI-01 | Node type JSON Schema endpoint (`GET .../node-types/{type}/schema`) | S |
| PP-AI-01 | Write validation dry-run (`POST /v1/kg/write/nodes/validate`) | S |
| PP-AI-02 | Projection status per node (`GET .../nodes/{id}/projection-status`) | M |
| PP-AI-03 | `/v1/access/me` alias + richer response | XS |
| PP-AI-06 | Enriched VALIDATION_FAILED với field + fix_hint + request ID | S |
| PP-ARC-01 | Rich MCP tool descriptions với examples và when_to_use | S |
| PP-ARC-01 | MCP context initialization event khi session open | M |
| PP-ARC-02 | Relevance score trong semantic search response | M |
| PP-ARC-03 | `token_budget` parameter + `format` modes | M |
| PP-OPS-01 | `KG_STARTUP_WAIT_BACKENDS=true` mode | S |
| PP-OPS-01 | Degraded mode capability table (documentation) | XS |
| PP-OPS-02 | `kg-validate` CLI với structured JSON output + exit codes | M |
| PP-OPS-03 | Prometheus metrics export (`GET /v1/kg/metrics/prometheus`) | M |
| PP-OPS-03 | Scheduled reconciliation job + alert webhook | M |

#### 🟠 P1 — Important (cải thiện production experience)

| PP | Đề xuất | Effort |
|:---|:---|:---:|
| PP-PA-02 | Dual-key grace period rotation | M |
| PP-PA-03 | Grant templates + expiry | M |
| PP-PA-04 | `GET /v1/health/detailed` per-subsystem | S |
| PP-TA-02 | Template versioning | L |
| PP-TA-03 | Access simulation (`POST /v1/access/simulate`) | M |
| PP-AI-02 | Wait-for-projection option (`?wait_for_projection=true`) | M |
| PP-AI-04 | Template spec endpoint với examples | S |
| PP-AI-05 | Official test SDK (`KGClientMock`) | L |
| PP-ARC-02 | Min score threshold + include_relationships | S |
| PP-ARC-03 | Context package endpoint | L |
| PP-ARC-04 | Canonical JSON format REST=MCP | M |
| PP-OPS-03 | Auto-reconcile trigger sau backend reconnect | M |
| PP-OPS-04 | Environment tagging (`KG_ENVIRONMENT` var) | XS |
| PP-OPS-04 | Config linting script (`validate-config.sh`) | S |
| PP-OPS-05 | W3C Trace Context propagation + request ID header | M |
| PP-OPS-05 | OpenTelemetry export | L |

#### 🟡 P2 — Nice to have

| PP | Đề xuất | Effort |
|:---|:---|:---:|
| PP-PA-05 | Usage metrics per tenant | M |
| PP-TA-04 | Lifecycle transition rules + validation API | L |
| PP-TA-05 | Domain docs endpoint + sandbox environment | L |
| PP-AI-05 | Local-dev Docker Compose profile (memory backends) | XS |
| PP-ARC-05 | HTTP cache headers + Redis read-through cache | M |
| PP-OPS-05 | Trace lookup endpoint (`GET /v1/admin/traces/{id}`) | L |

> **Effort key**: XS < 1 day | S = 1-3 days | M = 1-2 weeks | L = 2-4 weeks

---

## Nguồn tham chiếu

### Giải pháp đã có — Docs chính

| Resource | Link |
|:---|:---|
| API Reference | [docs/api/README.md](../../api/README.md) |
| Integration Workflows | [docs/guides/integration.md](../../guides/integration.md) |
| MCP Integration | [docs/guides/mcp.md](../../guides/mcp.md) |
| Quickstart | [docs/guides/quickstart.md](../../guides/quickstart.md) |
| Troubleshooting | [docs/guides/troubleshooting.md](../../guides/troubleshooting.md) |
| Testing Guide | [docs/guides/testing.md](../../guides/testing.md) |
| Environment Variables | [docs/deployment/environment.md](../../deployment/environment.md) |
| Docker Compose Deploy | [docs/deployment/compose.md](../../deployment/compose.md) |
| Reconciliation Runbook | [docs/operations/reconciliation-incident-handling.md](../../operations/reconciliation-incident-handling.md) |
| Replica Recovery | [docs/operations/replica-recovery.md](../../operations/replica-recovery.md) |

### Pain Points nguồn

| File | Actor |
|:---|:---|
| [painpoints/platform-admin.md](../painpoints/platform-admin.md) | Platform Admin |
| [painpoints/tenant-admin.md](../painpoints/tenant-admin.md) | Tenant Admin |
| [painpoints/app-integrator.md](../painpoints/app-integrator.md) | App Integrator |
| [painpoints/agent-runtime-client.md](../painpoints/agent-runtime-client.md) | Agent Runtime Client |
| [painpoints/operator-sre.md](../painpoints/operator-sre.md) | Operator / SRE |
| [painpoints/kg-digitization.md](../painpoints/kg-digitization.md) | Data Engineer, BA, Architect |
