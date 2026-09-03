# TASK-OV-009 — `services/openviking-fs` gRPC Server, NATS Events & Integration

**Wave:** 3 (Storage)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-007, TASK-OV-008  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-OV-002 §5, §6, §7](../solutions/SOL-OV-002-Filesystem-Service.md)

---

## Mục tiêu

Hoàn thiện `services/openviking-fs/` với: gRPC handler (toàn bộ FileSystemService), NATS event publisher/subscriber, gRPC → CryptoClient adapter, config, wire (DI), và main.go.

---

## Các file cần tạo

### 1. Protobuf Definition

**File: `api/proto/fs/v1/fs.proto`**

```protobuf
syntax = "proto3";
package openviking.fs.v1;
option go_package = "vnp-memory/services/openviking-fs/api/gen/fs/v1;fsv1";

import "google/protobuf/timestamp.proto";

service FileSystemService {
  // Basic CRUD
  rpc Read(ReadRequest)     returns (ReadResponse);
  rpc Write(WriteRequest)   returns (WriteResponse);
  rpc Mkdir(MkdirRequest)   returns (MkdirResponse);
  rpc Rm(RmRequest)         returns (RmResponse);
  rpc Mv(MvRequest)         returns (MvResponse);
  rpc Cp(CpRequest)         returns (CpResponse);
  rpc Stat(StatRequest)     returns (StatResponse);
  rpc Exists(ExistsRequest) returns (ExistsResponse);

  // Directory Operations
  rpc Ls(LsRequest)     returns (LsResponse);
  rpc Tree(TreeRequest) returns (TreeResponse);

  // Tiered Context
  rpc ReadBatch(ReadBatchRequest) returns (ReadBatchResponse);

  // Pattern Matching
  rpc Grep(GrepRequest) returns (GrepResponse);
  rpc Glob(GlobRequest) returns (GlobResponse);

  // Relations
  rpc GetRelations(GetRelationsRequest)       returns (GetRelationsResponse);
  rpc AddRelation(AddRelationRequest)         returns (AddRelationResponse);
  rpc RemoveRelation(RemoveRelationRequest)   returns (RemoveRelationResponse);

  // Privacy Config
  rpc GetPrivacyConfig(GetPrivacyConfigRequest)       returns (GetPrivacyConfigResponse);
  rpc UpsertPrivacyConfig(UpsertPrivacyConfigRequest) returns (UpsertPrivacyConfigResponse);

  // Lock (for Session service)
  rpc AcquireLock(AcquireLockRequest) returns (AcquireLockResponse);
  rpc ReleaseLock(ReleaseLockRequest) returns (ReleaseLockResponse);

  // JSONL helpers
  rpc AppendJSONL(AppendJSONLRequest)   returns (AppendJSONLResponse);
  rpc ReadJSONL(ReadJSONLRequest)       returns (ReadJSONLResponse);

  // Context packing
  rpc Pack(PackRequest) returns (PackResponse);
}

message ReadRequest {
  string uri   = 1;
  int32  level = 2;  // 0=Abstract, 1=Overview, 2=Detail
}

message ReadResponse {
  bytes  content = 1;
  string uri     = 2;
  bool   exists  = 3;
}

message WriteRequest {
  string uri        = 1;
  bytes  content    = 2;
  string account_id = 3;
}

message WriteResponse { string uri = 1; }

message GrepRequest {
  string          uri        = 1;
  string          pattern    = 2;
  string          account_id = 3;
  int32           max_depth  = 4;
  repeated string file_types = 5;
}

message GrepResponse {
  repeated GrepMatchProto matches = 1;
  int32 total_scanned = 2;
}

message GrepMatchProto {
  string uri     = 1;
  int32  line    = 2;
  string content = 3;
}

message TreeRequest {
  string uri       = 1;
  int32  max_depth = 2;  // default 3, max 10
  string format    = 3;  // "json" | "agent"
}

message AcquireLockRequest {
  string uri       = 1;
  string lock_type = 2;  // "point" | "subtree"
  int64  timeout_ms = 3; // default 5000ms
}

message AcquireLockResponse {
  string lock_id = 1;  // token to release
}

message ReleaseLockRequest { string lock_id = 1; }
message ReleaseLockResponse {}

message AppendJSONLRequest {
  string         uri   = 1;
  repeated bytes lines = 2;  // Each is a JSON-encoded line
}

message ReadJSONLRequest { string uri = 1; }
message ReadJSONLResponse { repeated bytes lines = 1; }
```

