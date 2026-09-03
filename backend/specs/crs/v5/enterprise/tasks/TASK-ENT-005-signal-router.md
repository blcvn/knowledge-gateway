# TASK-ENT-005 — NATS Signal Router

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-005 |
| **Wave** | 2 |
| **Solution** | [SOL-ENT-001](../solutions/SOL-ENT-001-Distributed-Leases.md) §1.2 |
| **Component** | `services/orchestration-service/internal/usecase/signal.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-ENT-004 |
| **Estimated** | 3h |

---

## Mục tiêu

NATS-backed signal routing: send typed signals between agents (handoff, alert, update, query).

---

## Công việc cụ thể

### `services/orchestration-service/internal/usecase/signal.go` [NEW]

```go
type SignalType string
const (
    SignalHandoff SignalType = "handoff"  // transfer task to another agent
    SignalAlert   SignalType = "alert"    // urgent notification
    SignalUpdate  SignalType = "update"   // status update
    SignalQuery   SignalType = "query"    // request info from agent
)

type Signal struct {
    Type          SignalType `json:"type"`
    FromAgentID   string     `json:"from_agent_id"`
    Payload       any        `json:"payload"`
    Timestamp     time.Time  `json:"timestamp"`
    CorrelationID string     `json:"correlation_id"` // for request-response
}

type SignalRouter struct {
    nats *nats.Conn
    subs map[string]*nats.Subscription
    mu   sync.Mutex
}

// SendSignal — publish to agent-specific subject
func (r *SignalRouter) SendSignal(ctx context.Context, req *SendSignalRequest) error {
    subject := fmt.Sprintf("agent.signal.%s.%s", req.TenantID, req.TargetAgentID)
    data, err := json.Marshal(Signal{
        Type: SignalType(req.SignalType), FromAgentID: req.FromAgentID,
        Payload: req.Payload, Timestamp: time.Now(),
        CorrelationID: uuid.NewString(),
    })
    if err != nil { return err }
    return r.nats.Publish(subject, data)
}

// Subscribe — agent subscribes to its own signal subject
func (r *SignalRouter) Subscribe(tenantID, agentID string, handler func(*Signal)) (*nats.Subscription, error) {
    subject := fmt.Sprintf("agent.signal.%s.%s", tenantID, agentID)
    return r.nats.Subscribe(subject, func(msg *nats.Msg) {
        var sig Signal
        if err := json.Unmarshal(msg.Data, &sig); err != nil { return }
        handler(&sig)
    })
}
```

---

## Acceptance Criteria

- [ ] Signal delivered to target agent via NATS subject
- [ ] Subject format: `agent.signal.{tenant_id}.{agent_id}`
- [ ] All 4 signal types (handoff, alert, update, query) work
- [ ] CorrelationID for request-response patterns
- [ ] Unit test with embedded NATS server

## Files

```
services/orchestration-service/internal/usecase/signal.go       [NEW]
services/orchestration-service/internal/usecase/signal_test.go  [NEW]
```
