package observability

import (
	"math"
	"sort"
	"time"

	"kg-service/internal/telemetry"
	"kg-service/internal/workers"
	"kg-service/internal/write"
)

type Store interface {
	ListOutboxEvents() []write.OutboxEvent
	ListProjectionVersions() []write.ProjectionVersionRecord
}

type Service struct {
	store   Store
	runtime *workers.Runtime
	now     func() time.Time
}

func NewService(store Store, runtime *workers.Runtime) *Service {
	return &Service{
		store:   store,
		runtime: runtime,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

type MetricsResponse struct {
	OutboxBacklog               int                `json:"kg_outbox_backlog"`
	GraphLagSeconds             telemetry.LagStats `json:"kg_graph_lag_seconds"`
	VectorLagSeconds            telemetry.LagStats `json:"kg_vector_lag_seconds"`
	OrphanRelationshipsCount    int                `json:"kg_orphan_relationships_count"`
	OrphanVectorDocsCount       int                `json:"kg_orphan_vector_docs_count"`
	BulkWriteBatchSize          telemetry.LagStats `json:"kg_bulk_write_batch_size"`
	BulkWritePartialFailureRate float64            `json:"kg_bulk_write_partial_failure_rate"`
	RealtimeReadFallbackCount   int                `json:"kg_realtime_read_fallback_count"`
	GraphScopeConflictCount     int                `json:"kg_graph_scope_conflict_count"`
	Alerts                      []string           `json:"alerts,omitempty"`
	LastUpdatedAt               time.Time          `json:"last_updated_at"`
}

func (s *Service) Snapshot() MetricsResponse {
	snap := telemetry.Default().Snapshot()
	resp := MetricsResponse{
		OutboxBacklog:               backlogCount(s.store),
		GraphLagSeconds:             lagStats(s.store, s.now, "graph"),
		VectorLagSeconds:            lagStats(s.store, s.now, "vector"),
		OrphanRelationshipsCount:    snap.OrphanRelationshipsCount,
		OrphanVectorDocsCount:       snap.OrphanVectorDocsCount,
		BulkWriteBatchSize:          snap.BulkWriteBatchSize,
		BulkWritePartialFailureRate: snap.BulkWritePartialFailureRate,
		RealtimeReadFallbackCount:   snap.RealtimeReadFallbackCount,
		GraphScopeConflictCount:     snap.GraphScopeConflictCount,
		Alerts:                      append([]string(nil), snap.Alerts...),
		LastUpdatedAt:               snap.LastUpdatedAt,
	}
	if s.runtime != nil {
		report := s.runtime.Reconcile()
		telemetry.RecordOrphanCounts(
			countIssues(report, "orphan_graph_relationship"),
			countIssues(report, "orphan_vector_doc")+countIssues(report, "orphan_graph_node"),
		)
		if report.GraphDriftCount > 0 || report.VectorDriftCount > 0 {
			resp.Alerts = append(resp.Alerts, "projection drift detected")
		}
	}
	if resp.OutboxBacklog > 1000 {
		resp.Alerts = append(resp.Alerts, "outbox backlog high")
	}
	return resp
}

func countIssues(report workers.ReconciliationReport, kind string) int {
	count := 0
	for _, issue := range report.Issues {
		if issue.Kind == kind {
			count++
		}
	}
	return count
}

func backlogCount(store Store) int {
	if store == nil {
		return 0
	}
	count := 0
	for _, event := range store.ListOutboxEvents() {
		switch event.Status {
		case "PENDING", "FAILED":
			count++
		}
	}
	return count
}

func lagStats(store Store, now func() time.Time, kind string) telemetry.LagStats {
	if store == nil {
		return telemetry.LagStats{}
	}
	values := make([]float64, 0, 64)
	for _, record := range store.ListProjectionVersions() {
		if kind == "graph" {
			if record.LastGraphSyncedAt.IsZero() {
				continue
			}
			values = append(values, now().Sub(record.LastGraphSyncedAt).Seconds())
			continue
		}
		if record.LastVectorSyncedAt.IsZero() {
			continue
		}
		values = append(values, now().Sub(record.LastVectorSyncedAt).Seconds())
	}
	return summarizeFloat(values)
}

func summarizeFloat(values []float64) telemetry.LagStats {
	if len(values) == 0 {
		return telemetry.LagStats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return telemetry.LagStats{
		Count:  len(sorted),
		Median: medianFloat(sorted),
		P95:    percentileFloat(sorted, 0.95),
		Max:    sorted[len(sorted)-1],
	}
}

func medianFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func percentileFloat(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if p <= 0 {
		return values[0]
	}
	if p >= 1 {
		return values[len(values)-1]
	}
	idx := int(math.Ceil(float64(len(values))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}
