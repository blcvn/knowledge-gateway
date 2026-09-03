---
id: FEAT-008
title: Profile Management Proxy API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.4 User Profiles"
---

## Mục Tiêu

Proxy Memobase profile management APIs cho Console UI User Profiles screen. Bao gồm profile explorer, config editor, buffer monitor, event timeline, context assembly preview.

## Scope

### In Scope
- `GET /v1/console/profiles` — List all user profiles (paginated)
- `GET /v1/console/profiles/{user_id}` — Profile detail
- `GET /v1/console/profiles/{user_id}/events` — Event timeline
- `GET /v1/console/profiles/{user_id}/context` — Context assembly preview
- `GET /v1/console/profiles/{user_id}/buffers` — Buffer zone status
- `GET /v1/console/profiles/config` — Profile schema config
- `PUT /v1/console/profiles/config` — Update profile schema

### Out of Scope
- Profile mutation (create/update profiles — Memobase does this automatically from conversations)

## Thiết Kế Kỹ Thuật

### API Contract

#### GET `/v1/console/profiles`
**Query params:** `?cursor=xxx&limit=50&search=keyword`

**Response (200):**
```json
{
  "users": [
    {
      "user_id": "user_123",
      "profile_count": 15,
      "last_updated": "2026-05-13T10:00:00Z",
      "preview": { "topic": "preferences", "sub_topic": "language", "content": "TypeScript" }
    }
  ],
  "next_cursor": "...",
  "total": 890
}
```

### Internal Architecture
- **Handler:** `adapter/http/profile_handler.go`
- **Proxy to:** `memobase-context`, `memobase-ingestion` via existing gRPC clients
- Gateway adds pagination wrapper and admin-level access control

## Acceptance Criteria
- [ ] AC-1: Profile list returns paginated users with profile preview
- [ ] AC-2: Profile detail returns full structured profile tree (topic/sub_topic/content)
- [ ] AC-3: Event timeline returns chronological events with gist search support
- [ ] AC-4: Context preview returns assembled prompt-ready string with token count
- [ ] AC-5: Buffer status shows active buffers, token accumulation, flush history
- [ ] AC-6: Config endpoint allows viewing/updating profile schema
- [ ] AC-7: All endpoints require admin role

## Test Requirements
- Unit tests: Response transformation, pagination
- Integration tests: Mock Memobase gRPC, verify proxy
- Minimum coverage: 80%