### 2. gRPC Handler

**File: `internal/adapter/grpc/handler.go`**

```go
type Handler struct {
    fsv1.UnimplementedFileSystemServiceServer
    readUC      *usecase.ReadFileUseCase
    writeUC     *usecase.WriteFileUseCase
    dirOpsUC    *usecase.DirectoryOpsUseCase
    fileOpsUC   *usecase.FileOpsUseCase
    grepUC      *usecase.GrepUseCase
    globUC      *usecase.GlobUseCase
    relationsUC *usecase.RelationsUseCase
    privacyUC   *usecase.PrivacyUseCase
    lock        *vikingfs.PathLock
    lockTokens  sync.Map  // lock_id → LockReleaser
}

func (h *Handler) Read(ctx context.Context, req *fsv1.ReadRequest) (*fsv1.ReadResponse, error)
func (h *Handler) Write(ctx context.Context, req *fsv1.WriteRequest) (*fsv1.WriteResponse, error)
// ... all 20+ methods

// AcquireLock: store LockReleaser in lockTokens map, return UUID token
func (h *Handler) AcquireLock(ctx context.Context, req *fsv1.AcquireLockRequest) (*fsv1.AcquireLockResponse, error)

// ReleaseLock: lookup token → call releaser → delete from map
func (h *Handler) ReleaseLock(ctx context.Context, req *fsv1.ReleaseLockRequest) (*fsv1.ReleaseLockResponse, error)

// Error mapping: OpenVikingError.Code → gRPC status codes
// ErrNotFound → codes.NotFound
// ErrPermissionDenied → codes.PermissionDenied
// ErrInvalidArgument → codes.InvalidArgument
// ErrResourceBusy → codes.ResourceExhausted
// ErrInternal → codes.Internal
```

### 3. NATS Publisher

**File: `internal/adapter/event/publisher.go`**

```go
type NATSPublisher struct {
    publisher *natspkg.Publisher
}

// Implements port.EventPublisher
func (p *NATSPublisher) PublishContentWritten(ctx context.Context, payload port.ContentWrittenPayload) {
    // Async: go func() { publisher.Publish(ctx, SubjectContentWritten, payload) }()
    // Log error but don't fail caller
}

func (p *NATSPublisher) PublishContentDeleted(ctx context.Context, payload port.ContentDeletedPayload) {
    // Async, same pattern
}
```

### 4. NATS Subscriber

**File: `internal/adapter/event/subscriber.go`**

```go
type Subscriber struct {
    dirOpsUC  *usecase.DirectoryOpsUseCase
    fileOpsUC *usecase.FileOpsUseCase
    js        nats.JetStreamContext
}

func (s *Subscriber) Start(ctx context.Context) error {
    // Subscribe admin.account.created → HandleAccountCreated
    // Subscribe admin.account.deleted → HandleAccountDeleted
}

func (s *Subscriber) HandleAccountCreated(msg *nats.Msg) {
    var payload natspkg.AccountCreatedPayload
    json.Unmarshal(msg.Data, &payload)
    
    // Create root directories for new account:
    roots := []string{
        "viking://user/" + payload.AccountID + "/",
        "viking://agent/" + payload.AccountID + "/",
        "viking://session/" + payload.AccountID + "/",
    }
    for _, root := range roots {
        s.dirOpsUC.Mkdir(context.Background(), root, true)  // existOK=true (idempotent)
    }
    msg.Ack()
}

func (s *Subscriber) HandleAccountDeleted(msg *nats.Msg) {
    var payload natspkg.AccountDeletedPayload
    json.Unmarshal(msg.Data, &payload)
    
    toDelete := []string{
        "viking://user/" + payload.AccountID + "/",
        "viking://agent/" + payload.AccountID + "/",
        "viking://session/" + payload.AccountID + "/",
    }
    for _, uri := range toDelete {
        s.fileOpsUC.Rm(context.Background(), uri, true)
    }
    msg.Ack()
}
```

