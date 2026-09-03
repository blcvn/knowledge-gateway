# SOL-AM-004 — Solution: Multi-Agent Orchestration Layer

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-004 |
| **CR** | CR-AM-004 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/orchestration-service` |

---

## 1. Giải pháp

See SOL-ENT-001 (Distributed Leases) for core implementation.
AgentMemory additions:
- Action DAG state machine
- Agent registry (which agents are active)
- Cross-agent memory sharing

### Action DAG

```go
// services/orchestration-service/internal/domain/dag.go [NEW]
type ActionDAG struct {
    ID       string
    TenantID string
    Nodes    []DAGNode  // each node is an agent action
    Edges    []DAGEdge  // dependency edges
    Status   string     // pending, running, completed, failed
}

type DAGNode struct {
    ID       string; Action string; AgentID string
    DependsOn []string; Status string
}

// Execute DAG: topological sort → execute ready nodes in parallel
func (e *DAGExecutor) Execute(ctx context.Context, dag *ActionDAG) error {
    order := topologicalSort(dag.Nodes)
    for _, batch := range order {
        var wg sync.WaitGroup
        for _, node := range batch {
            wg.Add(1)
            go func(n DAGNode) {
                defer wg.Done()
                e.executeNode(ctx, n)
            }(node)
        }
        wg.Wait()
    }
    return nil
}
```

## 2. Acceptance Criteria

- [ ] Lease acquire/release (SOL-ENT-001)
- [ ] Action DAG: parallel execution of independent steps
- [ ] Agent registry: list active agents per session
- [ ] Cross-agent signal routing via NATS

