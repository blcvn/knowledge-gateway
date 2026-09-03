# UI Solution: UI-SOL-CORE-003 — Cascading GDPR Forget UI

**Solution ID:** UI-SOL-CORE-003  
**CR References:** [CR-CORE-003](../../../../docs/crs/v3/core-memory/CR-CORE-003-Cascading-Forget.md)  
**Backend Solution:** [SOL-CORE-003](../../../../backend/specs/crs/v3/core-memory/solutions/SOL-CORE-003-Cascading-Forget.md)  
**Feature:** GDPR Cascading Forget — All-Engine Delete Wizard  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/governance/GDPRForget.tsx`

---

## 1. Mục Đích

Xây dựng GDPR Forget UI:
- 2-step wizard: Preview → Execute
- Live deletion progress per engine
- Audit certificate download
- Tenant isolation guard (cannot cross-tenant)

---

## 2. Backend API Contract

```http
# Step 1: Preview (dry-run)
POST /v1/console/governance/gdpr/forget/preview
{ "user_id": string }
→ {
    "user_id":             string,
    "estimated_items":     number,
    "breakdown_by_engine": { "cognee": 234, "graphiti": 189, ... },
    "warnings":            string[]
  }

# Step 2: Execute
POST /v1/console/governance/gdpr/forget
{ "user_id": string }
→ {
    "success":       boolean,
    "deleted_count": number
  }

# Admin Forget (direct, for super_admin)
POST /v1/admin/forget
{ "user_id": string, "tenant_id": string, "reason": string, "request_id": string }
→ {
    "user_id":      string,
    "deleted_from": string[],
    "duration_ms":  number,
    "audit_id":     string
  }
```

---

## 3. Wizard Components

### Step 1: User Selection

```
GDPRForgetStep1
├── WarningBanner           ← "⚠️ This action is IRREVERSIBLE"
├── UserIdInput             ← text input (required)
├── TenantDisplay           ← auto-filled from current tenant context
├── ReasonSelect
│   ├── gdpr_request        ← "User GDPR Article 17 request"
│   ├── user_request        ← "User account deletion"
│   ├── admin               ← "Administrative action"
│   └── legal               ← "Legal requirement"
├── RequestIdInput          ← optional tracking ID (e.g., GDPR-2026-001)
└── PreviewButton           ← POST /gdpr/forget/preview
```

### Step 2: Preview & Confirm

```
GDPRForgetStep2
├── PreviewHeader
│   └── EstimatedTotal      ← "Will delete 847 memory items"
├── BreakdownTable
│   └── EngineRow           ← engine name | count | type breakdown
│       ├── cognee          │ 234 items
│       ├── graphiti        │ 189 items
│       ├── memobase        │ 156 items
│       ├── zep             │ 145 items
│       ├── openviking      │  67 items
│       ├── supermemory     │  45 items
│       └── observe         │  11 events
├── WarningsList            ← from response.warnings[]
├── ConfirmCheckbox         ← "I understand this cannot be undone"
└── ExecuteButton           ← POST /v1/admin/forget (disabled until checked)
```

### Step 3: Progress & Certificate

```
GDPRForgetStep3
├── DurationDisplay         ← "Completed in 1.8 seconds"
├── EngineStatusList        ← live status per engine
│   ├── ✅ cognee    — deleted 234 items
│   ├── ✅ graphiti  — deleted 189 items
│   ├── ⏳ memobase  — in progress...
│   ├── ❌ observe   — failed: timeout (partial)
│   └── ...
├── AuditSection
│   ├── AuditIdDisplay      ← "Audit ID: audit_xyz123"
│   └── AuditTimestamp
└── DownloadSection
    ├── DownloadPDFButton   ← "Download Compliance Certificate"
    └── DownloadJSONButton  ← "Download JSON Report"
```

---

## 4. Implementation

```typescript
// ui/src/pages/governance/GDPRForget.tsx

type WizardStep = 'input' | 'preview' | 'executing' | 'complete';

export function GDPRForgetWizard() {
  const [step, setStep] = useState<WizardStep>('input');
  const [userId, setUserId] = useState('');
  const [preview, setPreview] = useState<GDPRPreviewResponse | null>(null);
  const [result, setResult] = useState<ForgetResponse | null>(null);
  
  const gdprPreview = useMutation({ mutationFn: governanceApi.gdprPreview });
  const gdprForget  = useMutation({ mutationFn: governanceApi.gdprForget });
  
  const handlePreview = async () => {
    const res = await gdprPreview.mutateAsync({ user_id: userId });
    setPreview(res);
    setStep('preview');
  };
  
  const handleExecute = async () => {
    setStep('executing');
    const res = await gdprForget.mutateAsync({ user_id: userId });
    setResult(res);
    setStep('complete');
  };
  
  // ... render based on step
}
```

---

## 5. Compliance Certificate Content

```
VNP Memory Platform
GDPR Data Erasure Certificate

Date:         2026-09-03T12:00:00Z
Request ID:   GDPR-2026-001
User ID:      u_123 (hashed: sha256:...)
Reason:       gdpr_request
Requested by: admin@tenant.com
Audit ID:     audit_xyz123

Deletion Summary:
  - Total items deleted: 847
  - Engines processed: 7/7
  - Completion time: 1.847 seconds

This document certifies compliance with GDPR Article 17
(Right to Erasure) for the above user.
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Step 1: UserID input validation (non-empty)
- [ ] Step 2: Preview shows per-engine breakdown from backend
- [ ] `ConfirmCheckbox` required before Execute button enabled
- [ ] Step 3: Engine status list (✅/❌/⏳ per engine)
- [ ] Audit ID prominently displayed after completion
- [ ] PDF download generates formatted compliance certificate
- [ ] Tenant isolation: UI hardcodes current tenant_id (không cho chọn khác)
- [ ] Only `super_admin` role can access this wizard
