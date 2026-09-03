# TASK-OV-005 — `services/openviking-crypto` Encryption Service

**Wave:** 2 (Security)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-001, TASK-OV-003 (pkg/adapters/kms), TASK-OV-004 (pkg/nats)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-OV-006 §2](../solutions/SOL-OV-006-Crypto-Admin-Services.md)  
**Port gRPC:** 9015

**Trạng thái:** ✅ Implemented  
**Ghi chú:** ov-crypto: 18 .go - crypto service complete  
---

## Mục tiêu

Tạo `services/openviking-crypto/` — service xử lý envelope encryption với OVE1 binary format. Service này được gọi bởi `openviking-fs` để encrypt/decrypt file content một cách transparent.

---

## Cấu trúc thư mục

```
services/openviking-crypto/
├── cmd/server/main.go
├── api/proto/crypto/v1/crypto.proto
├── internal/
│   ├── domain/
│   │   ├── envelope.go        # OVE1Header, IsOVE1(), ParseOVE1Header(), assembleOVE1()
│   │   └── key_hierarchy.go   # KeyScope
│   ├── usecase/
│   │   ├── encrypt.go         # EncryptUseCase
│   │   ├── decrypt.go         # DecryptUseCase
│   │   ├── rotate_keys.go     # RotateKeysUseCase
│   │   └── port/
│   │       ├── input.go       # CryptoUseCase interfaces
│   │       └── output.go      # KMSProvider, FSRawClient
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go     # gRPC server implementation
│   │   │   └── mapper.go      # Proto ↔ Domain
│   │   └── event/
│   │       └── publisher.go   # NATS ov.crypto.key.rotated
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 1. Domain — OVE1 Format

**File: `internal/domain/envelope.go`**

```go
const OVE1Magic = "OVE1"
const OVE1Version byte = 0x01

type OVE1Header struct {
    Magic            [4]byte
    Version          byte
    ProviderType     byte
    EncKeyLen        uint16
    KeyIVLen         uint16
    DataIVLen        uint16
    EncryptedFileKey []byte
    KeyIV            []byte  // 12 bytes (GCM nonce)
    DataIV           []byte  // 12 bytes (GCM nonce)
}

func IsOVE1(data []byte) bool
// Check first 4 bytes == "OVE1"

func ParseOVE1Header(data []byte) (*OVE1Header, int, error)
// Trả về header + offset (index vào data nơi ciphertext bắt đầu)
// offset = 12 + EncKeyLen + KeyIVLen + DataIVLen

func AssembleOVE1(header OVE1Header, ciphertext []byte) []byte
// Ghi header fields theo big-endian + append ciphertext
```

**Binary format (big-endian):**
```
Offset  Size  Field
0       4     "OVE1" magic
4       1     Version (0x01)
5       1     ProviderType
6       2     EncKeyLen
8       2     KeyIVLen
10      2     DataIVLen
12      N     EncryptedFileKey (N = EncKeyLen)
12+N    12    KeyIV
12+N+12 12   DataIV
...     var   AES-GCM Ciphertext + 16-byte Auth Tag
```

---

## 2. Use Cases

**File: `internal/usecase/encrypt.go`**

```go
type EncryptUseCase struct {
    kms     port.KMSProvider
}

func (uc *EncryptUseCase) Execute(ctx context.Context, plaintext []byte, accountID string) ([]byte, error) {
    // 1. Get Account Key from KMS
    // 2. Generate random File Key (32 bytes)
    // 3. Encrypt File Key với Account Key (AES-256-GCM)
    // 4. Encrypt plaintext với File Key (AES-256-GCM)
    // 5. Assemble OVE1 binary
    // 6. Zero File Key từ memory (security)
}

// Private helper
func aesGCMEncrypt(key, iv, plaintext []byte) ([]byte, error)
// cipher.NewGCM(aes.NewCipher(key)) → gcm.Seal(nil, iv, plaintext, nil)
```

**File: `internal/usecase/decrypt.go`**

```go
type DecryptUseCase struct {
    kms port.KMSProvider
}

func (uc *DecryptUseCase) Execute(ctx context.Context, ciphertext []byte, accountID string) ([]byte, error) {
    // 1. ParseOVE1Header
    // 2. Get Account Key from KMS
    // 3. Decrypt File Key với Account Key
    // 4. Decrypt content với File Key
    // 5. Zero keys từ memory
}

func aesGCMDecrypt(key, iv, ciphertext []byte) ([]byte, error)
// gcm.Open(nil, iv, ciphertext, nil)
```

**File: `internal/usecase/rotate_keys.go`**

```go
type RotateKeysUseCase struct {
    kms      port.KMSProvider
    fsClient port.FSRawClient  // read/write raw OVE1 bytes
    publisher port.EventPublisher
}

func (uc *RotateKeysUseCase) Execute(ctx context.Context, accountID string, fileURIs []string) error {
    // 1. Get OLD Account Key (trước khi rotate)
    // 2. Rotate root key trong KMS
    // 3. Get NEW Account Key
    // 4. errgroup: parallel re-wrap mỗi file (max 10 goroutines)
    //    → rewrapFile(uri, oldKey, newKey)
    // 5. Publish NATS ov.crypto.key.rotated
}

