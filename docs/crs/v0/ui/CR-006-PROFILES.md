# CR-006 — User Profiles: Mock → Real API (Memobase Engine)

| Field | Value |
|---|---|
| **CR ID** | CR-006 |
| **Title** | User Profiles: Kết nối structured profiles, buffer status, context assembly với Memobase backend |
| **Type** | Feature Implementation |
| **Priority** | P0 — Critical |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | User Profiles |
| **Files thay đổi** | `ui/src/mock/profile.mock.ts`, `ui/src/hooks/useProfiles.ts`, `ui/src/services/profile.service.ts` |

---

## 1. Hiện trạng

### Mock data ([`profile.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/profile.mock.ts))

Mock data cứng cho buffers, events, profiles, và context:
```typescript
export const profileMock = {
  users: [{ user_id: 'user_1', profiles: [] }],
  userDetail: {
    user_id: 'user_1',
    profiles: [{ topic: 'Preferences', sub_topic: 'Theme', content: 'Dark mode' }]
  },
  buffers: [{ user_id: 'user_1', buffer_type: 'core', token_count: 500, token_threshold: 1000, ... }],
  events: [{ id: 'evt_1', user_id: 'user_1', gist: 'User logged in', tags: ['auth'], ... }],
  context: { user_id: 'user_1', context_string: 'User prefers dark mode.', token_count: 5, ... }
};
```

---

## 2. Backend API cần implement

Base path: `/v1/console/profiles`
Data source chính: **Memobase Engine** (YOLO extraction).

### 2.1 Profile Retrieval

- `GET /v1/console/profiles` → Danh sách users có profile memory.
- `GET /v1/console/profiles/{user_id}` → Structured profile (hierarchical topics/sub-topics, score).

**Response schema** (`UserProfile`):
```json
{
  "user_id": "usr_abc123",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-06-16T00:00:00Z",
  "profiles": [
    { "topic": "Expertise", "sub_topic": "Programming", "content": "Advanced in Go and Rust" },
    { "topic": "Preference", "sub_topic": "Communication", "content": "Direct, minimal verbosity" }
  ]
}
```

### 2.2 Buffer Status

- `GET /v1/console/profiles/{user_id}/buffers`

**Response schema** (`BufferZone`):
Hiển thị trạng thái của Memory Blob buffer chờ flush.
```json
{
  "user_id": "usr_abc123",
  "buffer_type": "core",
  "token_count": 850,
  "token_threshold": 2000,
  "idle_timeout": "5m",
  "last_flush": "2026-06-16T12:00:00Z",
  "flush_count": 14
}
```

### 2.3 Context Assembly

- `GET /v1/console/profiles/{user_id}/context`

**Response schema** (`ContextAssembly`):
Payload mà Agent sẽ nhận được để bỏ vào LLM context prompt.
```json
{
  "user_id": "usr_abc123",
  "context_string": "<Profile>\nExpertise: Advanced in Go and Rust...\n</Profile>\n<Recent Events>...",
  "token_count": 156,
  "profile_section_tokens": 120,
  "event_section_tokens": 36,
  "latency_ms": 42
}
```

### 2.4 Event Timeline

- `GET /v1/console/profiles/{user_id}/events`

Lấy timeline event của user từ event-bus. Khớp [`UserEvent`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/profile.ts).

---

## 3. Frontend thay đổi

### 3.1 Xóa mock dependency trong `useProfiles.ts`

```typescript
// SAU — không còn mock
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { profileService } from '../services/profile.service';

export function useProfileList() {
  return useQuery({
    queryKey: ['profiles'],
    queryFn: () => profileService.listProfiles(),
  });
}

export function useProfileDetail(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId],
    queryFn: () => profileService.getProfile(userId),
    enabled: !!userId,
  });
}

// ... tương tự cho useBufferStatus, useUserEvents, useContextAssembly, useProfileConfig
```

---

## 4. Điều kiện hoàn thành

- [ ] Danh sách users có profile memory load từ API Memobase.
- [ ] User profile detail hiển thị hierarchical topics.
- [ ] Xem được payload assembled context (quan trọng cho debug).
- [ ] Xem được trạng thái flush của buffer.
- [ ] Xóa mock data cứng hoàn toàn.
