# UI Solution: UI-SOL-PLAT-002 — Rate Limiting & Subscription Tiers UI

**Solution ID:** UI-SOL-PLAT-002  
**CR References:** [CR-PLAT-002](../../../../docs/crs/v3/platform/CR-PLAT-002-Rate-Limiting-Subscription-Tiers.md)  
**Feature:** Rate Limiting — Quota Display, Tier Badges, Rate Limit Warnings  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/components/common/RateLimitToast.tsx`, `ui/src/pages/api-sdk/`

---

## 1. Mục Đích

Xây dựng UI cho Rate Limiting:
- Hiển thị current rate limit headers từ API responses
- Rate limit toast với countdown timer
- Subscription tier badge và quota usage
- Upgrade prompt khi gần đạt giới hạn

---

## 2. Backend Rate Limit Headers

```http
# Included in every API response when rate-limited:
X-RateLimit-Limit:     600
X-RateLimit-Remaining: 0
X-RateLimit-Reset:     1683820800    (Unix timestamp)
Retry-After:           30            (seconds)
```

### Tier Limits

| Tier | Req/min | Burst | Memory Store/h |
|------|---------|-------|----------------|
| Free | 60 | 10 | 100 |
| Pro | 600 | 50 | 10,000 |
| Enterprise | 6,000 | 200 | Unlimited |

---

## 3. Components

### 3.1 Rate Limit Toast (429 Handler)

```typescript
// ui/src/components/common/RateLimitToast.tsx
// Shown when API returns HTTP 429

function RateLimitToast({ retryAfter }: { retryAfter: number }) {
  const [countdown, setCountdown] = useState(retryAfter);
  
  useEffect(() => {
    const timer = setInterval(() => {
      setCountdown(c => {
        if (c <= 1) { clearInterval(timer); return 0; }
        return c - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, []);
  
  return (
    <Toast variant="warning">
      <ToastTitle>Rate Limit Reached</ToastTitle>
      <ToastDescription>
        Retry in {countdown}s
        <ProgressBar value={retryAfter - countdown} max={retryAfter} />
      </ToastDescription>
    </Toast>
  );
}
```

### 3.2 Rate Limit Header Reader

```typescript
// ui/src/lib/api-client.ts — parse rate limit headers
function extractRateLimitInfo(response: Response): RateLimitInfo | null {
  const limit     = response.headers.get('X-RateLimit-Limit');
  const remaining = response.headers.get('X-RateLimit-Remaining');
  const reset     = response.headers.get('X-RateLimit-Reset');
  const retryAfter = response.headers.get('Retry-After');
  
  if (!limit) return null;
  return {
    limit:      parseInt(limit),
    remaining:  parseInt(remaining ?? '0'),
    resetAt:    new Date(parseInt(reset ?? '0') * 1000),
    retryAfter: retryAfter ? parseInt(retryAfter) : null,
  };
}
```

### 3.3 Quota Usage Display (SDK Settings)

```
QuotaUsagePanel
├── TierBadge               ← [Free] / [Pro] / [Enterprise]
├── CurrentUsageBar
│   ├── RequestsPerMin      ← "47 / 60 req/min [███████░░░] 78%"
│   ├── StoragePerHour      ← "82 / 100 stores/h [████████░░] 82%"
│   └── BurstCapacity       ← "7 / 10 burst [███████░░░] 70%"
├── WarningBanner (if > 80%) ← "You're at 82% of your Free tier limit"
└── UpgradeButton           ← "Upgrade to Pro →"
```

---

## 4. Acceptance Criteria (Frontend)

- [ ] HTTP 429 → RateLimitToast with countdown (not generic error)
- [ ] Countdown timer decrements every 1 second to zero
- [ ] Tier badge displayed in SDK settings page
- [ ] Quota bars auto-refresh every 30s
- [ ] Warning banner at 80% usage
- [ ] `Retry-After` header correctly parsed for countdown start value
