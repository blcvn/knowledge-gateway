# UI Solution: UI-SOL-ENT-001 — Distributed Lease System UI

**Solution ID:** UI-SOL-ENT-001  
**CR References:** [CR-ENT-001](../../../../docs/crs/v5/enterprise/CR-ENT-001-Distributed-Leases.md)  
**Backend Solution:** [SOL-ENT-001](../../../../backend/specs/crs/v5/enterprise/solutions/SOL-ENT-001-Distributed-Leases.md)  
**Feature:** Distributed Leases — Multi-Agent Coordination, Conflict Detection  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/observability/AgentOrchestration.tsx`

---

## 1. Mục Đích

Xây dựng Distributed Lease UI cho multi-agent coordination:
- Active lease status panel với TTL countdown
- Agent registry: all known agents và health status
- Conflict detection: highlight race conditions
- Signal log: cross-agent communication
- Admin: force revoke lease

---

## 2. Backend API Contract

```http
GET    /v1/console/governance/leases        → AgentLease[]
DELETE /v1/console/governance/leases/{id}   → void (revoke)
GET    /v1/console/governance/agents        → AgentInfo[]

# WebSocket channel for realtime:
# agent.lease.acquired  → { lease_id, agent_id, resource, tenant_id, expires_at }
# agent.lease.released  → { lease_id }
# agent.conflict        → { resource, agents: string[], winner: string }
```

### TypeScript Types

```typescript
interface AgentLease {
  lease_id:    string;
  agent_id:    string;
  resource:    string;        // "user:u_123" | "session:s_456"
  tenant_id:   string;
  acquired_at: string;
  expires_at:  string;
  status:      'active' | 'expired' | 'revoked';
  priority:    number;
}

interface AgentConflict {
  resource:   string;
  agents:     string[];
  winner:     string;        // agent_id of winner
  resolved_at: string;
  method:     'first_write_wins';
}
```

---

## 3. Components

### 3.1 Orchestration Dashboard

```
AgentOrchestrationPage
├── ActiveAgentsHeader      ← "4 active agents, 12 leases held"
├── ConflictAlert (if any)  ← banner: "1 conflict detected!"
│   └── ConflictDetail      ← "resource: user:u_123, winner: agent-01"
├── LeaseStatusPanel
│   ├── LeaseFilters        ← agent_id, resource, status
│   └── LeaseTable
│       └── LeaseRow
│           ├── AgentBadge
│           ├── ResourceLabel    ← "user:u_123"
│           ├── ExpiryCountdown  ← "29s / 30s" with progress bar
│           ├── StatusBadge      ← active / expired
│           ├── PriorityBadge    ← priority number
│           └── RevokeButton     ← admin only
└── AgentRegistryPanel (bottom)
    └── AgentCard
        ├── AgentId
        ├── StatusBadge    ← active (green pulse) / idle / error
        ├── ActiveLeases   ← "3 leases"
        └── LastSeen       ← "2s ago"
```

### 3.2 Lease Expiry Countdown

```typescript
function LeaseCountdown({ expiresAt, acquiredAt }: LeaseCountdownProps) {
  const [remaining, setRemaining] = useState(0);
  const total = differenceInSeconds(new Date(expiresAt), new Date(acquiredAt));
  
  useEffect(() => {
    const update = () => {
      const rem = Math.max(0, differenceInSeconds(new Date(expiresAt), new Date()));
      setRemaining(rem);
    };
    update();
    const timer = setInterval(update, 1000);
    return () => clearInterval(timer);
  }, [expiresAt]);
  
  const pct = Math.round((remaining / total) * 100);
  const isWarning = remaining < 5;
  
  return (
    <div className={`flex items-center gap-2 ${isWarning ? 'text-red-500' : ''}`}>
      <div className="h-1.5 w-16 bg-gray-100 rounded overflow-hidden">
        <div className={`h-full ${isWarning ? 'bg-red-500' : 'bg-blue-500'}`}
             style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs">{remaining}s</span>
    </div>
  );
}
```

### 3.3 Realtime WebSocket Integration

```typescript
// Real-time lease updates
useEffect(() => {
  const cleanup1 = wsManager.on('agent.lease.acquired', (lease: AgentLease) => {
    qc.setQueryData(['leases'], (old: AgentLease[]) => [...(old ?? []), lease]);
  });
  
  const cleanup2 = wsManager.on('agent.lease.released', ({ lease_id }) => {
    qc.setQueryData(['leases'], (old: AgentLease[]) =>
      old?.filter(l => l.lease_id !== lease_id)
    );
  });
  
  const cleanup3 = wsManager.on('agent.conflict', (conflict: AgentConflict) => {
    setConflicts(prev => [...prev, conflict]);
    toast.warning(`Conflict resolved: ${conflict.winner} won "${conflict.resource}"`);
  });
  
  return () => { cleanup1(); cleanup2(); cleanup3(); };
}, [qc]);
```

---

## 4. Acceptance Criteria (Frontend)

- [ ] Active leases display with TTL countdown (per second)
- [ ] Lease table updates realtime via WebSocket
- [ ] Conflict alert banner when race condition detected
- [ ] Conflict shows winner agent and resolution method
- [ ] Revoke button requires confirmation (admin only)
- [ ] Agent registry shows status (active/idle/error)
- [ ] Priority badge shown on each lease
