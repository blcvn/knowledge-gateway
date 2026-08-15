package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"kg-service/internal/config"
	"kg-service/internal/integrity"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/session"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
	"kg-service/internal/runtimeobs"
	"kg-service/internal/searchprofile"
	"kg-service/internal/telemetry"
	"kg-service/internal/write"
)

type Repository interface {
	ClaimOutboxBatch(ctx context.Context, pageSize int) ([]write.OutboxEvent, error)
	ListOutboxEvents() []write.OutboxEvent
	GetOutboxEventByID(id string) (write.OutboxEvent, bool)
	GetNodeByID(id string) (write.NodeRecord, bool)
	GetNodesByIDs(ids []string) map[string]write.NodeRecord
	GetRelationshipByID(id string) (write.RelationshipRecord, bool)
	GetRelationshipsByIDs(ids []string) map[string]write.RelationshipRecord
	write.OutboxReader
	SoftDeleteRelationshipsWithOutbox(ctx context.Context, relationshipIDs []string, deletedAt time.Time) ([]write.RelationshipRecord, error)
	ListNodesBatch(afterID string, limit int) []write.NodeRecord
	ListRelationshipsBatch(afterID string, limit int) []write.RelationshipRecord
	UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error
	GetProjectionVersion(entityID, entityKind string) (write.ProjectionVersionRecord, bool)
	ListProjectionVersionsBatch(afterEntityKind, afterEntityID string, limit int) []write.ProjectionVersionRecord
	UpsertProjectionVersion(ctx context.Context, record write.ProjectionVersionRecord) error
	GetGraphVersionEntities(versionID string) []write.GraphVersionEntityRecord
	ListPendingGraphVersionsBefore(cutoff time.Time) []write.GraphVersionRecord
	ArchiveGraphVersions(ctx context.Context, keepCount int, olderThan time.Time) ([]string, error)
	GetGraphIdentityByID(ctx context.Context, identifierID string) (write.GraphIdentityRecord, bool)
	AbandonGraphVersion(ctx context.Context, versionID string) error
	CleanupExpiredSyncSession(ctx context.Context, versionID string) error
	ReleaseScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error
	UpsertGraphProjectionHead(ctx context.Context, record write.GraphProjectionHeadRecord) error
	GetGraphProjectionHead(identifierID, backendKind, backendName string) (write.GraphProjectionHeadRecord, bool)
}

type OntologyResolver interface {
	GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error)
}

type Runtime struct {
	store              Repository
	ontology           OntologyResolver
	cache              *rediscache.Client
	logger             *runtimeobs.Logger
	embedding          vector.EmbeddingRouter
	profiles           ontology.SearchProfileResolver
	graphAdapter       graphstore.GraphAdapter
	vectorAdapter      vectorstore.VectorAdapter
	ftsAdapter         fts.FTSAdapter
	tracer             trace.Tracer
	graph              *GraphStore
	vector             *VectorStore
	sessionManager     session.Manager
	mu                 sync.Mutex
	outbox             map[string]EventEnvelope
	maxRetries         int
	lagToleranceWindow time.Duration
	workerPoolSize     int
	outboxPageSize     int
}

type projectionWorkUnit struct {
	Events          []write.OutboxEvent
	Latest          write.OutboxEvent
	EntityKind      string
	EntityID        string
	SourceVersion   int64
	SourceUpdatedAt time.Time
}

type projectionUnitResult struct {
	Unit          projectionWorkUnit
	SourceNode    write.NodeRecord
	SourceRel     write.RelationshipRecord
	GraphNode     graphstore.GraphNode
	VectorDoc     vectorstore.VectorDocument
	GraphRel      graphstore.GraphRelationship
	GraphSynced   bool
	VectorSynced  bool
	GraphSkipped  bool
	VectorSkipped bool
	Err           error
	DeadLetter    bool
}

type RuntimeOption func(*Runtime)

type projectionVersionWriter interface {
	UpsertProjectionVersion(context.Context, write.ProjectionVersionRecord) error
}

func WithLagToleranceWindow(window time.Duration) RuntimeOption {
	return func(r *Runtime) {
		if window > 0 {
			r.lagToleranceWindow = window
		}
	}
}

func WithMaxRetries(maxRetries int) RuntimeOption {
	return func(r *Runtime) {
		if maxRetries > 0 {
			r.maxRetries = maxRetries
		}
	}
}

func WithWorkerPoolSize(workerPoolSize int) RuntimeOption {
	return func(r *Runtime) {
		if workerPoolSize > 0 {
			r.workerPoolSize = workerPoolSize
		}
	}
}

func WithOutboxPageSize(pageSize int) RuntimeOption {
	return func(r *Runtime) {
		if pageSize > 0 {
			r.outboxPageSize = pageSize
		}
	}
}

func WithSessionManager(manager session.Manager) RuntimeOption {
	return func(r *Runtime) {
		if manager != nil {
			r.sessionManager = manager
		}
	}
}

func WithLogger(logger *runtimeobs.Logger) RuntimeOption {
	return func(r *Runtime) {
		if logger != nil {
			r.logger = logger
		}
	}
}

func WithTracerProvider(provider trace.TracerProvider, name string) RuntimeOption {
	return func(r *Runtime) {
		if provider != nil {
			r.tracer = provider.Tracer(name)
		}
	}
}

func projectionBatchingEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("PROJECTION_BATCHING_ENABLED"))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func projectionBatchSizeFromEnv(name string, fallback int) int {
	if fallback <= 0 {
		fallback = 1
	}
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}

func (r *Runtime) graphBatchSize() int {
	return projectionBatchSizeFromEnv("GRAPH_BATCH_SIZE", 25)
}

func (r *Runtime) vectorBatchSize() int {
	return projectionBatchSizeFromEnv("VECTOR_BATCH_SIZE", 25)
}

func (r *Runtime) embedBatchSize() int {
	return projectionBatchSizeFromEnv("EMBED_BATCH_SIZE", 16)
}

func (r *Runtime) ftsBatchSize() int {
	return projectionBatchSizeFromEnv("FTS_BATCH_SIZE", 25)
}

func NewRuntime(store Repository, ontologyResolver OntologyResolver, cache *rediscache.Client, opts ...RuntimeOption) *Runtime {
	deterministic := vector.NewDeterministicProvider(8)
	router := vector.DirectRouter{Provider: deterministic}
	profileResolver, _ := ontologyResolver.(ontology.SearchProfileResolver)
	runtime := &Runtime{
		store:              store,
		ontology:           ontologyResolver,
		cache:              cache,
		logger:             runtimeobs.NewLogger(config.Config{}, "workers"),
		tracer:             otel.Tracer("kg-service/workers"),
		embedding:          router,
		profiles:           profileResolver,
		vectorAdapter:      vectorstore.NewInMemoryVectorAdapter(),
		ftsAdapter:         fts.NewInMemoryFTSAdapter(),
		graph:              &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector:             &VectorStore{Documents: map[string]VectorDocument{}},
		outbox:             map[string]EventEnvelope{},
		maxRetries:         3,
		lagToleranceWindow: 30 * time.Second,
		workerPoolSize:     10,
		outboxPageSize:     100,
	}
	runtime.graphAdapter = runtimeGraphAdapter{graph: runtime.graph}
	for _, opt := range opts {
		if opt != nil {
			opt(runtime)
		}
	}
	return runtime
}

func NewRuntimeFromConfig(store Repository, ontologyResolver OntologyResolver, cache *rediscache.Client, cfg config.Config) *Runtime {
	workerPoolSize := 10
	if raw := strings.TrimSpace(os.Getenv("WORKER_POOL_SIZE")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			workerPoolSize = parsed
		}
	}
	outboxPageSize := 100
	if raw := strings.TrimSpace(os.Getenv("OUTBOX_PAGE_SIZE")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			outboxPageSize = parsed
		}
	}
	return NewRuntime(
		store,
		ontologyResolver,
		cache,
		WithLogger(runtimeobs.NewLogger(cfg, "workers")),
		WithLagToleranceWindow(time.Duration(cfg.SyncLagToleranceMs)*time.Millisecond),
		WithMaxRetries(cfg.SyncLagStuckRetries),
		WithWorkerPoolSize(workerPoolSize),
		WithOutboxPageSize(outboxPageSize),
	)
}

func (r *Runtime) logf(format string, args ...any) {
	if r == nil || r.logger == nil {
		return
	}
	r.logger.Printf(format, args...)
}

func (r *Runtime) logEventf(event write.OutboxEvent, format string, args ...any) {
	if r == nil || r.logger == nil {
		return
	}
	meta := requestMetaFromPayload(event.Payload)
	if meta == (runtimeobs.RequestMeta{}) {
		r.logger.Printf(format, args...)
		return
	}
	message := fmt.Sprintf(format, args...)
	r.logger.Printf(
		"%s request_id=%s trace_id=%s span_id=%s",
		message,
		meta.RequestID,
		meta.TraceID,
		meta.SpanID,
	)
}

func requestMetaFromPayload(payload map[string]any) runtimeobs.RequestMeta {
	if len(payload) == 0 {
		return runtimeobs.RequestMeta{}
	}
	meta := runtimeobs.RequestMeta{}
	if value, ok := payload["request_id"].(string); ok {
		meta.RequestID = strings.TrimSpace(value)
	}
	if value, ok := payload["trace_id"].(string); ok {
		meta.TraceID = strings.TrimSpace(value)
	}
	if value, ok := payload["span_id"].(string); ok {
		meta.SpanID = strings.TrimSpace(value)
	}
	return meta
}

func eventSpanLinks(event write.OutboxEvent) []trace.Link {
	meta := requestMetaFromPayload(event.Payload)
	if meta == (runtimeobs.RequestMeta{}) {
		return nil
	}
	traceID, err := trace.TraceIDFromHex(meta.TraceID)
	if err != nil {
		return nil
	}
	spanID, err := trace.SpanIDFromHex(meta.SpanID)
	if err != nil {
		return nil
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	if !sc.IsValid() {
		return nil
	}
	return []trace.Link{{SpanContext: sc}}
}

func (r *Runtime) Graph() *GraphStore {
	return r.graph
}

func (r *Runtime) Vector() *VectorStore {
	return r.vector
}

func (r *Runtime) VectorAdapter() vectorstore.VectorAdapter {
	return r.vectorAdapter
}

func (r *Runtime) FTSAdapter() fts.FTSAdapter {
	return r.ftsAdapter
}

func (r *Runtime) SetEmbeddingRouter(router vector.EmbeddingRouter) {
	if r == nil || router == nil {
		return
	}
	r.embedding = router
}

func (r *Runtime) SetGraphAdapter(adapter graphstore.GraphAdapter) {
	if r == nil || adapter == nil {
		return
	}
	r.graphAdapter = adapter
}

func (r *Runtime) SetVectorAdapter(adapter vectorstore.VectorAdapter) {
	if r == nil || adapter == nil {
		return
	}
	r.vectorAdapter = adapter
}

func (r *Runtime) SetFTSAdapter(adapter fts.FTSAdapter) {
	if r == nil || adapter == nil {
		return
	}
	r.ftsAdapter = adapter
}

func (r *Runtime) PollOnce() WorkerReport {
	report := WorkerReport{}
	for {
		batch, err := r.store.ClaimOutboxBatch(context.Background(), r.outboxPageSize)
		if err != nil {
			r.logf("claim outbox batch failed: %v", err)
			report.Failed++
			return report
		}
		if len(batch) == 0 {
			return report
		}
		report.BatchCount++
		telemetry.RecordProjectionOutboxClaimSize(len(batch))
		graphEvents, legacyEvents := partitionOutboxEvents(batch)
		if len(graphEvents) > 0 {
			var reportMu sync.Mutex
			for _, event := range graphEvents {
				r.processOutboxEvent(event, &report, &reportMu)
			}
		}
		if len(legacyEvents) == 0 {
			r.mu.Lock()
			r.refreshStatusCascadesLocked()
			r.mu.Unlock()
			continue
		}
		if !projectionBatchingEnabled() {
			var reportMu sync.Mutex
			jobs := make(chan write.OutboxEvent)
			var wg sync.WaitGroup
			workerCount := r.workerPoolSize
			if workerCount <= 0 {
				workerCount = 1
			}
			for i := 0; i < workerCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for event := range jobs {
						r.processOutboxEvent(event, &report, &reportMu)
					}
				}()
			}
			for _, event := range legacyEvents {
				jobs <- event
			}
			close(jobs)
			wg.Wait()
			r.mu.Lock()
			r.refreshStatusCascadesLocked()
			r.mu.Unlock()
			continue
		}

		units := coalesceOutboxEvents(legacyEvents)
		telemetry.RecordProjectionCoalescedEntities(len(units))
		for _, unit := range units {
			telemetry.RecordProjectionQueueAge(time.Since(unit.Latest.CreatedAt).Seconds())
		}
		results := r.projectCoalescedUnits(units)
		for _, result := range results {
			r.commitProjectionResult(result, &report)
		}
		r.mu.Lock()
		r.refreshStatusCascadesLocked()
		r.mu.Unlock()
	}
}

func sessionCleanupIntervalFromEnv() time.Duration {
	const defaultMinutes = 30
	raw := strings.TrimSpace(os.Getenv("SYNC_SESSION_CLEANUP_INTERVAL_MINUTES"))
	if raw == "" {
		return time.Duration(defaultMinutes) * time.Minute
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return time.Duration(defaultMinutes) * time.Minute
	}
	return time.Duration(parsed) * time.Minute
}

