# ADR-001 — Monolith-first với InProcessRegistry

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-01 |
| **Deciders** | Platform Team |
| **Feature** | F01 (Unified Memory API), F24 (Infrastructure Health) |

---

## Context

VNP Memory cần orchestrate 35+ microservices (6 memory engines × nhiều services mỗi engine, AgentMemory layer, platform services). Câu hỏi đặt ra: **Cách tổ chức runtime cho development và early production?**

Vấn đề với microservices ngay từ đầu:
- Developer cần chạy 35+ services chỉ để test 1 API call
- Networking overhead giữa các services ngay cả khi chạy local
- Service discovery phức tạp (cần Consul/Etcd hoặc Kubernetes)
- Time to first API call: 2-3 tuần setup infrastructure

---

## Decision

**Monolith-first với InProcessRegistry pattern.**

```go
// InProcessRegistry: tất cả 35+ services chạy trong 1 process
type InProcessRegistry struct {
    services map[string]*grpc.ClientConn // bufconn connections
    mu       sync.RWMutex
}

// Development: register all services via bufconn
registry.Register("cognee-ingestion", bufconnConn)
registry.Register("graphiti-search", bufconnConn)
// ... 33+ more

// Production: switch to network gRPC transparently
// VNP_MEMORY_ENV=production → registry returns network connections
```

**Hai deployment modes, cùng API:**
- `make dev` → 1 binary, InProcessRegistry với bufconn
- `make docker-up` → distributed services, network gRPC

---

## Consequences

**Positive:**
- Time to first API call: **< 5 phút** (developer experience)
- Zero network latency trong development (bufconn = in-memory pipe)
- Cùng interfaces hoạt động cho cả 2 modes
- Graceful shutdown tự động: HTTP drain → NATS drain → gRPC stop → DB close

**Negative:**
- Single process failure = tất cả services down (development only)
- Memory footprint lớn hơn khi chạy 35+ services in-process
- Debugging phức tạp hơn khi nhiều goroutines cùng log

**Mitigations:**
- Production luôn dùng distributed mode
- Structured logging với service name field để phân biệt
- Panic recovery middleware cho mỗi gRPC handler

---

## Alternatives Considered

### A1 — Docker Compose ngay từ đầu
- **Rejected:** Developer phải chờ 5-10 phút startup, 35+ container logs khó follow, không suitable cho rapid iteration

### A2 — Kubernetes from day 1
- **Rejected:** Over-engineering cho development phase; k8s overhead cho local dev unacceptable

### A3 — gRPC network cho tất cả (ngay cả local)
- **Rejected:** localhost network overhead vẫn cao hơn bufconn; cần service discovery ngay cả local
