# Business Logic — Platform Admin

> **Actor**: Platform Admin  
> **Vai trò**: Quản trị viên toàn hệ thống — tạo/xóa tenant, quản lý app, kiểm soát access grants  
> **Phạm vi quyền**: Cao nhất — có thể thao tác trên mọi tenant

---

## Nghiệp vụ 1: Quản lý Tenant

### BL-PA-01: Tạo Tenant

**Mô tả**: Khi có khách hàng mới hoặc team mới cần sử dụng hệ thống, Platform Admin tạo một tenant để cô lập dữ liệu.

**Điều kiện tiên quyết**:
- Caller phải có quyền platform admin
- `slug` phải duy nhất trong toàn hệ thống

**Business Rules**:
- `slug` không thay đổi sau khi tạo (immutable identifier)
- Mỗi tenant được gán `tier` (free / pro / enterprise) — ảnh hưởng đến rate limit
- `default_sharing_policy` quyết định mặc định có chia sẻ data giữa apps trong cùng tenant không
- Tenant mới luôn có `status = active` khi tạo

**Luồng**:
```
Input: { slug, name, tier }
  → Validate slug không trùng
  → Tạo tenant với status = active
  → Trả về tenant object (không có API key — tenant chưa có app)
Output: tenant_id, slug, status = "active"
```

**Kết quả mong đợi**: Tenant được tạo, chưa có app, chưa có data, chưa dùng được service.

---

### BL-PA-02: Suspend / Unsuspend Tenant

**Mô tả**: Khi tenant vi phạm điều khoản hoặc chưa thanh toán, Platform Admin có thể tạm dừng toàn bộ hoạt động.

**Business Rules**:
- Suspend tenant → mọi API key của tenant đều trả 401
- Unsuspend → mọi API key active lại hoạt động
- Data không bị xóa khi suspend
- Suspend là soft operation — có thể undo

---

## Nghiệp vụ 2: Quản lý App & API Key

### BL-PA-03: Tạo App và cấp API Key

**Mô tả**: Mỗi ứng dụng hoặc service cần API key riêng để xác thực. Platform Admin tạo app dưới một tenant.

**Business Rules**:
- Mỗi app thuộc về **một tenant duy nhất** — không share app giữa tenant
- API key chỉ được **hiển thị một lần duy nhất** ngay lúc tạo — sau đó không thể recover
- Hệ thống chỉ lưu **SHA256 hash** của key, không bao giờ lưu plaintext
- Mỗi tenant có thể có nhiều app với `type` khác nhau:
  - `agent_consumer` — đọc và search
  - `ingestion_producer` — ghi dữ liệu
  - `admin_tool` — quản trị
  - `hybrid` — cả hai

**Luồng**:
```
Input: { tenant_id, slug, name, type }
  → Validate slug không trùng trong cùng tenant
  → Generate plaintext API key (kgsk_live_xxx)
  → Hash key (SHA256) → lưu hash
  → Trả về { api_key: "kgsk_live_xxx" }  ← CHỈ LẦN NÀY
Output: app_id, api_key (plaintext, one-time only)
```

> ⚠️ **Lưu key ngay**: Sau khi tạo, nếu mất key phải thực hiện rotate — không thể xem lại.

---

### BL-PA-04: Rotate API Key

**Mô tả**: Khi key bị lộ hoặc theo chính sách bảo mật định kỳ, cần tạo key mới.

**Business Rules**:
- Rotate tạo ra key hoàn toàn mới — key cũ bị vô hiệu **ngay lập tức** (cache cleared)
- Không có grace period — chỉ có thể dùng một key tại một thời điểm
- App vẫn giữ nguyên `app_id` sau khi rotate
- Key mới cũng chỉ hiển thị một lần

**Luồng**:
```
Input: { tenant_id, app_id }
  → Generate new key
  → Hash → update apps.api_key_hash
  → Xóa cache cũ ngay lập tức
  → Trả về { api_key: "kgsk_live_newxxx" }
Output: new api_key (plaintext, one-time only)
Effect: old key → 401 ngay lập tức
```

---

### BL-PA-05: Revoke App

**Mô tả**: Ngừng hoàn toàn một app — không cho phép access bằng key của app đó.

**Business Rules**:
- Revoke là **vĩnh viễn** — không thể undo (phải tạo app mới)
- Key bị vô hiệu trong < 30 giây (30s TTL + immediate cache clear)
- Data do app tạo ra vẫn còn trong hệ thống (soft delete, data không mất)
- App bị revoke vẫn xuất hiện trong danh sách với `status = revoked`

