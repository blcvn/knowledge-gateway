# Pain Points — Platform Admin

> **Actor**: Platform Admin  
> **Phạm vi**: Người quản lý toàn bộ platform — tạo/quản lý tenants, apps, API keys, access grants; đảm bảo service reachable và healthy  
> **Loại**: Platform governance & Multi-tenancy pain points  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Tổng quan

Platform Admin là **người cổng** của toàn bộ hệ thống — mọi tenant muốn dùng kg-service đều phải qua Platform Admin để được provisioned. Với multi-tenant architecture, Platform Admin cần quản lý nhiều tenants/apps/keys đồng thời trong khi đảm bảo isolation và security.

Hiện tại, kg-service không có admin portal, không có management CLI đầy đủ — mọi thao tác đều qua raw API calls. Đây là **bottleneck lớn nhất** cho platform scaling.

---

## Pain Points chi tiết

### PP-PA-01 — Không có tenant/app management portal — mọi thao tác qua raw API calls

**Mức độ**: 🔴 Critical  
**Tần suất**: Daily — mỗi khi onboard tenant mới hoặc manage existing tenants  

**Mô tả**:  
PRD explicitly nói "Non-Goals: Building a consumer UI or admin portal in this repository." Điều này đúng về scope sản phẩm — nhưng có nghĩa là Platform Admin phải làm việc với:

```bash
# Onboard một tenant mới: 5 API calls thủ công
curl -X POST /v1/tenants -d '{"name": "payment-team", "display_name": "Payment Team"}'
# Lưu tenant_id

curl -X POST /v1/tenants/{tenant_id}/apps -d '{"name": "payment-service"}'
# Lưu app_id VÀ api_key → phải share api_key an toàn với team

curl -X GET /v1/access/resolve -H "Authorization: Bearer {api_key}"
# Verify identity resolved đúng

curl -X POST /v1/tenants/{tenant_id}/grants -d '{"target_tenant": "...", "domains": [...]}'
# Setup grants nếu cần

curl -X POST /v1/tenants/{tenant_id}/apps/{app_id}/keys/rotate
# Nếu cần rotate key sau đó
```

Khi có 50 tenants và 200 apps → tracking state của tất cả qua curl là không feasible.

**Hệ quả kinh doanh**:
- Platform Admin trở thành bottleneck: mọi onboarding request phải queue qua một người
- Human error: sai tenant_id, quên activate grant, share api_key không an toàn
- Không có audit trail của admin actions
- Không có bulk operations: phải setup 20 apps một lúc → 20 × 4 API calls = 80 manual calls

**Giải pháp cần có**:
- Admin CLI: `kg-admin tenant create`, `kg-admin app provision`, `kg-admin grants list`
- Bulk provisioning: `kg-admin provision --from-config tenants.yaml`
- Audit log: `GET /v1/admin/audit?action=tenant.create&since=2026-01-01`
- Secure key delivery: API key delivery via secure channel, không qua email/Slack

---

### PP-PA-02 — Key rotation và revocation không có automated rollout

**Mức độ**: 🔴 Critical  
**Tần suất**: Khi security incident hoặc periodic rotation policy  

**Mô tả**:  
Khi một API key bị compromised hoặc cần rotate theo security policy:

```
Tình huống: payment-service API key bị leak lên GitHub

Bước 1: Revoke key ngay
→ POST /v1/tenants/{t}/apps/{a}/keys/revoke
→ Ngay lập tức payment-service không gọi được kg-service

Bước 2: Generate key mới  
→ POST /v1/tenants/{t}/apps/{a}/keys/rotate
→ Nhận new_api_key

Bước 3: Deliver new key đến payment-service team
→ Không có secure channel built-in
→ Phải dùng Slack/email → bản thân đây là security risk

Bước 4: payment-service team update key trong config
→ Deployment cần vài phút đến vài giờ
→ Trong khoảng thời gian đó: service DOWN

Bước 5: Verify new key hoạt động
→ Phải coordinate với payment-service team
→ Không có automated health check sau rotation
```

Không có:
- Graceful rotation: new_key và old_key cùng valid trong 24h transition window
- Automated notification đến app owner khi key rotate
- Key health monitoring: alert khi key không được dùng trong X ngày

**Hệ quả kinh doanh**:
- Security incident response slow → window of exposure dài hơn cần thiết
- Key rotation gây downtime → teams không muốn rotate → aging keys
- Key delivery không secure → leak lại

**Giải pháp cần có**:
- Dual-key rotation: `POST /v1/keys/rotate?grace_period_hours=24` — cả hai key valid trong grace period
- Webhook: notify app owner khi key sắp expire hoặc đã rotate
- Key expiry policy: `POST /v1/keys/{id}/policy --expiry-days 90 --auto-rotate true`

---

### PP-PA-03 — Cross-tenant access grants không có policy template — phải define từng grant thủ công

**Mức độ**: 🟠 High  
**Tần suất**: Khi setup collaboration giữa tenants  

**Mô tả**:  
Nhiều tenants cần share knowledge với nhau (vd: "payment-team" share risk domain với "compliance-team"). Hiện tại:

