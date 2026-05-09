---
id: DOC-S05
service: sm-project
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-project — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9079` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9124` | Yes | Health check port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `DB_DSN` | string | — | Yes | PostgreSQL connection |
| `MAX_SPACES_PER_ORG` | int | `100` | No | Space limit per org |
| `MAX_MEMBERS_PER_SPACE` | int | `50` | No | Member limit per space |
| `DEFAULT_VISIBILITY` | string | `private` | No | Default space visibility |

## Example .env

```env
GRPC_PORT=9079
HEALTH_PORT=9124
DB_DSN=postgres://user:pass@postgresql:5432/sm_project?sslmode=disable
MAX_SPACES_PER_ORG=100
DEFAULT_VISIBILITY=private
```