// staleSessionTTLFromEnv is how long a sync session may sit in PENDING_ENTITIES before the sweep
// abandons it and releases its scope lease.
//
// The default stays at 2h — the same value this loop used before it became configurable — because
// the sweep is service-wide: shortening it would abandon a long-running session belonging to any
// client, not just the one whose deployment wanted a shorter window. A client that needs faster
// recovery reclaims its own stale lease directly rather than lowering this for everyone.
func staleSessionTTLFromEnv() time.Duration {
	return durationEnv("KG_STALE_SESSION_TTL", 2*time.Hour)
}

// versionRetentionFromEnv is the age below which a sealed version is never archived.
func versionRetentionFromEnv() time.Duration {
	return durationEnv("KG_VERSION_RETENTION", 720*time.Hour)
}

// versionKeepCountFromEnv is how many recent versions of a graph stay ONLINE regardless of age.
func versionKeepCountFromEnv() int {
	const defaultKeep = 50
	raw := strings.TrimSpace(os.Getenv("KG_VERSION_KEEP_COUNT"))
	if raw == "" {
		return defaultKeep
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return defaultKeep
	}
	return parsed
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func (r *Runtime) RunSessionCleanupLoop(ctx context.Context) {
	if r == nil {
		return
	}
	interval := sessionCleanupIntervalFromEnv()
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.cleanupExpiredSyncSessions(ctx)
			r.archiveSupersededGraphVersions(ctx)
		}
	}
}

// archiveSupersededGraphVersions bounds the growth of kg_graph_versions and, more importantly, of
// kg_graph_version_entities: every sealed write appends one manifest row per touched entity, so a
// graph rewritten repeatedly accumulates manifest rows far faster than anything reads them.
func (r *Runtime) archiveSupersededGraphVersions(ctx context.Context) int {
	if r == nil || r.store == nil {
		return 0
	}
	keepCount := versionKeepCountFromEnv()
	cutoff := time.Now().UTC().Add(-versionRetentionFromEnv())
	archived, err := r.store.ArchiveGraphVersions(ctx, keepCount, cutoff)
	if err != nil {
		r.logf("graph version archive failed keep_count=%d cutoff=%s err=%v", keepCount, cutoff.Format(time.RFC3339), err)
		return 0
	}
	if len(archived) > 0 {
		r.logf("graph version archive processed=%d keep_count=%d cutoff=%s", len(archived), keepCount, cutoff.Format(time.RFC3339))
	}
	return len(archived)
}

func (r *Runtime) cleanupExpiredSyncSessions(ctx context.Context) int {
	if r == nil || r.store == nil {
		return 0
	}
	cutoff := time.Now().UTC().Add(-staleSessionTTLFromEnv())
	expired := r.store.ListPendingGraphVersionsBefore(cutoff)
	if len(expired) == 0 {
		return 0
	}
	cleaned := 0
	for _, version := range expired {
		if err := r.store.CleanupExpiredSyncSession(ctx, version.VersionID); err != nil {
			r.logf("session cleanup failed version_id=%s err=%v", version.VersionID, err)
			continue
		}
		cleaned++
	}
	if cleaned > 0 {
		r.logf("session cleanup processed=%d cutoff=%s", cleaned, cutoff.Format(time.RFC3339))
	}
	return cleaned
}

func (r *Runtime) processOutboxEvent(event write.OutboxEvent, report *WorkerReport, reportMu *sync.Mutex) {
	r.mu.Lock()
	defer r.mu.Unlock()

	env, ok := r.outbox[event.ID]
	if !ok {
		status := EventStatus(event.Status)
		if status == "" {
			status = EventPending
		}
		env = EventEnvelope{Event: event, Status: status, RetryCount: event.RetryCount}
	}
	if env.Status == EventDone || env.Status == EventDeadLetter {
		return
	}

	tracer := r.tracer
	if tracer == nil {
		tracer = otel.Tracer("kg-service/workers")
	}
	spanOpts := []trace.SpanStartOption{
		trace.WithAttributes(
			attribute.String("kg.event_id", event.ID),
			attribute.String("kg.aggregate_type", event.AggregateType),
			attribute.String("kg.event_type", event.EventType),
			attribute.String("kg.event_status", string(env.Status)),
			attribute.Int("kg.event_retry_count", env.RetryCount),
		),
	}
	if links := eventSpanLinks(event); len(links) > 0 {
		spanOpts = append(spanOpts, trace.WithLinks(links...))
	}
	ctx, span := tracer.Start(context.Background(), "worker.process_outbox_event", spanOpts...)
	defer span.End()

	r.logEventf(event,
		"projection event %s start aggregate=%s type=%s status=%s retry=%d payload=%v",
		event.ID,
		event.AggregateType,
		event.EventType,
		env.Status,
		env.RetryCount,
		event.Payload,
	)
	env.Status = EventProcessing
	span.SetAttributes(attribute.String("kg.event_status", string(EventProcessing)))
	_ = r.store.UpdateOutboxStatus(ctx, event.ID, string(EventProcessing), env.RetryCount, nil)
	if err := r.handleEvent(env.Event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		env.RetryCount++
		env.Error = err.Error()
		reportMu.Lock()
		if len(report.SampleErrors) < 3 {
			r.logEventf(event,
				"projection event %s failed aggregate=%s type=%s retry=%d/%d err=%v",
				event.ID,
				event.AggregateType,
				event.EventType,
				env.RetryCount,
				r.maxRetries,
				err,
			)
		}
		report.recordError(err)
		if env.RetryCount >= r.maxRetries {
			env.Status = EventDeadLetter
			report.DeadLetter++
		} else {
			env.Status = EventFailed
			report.Failed++
		}
		reportMu.Unlock()
		span.SetAttributes(attribute.String("kg.event_status", string(env.Status)))
		_ = r.store.UpdateOutboxStatus(ctx, event.ID, string(env.Status), env.RetryCount, nil)
		r.outbox[event.ID] = env
		return
	}
	env.Status = EventDone
	env.Error = ""
	now := time.Now().UTC()
	env.Event.ProcessedAt = &now
	span.SetAttributes(
		attribute.String("kg.event_status", string(EventDone)),
		attribute.String("kg.event_processed_at", now.Format(time.RFC3339Nano)),
	)
	_ = r.store.UpdateOutboxStatus(ctx, event.ID, string(EventDone), env.RetryCount, &now)
	r.outbox[event.ID] = env
	r.logEventf(event,
		"projection event %s done aggregate=%s type=%s retry=%d processed_at=%s",
		event.ID,
		event.AggregateType,
		event.EventType,
		env.RetryCount,
		now.Format(time.RFC3339Nano),
	)
	reportMu.Lock()
	report.Processed++
	reportMu.Unlock()
}

func coalesceOutboxEvents(events []write.OutboxEvent) []projectionWorkUnit {
	units := make([]projectionWorkUnit, 0, len(events))
	indexByKey := make(map[string]int, len(events))
	for _, event := range events {
		key, kind, entityID := projectionEventKey(event)
		if idx, ok := indexByKey[key]; ok {
			units[idx].Events = append(units[idx].Events, event)
			units[idx].Latest = event
			continue
		}
		unit := projectionWorkUnit{
			Events:     []write.OutboxEvent{event},
			Latest:     event,
			EntityKind: kind,
			EntityID:   entityID,
		}
		indexByKey[key] = len(units)
		units = append(units, unit)
	}
	return units
}

func partitionOutboxEvents(events []write.OutboxEvent) ([]write.OutboxEvent, []write.OutboxEvent) {
	graphEvents := make([]write.OutboxEvent, 0, len(events))
	legacyEvents := make([]write.OutboxEvent, 0, len(events))
	for _, event := range events {
		if hasGraphVersionMetadata(event) {
			graphEvents = append(graphEvents, event)
			continue
		}
		legacyEvents = append(legacyEvents, event)
	}
	return graphEvents, legacyEvents
}

func hasGraphVersionMetadata(event write.OutboxEvent) bool {
	identifierID, _ := event.Payload["graph_identifier_id"].(string)
	versionID, _ := event.Payload["graph_version_id"].(string)
	if identifierID == "" || versionID == "" {
		return false
	}
	switch raw := event.Payload["graph_version_number"].(type) {
	case int64:
		return raw > 0
	case int:
		return raw > 0
	case float64:
		return raw > 0
	case json.Number:
		if parsed, err := raw.Int64(); err == nil {
			return parsed > 0
		}
	}
	return false
}

func projectionEventKey(event write.OutboxEvent) (string, string, string) {
	entityKind := event.AggregateType
	switch event.AggregateType {
	case "kg_node":
		entityID, _ := event.Payload["node_id"].(string)
		if entityID == "" {
			entityID = event.AggregateID
		}
		return entityKind + ":" + entityID, entityKind, entityID
	case "kg_relationship":
		entityID, _ := event.Payload["relationship_id"].(string)
		if entityID == "" {
			entityID = event.AggregateID
		}
		return entityKind + ":" + entityID, entityKind, entityID
	default:
		return event.ID, entityKind, event.AggregateID
	}
}

func (r *Runtime) projectCoalescedUnits(units []projectionWorkUnit) []projectionUnitResult {
	results := make([]projectionUnitResult, len(units))
	if len(units) == 0 {
		return results
	}

	nodeIDs := make([]string, 0, len(units))
	relIDs := make([]string, 0, len(units))
	for _, unit := range units {
		switch unit.EntityKind {
		case "kg_node":
			nodeIDs = append(nodeIDs, unit.EntityID)
		case "kg_relationship":
			relIDs = append(relIDs, unit.EntityID)
		}
	}
	nodeSources := r.store.GetNodesByIDs(nodeIDs)
	relSources := r.store.GetRelationshipsByIDs(relIDs)

	nodeUpserts := make([]nodeProjectionWork, 0, len(nodeIDs))
	nodeDeletes := make([]nodeProjectionWork, 0, len(nodeIDs))
	relUpserts := make([]relationshipProjectionWork, 0, len(relIDs))
	relDeletes := make([]relationshipProjectionWork, 0, len(relIDs))

	for idx, unit := range units {
		results[idx].Unit = unit
		switch unit.EntityKind {
		case "kg_node":
			node, ok := nodeSources[unit.EntityID]
			if !ok {
				r.logf("projection skip stale event entity_id=%s kind=kg_node event_type=%s: node no longer exists",
					unit.EntityID, unit.Latest.EventType)
				results[idx].GraphSynced = true
				results[idx].VectorSynced = true
				continue
			}
			latest := unit.Latest
			if latest.EventType == "NODE_DELETED" {
				node.IsDeleted = true
			}
			work, err := r.buildNodeProjectionWork(idx, unit, node)
			if err != nil {
				results[idx].Err = err
				continue
			}
			results[idx].SourceNode = node
			results[idx].GraphNode = work.graphNode
			results[idx].VectorDoc = work.vectorDoc
			if latest.EventType == "NODE_DELETED" {
				nodeDeletes = append(nodeDeletes, work)
			} else {
				nodeUpserts = append(nodeUpserts, work)
			}
		case "kg_relationship":
			rel, ok := relSources[unit.EntityID]
			if !ok {
				r.logf("projection skip stale event entity_id=%s kind=kg_relationship event_type=%s: relationship no longer exists",
					unit.EntityID, unit.Latest.EventType)
				results[idx].GraphSynced = true
				results[idx].VectorSynced = true
				continue
			}
			work := r.buildRelationshipProjectionWork(idx, unit, rel)
			results[idx].SourceRel = rel
			results[idx].GraphRel = work.graphRel
			if unit.Latest.EventType == "RELATIONSHIP_DELETED" {
				relDeletes = append(relDeletes, work)
			} else {
				relUpserts = append(relUpserts, work)
			}
		default:
			if err := r.handleEvent(unit.Latest); err != nil {
				results[idx].Err = err
				continue
			}
			results[idx].GraphSynced = true
			results[idx].VectorSynced = true
		}
	}

	r.applyNodeProjectionWork(nodeUpserts, false, results)
	r.applyNodeProjectionWork(nodeDeletes, true, results)
	r.applyRelationshipProjectionWork(relUpserts, false, results)
	r.applyRelationshipProjectionWork(relDeletes, true, results)
	return results
}

type nodeProjectionWork struct {
	resultIndex       int
	unit              projectionWorkUnit
	source            write.NodeRecord
	graphNode         graphstore.GraphNode
	vectorDoc         vectorstore.VectorDocument
	ftsDoc            fts.FTSDocument
	embeddingProvider vector.EmbeddingProvider
	embeddingText     string
	graphStale        bool
	vectorStale       bool
}

type relationshipProjectionWork struct {
	resultIndex int
	unit        projectionWorkUnit
	source      write.RelationshipRecord
	graphRel    graphstore.GraphRelationship
	graphStale  bool
}

