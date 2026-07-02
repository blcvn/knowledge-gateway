package telemetry

import (
	"math"
	"sort"
	"sync"
	"time"
)

type Registry struct {
	mu sync.RWMutex

	bulkWriteBatchSizes               []int
	bulkWritePartialFailureRate       float64
	orphanRelationshipsCount          int
	orphanVectorDocsCount             int
	lagGraphSeconds                   []float64
	lagVectorSeconds                  []float64
	lastOutboxBacklog                 int
	realtimeReadFallbackCount         int
	graphScopeConflictCount           int
	projectionOutboxClaimSizes        []int
	projectionCoalescedEntityCounts   []int
	projectionQueueAgeSeconds         []float64
	projectionGraphBatchLatencies     []float64
	projectionVectorBatchLatencies    []float64
	projectionEmbeddingBatchLatencies []float64
	projectionPartialFailures         int
	projectionStaleSkips              int
	lastUpdate                        time.Time
}

type Snapshot struct {
	OutboxBacklog               int       `json:"kg_outbox_backlog"`
	GraphLagSeconds             LagStats  `json:"kg_graph_lag_seconds"`
	VectorLagSeconds            LagStats  `json:"kg_vector_lag_seconds"`
	OrphanRelationshipsCount    int       `json:"kg_orphan_relationships_count"`
	OrphanVectorDocsCount       int       `json:"kg_orphan_vector_docs_count"`
	BulkWriteBatchSize          LagStats  `json:"kg_bulk_write_batch_size"`
	BulkWritePartialFailureRate float64   `json:"kg_bulk_write_partial_failure_rate"`
	RealtimeReadFallbackCount   int       `json:"kg_realtime_read_fallback_count"`
	GraphScopeConflictCount     int       `json:"kg_graph_scope_conflict_count"`
	ProjectionOutboxClaimSize   LagStats  `json:"kg_projection_outbox_claim_size"`
	ProjectionCoalescedEntities LagStats  `json:"kg_projection_coalesced_entities"`
	ProjectionQueueAgeSeconds   LagStats  `json:"kg_projection_queue_age_seconds"`
	ProjectionGraphLatency      LagStats  `json:"kg_projection_graph_latency_seconds"`
	ProjectionVectorLatency     LagStats  `json:"kg_projection_vector_latency_seconds"`
	ProjectionEmbeddingLatency  LagStats  `json:"kg_projection_embedding_latency_seconds"`
	ProjectionPartialFailures   int       `json:"kg_projection_partial_failure_total"`
	ProjectionStaleSkips        int       `json:"kg_projection_stale_event_skips_total"`
	Alerts                      []string  `json:"alerts,omitempty"`
	LastUpdatedAt               time.Time `json:"last_updated_at"`
}

type LagStats struct {
	Count  int     `json:"count"`
	Median float64 `json:"median"`
	P95    float64 `json:"p95"`
	Max    float64 `json:"max"`
}

var defaultRegistry = &Registry{}

func Default() *Registry {
	return defaultRegistry
}

