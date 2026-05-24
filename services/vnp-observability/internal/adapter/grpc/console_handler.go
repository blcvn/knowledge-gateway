// Package grpc provides stub handlers for vnp-observability console endpoints.
// Returns mock data matching UI's observability.ts types.
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ObservabilityHandler provides stub console endpoints for observability.
type ObservabilityHandler struct{}

func NewObservabilityHandler() *ObservabilityHandler {
	return &ObservabilityHandler{}
}

type MetricPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type MetricsSummary struct {
	AvgLatency  float64 `json:"avgLatency"`
	RequestRate float64 `json:"requestRate"`
	ErrorRate   float64 `json:"errorRate"`
	Uptime      string  `json:"uptime"`
	P50         float64 `json:"p50"`
	P95         float64 `json:"p95"`
	P99         float64 `json:"p99"`
}

type TraceSpan struct {
	ID         string `json:"id,omitempty"`
	TraceID    string `json:"trace_id"`
	SpanID     string `json:"span_id"`
	Name       string `json:"name,omitempty"`
	Operation  string `json:"operation,omitempty"`
	Service    string `json:"service"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Duration   int64  `json:"duration,omitempty"`
	Status     string `json:"status,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

type ErrorEntry struct {
	ID             string `json:"id"`
	Message        string `json:"message"`
	Service        string `json:"service"`
	Timestamp      string `json:"timestamp,omitempty"`
	Count          int    `json:"count,omitempty"`
	LastOccurrence string `json:"lastOccurrence,omitempty"`
	Stack          string `json:"stack,omitempty"`
}

// GetMetrics returns observability metrics.
func (h *ObservabilityHandler) GetMetrics(_ context.Context) ([]byte, error) {
	now := time.Now().UTC()
	points := make([]MetricPoint, 0, 24)
	for i := 23; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * time.Hour)
		points = append(points, MetricPoint{
			Timestamp: t.Format(time.RFC3339),
			Value:     35.0 + float64(i%5)*8.0,
		})
	}
	data := map[string]interface{}{
		"points": points,
		"summary": MetricsSummary{
			AvgLatency:  42.5,
			RequestRate: 1250.0,
			ErrorRate:   0.12,
			Uptime:      "99.97%",
			P50:         35.2,
			P95:         120.5,
			P99:         250.3,
		},
	}
	return json.Marshal(data)
}

// ListTraces returns distributed traces.
func (h *ObservabilityHandler) ListTraces(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"traces": []TraceSpan{
			{ID: "tr-001", TraceID: "abc-123", SpanID: "span-001", Name: "SearchMemory", Service: "vnp-search-hub", DurationMs: 45, Status: "ok", Timestamp: now},
			{ID: "tr-002", TraceID: "abc-124", SpanID: "span-002", Name: "StoreMemory", Service: "cognee-ingestion", DurationMs: 120, Status: "ok", Timestamp: now},
			{ID: "tr-003", TraceID: "abc-125", SpanID: "span-003", Name: "GetSubgraph", Service: "graphiti-store", DurationMs: 85, Status: "error", Timestamp: now},
		},
		"total": 3,
	}
	return json.Marshal(data)
}

// GetTrace returns a specific trace.
func (h *ObservabilityHandler) GetTrace(_ context.Context, id string) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := TraceSpan{
		ID: id, TraceID: "abc-123", SpanID: "span-001",
		Name: "SearchMemory", Operation: "cross-engine-search",
		Service: "vnp-search-hub", DurationMs: 45, Status: "ok", Timestamp: now,
	}
	return json.Marshal(data)
}

// GetErrors returns error entries.
func (h *ObservabilityHandler) GetErrors(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"errors": []ErrorEntry{
			{ID: "err-001", Message: "Connection timeout to neo4j", Service: "graphiti-store", Timestamp: now, Count: 12, LastOccurrence: now},
			{ID: "err-002", Message: "Vector dimension mismatch", Service: "cognee-search", Timestamp: now, Count: 3, LastOccurrence: now},
		},
		"total": 2,
	}
	return json.Marshal(data)
}

// GetCosts returns cost analytics.
func (h *ObservabilityHandler) GetCosts(_ context.Context) ([]byte, error) {
	data := map[string]interface{}{
		"costs": []map[string]interface{}{
			{"service": "cognee-ingestion", "model": "text-embedding-3-small", "tokens": 1250000, "cost": 0.025},
			{"service": "memobase-context", "model": "gpt-4o-mini", "tokens": 850000, "cost": 0.127},
			{"service": "graphiti-knowledge", "model": "text-embedding-3-large", "tokens": 420000, "cost": 0.054},
		},
		"total_cost": 0.206,
		"period":     fmt.Sprintf("%s - %s", time.Now().AddDate(0, 0, -30).Format("2006-01-02"), time.Now().Format("2006-01-02")),
	}
	return json.Marshal(data)
}