---

## Nghiệp vụ 3: Quản lý Access Grants

### BL-PA-06: Cấp quyền Cross-Tenant (Access Grant)

**Mô tả**: Khi Tenant A muốn chia sẻ dữ liệu với Tenant B, Platform Admin (hoặc Tenant Admin) tạo AccessGrant.

**Business Rules**:

| Rule | Chi tiết |
|:---|:---|
| **Scope** | Grant có thể theo domain, node_type, dataset_tag, hoặc all |
| **Permission** | read / search / write / admin |
| **Expiry bắt buộc** | Grant `write` hoặc `admin` cross-tenant **bắt buộc** có `expires_at` |
| **Hiệu lực** | Grant có hiệu lực < 5 giây sau khi tạo (sync workers cập nhật acl_visible_to) |
| **Revoke ngay** | Revoke grant có hiệu lực **đồng bộ** (không qua queue) → grantee mất access tức thì |

**Luồng tạo grant**:
```
Input: { grantor_tenant_id, grantor_app_id?, grantee_tenant_id, grantee_app_id?,
         scope_type, scope_value, permission, expires_at? }
  → Validate: write/admin cross-tenant yêu cầu expires_at
  → INSERT access_grants
  → Ngay lập tức: clear Redis cache của grantee
  → Async: sync acl_visible_to lên Graph DB + Vector DB
Output: grant_id, status = "active"
Effect: grantee thấy data trong < 5s
```

**Ví dụ scenarios**:

| Scenario | Grant config |
|:---|:---|
| Tenant B đọc toàn bộ domain "payment" của Tenant A | scope_type=domain, scope_value=payment, permission=read |
| Tenant B search tất cả data Tenant A | scope_type=all, permission=search |
| Tenant B ghi vào domain "shared" của Tenant A (tạm thời) | scope_type=domain, permission=write, expires_at=30d |

---

### BL-PA-07: Thu hồi Access Grant

**Business Rules**:
- Revoke grant → grantee **mất access ngay lập tức** (cache cleared synchronously)
- Sync workers cập nhật `acl_visible_to` async sau đó (~< 5s)
- Audit trail ghi lại thời điểm revoke

---

## Nghiệp vụ 4: Giám sát và Xác thực

### BL-PA-08: Xác minh Access Resolution

**Mô tả**: Sau khi cấu hình tenant/app/grant, Platform Admin xác nhận rằng identity và visibility đúng như kỳ vọng.

**Luồng**:
```
Gọi: GET /v1/access/resolve  (với API key của app cần kiểm tra)
Expected output: {
  tenant_id: "tenant-A",
  app_id: "app-X",
  visible_domains: [
    { domain_id: "platform-base", owner: "platform", permission: "read" },
    { domain_id: "payment", owner: "tenant-A", permission: "read" },
    { domain_id: "shared-legal", owner: "tenant-B", permission: "search" }
  ]
}
```

**Khi nào cần dùng**: Sau khi tạo tenant mới, sau khi thêm grant, sau khi thay đổi sharing policy.

---

### BL-PA-09: Kiểm tra Audit Trail

**Mô tả**: Platform Admin kiểm tra ai đã truy cập data gì, khi nào, được cho phép hay từ chối.

**Business Rules**:
- Mọi access đều được log (cả allowed và denied)
- Audit log partition theo tháng, lưu 12 tháng
- Log bao gồm: requester, action, resource, allowed/denied, lý do (grant_id hoặc deny reason)

---

## Tóm tắt Business Rules — Platform Admin

| ID | Rule |
|:---:|:---|
| **BR-PA-01** | Chỉ platform admin mới được tạo/suspend/delete tenant |
| **BR-PA-02** | API key plaintext chỉ hiển thị một lần — không thể khôi phục |
| **BR-PA-03** | Cross-tenant grant `write`/`admin` bắt buộc có `expires_at` |
| **BR-PA-04** | Revoke app làm key vô hiệu trong < 30s |
| **BR-PA-05** | Revoke grant làm grantee mất access đồng bộ (không qua queue) |
| **BR-PA-06** | Tenant slug là immutable sau khi tạo |
| **BR-PA-07** | Mọi access (allow + deny) đều được audit log |
| **BR-PA-08** | Grant `read`/`search` cross-tenant không yêu cầu `expires_at` |