func (uc *RotateKeysUseCase) rewrapFile(ctx context.Context, uri string, oldKey, newKey []byte) error {
    // 1. ReadRaw OVE1 bytes
    // 2. ParseOVE1Header
    // 3. Decrypt EncryptedFileKey với oldKey
    // 4. New KeyIV, re-encrypt FileKey với newKey
    // 5. AssembleOVE1 với SAME ciphertext (chỉ header thay đổi)
    // 6. WriteRaw
}
```

**Passthrough mode** (khi encryption disabled):
```go
type PassthroughUseCase struct{}

func (uc *PassthroughUseCase) Encrypt(ctx, plaintext, accountID) ([]byte, error) {
    return plaintext, nil
}

func (uc *PassthroughUseCase) Decrypt(ctx, data, accountID) ([]byte, error) {
    if IsOVE1(data) {
        return nil, NewError(ErrInvalidArgument, "cannot decrypt OVE1: encryption disabled")
    }
    return data, nil
}
```

---

## 3. gRPC Proto

**File: `api/proto/crypto/v1/crypto.proto`**

```protobuf
syntax = "proto3";
package openviking.crypto.v1;
option go_package = "vnp-memory/services/openviking-crypto/api/gen/crypto/v1;cryptov1";

service CryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  rpc RotateKeys(RotateKeysRequest) returns (RotateKeysResponse);
  rpc GetKeyStatus(GetKeyStatusRequest) returns (GetKeyStatusResponse);
  rpc Bootstrap(BootstrapRequest) returns (BootstrapResponse);
}

message EncryptRequest {
  bytes plaintext = 1;
  string account_id = 2;
}

message EncryptResponse {
  bytes ciphertext = 1;
}

message DecryptRequest {
  bytes ciphertext = 1;
  string account_id = 2;
}

message DecryptResponse {
  bytes plaintext = 1;
}

message RotateKeysRequest {
  string account_id = 1;
  repeated string file_uris = 2;
}

message RotateKeysResponse {
  int32 rotated_count = 1;
}

message GetKeyStatusRequest { string account_id = 1; }
message GetKeyStatusResponse {
  string provider_type = 1;
  bool encryption_enabled = 2;
  google.protobuf.Timestamp last_rotated_at = 3;
}

message BootstrapRequest { string account_id = 1; }
message BootstrapResponse { bool success = 1; }
```

---

## 4. Config

**File: `internal/infra/config/config.go`**

```go
type Config struct {
    GRPC struct {
        Port int `yaml:"port"` // 9015
    } `yaml:"grpc"`
    Health struct {
        Port int `yaml:"port"` // 9095
    } `yaml:"health"`
    Encryption struct {
        Enabled      bool   `yaml:"enabled"`     // false = passthrough mode
        Provider     string `yaml:"provider"`    // local | vault | cloud
        LocalKeyPath string `yaml:"local_key_path"` // ~/.openviking/root.key
        VaultAddr    string `yaml:"vault_addr"`
        VaultToken   string `yaml:"vault_token"`
    } `yaml:"encryption"`
    NATS struct {
        URL string `yaml:"url"` // nats://nats:4222
    } `yaml:"nats"`
}
```

---

## 5. Main & Wire

**File: `cmd/server/main.go`**
```go
func main() {
    cfg := config.Load()
    
    // Wire KMS based on config
    var kmsProvider kms.KMSProvider
    if cfg.Encryption.Enabled {
        switch cfg.Encryption.Provider {
        case "local":
            kmsProvider, _ = kmslocal.NewLocalProvider(cfg.Encryption.LocalKeyPath)
        // vault, cloud: future
        }
    } else {
        kmsProvider = kmsdisabled.NewDisabledProvider()
    }
    
    // Wire use cases
    encryptUC := usecase.NewEncryptUseCase(kmsProvider)
    decryptUC := usecase.NewDecryptUseCase(kmsProvider)
    
    // Start gRPC server
    handler := grpchandler.New(encryptUC, decryptUC)
    grpcServer := server.NewGRPCServer(cfg.GRPC.Port, handler)
    grpcServer.Serve()
}
```

---

## Unit Tests

```
TestEncrypt_MagicBytes           → Encrypt(plain) → result starts with "OVE1"
TestEncrypt_UniqueFileKey        → Encrypt same content twice → different ciphertext (random IV)
TestDecrypt_RoundTrip            → Encrypt then Decrypt → same plaintext
TestDecrypt_WrongAccountKey      → different accountID → decrypt error
TestIsOVE1_True                  → bytes starting "OVE1" → true
TestIsOVE1_False                 → plaintext → false
TestParseOVE1Header_Correct      → parse known binary → correct fields
TestAssembleOVE1_ParseRoundTrip  → assemble then parse → same header
TestRotateKeys_ContentUnchanged  → rotate → ciphertext body unchanged, header different
TestRotateKeys_StillDecryptable  → rotate → Decrypt with new key → same plaintext
TestPassthrough_EncryptReturnsPlain → disabled → Encrypt returns input unchanged
TestPassthrough_DecryptOVE1Error → disabled → Decrypt OVE1 → error
TestAESGCMEncryptDecrypt         → known vectors test
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

# Generate protobuf
buf generate services/openviking-crypto/

# Build
go build ./services/openviking-crypto/...

# Test
go test ./services/openviking-crypto/... -v -count=1
```

---

## Ghi chú triển khai

- Package `crypto/rand` cho random IV và File Key generation
- `crypto/cipher` + `crypto/aes` từ standard library (không dùng external crypto libs)
- Zero sensitive bytes sau use: `for i := range key { key[i] = 0 }`
- `buf.yaml` cần có trong monorepo root để generate protobuf
- `RotateKeys` phải idempotent: gọi 2 lần với cùng key → cùng kết quả
