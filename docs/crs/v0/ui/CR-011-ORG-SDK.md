# CR-011 — Org Settings & API SDK: Inline Mock → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-011 |
| **Title** | Organization Settings & API SDK: Xóa bỏ inline mock và kết nối backend API |
| **Type** | Feature Implementation |
| **Priority** | P2 — Medium |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Organization, API SDK |
| **Files thay đổi** | `ui/src/hooks/useOrganizationSettings.ts`, `ui/src/hooks/useApiSdk.ts`, `ui/src/services/` (tạo mới) |

---

## 1. Hiện trạng

Khác với các module khác có file mock riêng, **Org Settings** và **API SDK** hiện đang định nghĩa *mock objects trực tiếp* trong file hooks.

**`useOrganizationSettings.ts`**:
```typescript
const mockSettings = { name: 'VNP Platform', ... };
const mockMembers = [{ id: 'm1', name: 'Nguyen Binh', ... }];
const mockRoles = [{ id: 'r1', name: 'owner', ... }];
// Trả về thẳng nếu VITE_USE_MOCK_DATA=true
```

**`useApiSdk.ts`**:
```typescript
const mockApiKeys = [{ id: 'key_1', name: 'Production Agent', ... }];
const mockRateLimits = [{ scope: 'Global (Default)', rps: 1000, ... }];
const mockWebhooks = [{ id: 'wh_1', url: 'https://...', ... }];
// Trả về thẳng
```

Việc này làm UI dính hardcode logic, vi phạm pattern tách biệt services.

---

## 2. Backend API cần implement

Base path: `/v1/admin/org` (hoặc có thể gom vào namespace console `/v1/console/org` cho nhất quán) và `/v1/console/sdk`.

### 2.1 Org Settings

- `GET /v1/console/org/settings`
- `PUT /v1/console/org/settings`
- `GET /v1/console/org/members`
- `GET /v1/console/org/roles`

### 2.2 API SDK & Webhooks

- `GET /v1/console/sdk/keys`
- `POST /v1/console/sdk/keys`
- `GET /v1/console/sdk/rate-limits`
- `GET /v1/console/sdk/webhooks`
- `POST /v1/console/sdk/webhooks`

---

## 3. Frontend thay đổi

### 3.1 Tạo file services

Tạo `ui/src/services/org.service.ts`:
```typescript
import { apiClient } from '../lib/api-client';

const BASE = '/v1/console/org';

export const orgService = {
  getSettings: () => apiClient.get(`${BASE}/settings`),
  getMembers: () => apiClient.get(`${BASE}/members`),
  getRoles: () => apiClient.get(`${BASE}/roles`),
};
```

Tạo `ui/src/services/sdk.service.ts`:
```typescript
import { apiClient } from '../lib/api-client';

const BASE = '/v1/console/sdk';

export const sdkService = {
  getKeys: () => apiClient.get(`${BASE}/keys`),
  getRateLimits: () => apiClient.get(`${BASE}/rate-limits`),
  getWebhooks: () => apiClient.get(`${BASE}/webhooks`),
};
```

### 3.2 Refactor hooks

Bỏ tất cả các `const mockXyz = [...]` khỏi `useOrganizationSettings.ts` và `useApiSdk.ts`.
Chỉ import queries từ `org.service.ts` và `sdk.service.ts` tương tự như các hooks khác.

---

## 4. Điều kiện hoàn thành

- [ ] Tách logic fetch API ra các service files (`org.service.ts`, `sdk.service.ts`).
- [ ] Không còn biến cục bộ nào bắt đầu bằng `mock` bên trong hooks.
- [ ] UI load thành viên và API keys từ database thực (bảng `api_keys` PostgreSQL).
