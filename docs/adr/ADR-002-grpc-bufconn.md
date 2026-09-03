# ADR-002 — gRPC + bufconn cho Inter-Service Communication

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-01 |
| **Deciders** | Platform Team |
| **Feature** | Tất cả inter-service communication |

---

## Context

VNP Memory có 35+ services cần communicate với nhau. Cần chọn communication protocol giữa:
- HTTP/JSON (REST)
- gRPC (protobuf)
- Message queue (NATS)
- In-process function calls

Requirements:
- Strong typing (contract-first)
- High performance (< 5ms internal latency)
- Testable (mockable interfaces)
- Supports both development (in-process) và production (network)

---

## Decision

**gRPC làm primary inter-service protocol, với bufconn cho development mode.**

```go
// bufconn: in-memory gRPC connection (zero network)
import "google.golang.org/grpc/test/bufconn"

listener := bufconn.Listen(1024 * 1024)  // 1MB buffer
conn, err := grpc.DialContext(ctx, "bufnet",
    grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
        return listener.Dial()
    }),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)

// Production: same interface, network connection
conn, err := grpc.Dial("cognee-ingestion:50051",
    grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
```

**Proto definitions** trong `backend/api/proto/` — source of truth cho tất cả service contracts.

---

## Consequences

**Positive:**
- Strong typing via proto — compile-time contract enforcement
- bufconn: **< 1ms** inter-service latency in development (vs 1-5ms với localhost TCP)
- Seamless switch production: thay connection string
- Streaming support (SSE observe hooks)
- Built-in interceptors cho tracing, logging, auth

**Negative:**
- Proto schema evolution cần backward compatibility discipline
- Thêm codegen step (protoc) vào build pipeline
- Binary protocol khó debug hơn JSON (cần grpcurl)

---

## Alternatives Considered

### A1 — REST/JSON giữa services
- **Rejected:** No compile-time type safety, slower serialization, no streaming native support

### A2 — Direct function calls (Go interfaces)
- **Rejected:** Không thể scale ra distributed mode; tightly coupled

### A3 — NATS cho tất cả communication
- **Rejected:** NATS là cho async events, không suitable cho request-response với timeout; no strong typing
