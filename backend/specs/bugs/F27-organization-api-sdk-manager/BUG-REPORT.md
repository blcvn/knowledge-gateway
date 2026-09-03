# Bug Report — F27: Organization, API & SDK Manager

> Feature: Org settings/members/roles, SDK API keys, webhooks
> Luồng: `apps/memory → gateway/console.go (OrgHandler, SDKHandler) → vnp-platform`

---

## BUG-F27-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:720-830`

---

## BUG-F27-002: `GetRoles` Trả Về Hardcoded Static Roles — Không Dynamic

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/console.go:746-759`

**Mô tả:**  
`OrgHandler.GetRoles` trả về hardcoded JSON với 4 static roles thay vì query từ database hay `vnp-platform`:

```go
func (h *OrgHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`{"roles":[` +
        `{"id":"owner","name":"Owner","permissions":["*"]},` +
        ...
    `]}`))
}
```

**Impact:**  
- Role customization không được hỗ trợ.
- Permissions model không flexible.

---

## BUG-F27-003: SDK API Keys `MigrateOrgSDKSchema` Tạo Tables Nhưng Handlers Forward Tới `vnp-platform`

**Severity:** HIGH  
**File:** `gateway/cmd/main.go:126-131`

**Mô tả:**  
Gateway tạo `sdk_api_keys` và `webhooks` tables trong PostgreSQL, nhưng `SDKHandler` forward tới `vnp-platform` thay vì sử dụng direct DB access. Không rõ `vnp-platform` có dùng cùng database hay không.

**Impact:**  
- SDK key creation có thể write vào wrong database hoặc wrong table.
- Inconsistency giữa schema migration và data flow.

---

## BUG-F27-004: Webhook Delivery Logic Không Tồn Tại

**Severity:** HIGH  
**File:** `gateway/`

**Mô tả:**  
`POST /v1/console/sdk/webhooks` create webhook configurations, nhưng không có webhook delivery mechanism trong gateway. Không có NATS subscriber nào deliver events tới webhook endpoints khi events xảy ra.

**Impact:**  
- Webhooks được create nhưng không bao giờ được deliver.