func (r *Runtime) buildNodeProjectionWork(resultIndex int, unit projectionWorkUnit, node write.NodeRecord) (nodeProjectionWork, error) {
	acl := nodeACLVisibleTo(node)
	graphNode := graphstore.GraphNode{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), acl...),
		Visibility:    node.Visibility,
		StatusValue:   node.StatusValue,
		IsDeleted:     node.IsDeleted,
		SyncVersion:   int64(node.DomainVersion),
		Properties:    cloneMap(node.Properties),
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
	}
	graphNode.Properties["_kg_sync_version"] = int64(node.DomainVersion)

	cfg, err := r.ontology.GetStatusFieldConfig(node.DomainID)
	if err != nil {
		cfg = nil
	}
	resolvedProfile := ontology.ResolvedSearchProfile{}
	if r.profiles != nil {
		if profile, err := r.profiles.Resolve(node.DomainID, node.OwnerTenantID, node.OwnerAppID); err == nil {
			resolvedProfile = profile
		}
	}
	text := searchprofile.BuildEmbeddingText(node, resolvedProfile)
	embeddingProvider := r.embedding.RouteContext(node.OwnerTenantID, node.DomainID)
	if embeddingProvider == nil {
		embeddingProvider = vector.NewDeterministicProvider(8)
	}
	vectorDoc := vectorstore.VectorDocument{
		NodeID:        node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), acl...),
		IsDeleted:     node.IsDeleted,
		StatusValue:   node.StatusValue,
		SyncVersion:   int64(node.DomainVersion),
		DomainProps:   cloneMap(node.Properties),
	}
	if cfg != nil && cfg.AuthorityFieldName != "" && len(cfg.AuthorityValuesMap) > 0 {
		if raw, ok := node.Properties[cfg.AuthorityFieldName]; ok {
			if score, ok := cfg.AuthorityValuesMap[fmt.Sprintf("%v", raw)]; ok {
				vectorDoc.AuthorityScore = &score
			}
		}
	}
	ftsDoc := buildFTSDocument(node, acl, vectorDoc.AuthorityScore)
	current, _ := r.store.GetProjectionVersion(node.ID, "kg_node")
	sourceVersion := int64(node.DomainVersion)
	return nodeProjectionWork{
		resultIndex:       resultIndex,
		unit:              unit,
		source:            node,
		graphNode:         graphNode,
		vectorDoc:         vectorDoc,
		ftsDoc:            ftsDoc,
		embeddingProvider: embeddingProvider,
		embeddingText:     text,
		graphStale:        current.GraphVersion > sourceVersion,
		vectorStale:       current.VectorVersion > sourceVersion,
	}, nil
}

func (r *Runtime) embedNodeProjectionWorkBatch(work []nodeProjectionWork) error {
	if len(work) == 0 {
		return nil
	}
	batchSize := r.embedBatchSize()
	if batchSize <= 0 {
		batchSize = 1
	}
	groups := map[string][]int{}
	for idx := range work {
		provider := work[idx].embeddingProvider
		if provider == nil {
			provider = vector.NewDeterministicProvider(8)
			work[idx].embeddingProvider = provider
		}
		key := fmt.Sprintf("%T|%s", provider, provider.ModelID())
		groups[key] = append(groups[key], idx)
	}
	for _, indexes := range groups {
		for start := 0; start < len(indexes); start += batchSize {
			end := start + batchSize
			if end > len(indexes) {
				end = len(indexes)
			}
			chunkIndexes := indexes[start:end]
			provider := work[chunkIndexes[0]].embeddingProvider
			texts := make([]string, 0, len(chunkIndexes))
			for _, idx := range chunkIndexes {
				texts = append(texts, work[idx].embeddingText)
			}
			embeddings, err := provider.EmbedBatch(context.Background(), texts)
			if err != nil {
				return err
			}
			if len(embeddings) != len(chunkIndexes) {
				return fmt.Errorf("embedding batch size mismatch: got %d want %d", len(embeddings), len(chunkIndexes))
			}
			for i, idx := range chunkIndexes {
				work[idx].vectorDoc.Embedding = append([]float64(nil), embeddings[i]...)
			}
		}
	}
	return nil
}

func (r *Runtime) buildRelationshipProjectionWork(resultIndex int, unit projectionWorkUnit, rel write.RelationshipRecord) relationshipProjectionWork {
	graphRel := graphstore.GraphRelationship{
		ID:          rel.ID,
		RelType:     rel.RelType,
		FromNodeID:  rel.FromNodeID,
		ToNodeID:    rel.ToNodeID,
		DomainID:    rel.DomainID,
		SyncVersion: int64(rel.DomainVersion),
		Properties:  cloneMap(rel.Properties),
	}
	current, _ := r.store.GetProjectionVersion(rel.ID, "kg_relationship")
	return relationshipProjectionWork{
		resultIndex: resultIndex,
		unit:        unit,
		source:      rel,
		graphRel:    graphRel,
		graphStale:  current.GraphVersion > int64(rel.DomainVersion),
	}
}

func (r *Runtime) applyNodeProjectionWork(work []nodeProjectionWork, deleteMode bool, results []projectionUnitResult) {
	if len(work) == 0 {
		return
	}
	graphBatch := r.graphBatchSize()
	if graphBatch <= 0 {
		graphBatch = 25
	}
	vectorBatch := r.vectorBatchSize()
	if vectorBatch <= 0 {
		vectorBatch = 25
	}
	ftsBatch := r.ftsBatchSize()
	if ftsBatch <= 0 {
		ftsBatch = 25
	}

	if deleteMode {
		docs := make([]vectorstore.VectorDocument, 0, len(work))
		graphNodes := make([]graphstore.GraphNode, 0, len(work))
		for _, item := range work {
			if !item.graphStale {
				graphNodes = append(graphNodes, item.graphNode)
			} else {
				results[item.resultIndex].GraphSkipped = true
			}
			if !item.vectorStale {
				docs = append(docs, item.vectorDoc)
			} else {
				results[item.resultIndex].VectorSkipped = true
			}
		}
		for i := 0; i < len(graphNodes); i += graphBatch {
			end := i + graphBatch
			if end > len(graphNodes) {
				end = len(graphNodes)
			}
			chunk := graphNodes[i:end]
			if batch, ok := r.graphAdapter.(graphstore.BatchGraphAdapter); ok {
				if err := batch.DeleteNodesBatch(context.Background(), chunk); err != nil {
					for _, node := range chunk {
						for _, item := range work {
							if item.graphNode.ID == node.ID {
								results[item.resultIndex].Err = err
							}
						}
					}
				}
			} else {
				for _, node := range chunk {
					if err := r.graphAdapter.DeleteNode(context.Background(), node.ID); err != nil {
						for _, item := range work {
							if item.graphNode.ID == node.ID {
								results[item.resultIndex].Err = err
							}
						}
					}
				}
			}
		}
		for i := 0; i < len(docs); i += vectorBatch {
			end := i + vectorBatch
			if end > len(docs) {
				end = len(docs)
			}
			chunk := docs[i:end]
			if batch, ok := r.vectorAdapter.(vectorstore.BatchVectorAdapter); ok {
				if err := batch.DeleteBatch(context.Background(), chunk); err != nil {
					for _, doc := range chunk {
						for _, item := range work {
							if item.vectorDoc.NodeID == doc.NodeID {
								results[item.resultIndex].Err = err
							}
						}
					}
				}
			} else {
				for _, doc := range chunk {
					_ = r.vectorAdapter.Delete(context.Background(), doc.NodeID)
				}
			}
		}
		for _, item := range work {
			result := &results[item.resultIndex]
			if !item.graphStale && result.Err == nil {
				result.GraphSynced = true
			}
			if !item.vectorStale && result.Err == nil {
				result.VectorSynced = true
			}
			if r.ftsAdapter != nil && result.Err == nil {
				_ = r.ftsAdapter.Delete(context.Background(), item.unit.EntityID)
			}
		}
		return
	}
	if err := r.embedNodeProjectionWorkBatch(work); err != nil {
		for _, item := range work {
			results[item.resultIndex].Err = err
		}
		return
	}
	for _, item := range work {
		results[item.resultIndex].VectorDoc = item.vectorDoc
	}

	graphNodes := make([]graphstore.GraphNode, 0, len(work))
	vectorDocs := make([]vectorstore.VectorDocument, 0, len(work))
	ftsDocs := make([]fts.FTSDocument, 0, len(work))
	for _, item := range work {
		if !item.graphStale {
			graphNodes = append(graphNodes, item.graphNode)
		} else {
			results[item.resultIndex].GraphSkipped = true
		}
		if !item.vectorStale {
			vectorDocs = append(vectorDocs, item.vectorDoc)
		} else {
			results[item.resultIndex].VectorSkipped = true
		}
		if item.unit.Latest.EventType != "NODE_DELETED" {
			ftsDocs = append(ftsDocs, item.ftsDoc)
		}
	}

	start := time.Now()
	if len(graphNodes) > 0 {
		for i := 0; i < len(graphNodes); i += graphBatch {
			end := i + graphBatch
			if end > len(graphNodes) {
				end = len(graphNodes)
			}
			chunk := graphNodes[i:end]
			if batch, ok := r.graphAdapter.(graphstore.BatchGraphAdapter); ok {
				if err := batch.UpsertNodesBatch(context.Background(), chunk); err != nil {
					for _, node := range chunk {
						for _, item := range work {
							if item.graphNode.ID == node.ID {
								results[item.resultIndex].Err = err
							}
						}
					}
				}
			} else {
				for _, node := range chunk {
					if err := r.graphAdapter.UpsertNode(context.Background(), node); err != nil {
						for _, item := range work {
							if item.graphNode.ID == node.ID {
								results[item.resultIndex].Err = err
							}
						}
					}
				}
			}
		}
		telemetry.RecordProjectionGraphLatency(time.Since(start).Seconds())
	}
	start = time.Now()
	if len(vectorDocs) > 0 {
		for i := 0; i < len(vectorDocs); i += vectorBatch {
			end := i + vectorBatch
			if end > len(vectorDocs) {
				end = len(vectorDocs)
			}
			chunk := vectorDocs[i:end]
			if batch, ok := r.vectorAdapter.(vectorstore.BatchVectorAdapter); ok {
				if err := batch.UpsertBatch(context.Background(), chunk); err != nil {
					for _, doc := range chunk {
						for _, item := range work {
							if item.vectorDoc.NodeID == doc.NodeID {
								results[item.resultIndex].Err = err
							}
						}
					}
				}
			} else {
				for _, doc := range chunk {
					_ = r.vectorAdapter.Upsert(context.Background(), doc)
				}
			}
		}
		telemetry.RecordProjectionVectorLatency(time.Since(start).Seconds())
	}
	start = time.Now()
	for i := 0; i < len(ftsDocs); i += ftsBatch {
		end := i + ftsBatch
		if end > len(ftsDocs) {
			end = len(ftsDocs)
		}
		for _, doc := range ftsDocs[i:end] {
			if r.ftsAdapter != nil {
				_ = r.ftsAdapter.Index(context.Background(), doc)
			}
		}
	}
	telemetry.RecordProjectionEmbeddingLatency(time.Since(start).Seconds())

	for _, item := range work {
		result := &results[item.resultIndex]
		if !item.graphStale && result.Err == nil {
			result.GraphSynced = true
		}
		if !item.vectorStale && result.Err == nil {
			result.VectorSynced = true
		}
	}
}

func (r *Runtime) applyRelationshipProjectionWork(work []relationshipProjectionWork, deleteMode bool, results []projectionUnitResult) {
	if len(work) == 0 {
		return
	}
	graphBatch := r.graphBatchSize()
	if graphBatch <= 0 {
		graphBatch = 25
	}
	if deleteMode {
		rels := make([]graphstore.GraphRelationship, 0, len(work))
		for _, item := range work {
			if !item.graphStale {
				rels = append(rels, item.graphRel)
			} else {
				results[item.resultIndex].GraphSkipped = true
			}
		}
		if len(rels) == 0 {
			return
		}
		if batch, ok := r.graphAdapter.(graphstore.BatchGraphAdapter); ok {
			if err := batch.DeleteRelationshipsBatch(context.Background(), rels); err != nil {
				for _, rel := range rels {
					for _, item := range work {
						if item.graphRel.ID == rel.ID {
							results[item.resultIndex].Err = err
						}
					}
				}
			}
		} else {
			for _, rel := range rels {
				if err := r.graphAdapter.DeleteRelationship(context.Background(), rel.ID); err != nil {
					for _, item := range work {
						if item.graphRel.ID == rel.ID {
							results[item.resultIndex].Err = err
						}
					}
				}
			}
		}
		for _, item := range work {
			if !item.graphStale && results[item.resultIndex].Err == nil {
				results[item.resultIndex].GraphSynced = true
			}
		}
		return
	}

	rels := make([]graphstore.GraphRelationship, 0, len(work))
	for _, item := range work {
		if !item.graphStale {
			rels = append(rels, item.graphRel)
		} else {
			results[item.resultIndex].GraphSkipped = true
		}
	}
	for i := 0; i < len(rels); i += graphBatch {
		end := i + graphBatch
		if end > len(rels) {
			end = len(rels)
		}
		chunk := rels[i:end]
		if batch, ok := r.graphAdapter.(graphstore.BatchGraphAdapter); ok {
			if err := batch.UpsertRelationshipsBatch(context.Background(), chunk); err != nil {
				for _, rel := range chunk {
					for _, item := range work {
						if item.graphRel.ID == rel.ID {
							results[item.resultIndex].Err = err
						}
					}
				}
			}
		} else {
			for _, rel := range chunk {
				if err := r.graphAdapter.UpsertRelationship(context.Background(), rel); err != nil {
					for _, item := range work {
						if item.graphRel.ID == rel.ID {
							results[item.resultIndex].Err = err
						}
					}
				}
			}
		}
	}
	for _, item := range work {
		if !item.graphStale && results[item.resultIndex].Err == nil {
			results[item.resultIndex].GraphSynced = true
		}
	}
}

func (r *Runtime) commitProjectionResult(result projectionUnitResult, report *WorkerReport) {
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commitProjectionResultLocked(result, report, now)
}

