---
id: SOL-002
title: In-Process Communication — gRPC bufconn + NATS Embedded
version: 1.0.0
status: Proposed
priority: P0
created: 2026-05-14
linked_sol: SOL-001
---

# SOL-002: In-Process Communication

## 1. Tổng Quan

Giải pháp chi tiết cho inter-module communication trong monolithic binary, thay thế TCP-based gRPC và external NATS bằng in-process alternatives.

---

## 2. gRPC In-Process via bufconn

### 2.1 Kiến Trúc

```
┌─────────────────────────────────────────────────────┐
│                   Single Process                     │
│                                                      │
│  ┌──────────┐     bufconn        ┌──────────────┐   │
│  │ Gateway   │ ◄──(memory)──────►│  gRPC Server  │   │
│  │ (Client)  │                   │  (All 35 svcs)│   │
│  └──────────┘                    └──────────────┘   │
│                                                      │
│  ┌──────────┐     bufconn        ┌──────────────┐   │
│  │ Service A │ ◄──(memory)──────►│  Same gRPC    │   │
│  │ (Client)  │                   │  Server       │   │
│  └──────────┘                    └──────────────┘   │
└─────────────────────────────────────────────────────┘
```

### 2.2 Implementation

```go
// apps/memory/internal/bus/grpc_bus.go
package bus

import (
    "context"
    "fmt"
    "net"
    "sync"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/test/bufconn"
)

const bufSize = 4 * 1024 * 1024 // 4MB buffer

// GRPCBus provides in-process gRPC communication
type GRPCBus struct {
    mu       sync.RWMutex
    listener *bufconn.Listener
    server   *grpc.Server
    conn     *grpc.ClientConn // single shared connection
    services map[string]bool  // registered service names
}

func NewGRPCBus(interceptors ...grpc.UnaryServerInterceptor) *GRPCBus {
    lis := bufconn.Listen(bufSize)
    srv := grpc.NewServer(
        grpc.ChainUnaryInterceptor(interceptors...),
        grpc.MaxRecvMsgSize(64 * 1024 * 1024), // 64MB
        grpc.MaxSendMsgSize(64 * 1024 * 1024),
    )
    return &GRPCBus{
        listener: lis,
        server:   srv,
        services: make(map[string]bool),
    }
}

// Register adds a gRPC service to the shared server
func (b *GRPCBus) Register(desc *grpc.ServiceDesc, impl interface{}) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.server.RegisterService(desc, impl)
    b.services[desc.ServiceName] = true
}

// Serve starts the in-process gRPC server
func (b *GRPCBus) Serve() error {
    return b.server.Serve(b.listener)
}

// GetConn returns shared in-process client connection
func (b *GRPCBus) GetConn() (*grpc.ClientConn, error) {
    b.mu.Lock()
    defer b.mu.Unlock()

    if b.conn != nil {
        return b.conn, nil
    }

    conn, err := grpc.NewClient("passthrough:///bufnet",
        grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
            return b.listener.DialContext(ctx)
        }),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, fmt.Errorf("bufconn dial: %w", err)
    }
    b.conn = conn
    return conn, nil
}

// Stop gracefully stops the gRPC server
func (b *GRPCBus) Stop() {
    b.server.GracefulStop()
    if b.conn != nil {
        b.conn.Close()
    }
}

// IsRegistered checks if a service is registered
func (b *GRPCBus) IsRegistered(serviceName string) bool {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return b.services[serviceName]
}
```

### 2.3 Service Registry Adapter

Adapter để gateway sử dụng bufconn thay vì TCP connections:

```go
// apps/memory/internal/bus/registry.go
package bus

import (
    "context"
    "time"

    gwDomain "github.com/vnp-community/vnp-memory/gateway/internal/domain"
    gwPort   "github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// InProcessRegistry implements gateway's ServiceRegistry port
// using in-process gRPC instead of TCP connections
type InProcessRegistry struct {
    bus    *GRPCBus
    logger *slog.Logger
}

var _ gwPort.ServiceRegistry = (*InProcessRegistry)(nil)

func NewInProcessRegistry(bus *GRPCBus, logger *slog.Logger) *InProcessRegistry {
    return &InProcessRegistry{bus: bus, logger: logger}
}

func (r *InProcessRegistry) Resolve(service string) (*gwDomain.RouteTarget, error) {
    if !r.bus.IsRegistered(service) {
        return nil, fmt.Errorf("service %s not registered in-process", service)
    }
    return &gwDomain.RouteTarget{
        Service: service,
        Address: "bufconn://inprocess",
        Timeout: 30 * time.Second,
    }, nil
}

func (r *InProcessRegistry) Forward(ctx context.Context, target *gwDomain.RouteTarget, payload []byte) ([]byte, error) {
    conn, err := r.bus.GetConn()
    if err != nil {
        return nil, err
    }
    // Use generic gRPC invoke with the service method from target
    // The gateway handler knows which method to call
    return r.invokeGRPC(ctx, conn, target, payload)
}

func (r *InProcessRegistry) HealthCheck(service string) (bool, error) {
    return r.bus.IsRegistered(service), nil
}
```

