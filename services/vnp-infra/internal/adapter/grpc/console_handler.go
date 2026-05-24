// Package grpc provides stub handlers for vnp-infra console endpoints.
// Returns mock data matching UI's infrastructure.ts types.
package grpc

import (
	"context"
	"encoding/json"
	"time"
)

// InfraHandler provides stub console endpoints for infrastructure monitoring.
type InfraHandler struct{}

func NewInfraHandler() *InfraHandler {
	return &InfraHandler{}
}

type ServiceInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Uptime  int64  `json:"uptime"`
}

type DatabaseHealth struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	LatencyMs float64 `json:"latency_ms"`
}

type ResourceMetrics struct {
	Service       string  `json:"service"`
	CPUUsagePct   float64 `json:"cpu_usage_pct"`
	MemoryUsageMB float64 `json:"memory_usage_mb"`
	DiskUsagePct  float64 `json:"disk_usage_pct"`
}

type ServiceNode struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ServiceEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Protocol string `json:"protocol"`
}

// GetTopology returns service topology.
func (h *InfraHandler) GetTopology(_ context.Context) ([]byte, error) {
	data := map[string]interface{}{
		"total_services":   35,
		"healthy_services": 33,
		"nodes": []ServiceNode{
			{Name: "vnp-gateway", Status: "healthy"},
			{Name: "vnp-dashboard", Status: "healthy"},
			{Name: "vnp-search-hub", Status: "healthy"},
			{Name: "cognee-search", Status: "healthy"},
			{Name: "graphiti-store", Status: "healthy"},
			{Name: "memobase-context", Status: "healthy"},
			{Name: "zep-core", Status: "healthy"},
			{Name: "ov-session", Status: "unhealthy"},
		},
		"connections": []ServiceEdge{
			{From: "vnp-gateway", To: "vnp-dashboard", Protocol: "grpc"},
			{From: "vnp-gateway", To: "vnp-search-hub", Protocol: "grpc"},
			{From: "vnp-gateway", To: "vnp-admin", Protocol: "grpc"},
			{From: "vnp-search-hub", To: "cognee-search", Protocol: "grpc"},
			{From: "vnp-search-hub", To: "graphiti-store", Protocol: "grpc"},
			{From: "vnp-search-hub", To: "memobase-context", Protocol: "grpc"},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	return json.Marshal(data)
}

// ListServices returns service health status.
func (h *InfraHandler) ListServices(_ context.Context) ([]byte, error) {
	data := []ServiceInfo{
		{Name: "vnp-gateway", Version: "1.0.0", Status: "Healthy", Uptime: 86400},
		{Name: "vnp-dashboard", Version: "1.0.0", Status: "Healthy", Uptime: 86400},
		{Name: "vnp-search-hub", Version: "1.0.0", Status: "Healthy", Uptime: 172800},
		{Name: "cognee-search", Version: "0.9.1", Status: "Healthy", Uptime: 43200},
		{Name: "graphiti-store", Version: "1.2.0", Status: "Healthy", Uptime: 259200},
		{Name: "zep-core", Version: "0.8.0", Status: "Warning", Uptime: 3600},
	}
	return json.Marshal(data)
}

// GetDatabases returns database health.
func (h *InfraHandler) GetDatabases(_ context.Context) ([]byte, error) {
	data := []DatabaseHealth{
		{Name: "Primary PostgreSQL", Type: "PostgreSQL", Status: "Healthy", LatencyMs: 2.3},
		{Name: "Cache Redis", Type: "Redis", Status: "Healthy", LatencyMs: 0.5},
		{Name: "Graph Neo4j", Type: "Neo4j", Status: "Healthy", LatencyMs: 8.1},
		{Name: "Vector Qdrant", Type: "Qdrant", Status: "Healthy", LatencyMs: 3.2},
		{Name: "Event Bus NATS", Type: "NATS", Status: "Warning", LatencyMs: 15.4},
	}
	return json.Marshal(data)
}

// GetResources returns resource usage metrics.
func (h *InfraHandler) GetResources(_ context.Context) ([]byte, error) {
	data := []ResourceMetrics{
		{Service: "vnp-gateway", CPUUsagePct: 15.2, MemoryUsageMB: 256, DiskUsagePct: 12.5},
		{Service: "cognee-search", CPUUsagePct: 45.8, MemoryUsageMB: 1024, DiskUsagePct: 35.2},
		{Service: "graphiti-store", CPUUsagePct: 32.1, MemoryUsageMB: 512, DiskUsagePct: 48.7},
		{Service: "memobase-context", CPUUsagePct: 22.4, MemoryUsageMB: 384, DiskUsagePct: 18.9},
	}
	return json.Marshal(data)
}
