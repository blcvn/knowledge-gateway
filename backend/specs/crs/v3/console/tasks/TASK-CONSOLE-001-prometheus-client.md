# TASK-CONSOLE-001 — Prometheus HTTP Client

| Field | Value |
|---|---|
| **Task ID** | TASK-CONSOLE-001 |
| **Wave** | 1 |
| **Solution** | [SOL-CONSOLE-001](../solutions/SOL-CONSOLE-001-Dashboard-APIs.md) §3 |
| **Component** | `gateway/internal/adapter/prometheus/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 2h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** gateway/internal/infra/server/observability.go: Prometheus client + /metrics endpoint
---

## Mục tiêu

Tạo `PrometheusClient` interface và implementation gọi Prometheus HTTP API.

---

## Công việc cụ thể

### 1. Tạo `gateway/internal/port/prometheus.go` [NEW]

```go
package port

type SeriesPoint struct {
    Timestamp int64   `json:"ts"`
    Value     float64 `json:"value"`
}

type PrometheusClient interface {
    QueryScalar(ctx context.Context, query string) (float64, error)
    QueryRange(ctx context.Context, query, duration, step string) ([]SeriesPoint, error)
}
```

### 2. Tạo `gateway/internal/adapter/prometheus/client.go` [NEW]

```go
package prometheus

import (
    "context", "encoding/json", "fmt", "net/http", "net/url", "strconv", "time"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

func New(baseURL string) *Client {
    return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) QueryScalar(ctx context.Context, query string) (float64, error) {
    u := fmt.Sprintf("%s/api/v1/query?query=%s", c.baseURL, url.QueryEscape(query))
    resp, err := c.httpClient.GetWithContext(ctx, u)
    if err != nil { return 0, err }
    defer resp.Body.Close()
    var result struct {
        Data struct {
            Result [][]any `json:"result"`
        } `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    if len(result.Data.Result) == 0 || len(result.Data.Result[0]) < 2 { return 0, nil }
    val, _ := strconv.ParseFloat(result.Data.Result[0][1].(string), 64)
    return val, nil
}

func (c *Client) QueryRange(ctx context.Context, query, duration, step string) ([]port.SeriesPoint, error) {
    end := time.Now()
    start := end.Add(-parseDuration(duration))
    u := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
        c.baseURL, url.QueryEscape(query), start.Unix(), end.Unix(), step)
    // ... parse matrix result
    return nil, nil
}
```

### 3. Tạo `gateway/internal/adapter/prometheus/client_test.go` [NEW]

```go
func TestQueryScalar_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]any{
            "data": map[string]any{
                "result": [][]any{{"1234567890", "42.5"}},
            },
        })
    }))
    client := prometheus.New(server.URL)
    val, err := client.QueryScalar(context.Background(), "test_metric")
    assert.NoError(t, err)
    assert.InDelta(t, 42.5, val, 0.001)
}
```

---

## Acceptance Criteria

- [ ] `QueryScalar` parses Prometheus instant query response
- [ ] `QueryRange` parses matrix response to SeriesPoint slice
- [ ] Timeout: 10s, no hanging
- [ ] Unit test with mock HTTP server passes

## Files

```
gateway/internal/port/prometheus.go                        [NEW]
gateway/internal/adapter/prometheus/client.go              [NEW]
gateway/internal/adapter/prometheus/client_test.go         [NEW]
```
