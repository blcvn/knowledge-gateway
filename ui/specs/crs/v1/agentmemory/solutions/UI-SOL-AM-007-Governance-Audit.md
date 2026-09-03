# UI Solution: UI-SOL-AM-007 — Governance, Audit & Diagnostics

**Solution ID:** UI-SOL-AM-007  
**CR References:** [CR-AM-007](../../../../docs/crs/v1/agentmemory/CR-AM-007-Governance-Audit-Diagnostics.md)  
**Backend Solution:** [SOL-007-Governance-Audit-Diagnostics.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-007-Governance-Audit-Diagnostics.md)  
**Feature:** Governance Center — GDPR, OPA Policies, Audit Trail  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/governance/`

---

## 1. Mục Đích

Xây dựng Governance Center UI với:
- Memory visibility: admin xem tất cả memories của bất kỳ user nào
- GDPR Forget wizard: 2-step confirmation → cascading delete → audit certificate
- OPA Policy editor: viết/test/deploy Rego policies
- Audit trail: immutable log với export (CSV/JSON)

---

## 2. Backend API Alignment

### API Endpoints

| Method | Path | Mô tả |
|--------|------|--------|
| `GET` | `/v1/console/governance/tenants` | List tenants |
| `POST` | `/v1/console/governance/tenants` | Create tenant |
| `GET` | `/v1/console/governance/policies` | List OPA policies |
| `POST` | `/v1/console/governance/policies` | Create policy |
| `PUT` | `/v1/console/governance/policies/{id}` | Update policy |
| `GET` | `/v1/console/governance/audit` | Audit logs (filtered) |
| `POST` | `/v1/console/governance/gdpr/forget/preview` | GDPR dry-run |
| `POST` | `/v1/console/governance/gdpr/forget` | GDPR execute |
| `GET` | `/v1/console/governance/memories` | User memories (admin) |

---

## 3. Components Architecture

### 3.1 Governance Navigation

```
GovernancePage (tabbed layout)
├── Tab: Memory Visibility
├── Tab: GDPR Forget
├── Tab: OPA Policies
├── Tab: Audit Trail
└── Tab: Tenants
```

### 3.2 Memory Visibility Panel

```
MemoryVisibilityPanel
├── UserSearchInput         ← search by user_id
├── EngineFilter            ← all engines checkboxes
├── MemoryTable             ← tabular list
│   └── MemoryRow
│       ├── EngineTag
│       ├── TypeBadge
│       ├── ContentPreview  ← first 80 chars
│       ├── CreatedAt
│       └── ViewButton      ← open in memory explorer
└── ExportButton            ← export as JSON/CSV
```

### 3.3 GDPR Forget Wizard

```
GDPRForgetWizard (modal, 3 steps)

Step 1: Select User
  ├── UserIdInput
  ├── TenantSelect
  └── ReasonSelect (gdpr_request | user_request | admin | legal)

Step 2: Preview (dry-run)
  ├── WarningBanner         ← "This will delete X memories"
  ├── BreakdownTable        ← per-engine count
  │   ├── cognee    │ 234 memories
  │   ├── graphiti  │ 189 memories
  │   └── ...
  └── ConfirmCheckbox       ← "I understand this is irreversible"

Step 3: Execute & Certificate
  ├── ProgressList          ← per-engine deletion status (live)
  │   ├── ✅ cognee    — deleted 234
  │   ├── ✅ graphiti  — deleted 189
  │   └── ⏳ memobase  — in progress...
  ├── AuditID               ← "Audit: audit_xyz123"
  └── DownloadButton        ← "Download Compliance Certificate (PDF)"
```

### 3.4 OPA Policy Editor

```
PolicyEditorPage
├── PolicyList (left sidebar)
│   └── PolicyItem          ← name, scope, enabled toggle
├── PolicyEditor (center)
│   ├── PolicyName          ← input
│   ├── RegoEditor          ← Monaco editor with Rego syntax highlighting
│   ├── TestPanel           ← input test data, run policy
│   │   ├── TestInput       ← JSON input
│   │   └── TestResult      ← allow/deny + reason
│   └── SaveButton
└── DeployedBadge           ← "Live" | "Draft"
```

### 3.5 Audit Trail Table

```
AuditTrailPage
├── AuditFilters
│   ├── ActionFilter        ← store | recall | forget | policy_check
│   ├── ActorInput          ← user_id / agent_id
│   ├── EntityTypeSelect    ← memory | user | tenant | policy
│   └── DateRange
├── AuditTable (immutable styling)
│   └── AuditRow
│       ├── Timestamp        ← precise ISO 8601
│       ├── ActionBadge      ← color coded
│       ├── Actor
│       ├── EntityType
│       ├── Result           ← success/denied
│       └── DetailLink
└── ExportButton            ← "Export CSV" / "Export JSON"
```

---

## 4. React Query Hooks

```typescript
// ui/src/api/hooks/useGovernance.ts

export function useGDPRPreview(userId: string, tenantId: string) {
  return useMutation({
    mutationFn: () => governanceApi.gdprPreview({ user_id: userId, tenant_id: tenantId }),
  });
}

export function useGDPRForget() {
  return useMutation({
    mutationFn: (req: { user_id: string; tenant_id: string; reason: string }) =>
      governanceApi.gdprForget(req),
  });
}

export function useAuditLogs(filters: AuditFilters) {
  return useQuery({
    queryKey: ['governance', 'audit', filters],
    queryFn:  () => governanceApi.getAuditLogs(filters),
  });
}
```

---

## 5. Security Guards

```typescript
// All Governance pages require super_admin role
// ui/src/pages/governance/GovernanceLayout.tsx

export function GovernanceLayout({ children }) {
  const { user } = useAuth();
  if (user?.role !== 'super_admin') {
    return <AccessDenied message="Governance Center requires super_admin role" />;
  }
  return children;
}
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Memory visibility: search by user_id + filter by engine → loads in `< 2s`
- [ ] GDPR wizard step 1→2: preview dry-run shows per-engine count
- [ ] GDPR wizard step 2→3: live status updates as engines delete
- [ ] GDPR completion: audit_id displayed + PDF download available
- [ ] OPA editor: Monaco editor với Rego syntax highlighting
- [ ] OPA test: run policy on JSON input → show allow/deny result
- [ ] Audit log: immutable (no edit/delete buttons visible)
- [ ] Audit export: CSV và JSON formats
- [ ] Access guard: non-super_admin gets `403 Access Denied` page