func (r *Runtime) commitProjectionResultLocked(result projectionUnitResult, report *WorkerReport, now time.Time) {
	unit := result.Unit
	record, ok := r.applyCommittedProjectionLocked(result, now)
	if ok {
		_ = r.store.UpsertProjectionVersion(context.Background(), record)
	}
	if result.Err != nil {
		telemetry.RecordProjectionPartialFailure(len(unit.Events))
		if len(report.SampleErrors) < 3 {
			r.logf(
				"projection unit failed entity_id=%s kind=%s events=%d retry=%d err=%v",
				unit.EntityID, unit.EntityKind, len(unit.Events),
				unit.Events[0].RetryCount+1, result.Err,
			)
		}
		report.recordError(result.Err)
		for _, event := range unit.Events {
			retryCount := event.RetryCount + 1
			status := EventFailed
			if retryCount >= r.maxRetries {
				status = EventDeadLetter
			}
			env := EventEnvelope{Event: event, Status: status, RetryCount: retryCount}
			if status == EventDeadLetter {
				env.Status = EventDeadLetter
			}
			r.outbox[event.ID] = env
			_ = r.store.UpdateOutboxStatus(context.Background(), event.ID, string(env.Status), env.RetryCount, nil)
		}
		if unit.Events[0].RetryCount+1 >= r.maxRetries {
			report.DeadLetter += len(unit.Events)
		} else {
			report.Failed += len(unit.Events)
		}
		return
	}
	for _, event := range unit.Events {
		env := EventEnvelope{Event: event, Status: EventDone, RetryCount: event.RetryCount}
		env.Event.ProcessedAt = &now
		r.outbox[event.ID] = env
		_ = r.store.UpdateOutboxStatus(context.Background(), event.ID, string(EventDone), env.RetryCount, &now)
	}
	report.Processed += len(unit.Events)
}

func (r *Runtime) applyCommittedProjectionLocked(result projectionUnitResult, now time.Time) (write.ProjectionVersionRecord, bool) {
	unit := result.Unit
	switch unit.EntityKind {
	case "kg_node":
		source := result.SourceNode
		if source.ID == "" {
			var ok bool
			source, ok = r.store.GetNodeByID(unit.EntityID)
			if !ok {
				return write.ProjectionVersionRecord{}, false
			}
		}
		deleteMode := unit.Latest.EventType == "NODE_DELETED"
		if result.GraphSynced {
			if deleteMode {
				delete(r.graph.Nodes, source.ID)
			} else {
				r.graph.Nodes[source.ID] = GraphNode{
					ID:            result.GraphNode.ID,
					NodeType:      result.GraphNode.NodeType,
					DomainID:      result.GraphNode.DomainID,
					OwnerTenantID: result.GraphNode.OwnerTenantID,
					OwnerAppID:    result.GraphNode.OwnerAppID,
					ACLVisibleTo:  append([]string(nil), result.GraphNode.ACLVisibleTo...),
					Visibility:    result.GraphNode.Visibility,
					StatusValue:   result.GraphNode.StatusValue,
					IsDeleted:     result.GraphNode.IsDeleted,
					SyncVersion:   result.GraphNode.SyncVersion,
					Properties:    cloneMap(result.GraphNode.Properties),
					CreatedAt:     result.GraphNode.CreatedAt,
					UpdatedAt:     result.GraphNode.UpdatedAt,
				}
			}
		}
		if result.VectorSynced {
			if deleteMode {
				delete(r.vector.Documents, source.ID)
			} else {
				r.vector.Documents[source.ID] = VectorDocument{
					NodeID:         result.VectorDoc.NodeID,
					NodeType:       result.VectorDoc.NodeType,
					DomainID:       result.VectorDoc.DomainID,
					OwnerTenantID:  result.VectorDoc.OwnerTenantID,
					OwnerAppID:     result.VectorDoc.OwnerAppID,
					ACLVisibleTo:   append([]string(nil), result.VectorDoc.ACLVisibleTo...),
					IsDeleted:      result.VectorDoc.IsDeleted,
					StatusValue:    result.VectorDoc.StatusValue,
					AuthorityScore: result.VectorDoc.AuthorityScore,
					SyncVersion:    result.VectorDoc.SyncVersion,
					DomainProps:    cloneMap(result.VectorDoc.DomainProps),
					Embedding:      append([]float64(nil), result.VectorDoc.Embedding...),
				}
			}
		}
		if result.GraphSkipped {
			telemetry.RecordProjectionStaleSkip(1)
		}
		if result.VectorSkipped {
			telemetry.RecordProjectionStaleSkip(1)
		}
		return r.mergeProjectionVersionLocked(unit, source.DomainVersion, source.UpdatedAt, result.GraphSynced, result.VectorSynced, result.GraphSkipped, result.VectorSkipped, now), true
	case "kg_relationship":
		source := result.SourceRel
		if source.ID == "" {
			var ok bool
			source, ok = r.store.GetRelationshipByID(unit.EntityID)
			if !ok {
				return write.ProjectionVersionRecord{}, false
			}
		}
		if result.GraphSynced {
			if unit.Latest.EventType == "RELATIONSHIP_DELETED" {
				delete(r.graph.Rels, source.ID)
			} else {
				r.graph.Rels[source.ID] = GraphRelationship{
					ID:          result.GraphRel.ID,
					RelType:     result.GraphRel.RelType,
					FromNodeID:  result.GraphRel.FromNodeID,
					ToNodeID:    result.GraphRel.ToNodeID,
					DomainID:    result.GraphRel.DomainID,
					SyncVersion: result.GraphRel.SyncVersion,
					Properties:  cloneMap(result.GraphRel.Properties),
				}
			}
		}
		if result.GraphSkipped {
			telemetry.RecordProjectionStaleSkip(1)
		}
		return r.mergeProjectionVersionLocked(unit, source.DomainVersion, source.CreatedAt, result.GraphSynced, false, result.GraphSkipped, false, now), true
	default:
		return write.ProjectionVersionRecord{}, false
	}
}

func (r *Runtime) mergeProjectionVersionLocked(unit projectionWorkUnit, sourceVersion int, sourceUpdatedAt time.Time, graphSynced, vectorSynced, graphSkipped, vectorSkipped bool, now time.Time) write.ProjectionVersionRecord {
	record, hasExisting := r.store.GetProjectionVersion(unit.EntityID, unit.EntityKind)
	if !hasExisting {
		record = write.ProjectionVersionRecord{}
	}
	if int64(sourceVersion) >= record.SourceVersion {
		record.EntityID = unit.EntityID
		record.EntityKind = unit.EntityKind
		record.SourceVersion = int64(sourceVersion)
		record.SourceEventID = unit.Latest.ID
		record.SourceUpdatedAt = sourceUpdatedAt
	}
	if graphSynced || graphSkipped {
		if int64(sourceVersion) >= record.GraphVersion {
			record.GraphVersion = maxInt64(record.GraphVersion, int64(sourceVersion))
			record.GraphBackend = "graph"
		}
		record.LastGraphSyncedAt = now
	}
	if vectorSynced || vectorSkipped {
		if int64(sourceVersion) >= record.VectorVersion {
			record.VectorVersion = maxInt64(record.VectorVersion, int64(sourceVersion))
			record.VectorBackend = "vector"
		}
		record.LastVectorSyncedAt = now
	}
	return record
}

func (r *Runtime) EventEnvelope(id string) (EventEnvelope, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	env, ok := r.outbox[id]
	return env, ok
}

func (r *Runtime) Reconcile() ReconciliationReport {
	r.mu.Lock()
	defer r.mu.Unlock()

	report := ReconciliationReport{}
	sourceNodes := loadNodeSnapshot(r.store)
	sourceRelationships := loadRelationshipSnapshot(r.store)
	graphNodes := map[string]GraphNode{}
	graphRelationships := map[string]GraphRelationship{}
	if r.graphAdapter != nil {
		if nodes, err := r.graphAdapter.ListNodes(context.Background()); err == nil {
			for _, node := range nodes {
				graphNodes[node.ID] = GraphNode{
					ID:            node.ID,
					NodeType:      node.NodeType,
					DomainID:      node.DomainID,
					OwnerTenantID: node.OwnerTenantID,
					OwnerAppID:    node.OwnerAppID,
					ACLVisibleTo:  append([]string(nil), node.ACLVisibleTo...),
					Visibility:    node.Visibility,
					StatusValue:   node.StatusValue,
					IsDeleted:     node.IsDeleted,
					SyncVersion:   node.SyncVersion,
					Properties:    cloneMap(node.Properties),
					CreatedAt:     node.CreatedAt,
					UpdatedAt:     node.UpdatedAt,
				}
			}
		}
		if rels, err := r.graphAdapter.ListRelationships(context.Background()); err == nil {
			for _, rel := range rels {
				graphRelationships[rel.ID] = GraphRelationship{
					ID:          rel.ID,
					RelType:     rel.RelType,
					FromNodeID:  rel.FromNodeID,
					ToNodeID:    rel.ToNodeID,
					DomainID:    rel.DomainID,
					SyncVersion: rel.SyncVersion,
					Properties:  cloneMap(rel.Properties),
				}
			}
		}
	}
	vectorDocs := map[string]VectorDocument{}
	if r.vectorAdapter != nil {
		if docs, err := r.vectorAdapter.Snapshot(context.Background()); err == nil {
			for _, doc := range docs {
				vectorDocs[doc.NodeID] = VectorDocument{
					NodeID:         doc.NodeID,
					NodeType:       doc.NodeType,
					DomainID:       doc.DomainID,
					OwnerTenantID:  doc.OwnerTenantID,
					OwnerAppID:     doc.OwnerAppID,
					ACLVisibleTo:   append([]string(nil), doc.ACLVisibleTo...),
					IsDeleted:      doc.IsDeleted,
					StatusValue:    doc.StatusValue,
					AuthorityScore: doc.AuthorityScore,
					SyncVersion:    doc.SyncVersion,
					DomainProps:    cloneMap(doc.DomainProps),
					Embedding:      append([]float64(nil), doc.Embedding...),
				}
			}
		}
	}

	for id, source := range sourceNodes {
		ledger, ledgerOk := r.store.GetProjectionVersion(id, "kg_node")
		expectedVersion := int64(source.DomainVersion)
		if ledgerOk && ledger.SourceVersion > 0 {
			expectedVersion = ledger.SourceVersion
		}
		if graphNode, ok := graphNodes[id]; ok {
			if graphNode.IsDeleted != source.IsDeleted || graphNode.StatusValue != source.StatusValue || graphNode.NodeType != source.NodeType || graphNode.DomainID != source.DomainID {
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_mismatch", Details: "graph projection does not match source node"})
			}
			if !reflect.DeepEqual(stripSyncVersionMetadata(graphNode.Properties), source.Properties) || !reflect.DeepEqual(graphNode.ACLVisibleTo, nodeACLVisibleTo(source)) {
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_payload_mismatch", Details: "graph node payload does not match source node"})
			}
			lagClass := classifyLag(
				graphNode.SyncVersion,
				expectedVersion,
				ledger.SourceEventID,
				ledger.LastGraphSyncedAt,
				r.store.GetOutboxEventByID,
				r.maxRetries,
				r.lagToleranceWindow,
			)
			switch lagClass {
			case SyncLagClassStuck:
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_stuck", Details: "graph node sync version does not match source version"})
			case SyncLagClassLagging:
				report.GraphLaggingCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_lagging", Details: "graph node sync version is behind source version"})
			case SyncLagClassInFlight:
				report.GraphInFlightCount++
			}
		} else {
			lagClass := classifyLag(
				0,
				expectedVersion,
				ledger.SourceEventID,
				ledger.LastGraphSyncedAt,
				r.store.GetOutboxEventByID,
				r.maxRetries,
				r.lagToleranceWindow,
			)
			switch lagClass {
			case SyncLagClassStuck:
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_stuck", Details: "graph node missing from graph projection"})
			case SyncLagClassLagging:
				report.GraphLaggingCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_lagging", Details: "graph node missing from graph projection"})
			case SyncLagClassInFlight:
				report.GraphInFlightCount++
			}
		}
		if doc, ok := vectorDocs[id]; ok {
			if doc.IsDeleted != source.IsDeleted || doc.StatusValue != source.StatusValue || doc.NodeType != source.NodeType || doc.DomainID != source.DomainID {
				report.VectorDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_mismatch", Details: "vector projection does not match source node"})
			}
			if !reflect.DeepEqual(stripSyncVersionMetadata(doc.DomainProps), source.Properties) || !reflect.DeepEqual(doc.ACLVisibleTo, nodeACLVisibleTo(source)) {
				report.VectorDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_payload_mismatch", Details: "vector document payload does not match source node"})
			}
			lagClass := classifyLag(
				doc.SyncVersion,
				expectedVersion,
				ledger.SourceEventID,
				ledger.LastVectorSyncedAt,
				r.store.GetOutboxEventByID,
				r.maxRetries,
				r.lagToleranceWindow,
			)
			switch lagClass {
			case SyncLagClassStuck:
				report.VectorDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_lag_stuck", Details: "vector document sync version does not match source version"})
			case SyncLagClassLagging:
				report.VectorLaggingCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_lag_lagging", Details: "vector document sync version is behind source version"})
			case SyncLagClassInFlight:
				report.VectorInFlightCount++
			}
		} else {
			lagClass := classifyLag(
				0,
				expectedVersion,
				ledger.SourceEventID,
				ledger.LastVectorSyncedAt,
				r.store.GetOutboxEventByID,
				r.maxRetries,
				r.lagToleranceWindow,
			)
			switch lagClass {
			case SyncLagClassStuck:
				report.VectorDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_lag_stuck", Details: "vector document missing from vector projection"})
			case SyncLagClassLagging:
				report.VectorLaggingCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "vector_lag_lagging", Details: "vector document missing from vector projection"})
			case SyncLagClassInFlight:
				report.VectorInFlightCount++
			}
		}
	}
	for id := range graphNodes {
		if _, ok := sourceNodes[id]; !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_graph_node", Details: "node exists in graph but not source"})
		}
	}
	for id := range vectorDocs {
		if _, ok := sourceNodes[id]; !ok {
			report.VectorDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_vector_doc", Details: "node exists in vector but not source"})
		}
	}
	for id, sourceRel := range sourceRelationships {
		graphRel, ok := graphRelationships[id]
		ledger, ledgerOk := r.store.GetProjectionVersion(id, "kg_relationship")
		expectedVersion := int64(sourceRel.DomainVersion)
		if ledgerOk && ledger.SourceVersion > 0 {
			expectedVersion = ledger.SourceVersion
		}
		if ok {
			if graphRel.RelType != sourceRel.RelType || graphRel.FromNodeID != sourceRel.FromNodeID || graphRel.ToNodeID != sourceRel.ToNodeID {
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "relationship_mismatch", Details: "graph relationship differs from source"})
			}
			if !reflect.DeepEqual(stripSyncVersionMetadata(graphRel.Properties), sourceRel.Properties) {
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "relationship_payload_mismatch", Details: "graph relationship payload does not match source"})
			}
			lagClass := classifyLag(
				graphRel.SyncVersion,
				expectedVersion,
				ledger.SourceEventID,
				ledger.LastGraphSyncedAt,
				r.store.GetOutboxEventByID,
				r.maxRetries,
				r.lagToleranceWindow,
			)
			switch lagClass {
			case SyncLagClassStuck:
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_stuck", Details: "graph relationship sync version does not match source version"})
			case SyncLagClassLagging:
				report.GraphLaggingCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_lagging", Details: "graph relationship sync version is behind source version"})
			case SyncLagClassInFlight:
				report.GraphInFlightCount++
			}
		} else {
			lagClass := classifyLag(
				0,
				expectedVersion,
				ledger.SourceEventID,
				ledger.LastGraphSyncedAt,
				r.store.GetOutboxEventByID,
				r.maxRetries,
				r.lagToleranceWindow,
			)
			switch lagClass {
			case SyncLagClassStuck:
				report.GraphDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_stuck", Details: "graph relationship missing from graph projection"})
			case SyncLagClassLagging:
				report.GraphLaggingCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "graph_lag_lagging", Details: "graph relationship missing from graph projection"})
			case SyncLagClassInFlight:
				report.GraphInFlightCount++
			}
		}
	}
	for id := range graphRelationships {
		if _, ok := sourceRelationships[id]; !ok {
			report.GraphDriftCount++
			report.Issues = append(report.Issues, ReconciliationIssue{ID: id, Kind: "orphan_graph_relationship", Details: "relationship exists in graph but not source"})
		}
	}
	for _, record := range loadProjectionVersionSnapshot(r.store) {
		switch record.EntityKind {
		case "kg_node":
			if _, ok := sourceNodes[record.EntityID]; !ok {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "orphan_projection_version", Details: "projection ledger contains node entry not present in source"})
				continue
			}
			source := sourceNodes[record.EntityID]
			if record.SourceVersion != int64(source.DomainVersion) {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "stale_projection_version", Details: "projection ledger source version differs from source node"})
			}
		case "kg_relationship":
			if _, ok := sourceRelationships[record.EntityID]; !ok {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "orphan_projection_version", Details: "projection ledger contains relationship entry not present in source"})
				continue
			}
			source := sourceRelationships[record.EntityID]
			if record.SourceVersion != int64(source.DomainVersion) {
				report.ProjectionVersionDriftCount++
				report.Issues = append(report.Issues, ReconciliationIssue{ID: record.EntityID, Kind: "stale_projection_version", Details: "projection ledger source version differs from source relationship"})
			}
		}
	}

	if report.GraphDriftCount == 0 && report.VectorDriftCount == 0 && report.ProjectionVersionDriftCount == 0 {
		if report.GraphLaggingCount > 0 || report.VectorLaggingCount > 0 {
			report.Overall = "warn"
		} else {
			report.Overall = "pass"
		}
	} else {
		report.Overall = "fail"
	}
	return report
}

