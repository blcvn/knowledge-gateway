# TASK-GR-024 — Grafana Dashboards + Alert Rules

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-024 |
| **Wave** | 4 (Admin & Observability) |
| **Component** | `deploy/dev/configs/grafana/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-007 §6 |
| **Priority** | Low |
| **Depends On** | TASK-GR-021 |
| **Estimated** | 2h |

**Trạng thái:** ⏳ Pending  
**Ghi chú:** Grafana dashboards not implemented  
---

## Context

Tạo Grafana dashboard JSON cho Graphiti observability. 2 dashboards: Ingestion Pipeline (episode rate, step latency, entity/edge rate) và Search Performance (query latency, cache hit rate, result distribution).

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `deploy/dev/configs/grafana/dashboards/graphiti-ingestion.json` |
| CREATE | `deploy/dev/configs/grafana/dashboards/graphiti-search.json` |
| MODIFY | `deploy/dev/configs/prometheus.yml` |

---

## Implementation

### Dashboard 1: `graphiti-ingestion.json` — Key Panels

```json
{
  "title": "Graphiti Ingestion Pipeline",
  "uid": "graphiti-ingestion-v1",
  "panels": [
    {
      "title": "Episode Ingestion Rate (per minute)",
      "type": "timeseries",
      "targets": [{
        "expr": "rate(graphiti_ingestion_episodes_total{status='success'}[1m]) * 60",
        "legendFormat": "{{group_id}} / {{source}}"
      }]
    },
    {
      "title": "Pipeline Step P95 Latency",
      "type": "timeseries",
      "targets": [{
        "expr": "histogram_quantile(0.95, rate(graphiti_ingestion_step_duration_seconds_bucket[5m]))",
        "legendFormat": "{{step}}"
      }]
    },
    {
      "title": "Entity Extraction Rate (new vs merged)",
      "type": "timeseries",
      "targets": [
        {"expr": "rate(graphiti_ingestion_entities_extracted_total{resolution='new'}[1m]) * 60", "legendFormat": "New Entities"},
        {"expr": "rate(graphiti_ingestion_entities_extracted_total{resolution='merged'}[1m]) * 60", "legendFormat": "Merged Entities"}
      ]
    },
    {
      "title": "Edge Resolution Distribution",
      "type": "piechart",
      "targets": [{
        "expr": "increase(graphiti_ingestion_edges_extracted_total[1h])",
        "legendFormat": "{{resolution}}"
      }]
    },
    {
      "title": "Worker Queue Depth",
      "type": "gauge",
      "targets": [{
        "expr": "sum(graphiti_ingestion_worker_queue_depth) by (group_id)",
        "legendFormat": "{{group_id}}"
      }]
    },
    {
      "title": "LLM Token Usage (per prompt type)",
      "type": "timeseries",
      "targets": [{
        "expr": "rate(graphiti_llm_tokens_total{type='completion'}[5m]) * 60",
        "legendFormat": "{{prompt_name}}"
      }]
    },
    {
      "title": "LLM Call Latency P99",
      "type": "timeseries",
      "targets": [{
        "expr": "histogram_quantile(0.99, rate(graphiti_llm_call_duration_seconds_bucket[5m]))",
        "legendFormat": "{{prompt_name}} ({{provider}})"
      }]
    }
  ]
}
```

### Dashboard 2: `graphiti-search.json` — Key Panels

```json
{
  "title": "Graphiti Search Performance",
  "uid": "graphiti-search-v1",
  "panels": [
    {
      "title": "Search Request Rate",
      "type": "timeseries",
      "targets": [{
        "expr": "rate(graphiti_search_requests_total[1m]) * 60",
        "legendFormat": "{{recipe}} / {{status}}"
      }]
    },
    {
      "title": "Search Latency P50/P95/P99",
      "type": "timeseries",
      "targets": [
        {"expr": "histogram_quantile(0.50, rate(graphiti_search_duration_seconds_bucket[5m]))", "legendFormat": "P50"},
        {"expr": "histogram_quantile(0.95, rate(graphiti_search_duration_seconds_bucket[5m]))", "legendFormat": "P95"},
        {"expr": "histogram_quantile(0.99, rate(graphiti_search_duration_seconds_bucket[5m]))", "legendFormat": "P99"}
      ]
    },
    {
      "title": "Redis Cache Hit Rate (%)",
      "type": "gauge",
      "targets": [{
        "expr": "rate(graphiti_search_cache_hits_total[5m]) / rate(graphiti_search_requests_total[5m]) * 100"
      }]
    },
    {
      "title": "Results Per Search (avg)",
      "type": "stat",
      "targets": [{
        "expr": "graphiti_search_results_returned_total / graphiti_search_requests_total"
      }]
    }
  ]
}
```

### MODIFY: `deploy/dev/configs/prometheus.yml`

Add scrape config + alert rules:

```yaml
scrape_configs:
  # Add to existing scrape configs:
  - job_name: 'graphiti-store'
    static_configs:
      - targets: ['graphiti-store:9091']
    
  - job_name: 'graphiti-knowledge'
    static_configs:
      - targets: ['graphiti-knowledge:9093']

  - job_name: 'graphiti-ingestion'
    static_configs:
      - targets: ['graphiti-ingestion:9095']

rule_files:
  - 'alert_rules.yml'
  - 'graphiti_alerts.yml'  # ADD NEW FILE
```

**Create `deploy/dev/configs/graphiti_alerts.yml`:**

```yaml
groups:
  - name: graphiti_pipeline
    rules:
      - alert: GraphitiHighIngestionLatency
        expr: histogram_quantile(0.95, graphiti_ingestion_step_duration_seconds_bucket) > 20
        for: 5m
        labels: { severity: warning }
        annotations:
          summary: "Graphiti pipeline step {{ $labels.step }} P95 > 20s"

      - alert: GraphitiWorkerQueueBacklog
        expr: sum(graphiti_ingestion_worker_queue_depth) > 100
        for: 2m
        labels: { severity: warning }
        annotations:
          summary: "Graphiti worker queue backlog > 100 episodes"

      - alert: GraphitiLLMHighTokenRate
        expr: rate(graphiti_llm_tokens_total{type="completion"}[5m]) > 500
        labels: { severity: warning }
        annotations:
          summary: "High LLM token consumption rate"

  - name: graphiti_search
    rules:
      - alert: GraphitiSearchHighLatency
        expr: histogram_quantile(0.99, graphiti_search_duration_seconds_bucket) > 5
        for: 3m
        labels: { severity: warning }
        annotations:
          summary: "Graphiti search P99 > 5 seconds"

      - alert: GraphitiSearchLowCacheHitRate
        expr: rate(graphiti_search_cache_hits_total[5m]) / rate(graphiti_search_requests_total[5m]) < 0.3
        for: 10m
        labels: { severity: info }
        annotations:
          summary: "Graphiti search cache hit rate < 30%"
```

---

## Verification

```bash
# Import dashboards via Grafana API
curl -X POST http://admin:admin@localhost:3000/api/dashboards/import \
    -H "Content-Type: application/json" \
    -d @deploy/dev/configs/grafana/dashboards/graphiti-ingestion.json

# Check Prometheus alerts loaded
curl http://localhost:9090/api/v1/rules | grep graphiti
```

**Expected:** Both dashboards visible in Grafana. All graphiti alert rules loaded.