---

## 3. NATS Embedded

### 3.1 Kiến Trúc

```
┌──────────────────────────────────────────────┐
│              Single Process                    │
│                                                │
│  ┌──────────┐    ┌──────────────┐             │
│  │Publisher  │───►│ NATS Server  │             │
│  │(Service) │    │ (embedded)   │             │
│  └──────────┘    └──────┬───────┘             │
│                         │                      │
│                  ┌──────▼───────┐             │
│                  │ JetStream    │             │
│                  │ (in-memory   │             │
│                  │  or file)    │             │
│                  └──────┬───────┘             │
│                         │                      │
│  ┌──────────┐    ┌──────▼───────┐             │
│  │Subscriber│◄───│ Consumer     │             │
│  │(Service) │    │ Groups       │             │
│  └──────────┘    └──────────────┘             │
└──────────────────────────────────────────────┘
```

### 3.2 Implementation

```go
// apps/memory/internal/bus/nats_embedded.go
package bus

import (
    "fmt"
    "log/slog"
    "time"

    natsserver "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
)

type NATSBus struct {
    server *natsserver.Server
    conn   *nats.Conn
    js     nats.JetStreamContext
    logger *slog.Logger
}

type NATSConfig struct {
    Mode     string // "embedded" | "external"
    URL      string // for external mode
    StoreDir string // JetStream store directory
}

func NewNATSBus(cfg NATSConfig, logger *slog.Logger) (*NATSBus, error) {
    if cfg.Mode == "external" {
        return newExternalNATS(cfg, logger)
    }
    return newEmbeddedNATS(cfg, logger)
}

func newEmbeddedNATS(cfg NATSConfig, logger *slog.Logger) (*NATSBus, error) {
    opts := &natsserver.Options{
        DontListen: true, // No TCP — in-process only
        JetStream:  true,
        StoreDir:   cfg.StoreDir,
        MaxPayload: 8 * 1024 * 1024, // 8MB
    }

    srv, err := natsserver.NewServer(opts)
    if err != nil {
        return nil, fmt.Errorf("create NATS server: %w", err)
    }

    // Silent logging for embedded
    srv.SetLoggerV2(newNATSLogger(logger), false, false, false)
    go srv.Start()

    if !srv.ReadyForConnections(10 * time.Second) {
        return nil, fmt.Errorf("NATS embedded not ready in 10s")
    }

    nc, err := nats.Connect("",
        nats.InProcessServer(srv),
        nats.MaxReconnects(-1),
    )
    if err != nil {
        return nil, fmt.Errorf("connect to embedded NATS: %w", err)
    }

    js, err := nc.JetStream(
        nats.PublishAsyncMaxPending(256),
    )
    if err != nil {
        return nil, fmt.Errorf("create JetStream context: %w", err)
    }

    bus := &NATSBus{
        server: srv,
        conn:   nc,
        js:     js,
        logger: logger,
    }

    // Create required streams
    if err := bus.createStreams(); err != nil {
        return nil, err
    }

    logger.Info("NATS embedded started", "mode", "in-process", "jetstream", true)
    return bus, nil
}

func (b *NATSBus) createStreams() error {
    streams := []struct {
        Name     string
        Subjects []string
    }{
        {"cognee", []string{"cognee.>"}},
        {"graphiti", []string{"graphiti.>"}},
        {"memobase", []string{"memobase.>"}},
        {"openviking", []string{"ov.>"}},
        {"zep", []string{"zep.>"}},
        {"supermemory", []string{"sm.>"}},
        {"admin", []string{"admin.>"}},
    }

    for _, s := range streams {
        _, err := b.js.AddStream(&nats.StreamConfig{
            Name:      s.Name,
            Subjects:  s.Subjects,
            Retention: nats.WorkQueuePolicy,
            MaxAge:    24 * time.Hour,
        })
        if err != nil {
            return fmt.Errorf("create stream %s: %w", s.Name, err)
        }
    }
    return nil
}

func (b *NATSBus) Publish(ctx context.Context, subject string, data []byte) error {
    _, err := b.js.Publish(subject, data)
    return err
}

func (b *NATSBus) Subscribe(subject, consumer string, handler nats.MsgHandler) (*nats.Subscription, error) {
    return b.js.QueueSubscribe(subject, consumer, handler,
        nats.Durable(consumer),
        nats.DeliverNew(),
        nats.ManualAck(),
        nats.AckWait(30*time.Second),
        nats.MaxDeliver(3),
    )
}

func (b *NATSBus) Close() {
    if b.conn != nil {
        b.conn.Drain()
    }
    if b.server != nil {
        b.server.Shutdown()
    }
}
```

