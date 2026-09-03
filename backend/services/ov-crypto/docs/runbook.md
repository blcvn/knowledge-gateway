---
id: DOC-S06
service: ov-crypto
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# ov-crypto — Runbook

## Startup

```bash
make run-ov-crypto
docker compose up ov-crypto postgresql
```

**Prerequisites**: PostgreSQL must be running. KMS provider must be configured and accessible.

## Shutdown

Graceful shutdown via SIGTERM (30s timeout). Active encryption/decryption requests complete. Key rotation jobs are paused and can be resumed.

## Health Check

```bash
curl http://localhost:9108/healthz
# Expected: {"status": "serving"}

curl http://localhost:9108/readyz
# Expected: {"status": "ready", "checks": {"db": "ok", "kms": "ok", "nats": "ok"}}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| gRPC UNAVAILABLE | Service not started | Check logs, verify port 9055 |
| `AuthenticationFailed` | AES-GCM tag mismatch | Wrong key version; check key rotation history |
| `InvalidMagic` | Not an OVE1 envelope | File is plaintext; decryption skipped |
| `CorruptedCiphertext` | Envelope parse failure | Check file integrity; may need backup restore |
| KMS timeout | Vault/Cloud KMS unreachable | Check KMS provider connectivity |
| High latency on Argon2 | API key validation slow | Expected (~100ms); consider caching validated keys |
| Key rotation stale | Re-wrap incomplete | Check `ov_key_rotation_log` status; resume via RotateKey |
| Local key file missing | `KMS_LOCAL_KEY_FILE` path wrong | Verify file path, check permissions |

## Key Rotation Procedure

```bash
# 1. Trigger rotation
grpcurl -d '{"account_id": "acc_123", "reason": "quarterly rotation"}' \
  ov-crypto:9055 openviking.crypto.v1.OvCryptoService/RotateKey

# 2. Monitor rotation progress
# Check ov_key_rotation_log table for status

# 3. Verify ov-fs re-wrap
# ov-fs subscribes to ov.crypto.key.rotated and re-wraps affected files
```

## Monitoring

- **Metrics**: Prometheus at `:9108/metrics`
  - `ov_crypto_encrypt_total` / `ov_crypto_decrypt_total` — operation counts
  - `ov_crypto_encrypt_duration_seconds` — encryption latency
  - `ov_crypto_key_rotation_total` — rotations performed
  - `ov_crypto_kms_latency_seconds` — KMS provider latency
  - `ov_crypto_api_key_validation_total` — API key validations
- **Traces**: Jaeger via OTel — trace per encrypt/decrypt with KMS span
- **Logs**: Structured JSON via slog with `request_id`, `account_id`, `operation`

## Security Considerations

- Root key file (`local` provider): restrict permissions to `0600`
- Vault token: use short-lived AppRole tokens, not static tokens
- API key hashes: Argon2id with time=3, memory=64MB — never store raw keys
- Key rotation: always verify all files re-wrapped before retiring old version

## Deployment Checklist

- [ ] PostgreSQL migrations applied (`make migrate-ov-crypto`)
- [ ] KMS provider configured and tested
- [ ] Root key provisioned (local: file, vault: transit key, cloud: KMS key)
- [ ] NATS JetStream stream `openviking` exists

## Escalation

1. On-call checks logs + metrics
2. If KMS provider down → escalate to infra/security team
3. If decryption failures → check key rotation log, verify key versions
4. If unresolved in 10min → escalate to security lead immediately