func (r *Runtime) ScanOrphans(ctx context.Context, tenantID string) (integrity.OrphanScanResponse, error) {
	_ = ctx
	sourceNodes := loadNodeSnapshot(r.store)
	nodeActive := func(nodeID string) bool {
		node, ok := sourceNodes[nodeID]
		return ok && !node.IsDeleted && node.OwnerTenantID == tenantID
	}

	result := integrity.OrphanScanResponse{}
	graphNodes := r.Graph().SnapshotNodes()
	graphRels := r.Graph().SnapshotRelationships()
	for _, rel := range graphRels {
		if rel.DomainID == "" {
			continue
		}
		fromActive := nodeActive(rel.FromNodeID)
		toActive := nodeActive(rel.ToNodeID)
		if !fromActive || !toActive {
			result.RelationshipOrphans = append(result.RelationshipOrphans, integrity.OrphanRelationshipItem{
				RelationshipID: rel.ID,
				RelType:        rel.RelType,
				FromNodeID:     rel.FromNodeID,
				ToNodeID:       rel.ToNodeID,
				DomainID:       rel.DomainID,
			})
		}
	}
	vectorDocs := r.Vector().SnapshotDocuments()
	for _, doc := range vectorDocs {
		if doc.OwnerTenantID != tenantID {
			continue
		}
		node, ok := sourceNodes[doc.NodeID]
		if !ok || node.IsDeleted {
			result.VectorOrphans = append(result.VectorOrphans, integrity.OrphanVectorDocItem{
				NodeID:   doc.NodeID,
				NodeType: doc.NodeType,
				DomainID: doc.DomainID,
			})
		}
	}
	telemetry.RecordOrphanCounts(len(result.RelationshipOrphans), len(result.VectorOrphans))
	_ = graphNodes
	return result, nil
}

func (r *Runtime) RebuildProjection(ctx context.Context, tenantID string) (integrity.RepairReport, error) {
	_ = ctx
	report := integrity.RepairReport{ScannedAt: time.Now().UTC()}
	sourceNodes := loadNodeSnapshot(r.store)
	sourceRelationships := loadRelationshipSnapshot(r.store)
	for _, node := range sourceNodes {
		if node.OwnerTenantID != tenantID {
			continue
		}
		if _, _, err := r.projectNode(node); err != nil {
			return report, err
		}
		report.RebuiltNodes++
	}
	for _, rel := range sourceRelationships {
		if rel.OwnerTenantID != tenantID {
			continue
		}
		if rel.IsDeleted {
			continue
		}
		relPayload := GraphRelationship{
			ID:          rel.ID,
			RelType:     rel.RelType,
			FromNodeID:  rel.FromNodeID,
			ToNodeID:    rel.ToNodeID,
			DomainID:    rel.DomainID,
			SyncVersion: int64(rel.DomainVersion),
			Properties:  cloneMap(rel.Properties),
		}
		r.graph.Rels[rel.ID] = relPayload
		if r.graphAdapter != nil {
			if err := r.graphAdapter.UpsertRelationship(context.Background(), graphstore.GraphRelationship{
				ID:          relPayload.ID,
				RelType:     relPayload.RelType,
				FromNodeID:  relPayload.FromNodeID,
				ToNodeID:    relPayload.ToNodeID,
				DomainID:    relPayload.DomainID,
				SyncVersion: relPayload.SyncVersion,
				Properties:  cloneMap(relPayload.Properties),
			}); err != nil {
				return report, err
			}
		}
		report.RebuiltRelationships++
	}
	telemetry.RecordOrphanCounts(0, 0)
	return report, nil
}

func (r *Runtime) PurgeOrphans(ctx context.Context, tenantID string) (integrity.RepairReport, error) {
	_ = ctx
	report := integrity.RepairReport{ScannedAt: time.Now().UTC()}
	scan, err := r.ScanOrphans(context.Background(), tenantID)
	if err != nil {
		return report, err
	}
	if len(scan.RelationshipOrphans) > 0 {
		ids := make([]string, 0, len(scan.RelationshipOrphans))
		for _, item := range scan.RelationshipOrphans {
			ids = append(ids, item.RelationshipID)
			if r.graphAdapter != nil {
				_ = r.graphAdapter.DeleteRelationship(context.Background(), item.RelationshipID)
			}
			delete(r.graph.Rels, item.RelationshipID)
		}
		deleted, err := r.store.SoftDeleteRelationshipsWithOutbox(context.Background(), ids, time.Now().UTC())
		if err != nil {
			return report, err
		}
		report.PurgedRelationships = len(deleted)
	}
	if len(scan.VectorOrphans) > 0 {
		for _, item := range scan.VectorOrphans {
			if r.vectorAdapter != nil {
				if err := r.vectorAdapter.Delete(context.Background(), item.NodeID); err != nil {
					return report, err
				}
			}
			if r.ftsAdapter != nil {
				_ = r.ftsAdapter.Delete(context.Background(), item.NodeID)
			}
			if r.graphAdapter != nil {
				_ = r.graphAdapter.DeleteNode(context.Background(), item.NodeID)
			}
			delete(r.vector.Documents, item.NodeID)
			delete(r.graph.Nodes, item.NodeID)
			report.PurgedVectorDocs++
		}
	}
	telemetry.RecordOrphanCounts(0, 0)
	return report, nil
}

func (r *Runtime) EntitySyncStatus(entityID, entityKind string) (EntitySyncStatus, bool) {
	record, ok := r.store.GetProjectionVersion(entityID, entityKind)
	if !ok {
		return EntitySyncStatus{}, false
	}
	eventByID := func(id string) (write.OutboxEvent, bool) {
		return r.store.GetOutboxEventByID(id)
	}
	graphProjectionReady, graphProjectionHeadVersion, graphProjectionHeadVersionID := r.graphProjectionReady(entityID, entityKind, record)
	graphVersion := maxInt64(record.GraphVersion, graphProjectionHeadVersion)
	if r.graphAdapter != nil {
		if actualVersion, err := r.graphAdapter.ReadSyncVersion(context.Background(), entityID); err == nil && actualVersion > 0 {
			graphVersion = maxInt64(graphVersion, actualVersion)
		}
	}
	graphLagClass := classifyGraphLag(graphVersion, record.SourceVersion, record.SourceEventID, record.LastGraphSyncedAt, eventByID, r.maxRetries, r.lagToleranceWindow, graphProjectionReady)
	vectorVersion := record.VectorVersion
	vectorLagClass := classifyLag(record.VectorVersion, record.SourceVersion, record.SourceEventID, record.LastVectorSyncedAt, eventByID, r.maxRetries, r.lagToleranceWindow)
	if r.vectorAdapter != nil {
		if actualVersion, err := r.vectorAdapter.ReadSyncVersion(context.Background(), entityID); err == nil && actualVersion > 0 {
			vectorVersion = actualVersion
			vectorLagClass = classifyLag(actualVersion, record.SourceVersion, record.SourceEventID, record.LastVectorSyncedAt, eventByID, r.maxRetries, r.lagToleranceWindow)
		}
	}
	status := EntitySyncStatus{
		EntityID:                     record.EntityID,
		EntityKind:                   record.EntityKind,
		SourceVersion:                record.SourceVersion,
		GraphVersion:                 graphVersion,
		GraphLagClass:                graphLagClass,
		GraphProjectionReady:         graphProjectionReady,
		GraphProjectionHeadVersion:   graphProjectionHeadVersion,
		GraphProjectionHeadVersionID: graphProjectionHeadVersionID,
		LastGraphSyncedAt:            record.LastGraphSyncedAt,
		VectorVersion:                vectorVersion,
		VectorLagClass:               vectorLagClass,
		LastVectorSyncedAt:           record.LastVectorSyncedAt,
	}
	if entityKind == "kg_relationship" && record.VectorBackend == "" && record.VectorVersion == 0 {
		status.VectorLagClass = SyncLagClassSynced
	}
	return status, true
}

func (r *Runtime) graphProjectionReady(entityID, entityKind string, record write.ProjectionVersionRecord) (bool, int64, string) {
	if r.store == nil {
		return false, 0, ""
	}
	headVersion := int64(0)
	headVersionID := ""
	expectedVersion := record.SourceVersion
	if expectedVersion <= 0 {
		expectedVersion = record.GraphVersion
	}
	if record.SourceEventID != "" {
		if event, ok := r.store.GetOutboxEventByID(record.SourceEventID); ok {
			identifierID, _, versionNumber, ok := r.graphVersionMetadata(event)
			if ok {
				if head, ok := r.store.GetGraphProjectionHead(identifierID, "graph", ""); ok {
					headVersion = head.AppliedVersionNumber
					headVersionID = head.AppliedVersionID
					if headVersion >= versionNumber {
						if r.graphAdapter == nil {
							return true, headVersion, headVersionID
						}
						return r.graphEntityQueryable(entityID, entityKind, expectedVersion), headVersion, headVersionID
					}
					return false, headVersion, headVersionID
				}
			}
		}
	}
	return false, headVersion, headVersionID
}

