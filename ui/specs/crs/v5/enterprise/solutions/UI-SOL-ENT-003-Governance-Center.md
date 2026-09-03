# UI Solution: UI-SOL-ENT-003 — Enterprise Governance Center UI

**Solution ID:** UI-SOL-ENT-003  
**CR References:** [CR-ENT-003](../../../../docs/crs/v5/enterprise/CR-ENT-003-Governance-Center.md)  
**Backend Solution:** [SOL-ENT-003](../../../../backend/specs/crs/v5/enterprise/solutions/SOL-ENT-003-Governance-Center.md)  
**Feature:** Governance Center — GDPR, OPA Policies, Audit Trail, Compliance Export  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/governance/`

---

## 1. Mục Đích

Enterprise Governance Center — full compliance UI:
- Memory visibility: admin view của tất cả memories theo user/tenant
- GDPR Forget Wizard với compliance certificate
- OPA Policy editor với real-time test
- Immutable audit trail với export (CSV/JSON)
- Role-based access control (super_admin only)

---

## 2. Backend API Contract

```http
# Memory visibility
GET /v1/console/governance/memories?user_id=u_123&tenant_id=t_456 → MemoryItem[]

# GDPR
POST /v1/console/governance/gdpr/forget/preview → GDPRPreviewResponse
POST /v1/console/governance/gdpr/forget          → GDPRForgetResponse
POST /v1/admin/forget                            → ForgetResponse (with audit_id)

# OPA Policies
GET  /v1/console/governance/policies        → Policy[]
POST /v1/console/governance/policies        → Policy
PUT  /v1/console/governance/policies/{id}   → Policy

# Audit
GET /v1/console/governance/audit?action=...&actor_id=...&from=...&to=... → AuditLogEntry[]

# Tenants
GET  /v1/console/governance/tenants         → Tenant[]
POST /v1/console/governance/tenants         → Tenant
PUT  /v1/console/governance/tenants/{id}    → Tenant
```

---

## 3. Components Architecture

### 3.1 Governance Navigation

```
GovernancePage
├── GovernanceTabs
│   ├── 🔍 Memory Visibility
│   ├── 🗑️ GDPR Forget
│   ├── ⚖️ OPA Policies
│   ├── 📋 Audit Trail
│   └── 🏢 Tenants
└── [Content per tab]
```

### 3.2 Memory Visibility Panel

```
MemoryVisibilityPanel
├── SearchBar               ← user_id text input
├── TenantSelect            ← filter by tenant (super_admin can cross-tenant view)
├── EngineFilter            ← checkboxes
├── DateRange
├── MemoryGrid              ← tabular view of all memories for user
│   └── MemoryRow
│       ├── EngineTag
│       ├── TypeBadge
│       ├── ContentSnippet  ← first 60 chars, click to expand
│       ├── CreatedAt
│       └── ActionButtons   ← View | Flag for Deletion
└── ExportButtons           ← "Export as JSON" | "Export as CSV"
```

### 3.3 GDPR Forget Wizard (3 steps)

```
Step 1: Identify User
├── UserIdInput (required)
├── TenantSelect (required)
├── ReasonSelect (gdpr_request | user_request | admin | legal)
└── RequestIdInput (optional, for compliance tracking)

Step 2: Preview (dry-run)
├── WarningBanner: "⚠️ Irreversible Action"
├── BreakdownTable: per-engine counts
├── warnings[]: any notes from backend
├── ConfirmCheckbox: required
└── ExecuteButton

Step 3: Progress & Certificate
├── EngineProgressList (live status)
├── AuditIdDisplay
└── DownloadButtons (PDF + JSON)
```

### 3.4 OPA Policy Editor

```typescript
// Monaco editor with Rego language support
// Policy structure:
interface Policy {
  id:           string;
  name:         string;
  description?: string;
  rego_code:    string;    // OPA Rego policy code
  scope:        string;    // "memory.store" | "memory.recall" | "admin"
  enabled:      boolean;
  tenant_id?:   string;
  created_at?:  string;
}

// Example Rego policy:
const examplePolicy = `
package vnp.memory

deny[reason] {
  input.operation == "store"
  input.memory.type == "semantic"
  contains_pii(input.memory.content)
  reason := "PII detected in semantic memory"
}
`;
```

### 3.5 Audit Trail Table

```
AuditTrailPanel
├── AuditFilters (collapsible)
│   ├── ActionMultiSelect    ← store | recall | forget | policy_check | key_create...
│   ├── ActorInput           ← user_id or agent_id
│   ├── EntityTypeSelect     ← memory | user | tenant | policy | api_key
│   └── DateRangePicker      ← from + to
├── AuditTable (immutable styling — no edit/delete)
│   └── AuditRow
│       ├── Timestamp         ← precise ISO 8601
│       ├── ActionBadge       ← color-coded by action type
│       ├── Actor             ← user or agent
│       ├── EntityType
│       ├── EntityId          ← linked to resource
│       ├── Result            ← ✅ success | ❌ denied | ⚠️ warning
│       └── DetailLink        ← expand payload
├── ResultCount               ← "1,247 events"
└── ExportRow                 ← "Export CSV" | "Export JSON"
    ↑ max 10,000 records per export
```

---

## 4. Tenant Management

```
TenantsPage
├── TenantSearch
├── TenantTable
│   └── TenantRow
│       ├── TenantId + Slug
│       ├── Name
│       ├── PlanBadge         ← free | pro | enterprise
│       ├── StatusBadge       ← Active | Suspended
│       ├── CreatedAt
│       └── EditButton        ← opens TenantEditModal
└── CreateTenantButton
```

---

## 5. Access Control Guards

```typescript
// GovernanceGuard.tsx — restricts access to super_admin only
export function GovernanceGuard({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  
  if (!user) return <Navigate to="/login" />;
  
  if (user.role !== 'super_admin') {
    return (
      <AccessDeniedPage
        title="Governance Center"
        message="This area requires super_admin privileges."
        contactMessage="Contact your system administrator."
      />
    );
  }
  
  return <>{children}</>;
}
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] All 5 governance tabs functional
- [ ] Memory visibility: search + filter → load in `< 2s`
- [ ] GDPR wizard: 3-step flow with confirmation guard
- [ ] GDPR completion: PDF certificate downloadable
- [ ] OPA editor: Monaco with syntax highlighting
- [ ] OPA test: input JSON → shows allow/deny + reason
- [ ] Audit trail: immutable (no delete/edit buttons)
- [ ] Audit export: max 10,000 records, CSV + JSON
- [ ] Tenant management: create + edit + status toggle
- [ ] Role guard: non-super_admin → AccessDenied page
