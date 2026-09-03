# Change Request: CR-PLAT-002 — Rate Limiting & Subscription Tiers

**CR ID:** CR-PLAT-002
**Component:** `backend/gateway`, `backend/shared/pkg/resilience`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F14](../../../features/14-authentication-multi-tenancy/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P4-01 | Enterprise Architect | Không có fair use enforcement giữa tenants |
| PP-P6-02 | Framework Integrator | Rate limits unclear cho SDK integration |

---

## 2. Subscription Tiers

| Tier | Store/min | Recall/min | Search/min | MCP calls/min |
|---|---|---|---|---|
| `free` | 20 | 50 | 30 | 100 |
| `pro` | 200 | 500 | 300 | 1000 |
| `enterprise` | Unlimited | Unlimited | Unlimited | Unlimited |

---

## 3. Rate Limit Implementation

```
Algorithm: Sliding Window Counter (Redis)
Key: rate_limit:{tenant_id}:{endpoint}:{window}

Window sizes:
  per-minute:  1m rolling window
  per-hour:    1h rolling window
  per-day:     24h rolling window

Response headers:
  X-RateLimit-Limit: 200
  X-RateLimit-Remaining: 187
  X-RateLimit-Reset: 1234567890 (Unix timestamp)

Exceeded → HTTP 429 Too Many Requests
Body: {"error": "rate_limit_exceeded", "retry_after": 30}
```

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/console/sdk/rate-limits` | Current usage vs limits |
| `GET` | `/v1/admin/rate-limits/{tenant_id}` | Admin: view any tenant's limits |
| `POST` | `/v1/admin/rate-limits/{tenant_id}/override` | Admin: temporary limit override |

---

## 5. Acceptance Criteria

- [ ] Sliding window enforced per tenant per endpoint
- [ ] 429 response with Retry-After header
- [ ] X-RateLimit-* headers on all memory API responses
- [ ] Enterprise tier: unlimited (no rate limit applied)
- [ ] Admin override API works
- [ ] NATS event published when rate limit exceeded: `rate_limit.exceeded`
