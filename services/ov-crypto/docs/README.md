---
id: DOC-S01
service: ov-crypto
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — OpenViking Team
---

# ov-crypto

> **Group**: OpenViking | **gRPC Port**: 9055 | **Health Port**: 9108 | **Origin**: OpenViking

## Purpose

Envelope encryption service providing **AES-256-GCM per-file encryption**, **multi-provider KMS abstraction** (Local/Vault/Cloud KMS), **key rotation**, and **API key hashing** (Argon2id). Replaces Python `openviking/crypto/encryptor.py` and `openviking/crypto/providers.py`.

### Business Capability

- **Envelope Encryption**: Each file gets a random File Key → encrypted with Account Key → derived from Root Key
- **AES-256-GCM**: Authenticated encryption with 12-byte IV, 32-byte key
- **Multi-Provider KMS**: Pluggable key providers (Local File, HashiCorp Vault, AWS KMS, GCP KMS, Volcengine KMS)
- **Key Rotation**: Rotate root/account keys without re-encrypting all files (re-wrap only)
- **Key Status**: Track key version, rotation history, active/expired status
- **Envelope Format**: `OVE1` magic header + version + provider type + encrypted file key + IVs + ciphertext

## Envelope Format

```
┌────────┬─────────┬──────────────┬──────────┬────────┬──────────┬───────────────────┐
│ Magic  │ Version │ Provider Type│ EFK Len  │KIV Len │ DIV Len  │ EFK + KIV + DIV   │
│ 4B     │ 1B      │ 1B           │ 2B       │ 2B     │ 2B       │ variable          │
│ "OVE1" │ 0x01    │ 0x01-0x03    │ big-end  │ big-end│ big-end  │                   │
└────────┴─────────┴──────────────┴──────────┴────────┴──────────┴───────────────────┘
│ ... Encrypted Content (AES-256-GCM ciphertext + 16B auth tag) ...                  │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Provider Types**: `0x01` = Local, `0x02` = Vault, `0x03` = Volcengine/Cloud KMS

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Crypto**: `crypto/aes`, `crypto/cipher` (AES-256-GCM), `golang.org/x/crypto/argon2` (API key hashing)
- **KMS**: HashiCorp Vault client, AWS/GCP KMS SDKs
- **Database**: PostgreSQL (key metadata, rotation history)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-ov-crypto
make run-ov-crypto
docker compose up ov-crypto postgresql
```

## API Surface

### gRPC Service

```protobuf
service OvCryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  rpc RotateKey(RotateKeyRequest) returns (RotateKeyResponse);
  rpc GetKeyStatus(KeyStatusRequest) returns (KeyStatus);
}
```

### KMS Backends

| Backend | Config Key | Use Case |
|---------|-----------|----------|
| Local (file-based) | `local` | Development / testing |
| HashiCorp Vault | `vault` | Production (on-prem) |
| AWS KMS | `aws_kms` | Cloud deployments (AWS) |
| GCP KMS | `gcp_kms` | Cloud deployments (GCP) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| ov-fs | gRPC (inbound) | Encrypt/decrypt file content |
| KMS Backend | Native SDK | Root key management |
| PostgreSQL | SQL | Key metadata, rotation audit log |

## NATS Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `ov.crypto.key.rotated` | Publish | Key rotation completed → ov-fs re-wraps files |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — OpenViking
