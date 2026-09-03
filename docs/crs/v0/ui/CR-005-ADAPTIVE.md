# CR-005 — Adaptive Memory: Mock → Real API (Supermemory Engine)

| Field | Value |
|---|---|
| **CR ID** | CR-005 |
| **Title** | Adaptive Memory: Kết nối memories, connectors, rules với Supermemory backend |
| **Type** | Feature Implementation |
| **Priority** | P1 — High |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Adaptive Memory |
| **Files thay đổi** | `ui/src/mock/adaptive.mock.ts`, `ui/src/hooks/useAdaptiveMemory.ts`, `ui/src/services/adaptive.service.ts` |

---

## 1. Hiện trạng

### Mock data ([`adaptive.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/adaptive.mock.ts))

Mock data cứng cho connectors, memory versions:
```typescript
export const adaptiveMock = {
  memories: [{ id: 'mem_a1', content: 'Adaptive memory mock', is_latest: true, ... }],
  versions: [{ id: 'v_1', memory_id: 'mem_a1', version_number: 1, is_latest: true, ... }],
  connectors: [{ id: 'conn_1', type: 'google_drive', status: 'Connected', document_count: 42, ... }],
};
```

### Hooks ([`useAdaptiveMemory.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/hooks/useAdaptiveMemory.ts))

Sử dụng `adaptiveMock` và trả về mock object cứng cho analytics (`defaultAnalyticsMock`).

```typescript
const defaultAnalyticsMock: AdaptiveAnalytics = {
  creation_rate: 12, deletion_rate: 3, contradiction_count: 5,
  connector_sync_count: 48, storage_usage_bytes: 52428800
};
// ... các hooks đều check useMock
```

---

## 2. Backend API cần implement

Base path: `/v1/console/adaptive`
Thực chất tất cả request đều được gateway route tới **Supermemory Engine** (`sm-memory`, `sm-connector`, `sm-analytics`).

### 2.1 GET /v1/console/adaptive/memories

Lấy danh sách current active adaptive memories.

**Response schema** (khớp [`AdaptiveMemory`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/adaptive.ts)):
```json
[
  {
    "id": "sm_abc123",
    "content": "User is a senior software engineer based in Vietnam.",
    "memory_type": "dynamic",
    "is_latest": true,
    "parent_id": "sm_abc000",
    "root_id": "sm_abc000",
    "relation_type": "updates",
    "created_at": "2026-06-15T10:00:00Z",
    "updated_at": "2026-06-16T11:00:00Z",
    "forget_after": "90d"
  }
]
```

### 2.2 GET /v1/console/adaptive/memories/{id}/versions

Lấy chuỗi lịch sử (version chain) của một memory.

**Response schema**: Array của [`MemoryVersion`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/adaptive.ts).

### 2.3 Connectors API

- `GET /v1/console/adaptive/connectors` → Lấy cấu hình các nguồn external data (Google Drive, Notion, GitHub).
- `POST /v1/console/adaptive/connectors` → Tạo connector mới (nhập API key/OAuth config).
- `POST /v1/console/adaptive/connectors/{id}/sync` → Trigger ingest job manually.

### 2.4 GET /v1/console/adaptive/analytics

Lấy metrics của Supermemory.

**Response schema**: Khớp [`AdaptiveAnalytics`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/adaptive.ts).
**Nguồn**: Aggregation count trên Supermemory PostgreSQL database + NATS events.

### 2.5 Rules API

- `GET /v1/console/adaptive/forget-rules`
- `PUT /v1/console/adaptive/forget-rules`

Quản lý policy auto-forget (thời gian expire, contradiction resolution).

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `useAdaptiveMemory.ts`

Chuyển tất cả queries sang gọi API trực tiếp. Thêm mutation cho sync connector.

```typescript
// SAU — không còn mock
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adaptiveService } from '../services/adaptive.service';

export function useAdaptiveMemories() {
  return useQuery({
    queryKey: ['adaptive', 'memories'],
    queryFn: () => adaptiveService.getMemories(),
  });
}

// ... tương tự cho useMemoryVersions, useConnectors, useAdaptiveAnalytics, useForgetRules

export function useSyncConnector() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => adaptiveService.syncConnector(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'connectors'] }),
  });
}

export function useUpdateForgetRules() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: adaptiveService.updateForgetRules,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'forget-rules'] }),
  });
}
```

---

## 4. Điều kiện hoàn thành

- [ ] Adaptive Dashboard UI tải data thực (Analytics, Rules).
- [ ] Connectors list hiển thị từ database.
- [ ] Tính năng Sync Connector hoạt động, gửi tín hiệu thành công tới `sm-connector` trigger NATS job.
- [ ] Memory versions chain thể hiện đúng theo `parent_id` và `root_id`.
- [ ] Không còn import mock.