func (r *Runtime) graphEntityQueryable(entityID, entityKind string, expectedVersion int64) bool {
	if r.graphAdapter == nil {
		return false
	}
	switch entityKind {
	case "kg_relationship":
		rels, err := r.graphAdapter.ListRelationships(context.Background())
		if err != nil {
			return false
		}
		for _, rel := range rels {
			if rel.ID == entityID && rel.SyncVersion >= expectedVersion {
				return true
			}
		}
	default:
		nodes, err := r.graphAdapter.ListNodes(context.Background())
		if err != nil {
			return false
		}
		for _, node := range nodes {
			if node.ID == entityID && !node.IsDeleted && node.SyncVersion >= expectedVersion {
				return true
			}
		}
	}
	return false
}

func classifyGraphLag(replicaVersion, sourceVersion int64, sourceEventID string, lastSyncedAt time.Time, getEvent func(id string) (write.OutboxEvent, bool), maxRetries int, lagToleranceWindow time.Duration, projectionReady bool) SyncLagClass {
	effectiveReplicaVersion := replicaVersion
	if !projectionReady && replicaVersion == sourceVersion && sourceVersion > 0 {
		effectiveReplicaVersion = sourceVersion - 1
	}
	return classifyLag(effectiveReplicaVersion, sourceVersion, sourceEventID, lastSyncedAt, getEvent, maxRetries, lagToleranceWindow)
}

func (r *Runtime) handleEvent(event write.OutboxEvent) error {
	if r.isGraphVersionEvent(event) {
		return r.handleGraphVersionEvent(event)
	}
	switch event.EventType {
	case "NODE_UPSERTED":
		nodeID, _ := event.Payload["node_id"].(string)
		r.logEventf(event, "handle node upsert event=%s node_id=%s", event.ID, nodeID)
		node, ok := r.fetchNodeForEvent(context.Background(), event, nodeID)
		if !ok {
			r.logEventf(event, "projection skip stale event=%s node_id=%s event_type=NODE_UPSERTED: node no longer exists", event.ID, nodeID)
			return nil
		}
		graphSynced, vectorSynced, err := r.projectNode(node)
		r.updateProjectionVersion(event, node.DomainVersion, node.ID, node.CreatedAt, int64(node.DomainVersion), int64(node.DomainVersion), graphSynced, vectorSynced)
		r.advanceProjectionHeadsForEvent(event, graphSynced, vectorSynced)
		if err != nil {
			return fmt.Errorf("handle node upsert event=%s node_id=%s: %w", event.ID, nodeID, err)
		}
		return r.applyStatusCascade(node.DomainID, node.ID)
	case "NODE_DELETED":
		nodeID, _ := event.Payload["node_id"].(string)
		r.logEventf(event, "handle node delete event=%s node_id=%s", event.ID, nodeID)
		node, ok := r.fetchNodeForEvent(context.Background(), event, nodeID)
		if !ok {
			r.logEventf(event, "projection skip stale event=%s node_id=%s event_type=NODE_DELETED: node no longer exists", event.ID, nodeID)
			return nil
		}
		node.IsDeleted = true
		if r.graphAdapter != nil {
			_ = r.graphAdapter.DeleteNode(context.Background(), nodeID)
		}
		if r.vectorAdapter != nil {
			_ = r.vectorAdapter.Delete(context.Background(), nodeID)
		}
		if r.ftsAdapter != nil {
			_ = r.ftsAdapter.Delete(context.Background(), nodeID)
		}
		graphSynced, vectorSynced, err := r.projectNode(node)
		r.updateProjectionVersion(event, node.DomainVersion, node.ID, node.UpdatedAt, int64(node.DomainVersion), int64(node.DomainVersion), graphSynced, vectorSynced)
		r.advanceProjectionHeadsForEvent(event, graphSynced, vectorSynced)
		if err != nil {
			return fmt.Errorf("handle node delete event=%s node_id=%s: %w", event.ID, nodeID, err)
		}
		return nil
	case "RELATIONSHIP_UPSERTED", "RELATIONSHIP_DELETED":
		relID, _ := event.Payload["relationship_id"].(string)
		r.logEventf(event, "handle relationship event=%s event_type=%s relationship_id=%s", event.ID, event.EventType, relID)
		rel, ok := r.fetchRelationshipForEvent(context.Background(), event, relID)
		if !ok {
			r.logEventf(event, "projection skip stale event=%s relationship_id=%s event_type=%s: relationship no longer exists", event.ID, relID, event.EventType)
			return nil
		}
		sourceVersion := int64(rel.DomainVersion)
		relPayload := GraphRelationship{
			ID:          rel.ID,
			RelType:     rel.RelType,
			FromNodeID:  rel.FromNodeID,
			ToNodeID:    rel.ToNodeID,
			DomainID:    rel.DomainID,
			SyncVersion: sourceVersion,
			Properties:  cloneMap(rel.Properties),
		}
		var graphErr error
		graphSynced := true
		if r.graphAdapter != nil {
			if event.EventType == "RELATIONSHIP_DELETED" {
				if err := r.graphAdapter.DeleteRelationship(context.Background(), relID); err != nil {
					graphSynced = false
					graphErr = err
				}
			} else {
				if err := r.graphAdapter.UpsertRelationship(context.Background(), graphstore.GraphRelationship{
					ID:          relPayload.ID,
					RelType:     relPayload.RelType,
					FromNodeID:  relPayload.FromNodeID,
					ToNodeID:    relPayload.ToNodeID,
					DomainID:    relPayload.DomainID,
					SyncVersion: sourceVersion,
					Properties:  cloneMap(relPayload.Properties),
				}); err != nil {
					graphSynced = false
					graphErr = err
				}
			}
		}
		r.graph.Rels[rel.ID] = relPayload
		r.graph.Rels[rel.ID].Properties["_kg_sync_version"] = sourceVersion
		r.updateProjectionVersion(event, rel.DomainVersion, rel.ID, rel.CreatedAt, int64(rel.DomainVersion), 0, graphSynced, false)
		r.advanceProjectionHeadsForEvent(event, graphSynced, false)
		if graphErr != nil {
			return fmt.Errorf("handle relationship event=%s relationship_id=%s: %w", event.ID, relID, graphErr)
		}
		return r.applyStatusCascade(rel.DomainID, rel.FromNodeID)
	case "ACCESS_GRANT_CHANGED":
		r.logEventf(event, "handle access grant change event=%s payload=%v", event.ID, event.Payload)
		return r.handleAccessGrantChanged(event.Payload)
	default:
		r.logEventf(event, "skip unsupported event=%s type=%s aggregate=%s", event.ID, event.EventType, event.AggregateType)
		return nil
	}
}

func (r *Runtime) fetchNodeForEvent(ctx context.Context, event write.OutboxEvent, nodeID string) (write.NodeRecord, bool) {
	if nodeID == "" {
		return write.NodeRecord{}, false
	}
	return withEventIdentityNodeRead(ctx, r.store, r.sessionManager, event, nodeID)
}

func (r *Runtime) fetchRelationshipForEvent(ctx context.Context, event write.OutboxEvent, relID string) (write.RelationshipRecord, bool) {
	if relID == "" {
		return write.RelationshipRecord{}, false
	}
	return withEventIdentityRelationshipRead(ctx, r.store, r.sessionManager, event, relID)
}

type txScopedWorkerRepository interface {
	WithTx(*sql.Tx) write.Repository
}

func withEventIdentityNodeRead(ctx context.Context, store Repository, manager session.Manager, event write.OutboxEvent, nodeID string) (write.NodeRecord, bool) {
	if manager == nil {
		return store.GetNodeByID(nodeID)
	}
	tenantID, _ := event.Payload["owner_tenant_id"].(string)
	appID, _ := event.Payload["owner_app_id"].(string)
	if tenantID == "" || appID == "" {
		return store.GetNodeByID(nodeID)
	}

	var result write.NodeRecord
	var found bool
	_, err := manager.Within(ctx, session.WriteIdentity{
		TenantID: tenantID,
		AppID:    appID,
	}, func(scope session.SessionScope) error {
		repo := store
		if scoped, ok := any(store).(txScopedWorkerRepository); ok && scope.Tx != nil {
			if typed, ok := any(scoped.WithTx(scope.Tx)).(Repository); ok {
				repo = typed
			}
		}
		result, found = repo.GetNodeByID(nodeID)
		return nil
	})
	if err != nil {
		return write.NodeRecord{}, false
	}
	return result, found
}

func withEventIdentityRelationshipRead(ctx context.Context, store Repository, manager session.Manager, event write.OutboxEvent, relID string) (write.RelationshipRecord, bool) {
	if manager == nil {
		return store.GetRelationshipByID(relID)
	}
	tenantID, _ := event.Payload["owner_tenant_id"].(string)
	appID, _ := event.Payload["owner_app_id"].(string)
	if tenantID == "" || appID == "" {
		return store.GetRelationshipByID(relID)
	}

	var result write.RelationshipRecord
	var found bool
	_, err := manager.Within(ctx, session.WriteIdentity{
		TenantID: tenantID,
		AppID:    appID,
	}, func(scope session.SessionScope) error {
		repo := store
		if scoped, ok := any(store).(txScopedWorkerRepository); ok && scope.Tx != nil {
			if typed, ok := any(scoped.WithTx(scope.Tx)).(Repository); ok {
				repo = typed
			}
		}
		result, found = repo.GetRelationshipByID(relID)
		return nil
	})
	if err != nil {
		return write.RelationshipRecord{}, false
	}
	return result, found
}

func (r *Runtime) refreshStatusCascadesLocked() {
	if r == nil || r.ontology == nil {
		return
	}
	seen := make(map[string]struct{}, len(r.graph.Nodes))
	for _, node := range r.graph.Nodes {
		key := node.DomainID + ":" + node.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		_ = r.applyStatusCascade(node.DomainID, node.ID)
	}
}

func (r *Runtime) projectNode(node write.NodeRecord) (bool, bool, error) {
	r.logf(
		"project node start node_id=%s node_type=%s domain=%s tenant=%s app=%s version=%d deleted=%t props=%d",
		node.ID,
		node.NodeType,
		node.DomainID,
		node.OwnerTenantID,
		node.OwnerAppID,
		node.DomainVersion,
		node.IsDeleted,
		len(node.Properties),
	)
	if node.IsDeleted {
		if r.graphAdapter != nil {
			if err := r.graphAdapter.DeleteNode(context.Background(), node.ID); err != nil {
				return false, false, err
			}
		}
		if r.vectorAdapter != nil {
			if err := r.vectorAdapter.Delete(context.Background(), node.ID); err != nil {
				return false, false, err
			}
		}
		if r.ftsAdapter != nil {
			if err := r.ftsAdapter.Delete(context.Background(), node.ID); err != nil {
				return false, false, err
			}
		}
		delete(r.graph.Nodes, node.ID)
		delete(r.vector.Documents, node.ID)
		return true, true, nil
	}
	acl := nodeACLVisibleTo(node)
	r.graph.Nodes[node.ID] = GraphNode{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), acl...),
		Visibility:    node.Visibility,
		StatusValue:   node.StatusValue,
		IsDeleted:     node.IsDeleted,
		SyncVersion:   int64(node.DomainVersion),
		Properties:    cloneMap(node.Properties),
	}
	r.graph.Nodes[node.ID].Properties["_kg_sync_version"] = int64(node.DomainVersion)
	graphSynced := true
	var graphErr error
	if r.graphAdapter != nil {
		r.logf(
			"project node graph upsert start node_id=%s node_type=%s domain=%s version=%d",
			node.ID,
			node.NodeType,
			node.DomainID,
			node.DomainVersion,
		)
		if err := r.graphAdapter.UpsertNode(context.Background(), graphstore.GraphNode{
			ID:            node.ID,
			NodeType:      node.NodeType,
			DomainID:      node.DomainID,
			OwnerTenantID: node.OwnerTenantID,
			OwnerAppID:    node.OwnerAppID,
			ACLVisibleTo:  append([]string(nil), acl...),
			Visibility:    node.Visibility,
			StatusValue:   node.StatusValue,
			IsDeleted:     node.IsDeleted,
			SyncVersion:   int64(node.DomainVersion),
			Properties:    cloneMap(node.Properties),
			CreatedAt:     node.CreatedAt,
			UpdatedAt:     node.UpdatedAt,
		}); err != nil {
			graphSynced = false
			graphErr = fmt.Errorf(
				"graph upsert node_id=%s node_type=%s domain=%s version=%d: %w",
				node.ID,
				node.NodeType,
				node.DomainID,
				node.DomainVersion,
				err,
			)
			r.logf("project node graph upsert failed node_id=%s err=%v", node.ID, graphErr)
		} else {
			r.logf("project node graph upsert done node_id=%s", node.ID)
		}
	}
	cfg, err := r.ontology.GetStatusFieldConfig(node.DomainID)
	if err != nil {
		cfg = nil
	}
	resolvedProfile := ontology.ResolvedSearchProfile{}
	if r.profiles != nil {
		if profile, err := r.profiles.Resolve(node.DomainID, node.OwnerTenantID, node.OwnerAppID); err == nil {
			resolvedProfile = profile
		}
	}
	text := searchprofile.BuildEmbeddingText(node, resolvedProfile)
	embeddingProvider := r.embedding.RouteContext(node.OwnerTenantID, node.DomainID)
	if embeddingProvider == nil {
		embeddingProvider = vector.NewDeterministicProvider(8)
	}
	r.logf(
		"project node embedding start node_id=%s provider=%T model=%s text_len=%d preview=%q",
		node.ID,
		embeddingProvider,
		embeddingProvider.ModelID(),
		len(text),
		textPreview(text, 160),
	)
	embedding, err := embeddingProvider.Embed(context.Background(), text)
	if err != nil {
		return graphSynced, false, fmt.Errorf(
			"embed node_id=%s node_type=%s domain=%s text_len=%d provider=%T model=%s: %w",
			node.ID,
			node.NodeType,
			node.DomainID,
			len(text),
			embeddingProvider,
			embeddingProvider.ModelID(),
			err,
		)
	}
	r.logf("project node embedding done node_id=%s dims=%d", node.ID, len(embedding))
	doc := VectorDocument{
		NodeID:        node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), acl...),
		IsDeleted:     node.IsDeleted,
		StatusValue:   node.StatusValue,
		SyncVersion:   int64(node.DomainVersion),
		DomainProps:   cloneMap(node.Properties),
		Embedding:     embedding,
	}
	if cfg != nil && cfg.AuthorityFieldName != "" && len(cfg.AuthorityValuesMap) > 0 {
		if raw, ok := node.Properties[cfg.AuthorityFieldName]; ok {
			if score, ok := cfg.AuthorityValuesMap[fmt.Sprintf("%v", raw)]; ok {
				doc.AuthorityScore = &score
			}
		}
	}
	vectorSynced := true
	var vectorErr error
	if r.vectorAdapter != nil {
		r.logf(
			"project node vector upsert start node_id=%s node_type=%s domain=%s dims=%d version=%d",
			node.ID,
			node.NodeType,
			node.DomainID,
			len(doc.Embedding),
			node.DomainVersion,
		)
		if err := r.vectorAdapter.Upsert(context.Background(), vectorstore.VectorDocument{
			NodeID:         doc.NodeID,
			NodeType:       doc.NodeType,
			DomainID:       doc.DomainID,
			OwnerTenantID:  doc.OwnerTenantID,
			OwnerAppID:     doc.OwnerAppID,
			ACLVisibleTo:   append([]string(nil), doc.ACLVisibleTo...),
			IsDeleted:      doc.IsDeleted,
			StatusValue:    doc.StatusValue,
			AuthorityScore: doc.AuthorityScore,
			SyncVersion:    doc.SyncVersion,
			DomainProps:    cloneMap(doc.DomainProps),
			Embedding:      append([]float64(nil), doc.Embedding...),
		}); err != nil {
			vectorSynced = false
			vectorErr = fmt.Errorf(
				"vector upsert node_id=%s node_type=%s domain=%s dims=%d version=%d: %w",
				node.ID,
				node.NodeType,
				node.DomainID,
				len(doc.Embedding),
				node.DomainVersion,
				err,
			)
			r.logf("project node vector upsert failed node_id=%s err=%v", node.ID, vectorErr)
		} else {
			r.logf("project node vector upsert done node_id=%s", node.ID)
		}
	}
	if vectorSynced {
		r.vector.Documents[node.ID] = doc
		r.vector.Documents[node.ID].DomainProps["_kg_sync_version"] = int64(node.DomainVersion)
	}
	if !node.IsDeleted && r.ftsAdapter != nil {
		r.logf("project node fts index start node_id=%s", node.ID)
		if err := r.ftsAdapter.Index(context.Background(), buildFTSDocument(node, acl, doc.AuthorityScore)); err != nil {
			return graphSynced, vectorSynced, fmt.Errorf(
				"fts index node_id=%s node_type=%s domain=%s: %w",
				node.ID,
				node.NodeType,
				node.DomainID,
				err,
			)
		}
		r.logf("project node fts index done node_id=%s", node.ID)
	}
	if vectorErr != nil {
		return graphSynced, vectorSynced, vectorErr
	}
	if graphErr != nil {
		return graphSynced, vectorSynced, graphErr
	}
	r.logf("project node done node_id=%s graph_synced=%t vector_synced=%t", node.ID, graphSynced, vectorSynced)
	return graphSynced, vectorSynced, nil
}

