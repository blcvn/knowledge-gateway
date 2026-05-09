---
id: TDD-ov-crypto
title: Technical Design — ov-crypto
service: ov-crypto
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: OpenViking
---

# Technical Design — ov-crypto

> **Group**: OpenViking | **gRPC Port**: 9055 | **Origin**: OpenViking (FileEncryptor + Providers)

## 1. Service Overview

Envelope encryption service: AES-256-GCM per-file encryption, multi-provider KMS abstraction (Local/Vault/Cloud), key rotation, and API key hashing (Argon2id).

**Origin mapping**: `openviking/crypto/encryptor.py` (324 lines) + `openviking/crypto/providers.py` + `openviking/crypto/exceptions.py`.

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── envelope.go              # Envelope, EnvelopeHeader, ProviderType, OVE1 magic
│   ├── key.go                   # KeyMaterial, KeyVersion, KeyStatus enum
│   └── kms.go                   # KMSProvider interface (domain-level)
├── repository/
│   └── key_repo.go              # KeyRepository (key metadata persistence)
├── event.go                     # KeyRotated domain event
└── errors.go                    # AuthFailed, CorruptedCipher, InvalidMagic, KeyMismatch
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── encrypt.go                   # Envelope encrypt: genFileKey → AES-GCM → wrapFk → OVE1
├── decrypt.go                   # Envelope parse → unwrapFk → AES-GCM decrypt
├── key_rotation.go             # Rotate root/account key → publish event → re-wrap
├── key_status.go               # Key version + status queries
├── port/
│   ├── input.go                # CryptoUseCase interface
│   └── output.go              # KMSProviderPort, EventPublisherPort, KeyRepoPort
└── dto/
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/handler.go              # OvCryptoService gRPC
├── event/publisher.go           # NATS: ov.crypto.key.rotated
└── kms/
    ├── local_provider.go        # Local file-based KMS (dev/test)
    ├── vault_provider.go        # HashiCorp Vault Transit
    └── cloud_provider.go        # AWS KMS / GCP KMS
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/key_repo.go      # PostgreSQL key metadata + rotation log
├── config/config.go
└── wire/wire.go                 # Wire provider set based on KMS_PROVIDER
```

## 3. gRPC API

```protobuf
service OvCryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  rpc RotateKey(RotateKeyRequest) returns (RotateKeyResponse);
  rpc GetKeyStatus(KeyStatusRequest) returns (KeyStatus);
}
```

## 4. NATS Events

### Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `ov.crypto.key.rotated` | `{account_id, old_version, new_version}` | Key rotation completed |

## 5. Data Model

- **ov_account_keys**: Key metadata (provider, version, status, references)
- **ov_key_rotation_log**: Rotation audit trail
- **ov_api_key_hashes**: Argon2id hashed API keys (key_prefix + hash + role)

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Purpose |
|---------|-----------|----------|---------|
| ov-fs | Inbound | gRPC | Encrypt/decrypt file content |
| KMS Backend | Outbound | Native SDK | Root key operations |
| PostgreSQL | Outbound | SQL | Key metadata, rotation audit |

## 7. Observability

- **Metrics**: Encrypt/decrypt count/latency, KMS provider latency, rotation count, API key validation
- **Traces**: OTel spans: `ov-crypto.Encrypt`, `ov-crypto.Decrypt`, `ov-crypto.RotateKey`
- **Health**: gRPC Health v1 + HTTP `/healthz` on port 9108

## 8. Multi-Tenancy

- `account_id` → per-account encryption keys, namespace isolated

---

> **Next Steps**: Decompose into FEAT specs: FEAT-001 (Envelope Encryption), FEAT-002 (Multi-Provider KMS), FEAT-003 (Key Rotation), FEAT-004 (API Key Hashing).
