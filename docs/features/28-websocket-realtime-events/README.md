# Feature 28 — WebSocket Real-time Events

> **Loại:** Platform | **Priority:** Medium | **Status:** Implemented

## Mô tả

WebSocket endpoint cung cấp real-time event streaming cho Console UI. Thay vì polling REST APIs, Console dùng WebSocket để nhận events ngay khi chúng xảy ra — health changes, new memories, pipeline completions, rate limit alerts.

---

## Business Logic

### Event Categories

Events được stream qua WebSocket:
- **health_change**: Service health status thay đổi (healthy → degraded)
- **memory_stored**: Memory mới được lưu (engine, type, tenant)
- **memory_forgotten**: Memory bị xóa
- **pipeline_complete**: Cognify/flush job hoàn thành
- **session_end**: Agent session kết thúc
- **rate_limit_exceeded**: Tenant vượt rate limit
- **circuit_breaker_open**: Circuit breaker mở (downstream failure)
- **observe_event**: Agent observation captured (từ Feature 08)

### Event Format

```json
{
  "event": "memory_stored",
  "tenant_id": "tenant-xyz",
  "data": {
    "engine": "cognee",
    "type": "semantic",
    "memory_id": "mem-abc123"
  },
  "timestamp": "2026-06-18T10:30:00Z"
}
```

### Reconnection

- Client tự động reconnect nếu WebSocket ngắt kết nối.
- Exponential backoff: 1s, 2s, 4s, max 30s.
- After reconnect: có thể request missed events (optional, event buffer).

### Tenant Scoping

WebSocket stream chỉ gửi events thuộc tenant của user đang connected — zero cross-tenant event leakage.

---

## Dataflow

```
Console UI
        │
        ├── Connect: GET /v1/console/ws  (WebSocket upgrade)
        │         ├── Auth: JWT token in query param hoặc header
        │         └── On success: connection established
        │
        ▼
WSHandler (gateway)
        │
        ├── Validate auth → extract TenantID
        ├── Subscribe to NATS subjects (tenant-scoped):
        │         ├── memory.*.{tenant_id}
        │         ├── gateway.circuit.*
        │         ├── gateway.ratelimit.*
        │         ├── pipeline.*.{tenant_id}
        │         └── observe.event.{tenant_id}
        │
        └── For each NATS message:
                  └── Format as WebSocket JSON frame
                      → Push to all connected clients for that tenant


NATS JetStream (event bus)
        │
        ├── Receives events from: all services, middleware, engines
        └── WSHandler subscribes and forwards to WebSocket clients
```

### Dashboard Live Updates

```
Dashboard component (UI)
        │
        ├── WebSocket connected
        │
        ├── On "health_change" event:
        │         └── Update service health indicator (no full reload)
        │
        ├── On "memory_stored" event:
        │         └── Increment memory counter on heatmap
        │
        └── On "pipeline_complete" event:
                  └── Update pipeline status badge
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/ws` | WebSocket upgrade (real-time events) |

---

## Related

- Feature 08 (Agent Observe) — generates `observe_event` events
- Feature 15 (Dashboard) — consumes events for live updates
- Feature 14 (Auth) — WS connection requires valid JWT

---

## Business Value

### Pain Points được giải quyết

- **PP-P2-05 (Pipeline failures im lặng)**

### Actors hưởng lợi

P2 Platform Engineer, P1 AI Agent Developer

### Giải pháp tham chiếu

- [S10 — Zero-config Infrastructure](../../bussiness/solutions/S10-infrastructure-simplicity.md)

### ROI / Kết quả đo được

> Real-time push alerts khi pipeline fails | No polling required

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
