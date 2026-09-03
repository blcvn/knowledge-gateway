// Package observability defines domain entities for obs-service.
//
// Absorbed from: vnp-observability, sm-engine
// (MERGE-P3-T2)
package observability

import "time"

// MetricPoint is a single time-series data point.
type MetricPoint struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// MetricSummary is an aggregated metrics view.
type MetricSummary struct {
	TotalRequests     int64           `json:"total_requests"`
	ErrorRate         float64         `json:"error_rate"`
	P50LatencyMs      float64         `json:"p50_latency_ms"`
	P95LatencyMs      float64         `json:"p95_latency_ms"`
	P99LatencyMs      float64         `json:"p99_latency_ms"`
	RequestsPerSecond float64         `json:"requests_per_second"`
	Services          []ServiceMetric `json:"services"`
	Timestamp         time.Time       `json:"timestamp"`
}

// ServiceMetric holds per-service metrics.
type ServiceMetric struct {
	Name         string  `json:"name"`
	Healthy      bool    `json:"healthy"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// Trace is a distributed trace.
type Trace struct {
	TraceID   string        `json:"trace_id"`
	Service   string        `json:"service"`
	Operation string        `json:"operation"`
	Duration  time.Duration `json:"duration_ms"`
	Status    string        `json:"status"` // "ok"|"error"
	Spans     []*Span       `json:"spans,omitempty"`
	StartedAt time.Time     `json:"started_at"`
}

// Span is a single trace span.
type Span struct {
	SpanID    string            `json:"span_id"`
	TraceID   string            `json:"trace_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Operation string            `json:"operation"`
	Service   string            `json:"service"`
	Duration  time.Duration     `json:"duration_ms"`
	Tags      map[string]string `json:"tags,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// ErrorEntry aggregates error occurrences.
type ErrorEntry struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Stack     string    `json:"stack,omitempty"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// CostEntry tracks LLM/storage costs per service.
type CostEntry struct {
	Service     string    `json:"service"`
	Period      string    `json:"period"` // "2026-06-11:hour" | "2026-06-11:day"
	LLMTokens   int64     `json:"llm_tokens"`
	EmbedTokens int64     `json:"embed_tokens"`
	StorageMB   int64     `json:"storage_mb"`
	EstCostUSD  float64   `json:"est_cost_usd"`
	Timestamp   time.Time `json:"timestamp"`
}

// TraceFilter filters trace queries.
type TraceFilter struct {
	Service string `json:"service,omitempty"`
	Status  string `json:"status,omitempty"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

// ErrorFilter filters error queries.
type ErrorFilter struct {
	Service string `json:"service,omitempty"`
	Limit   int    `json:"limit"`
}
