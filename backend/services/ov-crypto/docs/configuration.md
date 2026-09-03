---
id: DOC-S05
service: ov-crypto
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-crypto — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9055` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9108` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_MAX_CONNECTIONS` | int | `10` | No | Connection pool max size |
| `KMS_PROVIDER` | string | `local` | Yes | KMS backend: local / vault / aws_kms / gcp_kms |
| `KMS_LOCAL_KEY_FILE` | string | `./keys/root.key` | No | Local provider: root key file path |
| `KMS_VAULT_ADDR` | string | — | No | Vault address (vault provider) |
| `KMS_VAULT_TOKEN` | string | — | No | Vault token |
| `KMS_VAULT_KEY_PATH` | string | `transit/keys/ov-root` | No | Vault transit key path |
| `KMS_AWS_KEY_ID` | string | — | No | AWS KMS key ID (aws_kms provider) |
| `KMS_AWS_REGION` | string | `us-east-1` | No | AWS region |
| `KMS_GCP_KEY_NAME` | string | — | No | GCP KMS key resource name |
| `ARGON2_TIME` | int | `3` | No | Argon2id time cost |
| `ARGON2_MEMORY_KB` | int | `65536` | No | Argon2id memory cost (64MB) |
| `ARGON2_THREADS` | int | `4` | No | Argon2id parallelism |
| `KEY_ROTATION_MAX_BATCH` | int | `1000` | No | Max files to re-wrap per rotation batch |

## Example .env

```env
GRPC_PORT=9055
HEALTH_PORT=9108
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://vnp:secret@localhost:5432/ov_crypto?sslmode=disable
KMS_PROVIDER=local
KMS_LOCAL_KEY_FILE=./keys/root.key
ARGON2_TIME=3
ARGON2_MEMORY_KB=65536
```

## Production Example (Vault)

```env
KMS_PROVIDER=vault
KMS_VAULT_ADDR=https://vault.internal:8200
KMS_VAULT_TOKEN=${VAULT_TOKEN}
KMS_VAULT_KEY_PATH=transit/keys/ov-root
```
