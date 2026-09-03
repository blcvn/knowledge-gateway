# CR-003: Session API — Query Parameter Enhancement

**CR ID**: CR-003-session-query-params  
**Status**: Open  
**Priority**: 🟡 High  
**Target Components**: `vnp-gateway` (session handler), `zep-core` (backend service)  
**Frontend Source**: `ui/src/services/session.service.ts`, `ui/src/types/session.ts`  
**Created**: 2026-06-18

---

## Problem

The frontend's Session List page (`/console/sessions`) requires server-side **filtering, sorting, and pagination** for `GET /v1/console/sessions`. The backend handler (`session.ListSessions`) currently forwards the full request to `zep-core` but the gateway does not document, validate, or explicitly pass-through these query parameters. As a result, depending on `zep-core`'s behaviour, filtering may silently fail.

---

## Required Query Parameters

`GET /v1/console/sessions`

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | `active \| completed \| failed` | Filter by session status |
| `user_id` | `string` | Filter sessions by a specific user |
| `agent_id` | `string` | Filter sessions by a specific agent |
| `search` | `string` | Full-text search across session title/metadata |
| `sort` | `string` | Sort field, e.g. `created_at`, `updated_at` |
| `page` | `integer` (default: 1) | Page number |
| `page_size` | `integer` (default: 20) | Items per page |

---

## Required Response Shape

The frontend expects `PaginatedResponse<Session>`:

```json
{
  "data": [
    {
      "id":            "string",
      "user_id":       "string",
      "title":         "string",
      "agent_id":      "string | null",
      "status":        "active | completed | failed",
      "message_count": 10,
      "created_at":    "ISO 8601",
      "updated_at":    "ISO 8601"
    }
  ],
  "total":     100,
  "page":      1,
  "page_size": 20,
  "pageSize":  20,
  "has_more":  true,
  "hasMore":   true
}
```

> **Note**: Both `page_size` (snake_case) and `pageSize` (camelCase) must be present — the frontend uses both aliases.

---

## Related Endpoints (Already Implemented — Verify Response Schemas)

| Endpoint | Expected Response |
|----------|-------------------|
| `GET /v1/console/sessions/live` | `Session[]` (not paginated) |
| `GET /v1/console/sessions/{id}` | `Conversation { session_id, messages: Message[] }` |
| `GET /v1/console/sessions/{id}/timeline` | `SessionTimeline[]` |
| `GET /v1/console/sessions/{id}/diff` | `SessionDiff { session_id, added[], updated[], deleted[] }` |
| `GET /v1/console/sessions/{id}/working-memory` | `WorkingMemory { session_id, summary, entities[] }` |
| `GET /v1/console/sessions/{id}/user-summary` | `UserSummary { user_id, context_string, token_count }` |

---

## Implementation Notes

1. The gateway handler `SessionHandler.ListSessions` in `console.go` forwards raw HTTP to `zep-core` — query params are passed through automatically. The gap is that `zep-core` must be verified to:
   - Accept and honour all 7 query parameters
   - Return the `PaginatedResponse<Session>` shape with both camelCase and snake_case aliases

2. If `zep-core` returns a different shape (e.g., missing `hasMore` or `pageSize`), the gateway should add a **response transformer** to normalise the shape before returning to the frontend.

3. The `GET /v1/console/sessions/{id}` endpoint returns a `Conversation` (not a raw `Session`) — verify that `zep-core` returns `{ session_id, messages: [...] }` format.

---

## Acceptance Criteria

- [ ] `GET /v1/console/sessions?status=active&page=1&page_size=20` returns filtered, paginated `PaginatedResponse<Session>`
- [ ] `GET /v1/console/sessions?user_id=xxx` filters by user correctly
- [ ] `GET /v1/console/sessions?search=keyword` performs full-text search
- [ ] Response includes both `page_size` and `pageSize`, both `has_more` and `hasMore`
- [ ] `GET /v1/console/sessions/{id}` returns `Conversation` with `messages` array
- [ ] `GET /v1/console/sessions/{id}/diff` returns `SessionDiff` with `added`, `updated`, `deleted` arrays
- [ ] `GET /v1/console/sessions/{id}/working-memory` returns `WorkingMemory` with `summary` and `entities`
