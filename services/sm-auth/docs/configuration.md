---
id: DOC-S05
service: sm-auth
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-auth — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9077` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9122` | Yes | Health check port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS for events |
| `JWT_PUBLIC_KEY_PATH` | string | — | Yes | RS256 public key for validation |
| `JWT_ISSUER` | string | `vnp-memory` | No | Expected JWT issuer |
| `API_KEY_HASH_ALGO` | string | `sha256` | No | Key hashing algorithm |
| `API_KEY_PREFIX` | string | `sm_` | No | Key prefix |
| `SESSION_TTL_HOURS` | int | `24` | No | Session expiry |

## Example .env

```env
GRPC_PORT=9077
HEALTH_PORT=9122
DB_DSN=postgres://user:pass@postgresql:5432/sm_auth?sslmode=disable
NATS_URL=nats://nats:4222
JWT_PUBLIC_KEY_PATH=/etc/keys/jwt_public.pem
```