func (r *Runtime) applyStatusCascade(domainID, fromNodeID string) error {
	cfg, err := r.ontology.GetStatusFieldConfig(domainID)
	if err != nil || cfg == nil || len(cfg.CascadeRules) == 0 {
		return err
	}
	source, ok := r.graph.Nodes[fromNodeID]
	if !ok {
		return nil
	}
	for _, rule := range cfg.CascadeRules {
		for id, node := range r.graph.Nodes {
			if node.DomainID != domainID || node.NodeType != rule.ToNodeType {
				continue
			}
			if nodeHasIncomingRel(id, rule.ViaRel, fromNodeID, r.graph.Rels) {
				node.StatusValue = source.StatusValue
				r.graph.Nodes[id] = node
				if doc, ok := r.vector.Documents[id]; ok {
					doc.StatusValue = source.StatusValue
					r.vector.Documents[id] = doc
				}
			}
		}
	}
	return nil
}

func (r *Runtime) handleAccessGrantChanged(payload map[string]any) error {
	grantorTenantID, _ := payload["grantor_tenant_id"].(string)
	granteeTenantID, _ := payload["grantee_tenant_id"].(string)
	granteeAppID, _ := payload["grantee_app_id"].(string)
	scopeType, _ := payload["scope_type"].(string)
	scopeValue, _ := payload["scope_value"].(string)
	permission, _ := payload["permission"].(string)
	status, _ := payload["status"].(string)
	if grantorTenantID == "" || granteeTenantID == "" || granteeAppID == "" {
		return nil
	}
	token := granteeTenantID + ":" + granteeAppID
	for id, node := range r.graph.Nodes {
		if node.OwnerTenantID != grantorTenantID {
			continue
		}
		if scopeType == "domain" && scopeValue != "" && node.DomainID != scopeValue {
			continue
		}
		if status == "revoked" || status == "deleted" {
			node.ACLVisibleTo = removeString(node.ACLVisibleTo, token)
		} else {
			node.ACLVisibleTo = appendUnique(node.ACLVisibleTo, token)
		}
		r.graph.Nodes[id] = node
		if doc, ok := r.vector.Documents[id]; ok {
			if status == "revoked" || status == "deleted" {
				doc.ACLVisibleTo = removeString(doc.ACLVisibleTo, token)
			} else {
				doc.ACLVisibleTo = appendUnique(doc.ACLVisibleTo, token)
			}
			r.vector.Documents[id] = doc
		}
	}
	_ = permission
	if r.cache != nil {
		r.cache.Delete("acl:" + granteeTenantID + ":" + granteeAppID)
	}
	return nil
}

func (r *Runtime) updateProjectionVersion(event write.OutboxEvent, sourceVersion int, entityID string, sourceUpdatedAt time.Time, graphVersion, vectorVersion int64, graphSynced, vectorSynced bool) {
	writer, ok := any(r.store).(projectionVersionWriter)
	if !ok || writer == nil || sourceVersion == 0 {
		return
	}
	record, hasExisting := r.store.GetProjectionVersion(entityID, event.AggregateType)
	if !hasExisting {
		record = write.ProjectionVersionRecord{}
	}
	record.EntityID = entityID
	record.EntityKind = "kg_node"
	if event.AggregateType == "kg_relationship" {
		record.EntityKind = "kg_relationship"
	}
	record.SourceVersion = int64(sourceVersion)
	record.SourceEventID = event.ID
	record.SourceUpdatedAt = sourceUpdatedAt
	record.GraphBackend = "graph"
	if graphSynced {
		record.GraphVersion = graphVersion
	}
	record.VectorBackend = "vector"
	now := time.Now().UTC()
	if graphSynced {
		record.LastGraphSyncedAt = now
	}
	if vectorSynced {
		record.VectorVersion = vectorVersion
	}
	if vectorSynced {
		record.LastVectorSyncedAt = now
	}
	_ = writer.UpsertProjectionVersion(context.Background(), record)
}

func (r *Runtime) advanceGraphProjectionHead(ctx context.Context, identifierID, backendKind string, versionID string, versionNumber int64) {
	if r.store == nil || identifierID == "" || backendKind == "" || versionNumber <= 0 {
		return
	}
	_ = r.store.UpsertGraphProjectionHead(ctx, write.GraphProjectionHeadRecord{
		IdentifierID:         identifierID,
		BackendKind:          backendKind,
		BackendName:          "",
		AppliedVersionID:     versionID,
		AppliedVersionNumber: versionNumber,
		UpdatedAt:            time.Now().UTC(),
	})
}

func (r *Runtime) advanceProjectionHeadsForEvent(event write.OutboxEvent, graphSynced, vectorSynced bool) {
	identifierID, versionID, versionNumber, ok := r.graphVersionMetadata(event)
	if !ok {
		return
	}
	if graphSynced {
		r.advanceGraphProjectionHead(context.Background(), identifierID, "graph", versionID, versionNumber)
	}
	if vectorSynced {
		r.advanceGraphProjectionHead(context.Background(), identifierID, "vector", versionID, versionNumber)
	}
}

func (r *Runtime) isGraphVersionEvent(event write.OutboxEvent) bool {
	return event.AggregateType == "kg_graph_version" || event.EventType == "GRAPH_VERSION_SEALED"
}

func (r *Runtime) graphVersionMetadata(event write.OutboxEvent) (string, string, int64, bool) {
	identifierID, _ := event.Payload["graph_identifier_id"].(string)
	versionID, _ := event.Payload["graph_version_id"].(string)
	versionNumber := int64(0)
	switch raw := event.Payload["graph_version_number"].(type) {
	case int64:
		versionNumber = raw
	case int:
		versionNumber = int64(raw)
	case float64:
		versionNumber = int64(raw)
	case json.Number:
		if parsed, err := raw.Int64(); err == nil {
			versionNumber = parsed
		}
	}
	if identifierID == "" || versionID == "" || versionNumber <= 0 {
		return "", "", 0, false
	}
	return identifierID, versionID, versionNumber, true
}

func (r *Runtime) handleGraphVersionEvent(event write.OutboxEvent) error {
	identifierID, versionID, versionNumber, ok := r.graphVersionMetadata(event)
	if !ok {
		return nil
	}
	entities := r.store.GetGraphVersionEntities(versionID)
	if len(entities) == 0 {
		r.advanceGraphProjectionHead(context.Background(), identifierID, "graph", versionID, versionNumber)
		r.advanceGraphProjectionHead(context.Background(), identifierID, "vector", versionID, versionNumber)
		return nil
	}
	nodeEntities := make([]write.GraphVersionEntityRecord, 0, len(entities))
	relEntities := make([]write.GraphVersionEntityRecord, 0, len(entities))
	nodeIDs := make([]string, 0, len(entities))
	relIDs := make([]string, 0, len(entities))
	for _, entity := range entities {
		switch entity.EntityKind {
		case "node":
			nodeEntities = append(nodeEntities, entity)
			nodeIDs = append(nodeIDs, entity.EntityID)
		case "relationship", "embeddable_relationship":
			relEntities = append(relEntities, entity)
			relIDs = append(relIDs, entity.EntityID)
		}
	}
	nodeSources := r.store.GetNodesByIDs(nodeIDs)
	relSources := r.store.GetRelationshipsByIDs(relIDs)
	nodeUpserts := make([]nodeProjectionWork, 0, len(nodeEntities))
	nodeDeletes := make([]nodeProjectionWork, 0, len(nodeEntities))
	relUpserts := make([]relationshipProjectionWork, 0, len(relEntities))
	relDeletes := make([]relationshipProjectionWork, 0, len(relEntities))
	results := make([]projectionUnitResult, 0, len(entities))
	for _, entity := range entities {
		latestEvent := event
		switch entity.EntityKind {
		case "node":
			if entity.ChangeKind == "DELETE" {
				latestEvent.EventType = "NODE_DELETED"
			} else {
				latestEvent.EventType = "NODE_UPSERTED"
			}
		case "relationship", "embeddable_relationship":
			if entity.ChangeKind == "DELETE" {
				latestEvent.EventType = "RELATIONSHIP_DELETED"
			} else {
				latestEvent.EventType = "RELATIONSHIP_UPSERTED"
			}
		}
		unit := projectionWorkUnit{
			Events:        []write.OutboxEvent{event},
			Latest:        latestEvent,
			EntityKind:    "kg_node",
			EntityID:      entity.EntityID,
			SourceVersion: 0,
		}
		switch entity.EntityKind {
		case "node":
			node, found := nodeSources[entity.EntityID]
			if !found {
				results = append(results, projectionUnitResult{Unit: unit, GraphSynced: true, VectorSynced: true})
				continue
			}
			unit.EntityKind = "kg_node"
			work, err := r.buildNodeProjectionWork(len(results), unit, node)
			if err != nil {
				results = append(results, projectionUnitResult{Unit: unit, Err: err})
				continue
			}
			results = append(results, projectionUnitResult{Unit: unit, SourceNode: node, GraphNode: work.graphNode, VectorDoc: work.vectorDoc})
			if entity.ChangeKind == "DELETE" || node.IsDeleted {
				nodeDeletes = append(nodeDeletes, work)
			} else {
				nodeUpserts = append(nodeUpserts, work)
			}
		case "relationship", "embeddable_relationship":
			rel, found := relSources[entity.EntityID]
			if !found {
				results = append(results, projectionUnitResult{Unit: unit, GraphSynced: true, VectorSynced: true})
				continue
			}
			unit.EntityKind = "kg_relationship"
			work := r.buildRelationshipProjectionWork(len(results), unit, rel)
			results = append(results, projectionUnitResult{Unit: unit, SourceRel: rel, GraphRel: work.graphRel})
			if entity.ChangeKind == "DELETE" || rel.IsDeleted {
				relDeletes = append(relDeletes, work)
			} else {
				relUpserts = append(relUpserts, work)
			}
		}
	}
	r.applyNodeProjectionWork(nodeUpserts, false, results)
	r.applyNodeProjectionWork(nodeDeletes, true, results)
	r.applyRelationshipProjectionWork(relUpserts, false, results)
	r.applyRelationshipProjectionWork(relDeletes, true, results)
	for _, result := range results {
		if result.Err != nil {
			return result.Err
		}
		r.commitProjectionResultLocked(result, &WorkerReport{}, time.Now().UTC())
	}
	r.advanceGraphProjectionHead(context.Background(), identifierID, "graph", versionID, versionNumber)
	r.advanceGraphProjectionHead(context.Background(), identifierID, "vector", versionID, versionNumber)
	return nil
}

