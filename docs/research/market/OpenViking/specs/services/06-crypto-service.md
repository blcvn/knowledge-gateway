# 06 — OpenViking Crypto Service

> **Service**: `openviking-crypto`  
> **Port**: 9015 (gRPC) · 9095 (Health/Metrics)  
> **Origin**: L5 Crypto (encryptor.py + providers.py)  
> **Role**: Envelope encryption, KMS adapters, key rotation, per-file AES-256-GCM

---

## 1. Responsibilities

| Capability | Description |
|-----------|-------------|
| **Encrypt** | Per-file AES-256-GCM envelope encryption |
| **Decrypt** | Decrypt OVE1 envelope format |
| **Key Hierarchy** | Root Key → Account Key → File Key (random 32B) |
| **KMS Adapters** | Local file, HashiCorp Vault, Cloud KMS (pluggable) |
| **Key Rotation** | Re-wrap file keys without re-encrypting content |
| **Bootstrap** | Initialize key provider on first startup |

---

## 2. Clean Architecture Layout

```
services/openviking-crypto/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── envelope.go                 # OVE1 format, magic bytes, version
│   │   ├── key_hierarchy.go            # RootKey, AccountKey, FileKey
│   │   ├── provider_type.go            # Local=0x01, Vault=0x02, Cloud=0x03
│   │   └── errors.go
│   ├── usecase/
│   │   ├── encrypt.go                  # Encrypt plaintext → OVE1 ciphertext
│   │   ├── decrypt.go                  # Decrypt OVE1 → plaintext
│   │   ├── rotate_keys.go             # Re-wrap file keys with new root
│   │   ├── bootstrap.go               # Initialize key provider
│   │   ├── port/
│   │   │   ├── input.go               # CryptoUseCase interface
│   │   │   └── output.go             # KMSProvider, KeyStore
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── local/                 # LocalFileProvider
│   │   │   ├── vault/                 # HashiCorp Vault adapter
│   │   │   └── cloud/                 # Cloud KMS adapter (Volcengine/AWS/GCP)
│   │   └── event/
│   │       └── publisher.go            # NATS: ov.crypto.key.rotated
│   └── infra/
```

---

## 3. Envelope Encryption

```
Root Key (from KMS Provider)
  └── Account Key (derived per account via HKDF)
       └── File Key (random 32 bytes per file)
            └── AES-256-GCM encrypt content
```

### OVE1 Binary Format

| Offset | Size | Field |
|--------|------|-------|
| 0 | 4B | Magic: `"OVE1"` |
| 4 | 1B | Version: `0x01` |
| 5 | 1B | Provider type (0x01=Local, 0x02=Vault, 0x03=Cloud) |
| 6 | 2B | Encrypted File Key length (big-endian) |
| 8 | 2B | Key IV length (big-endian) |
| 10 | 2B | Data IV length (big-endian) |
| 12 | var | Encrypted File Key |
| — | var | Key IV |
| — | 12B | Data IV |
| — | var | AES-GCM ciphertext + 16B auth tag |

---

## 4. gRPC Service Definition

```protobuf
service CryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  rpc RotateKeys(RotateKeysRequest) returns (RotateKeysResponse);
  rpc GetKeyStatus(GetKeyStatusRequest) returns (GetKeyStatusResponse);
  rpc Bootstrap(BootstrapRequest) returns (BootstrapResponse);
}
```

---

## 5. KMS Provider Interface

```go
type KMSProvider interface {
    // GetRootKey retrieves the root encryption key
    GetRootKey(ctx context.Context) ([]byte, error)

    // DeriveAccountKey derives a per-account key from root
    DeriveAccountKey(ctx context.Context, accountID string) ([]byte, error)

    // RotateRootKey rotates the root key
    RotateRootKey(ctx context.Context) error

    // ProviderType returns the provider identifier
    ProviderType() byte
}
```

### Implementations

| Provider | Config | Description |
|----------|--------|-------------|
| `LocalFileProvider` | `encryption.local_key_path` | Root key stored in local file |
| `VaultProvider` | `encryption.vault_addr`, `vault_token` | HashiCorp Vault KV engine |
| `CloudKMSProvider` | `encryption.cloud_*` | AWS KMS / GCP KMS / Volcengine KMS |

---

## 6. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Separate crypto service | Security boundary isolation; independent audit surface |
| OVE1 format preserved | Binary-compatible with existing Python encryption |
| HKDF for account keys | Deterministic derivation; no extra storage needed |
| Key rotation = re-wrap | Only re-encrypts file keys, not file content (O(n) key wraps, not O(n·data)) |