---

## 4. Event Subject Mapping

Tất cả NATS subjects giữ nguyên như architecture spec:

| Stream | Subject | Publisher | Subscriber |
|--------|---------|-----------|------------|
| `cognee` | `cognee.data.ingested` | cognee-ingestion | cognee-cognify |
| `cognee` | `cognee.pipeline.completed` | cognee-cognify | cognee-search |
| `graphiti` | `graphiti.episode.ingested` | graphiti-ingestion | graphiti-search |
| `graphiti` | `graphiti.entity.resolved` | graphiti-knowledge | graphiti-search |
| `graphiti` | `graphiti.community.rebuilt` | graphiti-knowledge | graphiti-search |
| `memobase` | `memobase.buffer.ready` | memobase-ingestion | memobase-engine |
| `memobase` | `memobase.engine.completed` | memobase-engine | memobase-context |
| `memobase` | `memobase.profile.changed` | memobase-engine | memobase-context |
| `memobase` | `memobase.event.created` | memobase-engine | vnp-event |
| `openviking` | `ov.resource.ingested` | ov-resource | ov-search |
| `openviking` | `ov.session.committed` | ov-session | ov-search |
| `openviking` | `ov.session.memory.extracted` | ov-session | ov-fs |
| `openviking` | `ov.content.written` | ov-fs | ov-search |
| `openviking` | `ov.content.deleted` | ov-fs | ov-search |
| `openviking` | `ov.crypto.key.rotated` | ov-crypto | ov-fs |
| `zep` | `zep.memory.messages.ingested` | zep-memory | zep-graph |
| `zep` | `zep.graph.extraction.completed` | zep-graph | zep-search |
| `zep` | `zep.graph.fact.created` | zep-graph | zep-search |
| `zep` | `zep.graph.fact.invalidated` | zep-graph | zep-search |
| `zep` | `zep.thread.session.ended` | zep-thread | zep-memory |
| `zep` | `zep.user.deleted` | zep-user | zep-thread, zep-graph |
| `supermemory` | `sm.document.created` | sm-document | sm-memory, sm-search |
| `supermemory` | `sm.document.deleted` | sm-document | sm-memory, sm-search |
| `supermemory` | `sm.memory.created` | sm-memory | sm-search, sm-profile |
| `supermemory` | `sm.memory.forgotten` | sm-memory | sm-search, sm-profile |
| `supermemory` | `sm.connection.synced` | sm-connector | sm-document |
| `supermemory` | `sm.auth.api_key.used` | sm-auth | sm-analytics |
| `admin` | `admin.tenant.created` | vnp-admin | All |
| `admin` | `admin.tenant.deleted` | vnp-admin | All (cascade) |

---

## 5. Performance Comparison

| Metric | Microservices (TCP) | Monolithic (bufconn) | Improvement |
|--------|--------------------|--------------------|-------------|
| gRPC latency (p50) | ~2ms | ~0.05ms | **40x** |
| gRPC latency (p99) | ~10ms | ~0.2ms | **50x** |
| NATS publish | ~1ms | ~0.1ms | **10x** |
| Memory baseline | ~2GB (35 processes) | ~256MB (1 process) | **8x** |
| Startup time | ~30s (docker compose) | ~3s | **10x** |
| Context switch | OS-level | None (goroutines) | — |

---

## 6. Fallback Mode

Khi cần test individual service hoặc debug, vẫn có thể chạy microservices mode:

```go
// apps/memory/internal/bus/registry.go
func (r *InProcessRegistry) WithFallback(externalAddr string) *InProcessRegistry {
    // If service not registered in-process, fall back to external TCP
    r.fallbackAddr = externalAddr
    return r
}
```

Cho phép hybrid deployment: một số services chạy in-process, một số chạy external.

---

## 7. Acceptance Criteria

| # | Criteria | Verification |
|---|----------|-------------|
| AC-1 | gRPC bufconn works for all 35 services | Unit tests |
| AC-2 | NATS embedded creates all 7 streams | Stream list check |
| AC-3 | Event publish/subscribe works in-process | Integration test |
| AC-4 | Gateway uses InProcessRegistry seamlessly | API test |
| AC-5 | Graceful shutdown drains NATS + stops gRPC | Signal test |
| AC-6 | External NATS mode still works | Config toggle test |
