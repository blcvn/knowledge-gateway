---
id: DOC-S05
service: sm-mcp
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-mcp — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9076` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9121` | Yes | Health check port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `DB_DSN` | string | — | Yes | PostgreSQL (sessions) |
| `SM_DOCUMENT_ADDR` | string | `sm-document:9071` | Yes | Document service gRPC |
| `SM_SEARCH_ADDR` | string | `sm-search:9073` | Yes | Search service gRPC |
| `SM_PROFILE_ADDR` | string | `sm-profile:9074` | Yes | Profile service gRPC |
| `SM_AUTH_ADDR` | string | `sm-auth:9077` | Yes | Auth service gRPC |
| `SSE_KEEPALIVE_SEC` | int | `30` | No | SSE keepalive interval |
| `SESSION_TTL_HOURS` | int | `24` | No | MCP session expiry |

## Example .env

```env
GRPC_PORT=9076
HEALTH_PORT=9121
DB_DSN=postgres://user:pass@postgresql:5432/sm_mcp?sslmode=disable
SM_DOCUMENT_ADDR=sm-document:9071
SM_SEARCH_ADDR=sm-search:9073
SM_PROFILE_ADDR=sm-profile:9074
SM_AUTH_ADDR=sm-auth:9077
```
