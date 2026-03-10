DOC 3: KNOWLEDGE GRAPH SERVICE
Enterprise Production Architecture v3.0 (Fixed)
Version 3.1 — Gap-Fixed Edition
March 2026
1. Design Objectives
System Must Satisfy
• Strict tenant isolation (namespace per appId+tenantId)
• Snapshot + versioning (copy-on-write delta)
• Overlay graph for session-scoped temp writes
• Hybrid graph + vector search with score blending
• Role-based projection with PII masking
• Transaction-safe with multi-level locking (not coarse namespace lock)
• Version GC + compaction to prevent unbounded storage growth
• Structured overlay lifecycle (create/commit/discard/conflict)
2. Architecture Overview
API Gateway (KG Service)
    ↓
Namespace Resolver       (appId + tenantId → storage namespace)
    ↓
Request Router
    ├── Entity CRUD Handler
    ├── Edge CRUD Handler
    ├── Batch Handler        (bulk upsert/create)
    ├── Search Handler       (hybrid: vector + text + graph)
    ├── Traversal Handler    (path, subgraph, impact)
    ├── Overlay Handler      (create/commit/discard)
    ├── Version Handler      (snapshot, diff, restore)
    ├── Projection Handler   (role-based view)
    └── Analytics Handler    (coverage, clusters, traceability)
    ↓
Storage Layer
    ├── Graph DB  (Neo4j / ArangoDB)
    ├── Vector DB (Qdrant / Weaviate)
    ├── Version Store (Delta + Snapshot)
    └── Lock Manager