func RecordBulkWriteBatchSize(size int) {
	if size <= 0 {
		return
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.bulkWriteBatchSizes = append(defaultRegistry.bulkWriteBatchSizes, size)
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordBulkWritePartialFailure(total, failed int) {
	if total <= 0 {
		return
	}
	rate := 0.0
	if failed > 0 {
		rate = float64(failed) / float64(total)
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.bulkWritePartialFailureRate = rate
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordLagSeconds(kind string, seconds float64) {
	if seconds < 0 {
		seconds = 0
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	switch kind {
	case "graph":
		defaultRegistry.lagGraphSeconds = append(defaultRegistry.lagGraphSeconds, seconds)
	case "vector":
		defaultRegistry.lagVectorSeconds = append(defaultRegistry.lagVectorSeconds, seconds)
	}
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordOrphanCounts(relationships, vectorDocs int) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.orphanRelationshipsCount = relationships
	defaultRegistry.orphanVectorDocsCount = vectorDocs
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func SetOutboxBacklog(count int) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.lastOutboxBacklog = count
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordRealtimeReadFallback(domainID, appID string) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.realtimeReadFallbackCount++
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordGraphScopeConflict(scopeA, scopeB string) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.graphScopeConflictCount++
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionOutboxClaimSize(size int) {
	if size <= 0 {
		return
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionOutboxClaimSizes = append(defaultRegistry.projectionOutboxClaimSizes, size)
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionCoalescedEntities(size int) {
	if size <= 0 {
		return
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionCoalescedEntityCounts = append(defaultRegistry.projectionCoalescedEntityCounts, size)
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionQueueAge(seconds float64) {
	if seconds < 0 {
		seconds = 0
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionQueueAgeSeconds = append(defaultRegistry.projectionQueueAgeSeconds, seconds)
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionGraphLatency(seconds float64) {
	if seconds < 0 {
		seconds = 0
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionGraphBatchLatencies = append(defaultRegistry.projectionGraphBatchLatencies, seconds)
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionVectorLatency(seconds float64) {
	if seconds < 0 {
		seconds = 0
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionVectorBatchLatencies = append(defaultRegistry.projectionVectorBatchLatencies, seconds)
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionEmbeddingLatency(seconds float64) {
	if seconds < 0 {
		seconds = 0
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionEmbeddingBatchLatencies = append(defaultRegistry.projectionEmbeddingBatchLatencies, seconds)
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionPartialFailure(count int) {
	if count <= 0 {
		return
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionPartialFailures += count
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func RecordProjectionStaleSkip(count int) {
	if count <= 0 {
		return
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.projectionStaleSkips += count
	defaultRegistry.lastUpdate = time.Now().UTC()
}

func (r *Registry) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	snap := Snapshot{
		OutboxBacklog:               r.lastOutboxBacklog,
		GraphLagSeconds:             summarizeFloats(r.lagGraphSeconds),
		VectorLagSeconds:            summarizeFloats(r.lagVectorSeconds),
		OrphanRelationshipsCount:    r.orphanRelationshipsCount,
		OrphanVectorDocsCount:       r.orphanVectorDocsCount,
		BulkWriteBatchSize:          summarizeInts(r.bulkWriteBatchSizes),
		BulkWritePartialFailureRate: r.bulkWritePartialFailureRate,
		RealtimeReadFallbackCount:   r.realtimeReadFallbackCount,
		GraphScopeConflictCount:     r.graphScopeConflictCount,
		ProjectionOutboxClaimSize:   summarizeInts(r.projectionOutboxClaimSizes),
		ProjectionCoalescedEntities: summarizeInts(r.projectionCoalescedEntityCounts),
		ProjectionQueueAgeSeconds:   summarizeFloats(r.projectionQueueAgeSeconds),
		ProjectionGraphLatency:      summarizeFloats(r.projectionGraphBatchLatencies),
		ProjectionVectorLatency:     summarizeFloats(r.projectionVectorBatchLatencies),
		ProjectionEmbeddingLatency:  summarizeFloats(r.projectionEmbeddingBatchLatencies),
		ProjectionPartialFailures:   r.projectionPartialFailures,
		ProjectionStaleSkips:        r.projectionStaleSkips,
		LastUpdatedAt:               r.lastUpdate,
	}
	if snap.OutboxBacklog > 1000 {
		snap.Alerts = append(snap.Alerts, "outbox backlog high")
	}
	if snap.GraphLagSeconds.P95 > 300 || snap.VectorLagSeconds.P95 > 300 {
		snap.Alerts = append(snap.Alerts, "projection lag high")
	}
	return snap
}

func summarizeInts(values []int) LagStats {
	if len(values) == 0 {
		return LagStats{}
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	return LagStats{
		Count:  len(sorted),
		Median: float64(medianInt(sorted)),
		P95:    float64(percentileInt(sorted, 0.95)),
		Max:    float64(sorted[len(sorted)-1]),
	}
}

func summarizeFloats(values []float64) LagStats {
	if len(values) == 0 {
		return LagStats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return LagStats{
		Count:  len(sorted),
		Median: medianFloat(sorted),
		P95:    percentileFloat(sorted, 0.95),
		Max:    sorted[len(sorted)-1],
	}
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return int(math.Round(float64(values[mid-1]+values[mid]) / 2))
}

func percentileInt(values []int, p float64) int {
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
