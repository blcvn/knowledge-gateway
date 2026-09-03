package analytics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/wire"
)

const (
	defaultTraceMaxHops = 5
	maxTraceMaxHops     = 10
)

// ProviderSet wires analytics engine dependencies.
var ProviderSet = wire.NewSet(NewCache, NewEngine)

type QueryExecutor interface {
	ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error)
}

type AnalyticsEngine interface {
	CoverageReport(ctx context.Context, namespace, domain string) (*CoverageReport, error)
	TraceabilityMatrix(ctx context.Context, namespace string, sourceTypes, targetTypes []string, maxHops int) (*TraceabilityMatrix, error)
	ClusterAnalysis(ctx context.Context, namespace, entityType string) (*ClusterReport, error)
}

type Engine struct {
	query QueryExecutor
	cache *Cache
}

func NewEngine(query QueryExecutor, cache *Cache) *Engine {
	return &Engine{query: query, cache: cache}
}

type CoverageByType struct {
	EntityType      string  `json:"entity_type"`
	TotalEntities   int     `json:"total_entities"`
	CoveredEntities int     `json:"covered_entities"`
	CoveragePercent float64 `json:"coverage_percent"`
}

type CoverageReport struct {
	Domain          string           `json:"domain"`
	TotalEntities   int              `json:"total_entities"`
	CoveredEntities int              `json:"covered_entities"`
	CoveragePercent float64          `json:"coverage_percent"`
	UncoveredTypes  []string         `json:"uncovered_types"`
	GeneratedAt     time.Time        `json:"generated_at"`
	ByType          []CoverageByType `json:"by_type"`
}

type TraceabilityTarget struct {
	EntityID string   `json:"entity_id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Hops     int      `json:"hops"`
	Path     []string `json:"path"`
}

type TraceabilityRow struct {
	SourceID   string               `json:"source_id"`
	SourceName string               `json:"source_name"`
	SourceType string               `json:"source_type"`
	Targets    []TraceabilityTarget `json:"targets"`
}

type TraceabilityMatrix struct {
	Matrix            []TraceabilityRow `json:"matrix"`
	TotalSources      int               `json:"total_sources"`
	TotalTargets      int               `json:"total_targets"`
	ComputeDurationMs int64             `json:"compute_duration_ms"`
}

type Cluster struct {
	CommunityID int64    `json:"community_id"`
	NodeIDs     []string `json:"node_ids"`
	Size        int      `json:"size"`
}

type ClusterReport struct {
	EntityType string    `json:"entity_type"`
	Clusters   []Cluster `json:"clusters"`
}

func splitNamespace(namespace string) (appID, tenantID string, err error) {
	parts := strings.Split(strings.TrimSpace(namespace), "/")
	if len(parts) < 3 || parts[0] != "graph" {
		return "", "", fmt.Errorf("invalid namespace %q", namespace)
	}
	if parts[1] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("invalid namespace %q", namespace)
	}
	return parts[1], parts[2], nil
}

func toRows(raw map[string]any) []map[string]any {
	if raw == nil {
		return nil
	}
	if rows, ok := raw["data"].([]map[string]any); ok {
		return rows
	}
	items, ok := raw["data"].([]any)
	if !ok {
		return nil
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func asInt(value any) int {
	switch x := value.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	default:
		return 0
	}
}

func asInt64(value any) int64 {
	switch x := value.(type) {
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	default:
		return 0
	}
}

func asStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	if items, ok := value.([]string); ok {
		return append([]string(nil), items...)
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		out = append(out, asString(item))
	}
	return out
}

func normalizeMaxHops(maxHops int) int {
	if maxHops <= 0 {
		return defaultTraceMaxHops
	}
	if maxHops > maxTraceMaxHops {
		return maxTraceMaxHops
	}
	return maxHops
}
