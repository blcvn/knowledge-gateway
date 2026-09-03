# ov-crypto — Data Models

> **Service**: `services/ov-crypto`
> **Role**: OpenViking cryptography — envelope encryption (OVE1 format), account-scoped key management, and key rotation.

---

## AccountKey

```go
type AccountKey struct {
    ID               string
    AccountID        string
    KeyVersion       KeyVersion
    ProviderType     string     // "local" | "vault" | "aws" | "gcp"
    EncryptedRootKey []byte     // Only populated for Local provider
    KeyReference     string     // ARN or Vault Path
    Status           KeyStatus
    CreatedAt        time.Time
    RotatedAt        *time.Time
    ExpiresAt        *time.Time
}

type KeyStatus string
// active | expired | revoked

type KeyVersion int
```

---

## KeyRotationLog

```go
type KeyRotationLog struct {
    ID             string
    AccountID      string
    OldVersion     KeyVersion
    NewVersion     KeyVersion
    Reason         string
    InitiatedBy    string
    Status         string    // completed | failed | in_progress
    FilesReWrapped int
    DurationMs     int
    CreatedAt      time.Time
}
```

---

## Envelope (OVE1 Format)

```go
// Binary format: Magic(4) | Version(1) | ProviderType(1) | EFKLen(2) | KIVLen(2) | DIVLen(2) | EFK | KIV | DIV | Ciphertext+AuthTag

type EnvelopeHeader struct {
    Magic        string       // "OVE1"
    Version      byte
    ProviderType ProviderType
    EFKLen       uint16       // Encrypted File Key Length
    KIVLen       uint16       // Key Initialization Vector Length
    DIVLen       uint16       // Data Initialization Vector Length
}

type Envelope struct {
    Header     EnvelopeHeader
    EFK        []byte // Encrypted File Key
    KIV        []byte // Key Initialization Vector
    DIV        []byte // Data Initialization Vector
    Ciphertext []byte // Encrypted content including AES-GCM Auth Tag
}

type ProviderType byte
// 0x00 Unknown | 0x01 Local | 0x02 Vault | 0x03 Cloud (AWS/GCP)
```

---

## KMSProvider (Interface)

```go
type KMSProvider interface {
    EncryptFileKey(fileKey []byte, accountID string) (encryptedKey, iv []byte, err error)
    DecryptFileKey(encryptedKey, iv []byte, accountID string) (fileKey []byte, err error)
    RotateRootKey(accountID string) error
}
```

---

## Sources
- [`services/ov-crypto/internal/domain/model/key.go`](../../services/ov-crypto/internal/domain/model/key.go)
- [`services/ov-crypto/internal/domain/model/envelope.go`](../../services/ov-crypto/internal/domain/model/envelope.go)
- [`services/ov-crypto/internal/domain/model/kms.go`](../../services/ov-crypto/internal/domain/model/kms.go)
