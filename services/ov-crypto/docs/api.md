---
id: DOC-S02
service: ov-crypto
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-crypto — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9055

## gRPC Service Definition

```protobuf
// api/proto/openviking/crypto/v1/service.proto
service OvCryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  rpc RotateKey(RotateKeyRequest) returns (RotateKeyResponse);
  rpc GetKeyStatus(KeyStatusRequest) returns (KeyStatus);
}
```

## Endpoints

### Encrypt

Envelope encryption: generate random File Key → AES-256-GCM encrypt content → encrypt File Key with Account Key.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_id` | string | Yes | Account for key derivation |
| `plaintext` | bytes | Yes | Content to encrypt |

**Response**: `EncryptResponse { ciphertext: bytes, key_version: int32 }`

### Decrypt

Parse OVE1 envelope → decrypt File Key → AES-256-GCM decrypt content. Files without `OVE1` magic are returned as-is (backward-compatible).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_id` | string | Yes | Account for key derivation |
| `ciphertext` | bytes | Yes | Encrypted content (OVE1 envelope) |

**Response**: `DecryptResponse { plaintext: bytes }`

### RotateKey

Rotate the root or account key. Publishes `ov.crypto.key.rotated` for ov-fs to re-wrap files.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_id` | string | No | Specific account (empty = root key) |
| `reason` | string | No | Rotation reason for audit log |

**Response**: `RotateKeyResponse { new_version: int32, affected_accounts: int32 }`

### GetKeyStatus

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_id` | string | Yes | Account to check |

**Response**: `KeyStatus { version: int32, provider: string, created_at, last_rotated, status: ACTIVE/EXPIRED }`

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `INVALID_ARGUMENT` | 400 | Invalid account_id or empty plaintext |
| `NOT_FOUND` | 404 | Account key not found |
| `INTERNAL` | 500 | KMS provider error |
| `UNAUTHENTICATED` | 401 | Decryption authentication failed (wrong key / corrupted) |
| `DATA_LOSS` | 500 | Corrupted envelope (invalid magic, incomplete) |
