# Business Pain Points — Knowledge Graph Service

> **Project**: knowledge-graph-service  
> **Mục đích**: Tổng hợp pain points theo từng actor — product motivation cho việc phát triển tooling, AI assistance, và UX  
> **Ngày tạo**: 2026-08-03

---

## Tổng quan

Thư mục này chứa tài liệu phân tích **pain points** khi sử dụng Knowledge Graph Service, được tổ chức theo **actor** (từ PRD/URD). Mỗi file tập trung vào một actor cụ thể — phân tích ngữ cảnh, vấn đề thực tế, hệ quả kinh doanh, và lý do tại sao actor đó **phải** dùng kg-service.

---

## Files theo Actor

| File | Actor | Pain Points chính |
|:---|:---|:---|
| [kg-digitization.md](./kg-digitization.md) | Data Engineer, BA, Architect | Toàn bộ quá trình số hóa tài liệu phải làm thủ công (PP-KGD-01 → PP-KGD-08) |
| [platform-admin.md](./platform-admin.md) | Platform Admin | Quản lý tenant/key/grants qua raw API, không có admin CLI, rotation không automated |
| [tenant-admin.md](./tenant-admin.md) | Tenant Admin | Thiết kế ontology không có hỗ trợ, template lifecycle mù, visibility không rõ |
| [app-integrator.md](./app-integrator.md) | App Integrator / Developer | Schema discovery khó, projection lag không signal, auth model lạ, no mock mode |
| [agent-runtime-client.md](./agent-runtime-client.md) | AI Agent / Automation | MCP tool discovery kém, thiếu relevance score, token budget không được hỗ trợ |
| [operator-sre.md](./operator-sre.md) | Operator / SRE | Backend startup race, validation không CI/CD friendly, reconciliation drift thụ động |

---

## Cross-Actor Priority Matrix

