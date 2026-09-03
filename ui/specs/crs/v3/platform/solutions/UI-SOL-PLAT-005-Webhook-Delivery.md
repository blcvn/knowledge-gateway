# UI Solution: UI-SOL-PLAT-005 — Webhook Delivery System UI

**Solution ID:** UI-SOL-PLAT-005  
**CR References:** [CR-PLAT-005](../../../../docs/crs/v3/platform/CR-PLAT-005-Webhook-Delivery-System.md)  
**Feature:** Webhooks — CRUD, Delivery Log, Test Button  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/api-sdk/WebhooksPage.tsx`

---

## 1. Mục Đích

Xây dựng Webhook Management UI:
- Danh sách webhooks với status và success rate
- Create/Edit webhook với event multiselect
- Delivery history per webhook
- Test button để trigger test event

---

## 2. Backend API Contract

```http
GET    /v1/console/sdk/webhooks              → Webhook[]
POST   /v1/console/sdk/webhooks              → Webhook
PUT    /v1/console/sdk/webhooks/{id}         → Webhook
DELETE /v1/console/sdk/webhooks/{id}         → void
GET    /v1/console/sdk/webhooks/{id}/deliveries → delivery history
POST   /v1/console/sdk/webhooks/{id}/test    → trigger test event
```

### Supported Events (6 types)

```
memory.stored       → memory stored to an engine
memory.forgotten    → user data deleted (GDPR)
session.completed   → agent session ended
rate_limit.exceeded → tenant hit rate limit
health.degraded     → service health changed
pipeline.completed  → consolidation pipeline completed
```

---

## 3. Components

### 3.1 Webhooks List

```
WebhooksPage
├── CreateWebhookButton
├── WebhookCards (grid)
│   └── WebhookCard
│       ├── WebhookURL         ← masked: "https://api.example.co...endpoint"
│       ├── EventBadges        ← selected events as chips
│       ├── StatusBadge        ← active (green) / paused (gray) / failed (red)
│       ├── SuccessRateBar     ← "95% [█████████░]"
│       ├── TestButton         ← "Send Test →"
│       ├── DeliveryLogButton  ← "View 50 deliveries"
│       └── EditDeleteButtons
└── CreateWebhookModal
```

### 3.2 Create/Edit Webhook Form

```typescript
interface CreateWebhookForm {
  url:     string;       // URL validation required
  events:  string[];     // multi-select checkboxes
  secret?: string;       // optional HMAC signing secret
}

// Event selector:
const AVAILABLE_EVENTS = [
  { id: 'memory.stored',       label: 'Memory Stored' },
  { id: 'memory.forgotten',    label: 'Memory Forgotten (GDPR)' },
  { id: 'session.completed',   label: 'Session Completed' },
  { id: 'rate_limit.exceeded', label: 'Rate Limit Exceeded' },
  { id: 'health.degraded',     label: 'Health Degraded' },
  { id: 'pipeline.completed',  label: 'Pipeline Completed' },
];
```

### 3.3 Delivery History Drawer

```
DeliveryHistoryDrawer
├── DeliveryFilters        ← status: success | failed | pending
└── DeliveryList (last 50)
    └── DeliveryEntry
        ├── DeliveryId     ← short hash
        ├── EventType      ← "memory.stored" badge
        ├── Status         ← ✅ 200 OK / ❌ 503 failed / ⏳ pending
        ├── Duration       ← "234ms"
        ├── Timestamp
        └── RetryCount     ← "Attempt 2/3" if retried
```

### 3.4 Webhook Signature Info

```typescript
// Display HMAC verification instructions to user
const signatureInfo = `
Verify webhook signature in your server:
  const signature = req.headers['x-vnp-signature'];
  const expected  = HMAC_SHA256(secret, req.body);
  if (signature !== expected) reject();
`;
// Shown as code block in webhook detail modal
```

---

## 4. React Query Hooks

```typescript
export function useWebhooks() {
  return useQuery({
    queryKey: ['sdk', 'webhooks'],
    queryFn:  () => orgApi.getWebhooks(),
  });
}

export function useTestWebhook(webhookId: string) {
  return useMutation({
    mutationFn: () => orgApi.testWebhook(webhookId),
    onSuccess: () => toast.success('Test event sent!'),
    onError: ()   => toast.error('Test delivery failed'),
  });
}
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] Webhook list with status badge and success_rate bar
- [ ] Create form: URL validation (must be https://)
- [ ] Event multiselect: all 6 event types selectable
- [ ] Test button: sends test event → success/failure toast
- [ ] Delivery log shows last 50 deliveries with retry info
- [ ] Failed webhook shows `DISABLED` badge after 3 consecutive failures
- [ ] Signature verification instructions shown in webhook detail
