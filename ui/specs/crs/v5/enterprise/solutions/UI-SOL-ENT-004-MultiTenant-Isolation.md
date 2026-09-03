# UI Solution: UI-SOL-ENT-004 — Multi-Tenant Isolation UI

**Solution ID:** UI-SOL-ENT-004  
**CR References:** [CR-ENT-004](../../../../docs/crs/v5/enterprise/CR-ENT-004-MultiTenant-Isolation.md)  
**Backend Solution:** [SOL-ENT-004](../../../../backend/specs/crs/v5/enterprise/solutions/SOL-ENT-004-MultiTenant-Isolation.md)  
**Feature:** Multi-Tenant Isolation — Tenant Switcher, Isolation Badges, Cross-Tenant Guard  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/components/layouts/TenantContext.tsx`

---

## 1. Mục Đích

Xây dựng Multi-Tenant Isolation UI:
- Tenant context: current tenant clearly visible
- Tenant switcher (super_admin only)
- Isolation indicator: every request shows tenant_id context
- Cross-tenant access guard (UI prevention layer)
- Tenant-scoped data labeling

---

## 2. Backend Tenant Headers

```http
# Every API request MUST include:
X-Tenant-ID: {tenant_id}

# Every API response includes:
X-Tenant-ID: {tenant_id}   ← echo back for verification

# If tenant mismatch → 403 Forbidden
```

---

## 3. Components

### 3.1 Tenant Context Provider

```typescript
// ui/src/contexts/TenantContext.tsx

interface TenantContext {
  currentTenant: string;
  tenantName:    string;
  tenantPlan:    'free' | 'pro' | 'enterprise';
  switchTenant:  (tenantId: string) => void;  // super_admin only
  canSwitchTenant: boolean;                   // true if super_admin
}

export const TenantProvider = ({ children }) => {
  const { user } = useAuth();
  const [currentTenant, setCurrentTenant] = useState(
    localStorage.getItem('tenant_id') ?? user?.tenant_id ?? ''
  );
  
  // Inject X-Tenant-ID into all API requests (in api-client.ts)
  // On tenant switch: invalidate all queries
  const switchTenant = (tenantId: string) => {
    if (user?.role !== 'super_admin') {
      throw new Error('Only super_admin can switch tenants');
    }
    localStorage.setItem('tenant_id', tenantId);
    setCurrentTenant(tenantId);
    queryClient.clear();     // invalidate ALL cached data
    navigate('/dashboard');  // redirect to dashboard
  };
  
  return (
    <TenantContext.Provider value={{ currentTenant, switchTenant, ... }}>
      {children}
    </TenantContext.Provider>
  );
};
```

### 3.2 Tenant Badge in Navigation

```typescript
// Always visible in top navigation bar
function TenantBadge() {
  const { currentTenant, tenantName, tenantPlan } = useTenant();
  
  return (
    <div className="flex items-center gap-2 px-3 py-1 
                    bg-blue-50 border border-blue-200 rounded-full">
      <BuildingIcon className="w-4 h-4 text-blue-600" />
      <span className="text-sm font-medium text-blue-700">{tenantName}</span>
      <PlanBadge plan={tenantPlan} />
    </div>
  );
}

// PlanBadge: [Free] (gray) | [Pro] (blue) | [Enterprise] (gold)
```

### 3.3 Tenant Switcher (super_admin)

```typescript
// TenantSwitcher.tsx - shown in user menu for super_admin only
function TenantSwitcher() {
  const { user } = useAuth();
  const { switchTenant, currentTenant } = useTenant();
  
  if (user?.role !== 'super_admin') return null;
  
  const { data: tenants } = useQuery({
    queryKey: ['governance', 'tenants'],
    queryFn:  () => governanceApi.getTenants(),
  });
  
  return (
    <Select
      value={currentTenant}
      onChange={switchTenant}
      label="Switch Tenant"
    >
      {tenants?.map(t => (
        <Option key={t.id} value={t.id}>
          {t.name} — {t.plan}
          {t.status === 'Suspended' && <SuspendedBadge />}
        </Option>
      ))}
    </Select>
  );
}
```

### 3.4 Cross-Tenant Access Guard

```typescript
// When rendering data: verify tenant matches
function TenantScopedComponent({ data, expectedTenantId }) {
  const { currentTenant } = useTenant();
  
  if (data.tenant_id && data.tenant_id !== currentTenant) {
    // Should never happen if backend is correct
    console.error('SECURITY: Cross-tenant data detected!', {
      expected: currentTenant,
      received: data.tenant_id,
    });
    return <CrossTenantAlert />;
  }
  
  return <DataDisplay data={data} />;
}
```

### 3.5 Data Isolation Indicators

```typescript
// Every data row shows tenant scope
function TenantScopeIndicator({ tenantId }: { tenantId: string }) {
  const { currentTenant } = useTenant();
  
  if (tenantId === currentTenant) {
    return <span className="text-xs text-green-600">✓ Tenant-scoped</span>;
  }
  
  // This should never appear in normal operation
  return <span className="text-xs text-red-600 font-bold">⚠️ TENANT MISMATCH</span>;
}
```

---

## 4. Security Requirements (UI Layer)

1. `tenant_id` extracted from JWT on login, stored in localStorage
2. All API requests include `X-Tenant-ID` header (enforced in api-client.ts)
3. Tenant switch clears ALL query cache to prevent stale cross-tenant data
4. `currentTenant` state NEVER accepted from URL params (only from auth token)
5. Cross-tenant data received → log security warning + show alert component

---

## 5. Acceptance Criteria (Frontend)

- [ ] Tenant badge always visible in navigation header
- [ ] Plan badge (Free/Pro/Enterprise) shown in tenant badge
- [ ] Tenant switcher only visible for `super_admin` role
- [ ] Switching tenant: clears all query cache + redirects to dashboard
- [ ] `X-Tenant-ID` header included in every API request (verified in tests)
- [ ] Cross-tenant data detection: console warning + `CrossTenantAlert` component
- [ ] Suspended tenant: shown with `SUSPENDED` badge in switcher
- [ ] Tenant context NEVER from URL params (security requirement)
