# UI Solution: UI-SOL-004 — Session Query Params Alignment

**Solution ID:** UI-SOL-004  
**CR References:** [CR-003-session-query-params](../../../../docs/crs/v2/api-update/CR-003-session-query-params.md)  
**Backend Solution:** [SOL-003-session-query-params.md](../../../../backend/specs/crs/v2/api-update/solutions/SOL-003-session-query-params.md)  
**Feature:** Sessions Explorer — Query Params & Pagination  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/services/session.service.ts`, `ui/src/pages/sessions/`

---

## 1. Mục Đích

Align Session API query params với backend contract:
- Đảm bảo tất cả filter params gửi đúng format
- Fix pagination: `page` + `page_size` (snake_case)
- Add missing filters: `agent_id`, `search`, `sort`
- Handle `PaginatedResponse<Session>` đúng cách

---

## 2. Backend API Contract

```http
GET /v1/console/sessions
Query params:
  status?:    "active" | "completed" | "failed"
  user_id?:   string
  agent_id?:  string
  search?:    string          (full-text search in session title)
  sort?:      "created_at_desc" | "created_at_asc" | "message_count_desc"
  page?:      number          (1-indexed, default 1)
  page_size?: number          (default 20, max 100)

→ PaginatedResponse<Session>:
  {
    data:      Session[],
    total:     number,
    page:      number,
    page_size: number,
    has_more:  boolean,
    // aliases also present:
    pageSize:  number,
    hasMore:   boolean
  }
```

---

## 3. Frontend Implementation

### 3.1 Session Filters Hook

```typescript
// ui/src/api/hooks/useSessionFilters.ts

interface SessionFilters {
  status?:    'active' | 'completed' | 'failed';
  user_id?:   string;
  agent_id?:  string;
  search?:    string;
  sort?:      'created_at_desc' | 'created_at_asc' | 'message_count_desc';
  page?:      number;
  page_size?: number;
}

// URL sync: filters persist in URL query params
export function useSessionFilters() {
  const [searchParams, setSearchParams] = useSearchParams();
  
  const filters: SessionFilters = {
    status:    searchParams.get('status') as SessionFilters['status'] ?? undefined,
    user_id:   searchParams.get('user_id') ?? undefined,
    agent_id:  searchParams.get('agent_id') ?? undefined,
    search:    searchParams.get('search') ?? undefined,
    sort:      searchParams.get('sort') as SessionFilters['sort'] ?? 'created_at_desc',
    page:      Number(searchParams.get('page')) || 1,
    page_size: Number(searchParams.get('page_size')) || 20,
  };
  
  const setFilters = (updates: Partial<SessionFilters>) => {
    const next = { ...filters, ...updates, page: 1 };  // reset page on filter change
    setSearchParams(toSearchParams(next));
  };
  
  return { filters, setFilters };
}
```

### 3.2 Sessions API Service Fix

```typescript
// ui/src/services/session.service.ts
// FIX: snake_case query params, correct pagination handling

export async function getSessions(filters: SessionFilters): Promise<PaginatedResponse<Session>> {
  const params = new URLSearchParams();
  if (filters.status)    params.set('status', filters.status);
  if (filters.user_id)   params.set('user_id', filters.user_id);      // FIX: snake_case
  if (filters.agent_id)  params.set('agent_id', filters.agent_id);    // ADD: missing param
  if (filters.search)    params.set('search', filters.search);         // ADD: missing param
  if (filters.sort)      params.set('sort', filters.sort);             // ADD: missing param
  if (filters.page)      params.set('page', String(filters.page));
  if (filters.page_size) params.set('page_size', String(filters.page_size)); // FIX: snake_case
  
  return apiClient.get<PaginatedResponse<Session>>(`/v1/console/sessions?${params}`);
}
```

### 3.3 Pagination Component

```typescript
// ui/src/components/common/Pagination.tsx
// Uses page + page_size from PaginatedResponse

interface PaginationProps {
  page:      number;
  page_size: number;
  total:     number;
  has_more:  boolean;
  onChange:  (page: number) => void;
}

// Display: "Showing 21-40 of 156 sessions"
// Prev / [1] [2] [3] ... [8] / Next
```

### 3.4 Sessions Page Filters UI

```
SessionsPage
├── SearchBar               ← debounced, syncs to URL ?search=
├── FilterRow
│   ├── StatusSelect        ← All | Active | Completed | Failed
│   ├── AgentIdInput        ← agent_id filter (text input)  ← ADD
│   ├── UserIdInput         ← user_id filter (text input)
│   └── SortSelect          ← Newest | Oldest | Most Messages ← ADD
├── SessionsTable           ← from PaginatedResponse.data
└── Pagination              ← from PaginatedResponse metadata
```

---

## 4. Acceptance Criteria (Frontend)

- [ ] `status` filter gửi đúng giá trị: `"active"` / `"completed"` / `"failed"`
- [ ] `agent_id` filter hiển thị và gửi đúng param name
- [ ] `search` full-text với debounce 300ms
- [ ] `sort` selector với 3 options (newest/oldest/most messages)
- [ ] Pagination hiển thị "Showing X-Y of Z" format
- [ ] URL sync: filters persist qua browser back/forward/refresh
- [ ] `page_size` default 20, có thể chọn 10/20/50/100
- [ ] Page reset về 1 khi thay đổi bất kỳ filter nào
