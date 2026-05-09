---
id: DOC-S05
service: sm-connector
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-connector — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9075` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9120` | Yes | Health check port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS for sync events |
| `GOOGLE_CLIENT_ID` | string | — | Yes* | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | string | — | Yes* | Google OAuth secret |
| `NOTION_CLIENT_ID` | string | — | Yes* | Notion OAuth client ID |
| `NOTION_CLIENT_SECRET` | string | — | Yes* | Notion OAuth secret |
| `ONEDRIVE_CLIENT_ID` | string | — | Yes* | OneDrive client ID |
| `ONEDRIVE_CLIENT_SECRET` | string | — | Yes* | OneDrive secret |
| `OAUTH_REDIRECT_BASE` | string | — | Yes | OAuth callback base URL |
| `TOKEN_ENCRYPTION_KEY` | string | — | Yes | AES-256 key for token encryption |
| `SYNC_BATCH_SIZE` | int | `50` | No | Documents per sync batch |
| `SYNC_CONCURRENCY` | int | `3` | No | Parallel sync workers |
| `STATE_TTL_MINUTES` | int | `10` | No | OAuth state token TTL |

*Required unless org-level custom keys are configured in sm-profile.

## Example .env

```env
GRPC_PORT=9075
HEALTH_PORT=9120
DB_DSN=postgres://user:pass@postgresql:5432/sm_connector?sslmode=disable
NATS_URL=nats://nats:4222
GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=xxx
OAUTH_REDIRECT_BASE=https://api.vnp-memory.io
TOKEN_ENCRYPTION_KEY=base64_encoded_32_byte_key
```
