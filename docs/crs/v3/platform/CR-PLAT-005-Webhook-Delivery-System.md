# Change Request: CR-PLAT-005 — Webhook Delivery System

**CR ID:** CR-PLAT-005
**Component:** `backend/gateway`, `backend/services/vnp-platform`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F27](../../../features/27-organization-api-sdk-manager/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P6-01 | Framework Integrator | Không có webhook → phải poll API |
| PP-P2-04 | Platform Engineer | No event notification for external systems |

**Before:** Integrators must poll APIs to detect changes.
**After:** Push events via webhooks on: memory stored, session ended, rate limit hit.

---

## 2. Supported Events

```
memory.stored        → memory_id, engine, type, tenant_id
memory.forgotten     → user_id, engines_deleted, tenant_id
session.completed    → session_id, observation_count, tenant_id
rate_limit.exceeded  → tenant_id, endpoint, limit, window
health.degraded      → service_name, previous_status, current_status
pipeline.completed   → session_id, tiers_completed, tenant_id
```

---

## 3. Webhook Delivery

```
Trigger: NATS event published
  ↓
Webhook service subscribes to NATS
  ↓
Lookup webhooks for tenant (filter by event type)
  ↓
HTTP POST to webhook URL
  Body: {"event": "memory.stored", "data": {...}, "timestamp": "..."}
  Headers:
    X-VNP-Signature: HMAC-SHA256(secret, body)
    X-VNP-Event:     memory.stored
    X-VNP-Delivery:  {delivery_id}
  ↓
On failure: Retry with exponential backoff
  Attempts: 1 → 5s → 25s → 125s (max 3 retries)
  After 3 failures: mark webhook as degraded, alert tenant
```

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/console/sdk/webhooks` | List configured webhooks |
| `POST` | `/v1/console/sdk/webhooks` | Create webhook |
| `PUT` | `/v1/console/sdk/webhooks/{id}` | Update webhook |
| `DELETE` | `/v1/console/sdk/webhooks/{id}` | Delete webhook |
| `GET` | `/v1/console/sdk/webhooks/{id}/deliveries` | Delivery history |
| `POST` | `/v1/console/sdk/webhooks/{id}/test` | Send test event |

---

## 5. Acceptance Criteria

- [ ] All 6 event types delivered correctly
- [ ] HMAC-SHA256 signature on every delivery
- [ ] Exponential backoff: 3 retries max
- [ ] Delivery history: last 50 deliveries per webhook
- [ ] Test endpoint sends sample event immediately
- [ ] Webhook disabled after 3 consecutive complete failures
- [ ] X-VNP-Signature verifiable by receiver
