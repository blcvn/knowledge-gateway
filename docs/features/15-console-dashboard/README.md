# Feature 15 — Console Dashboard

> **Loại:** Console UI | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

Dashboard là trang tổng quan đầu tiên khi vào Console UI. Hiển thị health status của toàn bộ 35 services, throughput metrics, và memory usage heatmap theo thời gian thực.

---

## Business Logic

### Health Overview

Dashboard aggregate health status từ tất cả 35 services:
- Mỗi service có trạng thái: `healthy` / `degraded` / `unhealthy`
- Aggregated health: healthy nếu tất cả OK, degraded nếu 1+ service lỗi
- Auto-refresh mỗi 30 giây

### Metrics Display

- **Throughput**: Requests/second across all endpoints (Prometheus data)
- **Error Rate**: Tỷ lệ 4xx/5xx per service
- **Latency**: P50/P95/P99 per endpoint
- **Memory Usage**: Storage consumption per engine

### Memory Heatmap

Heatmap hiển thị hoạt động memory theo thời gian:
- X-axis: Time (last 24h)
- Y-axis: Memory types (semantic, episodic, conversational, profile, procedural, adaptive)
- Color: Intensity = request volume

---

## Dataflow

```
Console UI (Dashboard component)
        │
        ├── GET /v1/console/dashboard/health     → 35-service health status
        ├── GET /v1/console/dashboard/metrics     → Prometheus metrics (last 1h)
        ├── GET /v1/console/dashboard/throughput  → Throughput timeseries
        └── GET /v1/console/dashboard/heatmap     → Memory activity heatmap
        │
        ▼
DashboardHandler (gateway)
        │
        ├── health: aggregate GET /healthz data from all services
        ├── metrics: query Prometheus via obs-service
        ├── throughput: query NATS message counts + HTTP metrics
        └── heatmap: query per-engine ingestion counts (time-bucketed)


Real-time updates:
        └── WebSocket /v1/console/ws
                  └── Push events: health_change, metric_update
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/dashboard/health` | All-service health |
| `GET` | `/v1/console/dashboard/metrics` | Key metrics |
| `GET` | `/v1/console/dashboard/throughput` | Throughput timeseries |
| `GET` | `/v1/console/dashboard/heatmap` | Memory activity heatmap |
