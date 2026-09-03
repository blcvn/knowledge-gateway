# UI Solution: UI-SOL-AM-004 — Multi-Agent Orchestration

**Solution ID:** UI-SOL-AM-004  
**CR References:** [CR-AM-004](../../../../docs/crs/v1/agentmemory/CR-AM-004-Multi-Agent-Orchestration.md)  
**Backend Solution:** [SOL-004-Multi-Agent-Orchestration.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-004-Multi-Agent-Orchestration.md)  
**Feature:** Multi-Agent Orchestration — Lease System, Signal Log  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/observability/` + `ui/src/pages/governance/`

---

## 1. Mục Đích

Xây dựng UI cho Multi-Agent Orchestration:
- Xem tất cả active leases và agent assignments
- Monitor agent conflicts và race conditions
- Xem signal log (messages giữa agents)
- Admin actions: revoke lease, force release

---

## 2. Backend API Alignment

### API Endpoints (từ admin APIs)

| Method | Path | Mô tả |
|--------|------|--------|
| `GET` | `/v1/console/governance/agents` | Danh sách active agents |
| `GET` | `/v1/console/governance/leases` | Active lease assignments |
| `DELETE` | `/v1/console/governance/leases/{id}` | Revoke a lease |
| `GET` | `/v1/console/observability/traces` | Agent operation traces |

### TypeScript Types

```typescript
// ui/src/types/orchestration.ts

interface AgentLease {
  lease_id:    string;
  agent_id:    string;
  resource:    string;        // "user:u_123", "session:s_456"
  tenant_id:   string;
  acquired_at: string;
  expires_at:  string;
  status:      'active' | 'expired' | 'revoked';
  priority:    number;
}

interface AgentInfo {
  agent_id:     string;
  name:         string;
  status:       'active' | 'idle' | 'error';
  active_leases: number;
  last_seen:    string;
}

interface SignalEvent {
  signal_id:   string;
  from_agent:  string;
  to_agent:    string;
  signal_type: 'release' | 'acquire' | 'heartbeat' | 'conflict';
  payload:     Record<string, unknown>;
  timestamp:   string;
}
```

---

## 3. Components Architecture

```
MultiAgentDashboard
├── AgentStatusGrid             ← card grid: agent_id, status, lease count
│   └── AgentCard
│       ├── StatusBadge         ← active (green pulse) / idle / error
│       ├── LeaseCount          ← "3 active leases"
│       └── LastSeen            ← relative time
├── LeaseTable                  ← active leases với TTL countdown
│   └── LeaseRow
│       ├── AgentBadge
│       ├── ResourceLabel       ← "user:u_123"
│       ├── ExpiryCountdown     ← "expires in 42s"
│       └── RevokeButton        ← admin only
├── SignalLog (right panel)     ← chronological signal events
│   └── SignalEntry             ← from/to agent, signal_type badge
└── ConflictAlert               ← banner nếu có active conflicts
```

---

## 4. Real-time Updates

```typescript
// WebSocket channel subscription
// ui/src/api/hooks/useAgentOrchestration.ts

export function useActiveLeasesRealtime() {
  const qc = useQueryClient();
  
  useEffect(() => {
    const ws = connectWebSocket('/v1/console/ws');
    ws.on('agent.lease.acquired', (data: AgentLease) => {
      qc.setQueryData(['orchestration', 'leases'], 
        (old: AgentLease[]) => [...(old ?? []), data]);
    });
    ws.on('agent.lease.released', (data: { lease_id: string }) => {
      qc.setQueryData(['orchestration', 'leases'],
        (old: AgentLease[]) => old?.filter(l => l.lease_id !== data.lease_id));
    });
    return () => ws.close();
  }, [qc]);
}
```

---

## 5. UI Design

### Lease Expiry Countdown
```
[████████░░] 23s/30s  resource: user:u_123 (agent: claude-code-01)
[██░░░░░░░░]  8s/30s  resource: session:s_789 (agent: gpt-4-agent)  ← warning color
```

### Conflict Detection Alert
```
⚠️ CONFLICT DETECTED: 2 agents attempting to acquire lease for "user:u_123"
  - agent-01: requested 10:00:01.123
  - agent-02: requested 10:00:01.456
  → Winner: agent-01 (first-write-wins)
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Agent status grid cập nhật realtime qua WebSocket
- [ ] Lease table hiển thị countdown TTL (đếm ngược)
- [ ] Revoke button chỉ hiện với admin role
- [ ] Conflict alert hiển thị khi có race condition
- [ ] Signal log scrolls và filter by agent/signal_type