3. Namespace & Tenant Isolation Model
// Storage namespace
namespace = fmt.Sprintf("graph/%s/%s", appId, tenantId)
// Every entity/edge MUST carry tenant context
// API layer rejects any operation missing valid namespace
// Cross-namespace queries: forbidden (returns 403)
// Enforcement: middleware validates namespace matches JWT claims
func namespaceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ns := r.Header.Get("X-KG-Namespace")
        claims := jwtFromContext(r.Context())
        if !authorizeNamespace(claims, ns) {
            http.Error(w, "forbidden", 403); return
        }
        next.ServeHTTP(w, r.WithContext(withNamespace(r.Context(), ns)))
    })
}
4. Core Data Model
type Entity struct {
    AppID          string
    TenantID       string
    EntityID       string
    EntityType     string
    Properties     map[string]interface{}
    Embedding      []float32          // for vector search
    Confidence     float64
    SourceFile     string
    ChunkID        string
    SkillID        string
    VersionID      string
    ProvenanceType ProvenanceType     // EXTRACTED|GENERATED|MANUAL
    CreatedAt      time.Time
    UpdatedAt      time.Time
    Version        int                // optimistic lock field
    IsDeleted      bool               // soft delete
}
type Edge struct {
    AppID        string
    TenantID     string
    EdgeID       string
    FromEntityID string
    ToEntityID   string
    RelationType string
    Properties   map[string]interface{}
    Confidence   float64
    VersionID    string
    CreatedAt    time.Time
}
5. Concurrency — Multi-Level Locking (Gap #12 Fixed)
Original Gap #12
Namespace-level write lock was too coarse-grained.
100 parallel chunk extractions all blocked on a single namespace lock.
Would cause severe throughput bottleneck in production.
5.1 Locking Levels
Level
Scope
Mechanism
When Used
Node-level
Single entity
Optimistic lock (version field)
Default for all entity writes
Subgraph-level
Connected node set
Pessimistic lock set (sorted to prevent deadlock)
Writing related entities with edges
Version-level
Entire version snapshot
Shared/Exclusive lock
Snapshot creation
Namespace-level
Entire tenant graph
Exclusive lock (heavy)
Schema migration, full rebuild ONLY
5.2 Node-Level Optimistic Lock
// Write entity with optimistic lock
func (s *EntityStore) Upsert(ctx context.Context, e Entity) error {
    for attempt := 0; attempt < maxRetries; attempt++ {
        current, err := s.get(ctx, e.EntityID, e.TenantID)
        if err == ErrNotFound {
            return s.insert(ctx, e)  // new entity, no conflict
        }
        // Check version
        if current.Version != e.Version {
            // Merge conflict resolution
            e = resolveConflict(current, e)  // see conflict policy
            e.Version = current.Version       // use latest version
        }
        // Atomic update with version check
        affected := s.db.ExecCAS(
            `UPDATE entities SET properties=$1, version=version+1
             WHERE entity_id=$2 AND tenant_id=$3 AND version=$4`,
            e.Properties, e.EntityID, e.TenantID, e.Version)
        if affected == 1 { return nil }  // success
        // CAS failed: another write happened, retry
        time.Sleep(jitter(50*time.Millisecond, attempt))
    }
    return ErrMaxRetriesExceeded
}
// Subgraph lock: acquire locks in sorted order to prevent deadlock
func acquireSubgraphLock(entityIDs []string) []Lock {
    sort.Strings(entityIDs)  // consistent order
    locks := []Lock{}
    for _, id := range entityIDs {
        locks = append(locks, acquireLock(id, timeout=500*ms))
    }
    return locks
}
6. Versioning Model — Copy-on-Write Delta
type GraphVersion struct {
    AppID           string
    TenantID        string
    VersionID       string
    ParentVersionID string
    CreatedAt       time.Time
    IsSnapshot      bool
    Label           string    // optional human-readable tag
    DeltaRef        string    // pointer to VersionDelta storage
}
type VersionDelta struct {
    BaseVersionID  string
    AddedEntities  []Entity
    ModifiedEntities []EntityDiff
    RemovedEntityIDs []string
    AddedEdges     []Edge
    RemovedEdgeIDs []string
}
7. Version GC + Compaction (Gap #15 Fixed)
Original Gap #15
Copy-on-write creates unbounded version history. No GC policy defined.
Production system would accumulate versions indefinitely → storage explosion.
type RetentionPolicy struct {
    KeepNamedSnapshots  bool          // forever (or explicit TTL)
    KeepRecentDeltas    int           // default: last 50 versions
    CompactOlderThan    time.Duration // default: 7 days
    CompactThreshold    int           // compact when > N old deltas
}
// Background compaction job (runs per tenant, daily)
func (gc *VersionGC) Compact(ctx context.Context, tenantID string) error {
    policy := gc.getPolicy(tenantID)
    versions := gc.listVersions(tenantID,
        olderThan: time.Now().Add(-policy.CompactOlderThan),
        excludeSnapshots: true,
        excludeNamed: true)
    if len(versions) <= policy.CompactThreshold { return nil }
    // Merge old deltas into single snapshot
    merged := gc.mergeDeltas(versions)
    snapshotID := gc.storeAsSnapshot(tenantID, merged)
    // Re-link children of last merged version to new snapshot
    gc.relinkChildren(versions[len(versions)-1].VersionID, snapshotID)
    // Delete merged deltas
    return gc.deleteDeltas(versions)
}
// Retention summary:
// Named snapshots:  keep forever (user-tagged)
// Recent deltas:    keep last 50
// Old deltas:       compact → snapshot after 7 days
// Compacted data:   S3 cold storage for audit, local delete
8. Overlay Graph — Full Lifecycle (Gap #14 Fixed)
Original Gap #14
OverlayGraph had TempEntities and TempEdges but no write-back strategy.
No commit/discard rules. No conflict handling when base changes during overlay.
8.1 Overlay Lifecycle States
States: CREATED → ACTIVE → COMMITTED | DISCARDED
type OverlayGraph struct {
    OverlayID    string
    AppID        string
    TenantID     string
    SessionID    string
    BaseVersionID string   // version at overlay creation time
    TempEntities []Entity
    TempEdges    []Edge
    Status       OverlayStatus
    CreatedAt    time.Time
    TTL          time.Duration  // auto-discard if session expires
}
8.2 Commit Protocol
func (o *OverlayService) Commit(ctx context.Context, overlayID string,
    policy ConflictPolicy) (string, error) {
    overlay := o.load(overlayID)
    if overlay.Status != ACTIVE {
        return "", ErrOverlayNotActive
    }
    // Check for conflicts: has base version changed?
    currentBase := o.getCurrentVersion(overlay.TenantID)
    if currentBase.VersionID != overlay.BaseVersionID {
        diff := o.diffVersions(overlay.BaseVersionID, currentBase.VersionID)
        overlapping := findOverlap(diff, overlay.TempEntities)
        if len(overlapping) > 0 {
            switch policy {
            case KEEP_OVERLAY:
                // overlay wins, overwrite base changes
            case KEEP_BASE:
                // discard overlapping overlay changes
                overlay.TempEntities = removeOverlapping(overlay.TempEntities, overlapping)
            case MERGE:
                // field-level merge: newer timestamp wins per property
                overlay.TempEntities = mergeEntities(overlay.TempEntities, overlapping)
            case REQUIRE_MANUAL:
                return "", &ConflictError{Conflicts: overlapping}
            }
        }
    }
    // Create new version delta from overlay
    delta := VersionDelta{
        BaseVersionID:    currentBase.VersionID,
        AddedEntities:    overlay.TempEntities,
        AddedEdges:       overlay.TempEdges,
    }
    newVersionID := o.versionStore.CreateDelta(overlay.TenantID, delta)
    overlay.Status = COMMITTED
    o.save(overlay)
    return newVersionID, nil
}
// Discard: simply mark as DISCARDED, GC handles cleanup
func (o *OverlayService) Discard(ctx context.Context, overlayID string) error {
    overlay := o.load(overlayID)
    overlay.Status = DISCARDED
    return o.save(overlay)
}
// Session-overlay binding rules:
// task=EXTRACT+BUILD → commit (persist knowledge)
// task=QUERY only   → discard (no KG change)
// task=GENERATE     → commit with ProvenanceType=GENERATED
// session timeout   → auto-discard via TTL
9. Hybrid Search — Score Blending + Reranking (Gap #13 Fixed)
Original Gap #13
hybrid_search() was: vector_search → graph_filter. No re-ranking. No score blending.
Empty semantic results would produce empty output with no fallback.
func (s *SearchService) HybridSearch(ctx context.Context,
    req HybridSearchRequest) ([]SearchResult, error) {
    ns := namespaceFromCtx(ctx)
    // Phase 1: Candidate generation (parallel)
    var semantic, textual []SearchResult
    g, gctx := errgroup.WithContext(ctx)
    g.Go(func() error {
        embedding := s.embedder.Embed(req.Query)
        semantic, _ = s.vectorDB.Search(gctx, ns, embedding, topK=100)
        return nil
    })
    g.Go(func() error {
        textual, _ = s.textDB.BM25Search(gctx, ns, req.Query, topK=100)
        return nil
    })
    g.Wait()
    // Phase 2: Merge and deduplicate
    candidates := deduplicateByID(append(semantic, textual...))
    // Phase 3: Apply structural filters (node type, domain, confidence)
    if req.Filters != nil {
        candidates = applyFilters(candidates, req.Filters)
    }
    // Phase 4: Graph-aware reranking
    for i := range candidates {
        c := &candidates[i]
        semScore  := getSemanticScore(c.EntityID, semantic)   // 0–1
        textScore := getTextScore(c.EntityID, textual)        // 0–1
        centrality := s.graphDB.PageRank(ctx, ns, c.EntityID) // normalized 0–1
        neighborRel := s.computeNeighborRelevance(ctx, ns, c.EntityID, req.Query)
        alpha := req.Alpha  // 0=pure text, 1=pure semantic
        c.FinalScore = alpha*semScore + (1-alpha)*textScore
        c.FinalScore = 0.80*c.FinalScore + 0.15*centrality + 0.05*neighborRel
        // Boost nodes marked as EXTRACTED (more reliable than GENERATED)
        if c.ProvenanceType == EXTRACTED { c.FinalScore *= 1.10 }
    }
    // Phase 5: Sort and truncate
    sort.Slice(candidates, func(i,j int) bool {
        return candidates[i].FinalScore > candidates[j].FinalScore
    })
    if len(candidates) > req.TopK { candidates = candidates[:req.TopK] }
    // Fallback: if still empty, text-only search without filters
    if len(candidates) == 0 && req.Filters != nil {
        return s.HybridSearch(ctx, HybridSearchRequest{Query:req.Query, TopK:req.TopK})
    }
    return candidates, nil
}
10. Projection Engine — Full Filter Logic (Gap #16 Fixed)
Original Gap #16
filter_entities(entities, ontology_slice) had no definition of filter rules.
No PII masking. No confidence threshold. No edge filtering.
type ProjectionRule struct {
    IncludeEntityTypes []string
    IncludeEdgeTypes   []string
    ExcludeProperties  []string  // PII fields to mask
    MinConfidence      float64
    DomainFilter       []string
    ProvenanceFilter   []ProvenanceType  // nil = allow all
}
// Projection rules per role (stored in Ontology Service)
var defaultProjectionRules = map[string]ProjectionRule{
    "BA": {
        IncludeEntityTypes: []string{"Requirement","UseCase","Actor","BusinessRule","Risk"},
        IncludeEdgeTypes:   []string{"DEPENDS_ON","CONFLICTS_WITH","TRACED_TO"},
        ExcludeProperties:  []string{"internal_code","implementation_detail"},
        MinConfidence:      0.70,
    },
    "DEV": {
        IncludeEntityTypes: []string{"APIEndpoint","DataModel","Integration","NFR","Sequence"},
        IncludeEdgeTypes:   []string{"CALLS","IMPLEMENTS","EXTENDS"},
        ExcludeProperties:  []string{"stakeholder_name","business_justification"},
        MinConfidence:      0.65,
    },
    "PO": {
        IncludeEntityTypes: []string{"Epic","UserStory","Feature","Stakeholder"},
        IncludeEdgeTypes:   []string{"PART_OF","BLOCKS","DELIVERS_VALUE_TO"},
        MinConfidence:      0.60,
    },
    "DESIGNER": {
        IncludeEntityTypes: []string{"UserFlow","Screen","Persona","Interaction"},
        IncludeEdgeTypes:   []string{"NAVIGATES_TO","TRIGGERED_BY"},
        MinConfidence:      0.65,
    },
}
func (pe *ProjectionEngine) Project(ctx context.Context,
    versionID, role string, domains []string) ProjectedGraph {
    rules := pe.getRules(role, domains)  // merge role rules + domain overlay
    entities := pe.entityStore.LoadVersion(ctx, versionID)
    edges    := pe.edgeStore.LoadVersion(ctx, versionID)
    // Filter entities
    filteredEntities := []Entity{}
    entitySet := map[string]bool{}
    for _, e := range entities {
        if !contains(rules.IncludeEntityTypes, e.EntityType) { continue }
        if e.Confidence < rules.MinConfidence { continue }
        if len(rules.DomainFilter) > 0 && !overlap(rules.DomainFilter, e.Domains) { continue }
        if rules.ProvenanceFilter != nil && !contains(rules.ProvenanceFilter, e.ProvenanceType) { continue }
        // Mask PII properties
        e.Properties = maskProperties(e.Properties, rules.ExcludeProperties)
        filteredEntities = append(filteredEntities, e)
        entitySet[e.EntityID] = true
    }
    // Filter edges: both endpoints must be in filtered set
    filteredEdges := []Edge{}
    for _, ed := range edges {
        if !contains(rules.IncludeEdgeTypes, ed.RelationType) { continue }
        if !entitySet[ed.FromEntityID] || !entitySet[ed.ToEntityID] { continue }
        filteredEdges = append(filteredEdges, ed)
    }
    return ProjectedGraph{Entities: filteredEntities, Edges: filteredEdges}
}
11. Diff Engine
func (d *DiffEngine) Diff(ctx context.Context, v1, v2 string, tenantID string) DiffReport {
    e1 := d.loadEntities(ctx, v1, tenantID)
    e2 := d.loadEntities(ctx, v2, tenantID)
    added   := setDiff(e2, e1)   // in v2 but not v1
    removed := setDiff(e1, e2)   // in v1 but not v2
    modified := []EntityDiff{}
    for id := range intersect(e1, e2) {
        if !deepEqual(e1[id].Properties, e2[id].Properties) {
            modified = append(modified, EntityDiff{
                EntityID: id,
                Before:   e1[id].Properties,
                After:    e2[id].Properties,
                Delta:    computeDelta(e1[id], e2[id]),
            })
        }
    }
    return DiffReport{Added: added, Removed: removed, Modified: modified,
        V1: v1, V2: v2, TenantID: tenantID}
}
12. Deployment Model
Component
Technology
Scaling
Notes
Graph DB
Neo4j Cluster / ArangoDB
Horizontal read replicas
Write to primary, read from replicas
Vector DB
Qdrant Cluster
Shard by tenantId
Separate index per namespace
Version Store
PostgreSQL + S3
PG for metadata, S3 for delta blobs
S3 for snapshots (immutable)
Lock Manager
Redis (Redlock)
3-node Redis cluster
500ms lock timeout, retry 3x
Overlay Store
Redis (TTL)
Co-located with session Redis
TTL matches session TTL
GC Job
Kubernetes CronJob
1 pod per tenant batch
Low priority, preemptible
Ontology Service
Go service + PostgreSQL
Read replicas
Projection rules cached in-memory