### 5. gRPC CryptoClient (adapter)

**File: `internal/adapter/client/crypto_client.go`**

```go
// Wraps cryptov1.CryptoServiceClient gRPC client
// Implements usecase/port.CryptoClient interface

type GRPCCryptoClient struct {
    client  cryptov1.CryptoServiceClient
    breaker *resilience.CircuitBreaker
    enabled bool  // From config: encryption.enabled
}

func (c *GRPCCryptoClient) IsEnabled() bool { return c.enabled }

func (c *GRPCCryptoClient) Encrypt(ctx context.Context, plaintext []byte, accountID string) ([]byte, error) {
    if !c.enabled {
        return plaintext, nil  // Passthrough mode
    }
    result, err := c.breaker.Execute(func() (interface{}, error) {
        resp, err := c.client.Encrypt(ctx, &cryptov1.EncryptRequest{
            Plaintext: plaintext, AccountId: accountID,
        })
        return resp, err
    })
    // ...
}
```

### 6. Config

**File: `internal/infra/config/config.go`**

```yaml
# config.yaml template
fs:
  grpc:
    port: 9011
  health:
    port: 9091
  storage:
    workspace: "~/.openviking/data"
    max_tree_depth: 10
    max_grep_goroutines: 20
    max_file_size_mb: 100
  crypto:
    service_url: "openviking-crypto:9015"
    enabled: true
    timeout: 5s
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
  telemetry:
    service_name: "openviking-fs"
    otel_endpoint: "otel-collector:4317"
```

### 7. Wire & Main

**File: `internal/infra/wire/wire.go`**

```go
func InitializeDependencies(cfg *config.Config) (*Dependencies, error) {
    // 1. VikingFS LocalFileSystem (workspace from config)
    // 2. PathLock (single shared instance)
    // 3. FSAdapter (wraps VikingFS)
    // 4. gRPC CryptoClient (with circuit breaker)
    // 5. NATS Publisher
    // 6. All use cases (inject deps)
    // 7. gRPC Handler
    // 8. NATS Subscriber
}
```

**File: `cmd/server/main.go`**

```go
func main() {
    cfg := config.Load()
    deps, _ := wire.InitializeDependencies(cfg)
    
    // Startup redo-log recovery (if any)
    // deps.Subscriber.Start(ctx)  // NATS subscriptions
    
    // gRPC server
    s := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            otelgrpc.UnaryServerInterceptor(),
            recovery.GRPCRecovery(),
        ),
    )
    fsv1.RegisterFileSystemServiceServer(s, deps.Handler)
    
    // Health server on separate port
    // Start serving
}
```

---

## Integration Test (`_test/integration_test.go`)

> Dùng `testcontainers-go` hoặc local NATS/fake-crypto để test

```
TestWriteReadE2E                    → write → read → same content
TestWriteReadE2E_WithEncryption     → encrypted write → read → plaintext
TestGrepE2E                         → write 10 files → grep → correct matches
TestAccountLifecycleFS              → NATS account.created → dirs created
TestAccountDeletedFS                → NATS account.deleted → dirs removed
TestLockAcquireRelease              → AcquireLock → ReleaseLock → unlock success
TestLockAcquire_ContentionFails     → hold lock → second AcquireLock → timeout
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

# Generate protobuf
buf generate services/openviking-fs/

# Build full service
go build ./services/openviking-fs/...

# Unit tests
go test ./services/openviking-fs/... -v -count=1

# Run service (dev mode)
cd services/openviking-fs && go run cmd/server/main.go
# → gRPC listening on :9011
```

---

## Ghi chú triển khai

- `lockTokens sync.Map` stores `LockReleaser` functions, keyed by UUID token
- Lock tokens expire via context deadline (không cần explicit TTL mechanism)
- `AcquireLock` gRPC method: timeout từ `req.TimeoutMs`, default 5000ms
- gRPC max message size: `grpc.MaxRecvMsgSize(10 * 1024 * 1024)` (10MB)
- NATS stream `"openviking"` phải được tạo trước khi subscribe — dùng `MustCreateStream()`