| ID | Pain Point | Actor | Mức độ | Giải pháp cần có |
|:---|:---|:---|:---:|:---|
| PP-KGD-01 | Xác định ontology domain thủ công | Data Eng, BA, Arch | 🔴 P0 | `POST /v1/ontology/match` — AI ontology matching |
| PP-KGD-02 | Mở rộng ontology phức tạp, 3–10 ngày | Data Eng, Arch | 🔴 P0 | AI-Assisted RFC + `pikb-cli ontology extend` |
| PP-KGD-03 | Schema provision thủ công nhiều bước | Data Eng, Dev | 🔴 P0 | `pikb-cli provision --entity X --domain Y` auto-pipeline |
| PP-KGD-04 | Chuyển đổi tài liệu → nodes thủ công | Data Eng, BA | 🔴 P0 | `POST /v1/kg/write/ingest/batch` + AI extraction |
| PP-KGD-05 | Tạo relationships thủ công | Data Eng, Arch | 🔴 P0 | AI relationship suggestion API + Relationship Builder UI |
| PP-PA-01 | Không có admin portal/CLI | Platform Admin | 🔴 P0 | Admin CLI + bulk provisioning |
| PP-PA-02 | Key rotation không automated | Platform Admin | 🔴 P0 | Dual-key rotation + expiry policy |
| PP-TA-01 | Không có hỗ trợ thiết kế ontology | Tenant Admin | 🔴 P0 | Ontology templates + AI assist |
| PP-TA-02 | Query template lifecycle thiếu preview | Tenant Admin | 🔴 P0 | Template preview + versioning |
| PP-TA-03 | Không có visibility vào effective access | Tenant Admin | 🔴 P0 | Access summary API |
| PP-AI-01 | Không có schema discovery | App Integrator | 🔴 P0 | Node type schema endpoint |
| PP-AI-02 | Projection lag không có signal | App Integrator | 🔴 P0 | Projection status API + wait mode |
| PP-AI-03 | Auth model khác convention, dễ làm sai | App Integrator | 🔴 P0 | `/v1/access/me` endpoint + SDK |
| PP-ARC-01 | MCP tool discovery không đủ cho agent | Agent Client | 🔴 P0 | Rich tool descriptions + context init |
| PP-ARC-02 | Semantic search thiếu relevance score | Agent Client | 🔴 P0 | Score + match_reason in response |
| PP-ARC-03 | Không có token-budget-aware retrieval | Agent Client | 🔴 P0 | Token budget param + summarization |
| PP-OPS-01 | Backend startup race condition | Operator/SRE | 🔴 P0 | Wait-for-backend mode + startup probe |
| PP-OPS-02 | Validation không có PASS/FAIL output | Operator/SRE | 🔴 P0 | `kg-validate` tool với exit code |
| PP-OPS-03 | Reconciliation drift không proactive detect | Operator/SRE | 🔴 P0 | Scheduled recon + drift alerts |
| PP-KGD-06 | Mapping nhiều nguồn data phức tạp | Data Eng, Arch | 🟠 P1 | AI Entity Resolution + `POST /v1/kg/entity/resolve` |
| PP-KGD-07 | Không có observability khi digitize | Data Eng, PO | 🟠 P1 | Ingest Job Dashboard + `GET /v1/kg/jobs/{id}` |
| PP-PA-03 | Cross-tenant grants thủ công | Platform Admin | 🟠 P1 | Grant templates + expiry |
| PP-PA-04 | Health không granular | Platform Admin | 🟠 P1 | Per-subsystem health + alerting |
| PP-AI-04 | Templates là black box | App Integrator | 🟠 P1 | Template self-documentation |
| PP-AI-05 | Không có offline/mock mode | App Integrator | 🟠 P1 | Memory profile + test SDK |
| PP-AI-06 | Error messages không actionable | App Integrator | 🟠 P1 | Structured errors + fix hints |
| PP-ARC-04 | REST và MCP format khác nhau | Agent Client | 🟠 P1 | Canonical format + unified SDK |
| PP-OPS-04 | Multi-env config không có isolation guard | Operator/SRE | 🟠 P1 | Env validation + environment tagging |
| PP-OPS-05 | Không có correlation ID xuyên write→projection | Operator/SRE | 🟠 P1 | Distributed tracing + W3C Trace Context |
| PP-KGD-08 | Tribal knowledge, không có documentation | Tất cả | 🟡 P2 | Digitization Playbook + CLI interactive wizard |
| PP-TA-04 | Lifecycle rules không có tooling | Tenant Admin | 🟡 P2 | Lifecycle editor + enforcement |
| PP-TA-05 | Onboard app mới phải support thủ công | Tenant Admin | 🟡 P2 | Auto-generated docs + sandbox |
| PP-PA-05 | Không có usage visibility | Platform Admin | 🟡 P2 | Usage metrics API |
| PP-ARC-05 | Không có caching → repeated queries tốn kém | Agent Client | 🟡 P2 | HTTP cache headers + Redis read cache |

---

## Ước tính impact nếu giải quyết được

| Công đoạn | Thủ công | Với tool | Tiết kiệm |
|:---|:---:|:---:|:---:|
| Xác định ontology domain | 2–4 giờ | < 5 phút | **96%** |
| Mở rộng ontology | 3–10 ngày | 2–4 giờ | **90%** |
| Provision schema + register | 2–4 giờ | < 1 phút | **99%** |
| Chuyển đổi 100 docs → nodes | 5–10 ngày | 1–2 giờ | **95%** |
| Tạo relationships | 3–7 ngày | 4–8 giờ | **85%** |
| Mapping nhiều nguồn data | 5–15 ngày | 1–3 ngày | **80%** |
| Onboard tenant mới (Platform Admin) | 1–2 ngày | < 30 phút | **90%** |
| Onboard App Integrator vào domain | 1 tuần | 1 ngày | **80%** |
| Deploy và validate | 4–8 giờ | < 1 giờ | **85%** |
| Debug production incident | 2–4 giờ | 20–30 phút | **80%** |
| **Tổng 1 domain vừa (end-to-end)** | **3–6 tuần** | **2–4 ngày** | **~90%** |

---

## Tài liệu tham chiếu

- [PRD — Knowledge Graph Service](../requirements/prd.md)
- [URD — Knowledge Graph Service](../requirements/urd.md)
- [SRS — Knowledge Graph Service](../requirements/srs.md)
- [API Documentation](../api/)

## Cross-project references

- **vnp-ontology**: [document-scatter.md](../../../vnp-ontology/docs/business/painpoints/document-scatter.md)
- **vnp-products**: [kg-digitization stub](../../../vnp-products/docs/business/painpoints/kg-digitization.md)
