# Memobase App — API Reference

> REST API exposed through the embedded vnp-gateway.

All endpoints require authentication (Bearer token) unless noted otherwise.

## Health

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/healthcheck` | ❌ No | Gateway health check |

## User API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| POST | `/api/v1/users` | memobase-admin | `CreateUser` |
| GET | `/api/v1/users/{user_id}` | memobase-admin | `GetUser` |
| PUT | `/api/v1/users/{user_id}` | memobase-admin | `UpdateUser` |
| DELETE | `/api/v1/users/{user_id}` | memobase-admin | `DeleteUser` |

## Blob API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| POST | `/api/v1/blobs/insert/{user_id}` | memobase-ingestion | `InsertBlob` |
| GET | `/api/v1/blobs/{user_id}/{blob_id}` | memobase-ingestion | `GetBlob` |
| DELETE | `/api/v1/blobs/{user_id}/{blob_id}` | memobase-ingestion | `DeleteBlob` |

## Profile API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| GET | `/api/v1/users/profile/{user_id}` | memobase-context | `GetProfiles` |
| POST | `/api/v1/users/profile/{user_id}` | memobase-context | `AddProfile` |
| PUT | `/api/v1/users/profile/{user_id}/{profile_id}` | memobase-context | `UpdateProfile` |
| DELETE | `/api/v1/users/profile/{user_id}/{profile_id}` | memobase-context | `DeleteProfile` |

## Buffer API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| POST | `/api/v1/users/buffer/{user_id}/{buffer_type}` | memobase-ingestion | `FlushBuffer` |
| GET | `/api/v1/users/buffer/capacity/{user_id}/{buffer_type}` | memobase-ingestion | `GetBufferCapacity` |

## Event API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| GET | `/api/v1/users/event/{user_id}` | memobase-event | `GetEvents` |
| PUT | `/api/v1/users/event/{user_id}/{event_id}` | memobase-event | `UpdateEvent` |
| DELETE | `/api/v1/users/event/{user_id}/{event_id}` | memobase-event | `DeleteEvent` |
| GET | `/api/v1/users/event/search/{user_id}` | memobase-event | `SearchEvents` |
| GET | `/api/v1/users/event_gist/search/{user_id}` | memobase-event | `SearchEventGists` |
| GET | `/api/v1/users/event_tags/search/{user_id}` | memobase-event | `FilterByTags` |

## Context API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| GET | `/api/v1/users/context/{user_id}` | memobase-context | `GetContext` |

## Project API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| POST | `/api/v1/project/profile_config` | memobase-admin | `UpdateProfileConfig` |
| GET | `/api/v1/project/profile_config` | memobase-admin | `GetProfileConfig` |
| GET | `/api/v1/project/billing` | memobase-admin | `GetBilling` |
| GET | `/api/v1/project/users` | memobase-admin | `ListProjectUsers` |
| GET | `/api/v1/project/usage` | memobase-admin | `GetUsage` |

## Admin API

| Method | Path | Target Service | gRPC Method |
|--------|------|----------------|-------------|
| GET | `/api/v1/admin/status_check` | memobase-admin | `StatusCheck` |

## MCP Tools (Port 8082)

| Tool Name | Description | Target |
|-----------|-------------|--------|
| `save_memory` | Insert blob + flush buffer | memobase-ingestion |
| `get_user_profiles` | Get user profiles | memobase-context |
| `search_memories` | Search events | memobase-event |

## Authentication

- **Bearer Token**: `Authorization: Bearer sk-proj-{project_id}-{secret}`
- **Root Token**: `Authorization: Bearer {ROOT_ACCESS_TOKEN}`
- **Dev Mode**: `AUTH_DEV_MODE=true` skips authentication

## Error Responses

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "invalid or missing bearer token"
  }
}
```

| HTTP Status | gRPC Code | Meaning |
|-------------|-----------|---------|
| 400 | INVALID_ARGUMENT | Bad request |
| 401 | UNAUTHENTICATED | Missing/invalid auth |
| 403 | PERMISSION_DENIED | Forbidden |
| 404 | NOT_FOUND | Resource not found |
| 429 | RESOURCE_EXHAUSTED | Rate limited |
| 500 | INTERNAL | Server error |
| 503 | UNAVAILABLE | Service unavailable |
