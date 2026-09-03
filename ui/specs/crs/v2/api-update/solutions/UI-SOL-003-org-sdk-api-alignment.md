# UI Solution: UI-SOL-003 — Org & SDK API Frontend Alignment

**Solution ID:** UI-SOL-003  
**CR References:** [CR-002-org-sdk-api](../../../../docs/crs/v2/api-update/CR-002-org-sdk-api.md)  
**Backend Solution:** [SOL-002-org-sdk-api.md](../../../../backend/specs/crs/v2/api-update/solutions/SOL-002-org-sdk-api.md)  
**Feature:** Org Settings + SDK/API Keys + Webhooks  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/services/org.service.ts`, `ui/src/pages/org-settings/`

---

## 1. Mục Đích

Align frontend Org & SDK services với backend API contracts:
- Org Settings: GET/PUT `/v1/console/org/settings`
- Members & Roles: GET `/v1/console/org/members`, `/v1/console/org/roles`
- API Keys: list, create (raw_key shown ONCE), revoke
- Rate Limits: display tier config
- Webhooks: CRUD + test

---

## 2. Backend API Contracts

### Org Settings
```http
GET /v1/console/org/settings → OrgSettings
PUT /v1/console/org/settings (partial) → OrgSettings
GET /v1/console/org/members → OrgMember[]
GET /v1/console/org/roles → OrgRole[]
```

### SDK / API Keys
```http
GET    /v1/console/sdk/keys         → APIKey[]   (raw_key NOT included)
POST   /v1/console/sdk/keys         → { key: APIKey, raw_key: string }  ← show ONCE
DELETE /v1/console/sdk/keys/{id}    → void
GET    /v1/console/sdk/rate-limits  → RateLimitConfig[]
GET    /v1/console/sdk/webhooks     → Webhook[]
POST   /v1/console/sdk/webhooks     → Webhook
DELETE /v1/console/sdk/webhooks/{id}→ void
```

---

## 3. Critical: raw_key Display Logic

```typescript
// IMPORTANT: raw_key shown ONLY ONCE after creation
// Must display in modal BEFORE closing

export function CreateAPIKeyModal() {
  const [rawKey, setRawKey] = useState<string | null>(null);
  
  const createKey = useMutation({
    mutationFn: orgApi.createKey,
    onSuccess: (data) => {
      setRawKey(data.raw_key);           // show immediately
      queryClient.invalidateQueries(['sdk', 'keys']);
    },
  });

  if (rawKey) {
    return (
      <RawKeyDisplay
        rawKey={rawKey}
        warning="This key will never be shown again. Copy it now!"
        onClose={() => setRawKey(null)}
      />
    );
  }

  return <CreateKeyForm onSubmit={createKey.mutate} />;
}
```

---

## 4. Components Architecture

### 4.1 Org Settings Page

```
OrgSettingsPage (tabbed)
├── Tab: General
│   ├── OrgNameInput
│   ├── SlugInput (read-only after creation)
│   ├── TimezoneSelect
│   └── PlanBadge         ← free / pro / enterprise
├── Tab: Members
│   ├── MembersTable
│   │   └── MemberRow     ← name, email, role badge, status, actions
│   └── InviteButton      ← invite new member (future)
└── Tab: Roles
    └── RolesTable        ← role name, permissions list
```

### 4.2 SDK / API Keys Page

```
SDKPage (tabbed)
├── Tab: API Keys
│   ├── CreateKeyButton   ← opens CreateAPIKeyModal
│   ├── KeysTable
│   │   └── KeyRow
│   │       ├── KeyName
│   │       ├── PrefixDisplay   ← "vnp_prod_sk_3f9a..."
│   │       ├── ScopesBadges
│   │       ├── LastUsed
│   │       ├── ExpiryDate
│   │       ├── StatusBadge     ← active / revoked / expired
│   │       └── RevokeButton    ← confirm dialog before revoke
│   └── CreateAPIKeyModal
│       ├── NameInput
│       ├── ScopesCheckboxes
│       ├── ExpiryDaysInput
│       └── RawKeyDisplay (step 2) ← copy-to-clipboard + warning
├── Tab: Rate Limits
│   └── RateLimitsTable    ← scope, rps, rpm, burst, tier
└── Tab: Webhooks
    ├── WebhookList
    │   └── WebhookRow     ← url, events, status, success_rate, actions
    └── CreateWebhookForm  ← url, events multiselect, secret
```

### 4.3 Raw Key Display Component

```typescript
// Full-screen overlay with dimmed background
// "⚠️ COPY THIS KEY NOW — it will never be shown again"
// [Copy to Clipboard] button with success animation
// [I have copied the key] button to close
```

---

## 5. TypeScript Types Alignment

```typescript
// Ensure exact match with backend contracts

interface APIKey {
  id:          string;
  name:        string;
  prefix:      string;    // e.g., "vnp_prod_sk_3f9a..."
  scopes:      string[];
  created_at:  string;
  last_used?:  string;
  expires_at?: string;
  status:      'active' | 'revoked' | 'expired';
  // NOTE: raw_key is NOT in APIKey — only in CreateKeyResponse
}

interface CreateKeyResponse {
  key:     APIKey;
  raw_key: string;   // SHOWN ONCE
}

interface Webhook {
  id:           string;
  url:          string;
  events:       string[];
  status:       'active' | 'paused' | 'failed';
  secret?:      string;     // masked display
  success_rate: number;     // 0.0–1.0
  created_at:   string;
}
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Org settings load và save với partial update (PUT)
- [ ] Members list với role badges
- [ ] API key creation: raw_key displayed in overlay immediately
- [ ] Raw key overlay: cannot be closed without "I copied it" confirmation
- [ ] Key revoke: requires confirmation dialog ("Type key name to confirm")
- [ ] Rate limits table shows tier (free/pro/enterprise) badges
- [ ] Webhook creation: URL validation + events multiselect
- [ ] Webhook success_rate displayed as percentage bar
