# v3/platform — Solution Index

| Solution | CR | Priority | Status | Description |
|---|---|---|---|---|
| [SOL-PLAT-001](./SOL-PLAT-001-Auth-API-Key-JWT.md) | [CR-PLAT-001](../../../../docs/crs/v3/platform/CR-PLAT-001-Auth-API-Key-JWT.md) | 🔴 Critical | Open | API Key Lifecycle (create/rotate/revoke/expire) + JWT RS256 + JWK endpoint |
| [SOL-PLAT-002](./SOL-PLAT-002-Rate-Limiting-Subscription-Tiers.md) | [CR-PLAT-002](../../../../docs/crs/v3/platform/CR-PLAT-002-Rate-Limiting-Subscription-Tiers.md) | 🔴 Critical | Open | Redis sliding window rate limiter + tier limits + admin override |
| [SOL-PLAT-003](./SOL-PLAT-003-SSO-Google-OAuth.md) | [CR-PLAT-003](../../../../docs/crs/v3/platform/CR-PLAT-003-SSO-Google-OAuth.md) | 🟡 High | Open | Google OAuth2 PKCE flow + user provisioning + refresh token rotation |
| [SOL-PLAT-004](./SOL-PLAT-004-OpenTelemetry-Tracing.md) | [CR-PLAT-004](../../../../docs/crs/v3/platform/CR-PLAT-004-OpenTelemetry-Tracing.md) | 🟡 High | Open | OTel distributed tracing + W3C propagation + OTLP exporter + secret redaction |
| [SOL-PLAT-005](./SOL-PLAT-005-Webhook-Delivery-System.md) | [CR-PLAT-005](../../../../docs/crs/v3/platform/CR-PLAT-005-Webhook-Delivery-System.md) | 🟡 High | Open | NATS-triggered webhook delivery + HMAC signature + exponential backoff retry |
| [SOL-PLAT-006](./SOL-PLAT-006-WebSocket-Realtime-Events.md) | [CR-PLAT-006](../../../../docs/crs/v3/platform/CR-PLAT-006-WebSocket-Realtime-Events.md) | 🟡 High | Open | WebSocket real-time event streaming + tenant isolation + Redis event buffer |

## Implementation Order

```
1. SOL-PLAT-001  (Auth/JWT)     ← prerequisite for everything
2. SOL-PLAT-002  (Rate Limit)   ← depends on AuthContext.RateTier from PLAT-001
3. SOL-PLAT-003  (SSO)          ← depends on JWT service from PLAT-001
4. SOL-PLAT-004  (OTel)         ← independent, instrument as we go
5. SOL-PLAT-005  (Webhooks)     ← depends on NATS events from all services
6. SOL-PLAT-006  (WebSocket)    ← depends on NATS events + Redis
```

## Key Services

| Service | Role |
|---|---|
| `gateway` | Auth middleware, rate limit middleware, WebSocket handler, tracing |
| `services/vnp-platform` | API key CRUD, webhook management, SSO use cases |
| `shared/pkg/resilience` | Rate limiter (Lua + Redis sliding window) |
| `shared/pkg/telemetry` | OTel tracer, LLM spans, secret redaction |