func filterGraphVersionVectorEntities(entities []write.GraphVersionEntityRecord) []write.GraphVersionEntityRecord {
	filtered := make([]write.GraphVersionEntityRecord, 0, len(entities))
	for _, entity := range entities {
		if entity.EntityKind == "node" || entity.EntityKind == "embeddable_relationship" {
			filtered = append(filtered, entity)
		}
	}
	return filtered
}

func nodeACLVisibleTo(node write.NodeRecord) []string {
	if len(node.ACLVisibleTo) > 0 {
		return append([]string(nil), node.ACLVisibleTo...)
	}
	return []string{node.OwnerTenantID + ":" + node.OwnerAppID}
}

func nodeHasIncomingRel(nodeID, relType, fromNodeID string, rels map[string]GraphRelationship) bool {
	for _, rel := range rels {
		if rel.RelType == relType && rel.ToNodeID == nodeID && rel.FromNodeID == fromNodeID {
			return true
		}
	}
	return false
}

func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func loadNodeSnapshot(store Repository) map[string]write.NodeRecord {
	result := map[string]write.NodeRecord{}
	var afterID string
	for {
		batch := store.ListNodesBatch(afterID, 500)
		if len(batch) == 0 {
			return result
		}
		for _, node := range batch {
			result[node.ID] = node
		}
		afterID = batch[len(batch)-1].ID
	}
}

func loadRelationshipSnapshot(store Repository) map[string]write.RelationshipRecord {
	result := map[string]write.RelationshipRecord{}
	var afterID string
	for {
		batch := store.ListRelationshipsBatch(afterID, 500)
		if len(batch) == 0 {
			return result
		}
		for _, rel := range batch {
			result[rel.ID] = rel
		}
		afterID = batch[len(batch)-1].ID
	}
}

func loadProjectionVersionSnapshot(store Repository) []write.ProjectionVersionRecord {
	result := make([]write.ProjectionVersionRecord, 0)
	var afterKind, afterID string
	for {
		batch := store.ListProjectionVersionsBatch(afterKind, afterID, 500)
		if len(batch) == 0 {
			return result
		}
		result = append(result, batch...)
		afterKind = batch[len(batch)-1].EntityKind
		afterID = batch[len(batch)-1].EntityID
	}
}

func textPreview(text string, max int) string {
	cleaned := strings.Join(strings.Fields(text), " ")
	if max <= 0 || len(cleaned) <= max {
		return cleaned
	}
	return cleaned[:max] + "..."
}

func classifyLag(replicaVersion, sourceVersion int64, sourceEventID string, lastSyncedAt time.Time, getEvent func(id string) (write.OutboxEvent, bool), maxRetries int, lagToleranceWindow time.Duration) SyncLagClass {
	if replicaVersion == sourceVersion {
		return SyncLagClassSynced
	}
	if getEvent != nil {
		if event, ok := getEvent(sourceEventID); ok {
			if event.RetryCount >= maxRetries {
				return SyncLagClassStuck
			}
			if time.Since(event.CreatedAt) > lagToleranceWindow {
				return SyncLagClassLagging
			}
			return SyncLagClassInFlight
		}
	}
	if lastSyncedAt.IsZero() {
		return SyncLagClassStuck
	}
	if time.Since(lastSyncedAt) <= lagToleranceWindow {
		return SyncLagClassInFlight
	}
	return SyncLagClassStuck
}

func buildFTSDocument(node write.NodeRecord, acl []string, authorityScore *int) fts.FTSDocument {
	fields := map[string]string{
		"id":           node.ID,
		"node_type":    node.NodeType,
		"domain_id":    node.DomainID,
		"external_ref": node.ExternalRef,
		"status_value": node.StatusValue,
	}
	for key, value := range node.Properties {
		fields[key] = fmt.Sprint(value)
	}
	return fts.FTSDocument{
		NodeID:         node.ID,
		NodeType:       node.NodeType,
		DomainID:       node.DomainID,
		OwnerTenantID:  node.OwnerTenantID,
		OwnerAppID:     node.OwnerAppID,
		ACLVisibleTo:   append([]string(nil), acl...),
		IsDeleted:      node.IsDeleted,
		StatusValue:    node.StatusValue,
		AuthorityScore: authorityScore,
		DomainProps:    cloneMap(node.Properties),
		Fields:         fields,
		CreatedAt:      node.CreatedAt,
	}
}

func appendUnique(values []string, item string) []string {
	for _, value := range values {
		if value == item {
			return values
		}
	}
	return append(values, item)
}

func removeString(values []string, item string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == item {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func stripSyncVersionMetadata(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(values))
	for key, value := range values {
		if key == "_kg_sync_version" {
			continue
		}
		result[key] = value
	}
	return result
}

type runtimeGraphAdapter struct {
	graph *GraphStore
}

func (a runtimeGraphAdapter) UpsertNode(_ context.Context, node graphstore.GraphNode) error {
	if a.graph == nil {
		return nil
	}
	a.graph.Nodes[node.ID] = GraphNode{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		ACLVisibleTo:  append([]string(nil), node.ACLVisibleTo...),
		Visibility:    node.Visibility,
		StatusValue:   node.StatusValue,
		IsDeleted:     node.IsDeleted,
		SyncVersion:   node.SyncVersion,
		Properties:    cloneMap(node.Properties),
	}
	a.graph.Nodes[node.ID].Properties["_kg_sync_version"] = node.SyncVersion
	return nil
}

func (a runtimeGraphAdapter) DeleteNode(_ context.Context, nodeID string) error {
	if a.graph == nil {
		return nil
	}
	delete(a.graph.Nodes, nodeID)
	for id, rel := range a.graph.Rels {
		if rel.FromNodeID == nodeID || rel.ToNodeID == nodeID {
			delete(a.graph.Rels, id)
		}
	}
	return nil
}

func (a runtimeGraphAdapter) UpsertRelationship(_ context.Context, rel graphstore.GraphRelationship) error {
	if a.graph == nil {
		return nil
	}
	a.graph.Rels[rel.ID] = GraphRelationship{
		ID:          rel.ID,
		RelType:     rel.RelType,
		FromNodeID:  rel.FromNodeID,
		ToNodeID:    rel.ToNodeID,
		DomainID:    rel.DomainID,
		SyncVersion: rel.SyncVersion,
		Properties:  cloneMap(rel.Properties),
	}
	a.graph.Rels[rel.ID].Properties["_kg_sync_version"] = rel.SyncVersion
	return nil
}

func (a runtimeGraphAdapter) DeleteRelationship(_ context.Context, relID string) error {
	if a.graph == nil {
		return nil
	}
	delete(a.graph.Rels, relID)
	return nil
}

func (a runtimeGraphAdapter) ExecuteQuery(_ context.Context, query graphstore.GraphQuery, params map[string]any) ([]map[string]any, error) {
	if a.graph == nil {
		return nil, nil
	}
	return runtimeGraphQuery(a.graph.SnapshotNodes(), a.graph.SnapshotRelationships(), query, params), nil
}

func (a runtimeGraphAdapter) ListNodes(_ context.Context) ([]graphstore.GraphNode, error) {
	if a.graph == nil {
		return nil, nil
	}
	result := make([]graphstore.GraphNode, 0, len(a.graph.Nodes))
	for _, node := range a.graph.Nodes {
		result = append(result, graphstore.GraphNode{
			ID:            node.ID,
			NodeType:      node.NodeType,
			DomainID:      node.DomainID,
			OwnerTenantID: node.OwnerTenantID,
			OwnerAppID:    node.OwnerAppID,
			ACLVisibleTo:  append([]string(nil), node.ACLVisibleTo...),
			Visibility:    node.Visibility,
			StatusValue:   node.StatusValue,
			IsDeleted:     node.IsDeleted,
			SyncVersion:   node.SyncVersion,
			Properties:    cloneMap(node.Properties),
			CreatedAt:     node.CreatedAt,
			UpdatedAt:     node.UpdatedAt,
		})
	}
	return result, nil
}

func (a runtimeGraphAdapter) ListRelationships(_ context.Context) ([]graphstore.GraphRelationship, error) {
	if a.graph == nil {
		return nil, nil
	}
	result := make([]graphstore.GraphRelationship, 0, len(a.graph.Rels))
	for _, rel := range a.graph.Rels {
		result = append(result, graphstore.GraphRelationship{
			ID:          rel.ID,
			RelType:     rel.RelType,
			FromNodeID:  rel.FromNodeID,
			ToNodeID:    rel.ToNodeID,
			DomainID:    rel.DomainID,
			SyncVersion: rel.SyncVersion,
			Properties:  cloneMap(rel.Properties),
		})
	}
	return result, nil
}

func (a runtimeGraphAdapter) ReadSyncVersion(_ context.Context, entityID string) (int64, error) {
	if a.graph == nil {
		return 0, nil
	}
	if node, ok := a.graph.Nodes[entityID]; ok {
		return node.SyncVersion, nil
	}
	if rel, ok := a.graph.Rels[entityID]; ok {
		return rel.SyncVersion, nil
	}
	return 0, nil
}

func runtimeGraphQuery(nodes map[string]GraphNode, rels map[string]GraphRelationship, query graphstore.GraphQuery, params map[string]any) []map[string]any {
	nodeByID := map[string]GraphNode{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	visibility := visibleOwnerTokensFromAny(params[query.ACLTokensParam])
	results := []map[string]any{}
	for _, node := range nodes {
		if node.IsDeleted || node.NodeType != query.StartNodeType {
			continue
		}
		if !runtimeMatchesNode(node.Properties, query.StartMatch, params) {
			continue
		}
		if !runtimeIsVisible(node, visibility) {
			continue
		}
		payload := cloneMap(node.Properties)
		payload["id"] = node.ID
		payload["node_type"] = node.NodeType
		payload["domain_id"] = node.DomainID
		payload["_kg_sync_version"] = node.SyncVersion
		current := node
		ok := true
		for idx, hop := range query.Hops {
			next, found := runtimeNextHop(current, hop, nodeByID, rels, visibility, params)
			if !found {
				ok = false
				break
			}
			current = next
			payload[fmt.Sprintf("hop_%d", idx+1)] = current.ID
		}
		if !ok {
			continue
		}
		selected := map[string]any{}
		for _, field := range query.ReturnFields {
			selected[field] = runtimeResolveFieldValue(node, current, field)
		}
		for k, v := range payload {
			selected[k] = v
		}
		results = append(results, selected)
	}
	return results
}

func runtimeIsVisible(node GraphNode, visibility map[string]struct{}) bool {
	if len(visibility) == 0 {
		return true
	}
	for _, token := range node.ACLVisibleTo {
		if _, ok := visibility[token]; ok {
			return true
		}
	}
	return false
}

func runtimeMatchesNode(props map[string]any, match map[string]any, params map[string]any) bool {
	for key, raw := range match {
		want, ok := raw.(string)
		if ok && strings.HasPrefix(want, "$") {
			if value, exists := params[strings.TrimPrefix(want, "$")]; exists {
				want = fmt.Sprint(value)
			}
		}
		got, exists := props[key]
		if !exists || fmt.Sprint(got) != want {
			return false
		}
	}
	return true
}

func runtimeNextHop(current GraphNode, hop graphstore.GraphQueryHop, nodeByID map[string]GraphNode, relationships map[string]GraphRelationship, visibility map[string]struct{}, params map[string]any) (GraphNode, bool) {
	for _, rel := range relationships {
		if rel.RelType != hop.RelType {
			continue
		}
		var candidateID string
		switch strings.ToLower(hop.Direction) {
		case "in":
			if rel.ToNodeID != current.ID {
				continue
			}
			candidateID = rel.FromNodeID
		default:
			if rel.FromNodeID != current.ID {
				continue
			}
			candidateID = rel.ToNodeID
		}
		candidate, ok := nodeByID[candidateID]
		if !ok || candidate.IsDeleted || candidate.NodeType != hop.ToNodeType {
			continue
		}
		if !runtimeIsVisible(candidate, visibility) {
			continue
		}
		if !runtimeMatchesNode(candidate.Properties, hop.Filter, params) {
			continue
		}
		return candidate, true
	}
	return GraphNode{}, false
}

func runtimeResolveFieldValue(start, current GraphNode, field string) any {
	if strings.Contains(field, ".") {
		parts := strings.SplitN(field, ".", 2)
		switch parts[0] {
		case start.NodeType:
			return start.Properties[parts[1]]
		case current.NodeType:
			return current.Properties[parts[1]]
		}
	}
	return current.Properties[field]
}

func visibleOwnerTokensFromAny(raw any) map[string]struct{} {
	result := map[string]struct{}{}
	switch value := raw.(type) {
	case []string:
		for _, token := range value {
			result[token] = struct{}{}
		}
	case []any:
		for _, item := range value {
			if token, ok := item.(string); ok {
				result[token] = struct{}{}
			}
		}
	}
	return result
}
