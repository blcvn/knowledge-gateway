// Package grpc provides stub console handlers for vnp-dashboard service.
// These handlers return realistic mock data matching the UI's TypeScript types.
// TODO: Replace with real aggregation logic from downstream engine services.
package grpc

import (
	"context"
	"encoding/json"
	"time"
)

// DashboardHandler provides dashboard console endpoints (FEAT-006).
// Stub implementation — returns mock data matching UI expectations.
type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// ──── Response types (match UI's dashboard.ts) ─────────────────

type EngineHealth struct {
	Name          string  `json:"name"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	LatencyP50Ms  float64 `json:"latencyP50Ms"`
	LatencyP95Ms  float64 `json:"latencyP95Ms"`
	QueueDepth    int     `json:"queueDepth"`
	UptimeSeconds int64   `json:"uptimeSeconds"`
	LastCheck     string  `json:"lastCheck"`
}

type KPIData struct {
	ActiveAgents       int     `json:"activeAgents"`
	RecallLatencyP50Ms float64 `json:"recallLatencyP50Ms"`
	RecallLatencyP95Ms float64 `json:"recallLatencyP95Ms"`
	ContextSavingsPct  float64 `json:"contextSavingsPct"`
	GraphNodesTotal    int64   `json:"graphNodesTotal"`
	GraphEdgesTotal    int64   `json:"graphEdgesTotal"`
	GraphGrowth24h     int     `json:"graphGrowth24h"`
	ErrorRatePct       float64 `json:"errorRatePct"`
	ActiveSessions     int     `json:"activeSessions"`
	ActiveProfiles     int     `json:"activeProfiles"`
	MemoryVersions     int     `json:"memoryVersions"`
}

type MemoryFlowMetrics struct {
	IngestPerSec               float64 `json:"ingestPerSec"`
	RecallPerSec               float64 `json:"recallPerSec"`
	EmbedPerSec                float64 `json:"embedPerSec"`
	ProfileExtractionsPerSec   float64 `json:"profileExtractionsPerSec,omitempty"`
	QueueBacklog               int     `json:"queueBacklog,omitempty"`
}

type ThroughputData struct {
	Window  string                       `json:"window"`
	Engines map[string]MemoryFlowMetrics `json:"engines"`
}

type HeatmapPoint struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Density float64 `json:"density"`
}

// ──── Handler Methods ──────────────────────────────────────────

// GetHealth returns aggregated engine health (stub).
func (h *DashboardHandler) GetHealth(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := []EngineHealth{
		{Name: "cognee", Role: "Semantic Knowledge Processing", Status: "Healthy", LatencyP50Ms: 45, LatencyP95Ms: 120, QueueDepth: 3, UptimeSeconds: 86400, LastCheck: now},
		{Name: "graphiti", Role: "Temporal Knowledge Graph", Status: "Healthy", LatencyP50Ms: 30, LatencyP95Ms: 85, QueueDepth: 1, UptimeSeconds: 172800, LastCheck: now},
		{Name: "zep", Role: "Conversational Memory", Status: "Healthy", LatencyP50Ms: 25, LatencyP95Ms: 60, QueueDepth: 5, UptimeSeconds: 259200, LastCheck: now},
		{Name: "openviking", Role: "Filesystem & Sessions", Status: "Warning", LatencyP50Ms: 55, LatencyP95Ms: 200, QueueDepth: 12, UptimeSeconds: 43200, LastCheck: now},
		{Name: "memobase", Role: "User Profiles & Context", Status: "Healthy", LatencyP50Ms: 20, LatencyP95Ms: 50, QueueDepth: 0, UptimeSeconds: 345600, LastCheck: now},
		{Name: "supermemory", Role: "Adaptive Memory & Connectors", Status: "Healthy", LatencyP50Ms: 35, LatencyP95Ms: 90, QueueDepth: 2, UptimeSeconds: 86400, LastCheck: now},
	}
	return json.Marshal(data)
}

// GetMetrics returns KPI metrics (stub).
func (h *DashboardHandler) GetMetrics(_ context.Context) ([]byte, error) {
	data := KPIData{
		ActiveAgents:       42,
		RecallLatencyP50Ms: 35.2,
		RecallLatencyP95Ms: 120.5,
		ContextSavingsPct:  67.8,
		GraphNodesTotal:    125430,
		GraphEdgesTotal:    387650,
		GraphGrowth24h:     1247,
		ErrorRatePct:       0.12,
		ActiveSessions:     18,
		ActiveProfiles:     3254,
		MemoryVersions:     89201,
	}
	return json.Marshal(data)
}

// GetThroughput returns per-engine throughput (stub).
func (h *DashboardHandler) GetThroughput(_ context.Context) ([]byte, error) {
	data := ThroughputData{
		Window: "5m",
		Engines: map[string]MemoryFlowMetrics{
			"cognee":      {IngestPerSec: 12.5, RecallPerSec: 45.2, EmbedPerSec: 8.3, QueueBacklog: 3},
			"graphiti":    {IngestPerSec: 8.1, RecallPerSec: 32.7, EmbedPerSec: 5.6, QueueBacklog: 1},
			"zep":         {IngestPerSec: 25.3, RecallPerSec: 67.8, EmbedPerSec: 12.1, QueueBacklog: 5},
			"openviking":  {IngestPerSec: 3.2, RecallPerSec: 15.4, EmbedPerSec: 2.1, QueueBacklog: 12},
			"memobase":    {IngestPerSec: 18.7, RecallPerSec: 28.3, EmbedPerSec: 9.8, ProfileExtractionsPerSec: 4.2, QueueBacklog: 0},
			"supermemory": {IngestPerSec: 7.4, RecallPerSec: 22.1, EmbedPerSec: 4.5, QueueBacklog: 2},
		},
	}
	return json.Marshal(data)
}

// GetHeatmap returns memory density heatmap (stub).
func (h *DashboardHandler) GetHeatmap(_ context.Context) ([]byte, error) {
	points := make([]HeatmapPoint, 0, 50)
	for i := 0; i < 50; i++ {
		points = append(points, HeatmapPoint{
			X:       float64(i%10) * 0.1,
			Y:       float64(i/10) * 0.2,
			Density: float64(50+i*3) / 100.0,
		})
	}
	data := map[string]interface{}{
		"points": points,
	}
	return json.Marshal(data)
}
