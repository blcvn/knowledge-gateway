# ADR-003 — NATS JetStream làm Message Broker

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-02 |
| **Deciders** | Platform Team |
| **Feature** | F08 (Observe), F11 (Orchestration), F12 (Consolidation), F28 (WebSocket Events) |

---

## Context

VNP Memory cần async event bus cho:
- Observe hook events (high volume, burst)
- Session lifecycle signals (agent coordination)
- Consolidation pipeline triggers
- Real-time console events (WebSocket fan-out)

Requirements:
- Guaranteed delivery (at-least-once) — hook events không được mất
- Low latency (< 10ms publish)
- Embeddable (development không cần external broker)
- Horizontal scale (production)
- Replay (failed consumers có thể reprocess)

---

## Decision

**NATS JetStream với embedded mode cho development.**

```go
// Embedded NATS (development)
opts := natsserver.Options{
    JetStream: true,
    NoLog:     true,
    Port:      -1, // random port
}
ns, _ := natsserver.NewServer(&opts)
go ns.Start()

// External NATS (production)
// VNP_MEMORY_NATS_MODE=external
nc, _ := nats.Connect("nats://nats:4222")
js, _ := nc.JetStream()

// Stream definitions
js.AddStream(&nats.StreamConfig{
    Name:      "MEMORY_EVENTS",
    Subjects:  []string{"memory.*"},
    Retention: nats.WorkQueuePolicy,  // one consumer processes each message
    MaxAge:    24 * time.Hour,
})

// Key subjects:
// memory.blob.inserted      → consolidation trigger
// agent.hook.captured       → session state update
// agent.session.complete    → tier-1 consolidation start
// orchestration.signal      → multi-agent signals
// platform.event.created    → WebSocket fan-out
```

---

## Consequences

**Positive:**
- **Embedded mode:** Zero infra cho development — 1 binary, 0 external dependencies
- **Switch production:** 1 env var: `VNP_MEMORY_NATS_MODE=external`
- **At-least-once delivery** via JetStream acknowledgements
- **Consumer groups** cho load balancing consolidation workers
- **Replay** từ sequence number khi consumer crash và restart

**Negative:**
- JetStream state persistence cần storage (file/memory mode)
- NATS cluster setup phức tạp hơn Kafka cho HA (cần 3 nodes)
- Less ecosystem so với Kafka (ít connectors)

---

## Alternatives Considered

### A1 — Apache Kafka
- **Rejected:** Quá heavy cho development; requires Zookeeper hoặc KRaft; không embeddable; JVM latency; over-engineered cho volume hiện tại

### A2 — Redis Pub/Sub
- **Rejected:** No persistence (fire-and-forget); không suitable cho guaranteed delivery; hook events có thể mất nếu consumer down

### A3 — RabbitMQ
- **Rejected:** Erlang runtime dependency; không embeddable; setup phức tạp hơn NATS