```
Không có policy template:
→ Mỗi grant phải define thủ công: tenant A → tenant B → domain D → permissions [read]
→ Khi có 10 tenants và mỗi cặp cần grants → tổ hợp grants phức tạp
→ Không có way để define: "compliance-team có quyền đọc tất cả risk-related domains từ mọi tenants"

Không có grant visibility:
→ GET /v1/grants → flat list, không có visualization
→ "App X hiện đang nhận data từ những tenants nào?" → phải query và filter

Không có grant lifecycle:
→ Grants tồn tại mãi → không có expiry
→ Không có way để grant tạm thời: "share trong 30 ngày cho audit purpose"
→ Grant leak: tenant share nhầm → không biết → data exposure
```

**Hệ quả kinh doanh**:
- Over-granting để tránh phải configure nhiều lần → data visibility rộng hơn cần thiết
- Grant sprawl: sau 1 năm, không ai biết grant nào còn cần thiết
- Compliance risk: không có audit trail về "ai được xem data gì và khi nào"

**Giải pháp cần có**:
- Grant templates: `POST /v1/grants/templates` — define reusable access policies
- Grant expiry: `POST /v1/grants --expires-at "2026-12-31"`
- Grant graph: visualization của who-can-see-what
- Grant review: quarterly prompt "Review và confirm các grants này còn hợp lệ không?"

---

### PP-PA-04 — Service health không đủ granular — biết "up" nhưng không biết "degraded"

**Mức độ**: 🟠 High  
**Tần suất**: Khi diagnose performance issues hoặc backend connectivity problems  

**Mô tả**:  
SRS yêu cầu health endpoints. Nhưng có sự khác biệt giữa:
- **UP**: Service respond HTTP 200 → "alive"
- **DEGRADED**: Graph backend laggy → graph queries slow nhưng write vẫn work
- **PARTIAL**: Vector backend down → semantic search fail nhưng graph read vẫn work

Hiện tại, nếu graph backend có latency issue:
- `/health` vẫn trả về 200 → monitoring không alert
- App integrators thấy slow queries → tạo support ticket
- Platform Admin không có dashboard để correlate: "Ah, graph backend latency spike vào 14:30, explain why payment-service templates chậm"

**Hệ quả kinh doanh**:
- MTTR cao: phát hiện degradation muộn vì monitoring không granular
- "Works for write, broken for search" — hard to diagnose without per-subsystem health

**Giải pháp cần có**:
- `GET /v1/health/detailed` — per-subsystem health: postgres_ok, redis_ok, graph_latency_ms, vector_latency_ms
- Alerting hooks: webhook khi backend latency vượt threshold
- Historical health metrics endpoint: "graph backend latency trong 24h qua"

---

### PP-PA-05 — Không có usage visibility — không biết tenant nào dùng gì và bao nhiêu

**Mức độ**: 🟡 Medium  
**Tần suất**: Monthly — khi planning capacity hoặc billing  

**Mô tả**:  
Platform Admin cần biết:
- Tenant nào đang grow nhanh nhất? → capacity planning
- Domain nào được query nhiều nhất? → performance optimization priority
- API key nào không được dùng trong 90 ngày? → cleanup/security
- Tenant nào sắp hết quota? → billing/throttling

Hiện tại không có:
- Usage metrics per tenant/app
- API call volume history
- Storage usage per tenant (node count, relationship count)
- Query performance by template

**Hệ quả kinh doanh**:
- Capacity planning không data-driven → over/under-provision
- Zombie tenants (không dùng nhưng vẫn tốn resources) không được cleanup
- Không có basis để charge tenants theo usage

**Giải pháp cần có**:
- `GET /v1/admin/usage?tenant={t}&period=30d` — nodes written, queries executed, storage used
- Usage dashboard per tenant
- Inactivity alerts: "App X hasn't made any API calls in 30 days"

---

## Ma trận Pain Points — Platform Admin

| ID | Pain Point | Mức độ | Impact | Giải pháp cần có |
|:---|:---|:---:|:---|:---|
| PP-PA-01 | Không có admin portal/CLI | 🔴 | Bottleneck, human error | Admin CLI + bulk provisioning |
| PP-PA-02 | Key rotation không automated | 🔴 | Security risk, downtime | Dual-key rotation + expiry policy |
| PP-PA-03 | Cross-tenant grants thủ công | 🟠 | Over-granting, grant sprawl | Grant templates + expiry |
| PP-PA-04 | Health không granular | 🟠 | Slow incident detection | Per-subsystem health + alerting |
| PP-PA-05 | Không có usage visibility | 🟡 | Capacity misjudgment | Usage metrics API |

---

## Tại sao Platform Admin phải dùng kg-service

1. **Multi-tenancy với isolation built-in**: Không cần implement ACL cho từng service riêng — kg-service enforce tenant isolation at the infrastructure level
2. **Single control plane cho knowledge**: Thay vì quản lý knowledge access trên nhiều systems (Confluence permissions, database ACL, S3 bucket policies...) — một nơi, một grant model
3. **API-first management**: Mọi thao tác có thể automate qua API → integrate vào IDP/ITSM workflows của công ty
4. **Auditable**: Mọi access decision đều traceable qua access-resolution API — compliance requirement được đáp ứng

> **Kết luận**: Platform Admin chịu đau nhiều nhất ở phase mở rộng — khi service tăng từ 5 tenants lên 50. Giải quyết PP-PA-01 (admin CLI) trước sẽ unblock toàn bộ scaling pain points còn lại.
