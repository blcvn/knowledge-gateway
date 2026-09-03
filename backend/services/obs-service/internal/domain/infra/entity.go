// Package infra defines domain entities for infrastructure topology.
//
// Absorbed from: vnp-infra
// (MERGE-P3-T2)
package infra

import "time"

// ServiceInfo describes a running microservice.
type ServiceInfo struct {
	Name        string        `json:"name"`
	Version     string        `json:"version,omitempty"`
	Status      string        `json:"status"` // "healthy"|"degraded"|"down"
	Uptime      time.Duration `json:"uptime_seconds"`
	Replicas    int           `json:"replicas"`
	Port        int           `json:"port,omitempty"`
	Address     string        `json:"address,omitempty"`
	LastCheckAt time.Time     `json:"last_check_at"`
}

// TopologyGraph is the service dependency graph.
type TopologyGraph struct {
	Services  []*ServiceInfo `json:"services"`
	Edges     []*ServiceEdge `json:"edges"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ServiceEdge is a directed dependency between two services.
type ServiceEdge struct {
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	Protocol string  `json:"protocol"` // "grpc"|"http"|"nats"
	LatencyMs float64 `json:"latency_ms,omitempty"`
}

// Database describes a backend data store.
type Database struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "postgres"|"redis"|"neo4j"|"nats"
	Status      string `json:"status"`
	SizeMB      int64  `json:"size_mb"`
	Connections int    `json:"connections"`
}

// Deployment describes a service deployment state.
type Deployment struct {
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	Replicas  int       `json:"replicas"`
	CreatedAt time.Time `json:"created_at"`
}

// Resource is a system-level resource (CPU/memory/disk/network).
type Resource struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"` // "cpu"|"memory"|"disk"|"network"
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
	Unit  string  `json:"unit"`
}

// knownServices lists all services in the VNP Memory architecture.
var KnownServices = []string{
	"gateway",
	"vnp-platform",
	"storage-service",
	"kg-service",
	"memory-service",
	"search-service",
	"pipeline-service",
	"obs-service",
}

// staticEdges defines the known service dependency graph.
var StaticEdges = []*ServiceEdge{
	{Source: "gateway", Target: "vnp-platform", Protocol: "grpc"},
	{Source: "gateway", Target: "storage-service", Protocol: "grpc"},
	{Source: "gateway", Target: "kg-service", Protocol: "grpc"},
	{Source: "gateway", Target: "memory-service", Protocol: "grpc"},
	{Source: "gateway", Target: "search-service", Protocol: "grpc"},
	{Source: "gateway", Target: "pipeline-service", Protocol: "grpc"},
	{Source: "gateway", Target: "obs-service", Protocol: "grpc"},
	{Source: "kg-service", Target: "memory-service", Protocol: "grpc"},
	{Source: "search-service", Target: "kg-service", Protocol: "http"},
	{Source: "search-service", Target: "memory-service", Protocol: "http"},
	{Source: "search-service", Target: "storage-service", Protocol: "http"},
	{Source: "pipeline-service", Target: "kg-service", Protocol: "grpc"},
}
