DOC 7: KG SERVICE — IMPLEMENTATION DETAIL
Multi-level Locking · Overlay Lifecycle · Hybrid Search · Projection · Version GC
Version 1.0
March 2026
1. Package Structure (Golang)
kg-service/
├── cmd/server/
├── internal/
│   ├── api/                HTTP handlers + middleware
│   ├── namespace/          Resolver + enforcement
│   ├── entity/             EntityStore (CRUD + optimistic lock)
│   ├── edge/               EdgeStore
│   ├── batch/              BulkUpsert, BulkEdge
│   ├── search/
│   │   ├── vector/         VectorDB client (Qdrant)
│   │   ├── text/           BM25 search
│   │   └── hybrid/         Score blending + reranking
│   ├── traversal/          Path, Subgraph, Impact, Cluster
│   ├── overlay/            OverlayService (create/commit/discard)
│   ├── version/
│   │   ├── store/          Delta + Snapshot store
│   │   ├── diff/           DiffEngine
│   │   └── gc/             CompactionJob
│   ├── projection/         ProjectionEngine + OntologyService
│   ├── lock/               LockManager (multi-level)
│   └── analytics/          Coverage, Traceability, Clusters
└── pkg/contracts/
2. Batch Upsert — Performance Critical Path
// Bulk upsert: max 1000 entities per call
func (b *BatchHandler) UpsertEntities(ctx context.Context,
    entities []Entity, ns Namespace, overlayID *string) BatchResult {
    // Step 1: semantic dedup check (find existing similar)
    embeddings := embedAll(entities)    // parallel
    existing   := b.vectorDB.BatchSearch(ctx, ns, embeddings, threshold=0.92)
    toInsert, toUpdate, toMerge := classifyEntities(entities, existing)
    // Step 2: resolve conflicts
    merged := resolveConflicts(toMerge, existing, ConflictPolicy)
    // Step 3: write
    if overlayID != nil {
        // Write to overlay (no lock needed, overlay is session-scoped)
        b.overlayStore.AppendEntities(ctx, *overlayID, append(toInsert, merged...))
    } else {
        // Write to base graph with optimistic lock per entity
        b.entityStore.BulkUpsert(ctx, ns, append(toInsert, merged...))
    }
    // Step 4: update vector index
    b.vectorDB.BulkIndex(ctx, ns, append(toInsert, merged...))
    return BatchResult{
        Created:    len(toInsert),
        Updated:    len(merged),
        Skipped:    len(toUpdate),  // identical, no change
        Conflicted: countConflicts(toMerge, existing),
    }
}
3. Graph Traversal APIs — Implementation
3.1 Subgraph Extraction (BFS)
func (t *TraversalService) Subgraph(ctx context.Context,
    ns Namespace, rootID string, depth int,
    nodeTypes, edgeTypes []string) (*SubgraphResult, error) {
    visited  := map[string]bool{rootID: true}
    queue    := []string{rootID}
    entities := []Entity{}
    edges    := []Edge{}
    for d := 0; d < depth && len(queue) > 0; d++ {
        nextQueue := []string{}
        // Batch fetch neighbors (single DB query per level)
        neighbors, edgesOut := t.graphDB.GetNeighborsBatch(ctx, ns, queue, edgeTypes)
        for _, e := range edgesOut {
            if !visited[e.ToEntityID] {
                neighbor := neighbors[e.ToEntityID]
                if len(nodeTypes) == 0 || contains(nodeTypes, neighbor.EntityType) {
                    visited[e.ToEntityID] = true
                    nextQueue = append(nextQueue, e.ToEntityID)
                    entities = append(entities, neighbor)
                    edges    = append(edges, e)
                }
            }
        }
        queue = nextQueue
    }
    return &SubgraphResult{Root: rootID, Entities: entities, Edges: edges}, nil
}
3.2 Impact Analysis (Upstream/Downstream)
func (t *TraversalService) Impact(ctx context.Context,
    ns Namespace, nodeID string, dir Direction, edgeTypes []string,
    maxDepth int) ([]ImpactNode, error) {
    // direction=UPSTREAM:   follow incoming edges (what uses this?)
    // direction=DOWNSTREAM: follow outgoing edges (what does this affect?)
    // direction=BOTH:       union of both
    visited := map[string]bool{}
    result  := []ImpactNode{}
    var dfs func(id string, depth int, path []string)
    dfs = func(id string, depth int, path []string) {
        if depth > maxDepth || visited[id] { return }
        visited[id] = true
        neighbors := t.graphDB.GetNeighbors(ctx, ns, id, dir, edgeTypes)
        for _, n := range neighbors {
            result = append(result, ImpactNode{
                Entity: n,
                Depth:  depth,
                Path:   append(path, n.EntityID),
            })
            dfs(n.EntityID, depth+1, append(path, n.EntityID))
        }
    }
    dfs(nodeID, 1, []string{nodeID})
    return result, nil
}
4. Analytics — Coverage & Traceability
4.1 Domain Coverage
func (a *AnalyticsService) Coverage(ctx context.Context,
    ns Namespace, versionID, domain string) CoverageReport {
    ontology := a.ontologyStore.Get(domain)
    entities := a.entityStore.LoadVersion(ctx, ns, versionID)
    foundTypes := map[string]int{}
    for _, e := range entities {
        if e.Domain == domain { foundTypes[e.EntityType]++ }
    }
    missing  := []string{}
    coverage := 0.0
    for _, required := range ontology.RequiredEntityTypes {
        if foundTypes[required] > 0 {
            coverage += 1.0
        } else {
            missing = append(missing, required)
        }
    }
    coverage /= float64(len(ontology.RequiredEntityTypes))
    return CoverageReport{
        Domain:        domain,
        CoverageScore: coverage,
        FoundTypes:    foundTypes,
        MissingTypes:  missing,
        TotalEntities: len(entities),
    }
}
4.2 Traceability Matrix
// POST /kg/{ns}/traceability
// { from_type: 'Requirement', to_type: 'UserStory',
//   via: ['EXPRESSED_AS', 'IMPLEMENTS'], domains: ['payment'] }
func (a *AnalyticsService) Traceability(ctx context.Context,
    ns Namespace, req TraceRequest) TraceMatrix {
    fromNodes := a.entityStore.LoadByType(ctx, ns, req.FromType, req.Domains)
    matrix    := []TraceRow{}
    for _, from := range fromNodes {
        // BFS following only allowed edge types
        targets := a.traversal.FindReachable(ctx, ns, from.EntityID,
            req.Via, req.ToType, maxDepth=5)
        matrix = append(matrix, TraceRow{
            From:    from,
            To:      targets,
            Covered: len(targets) > 0,
        })
    }
    covered  := countCovered(matrix)
    return TraceMatrix{
        Rows:          matrix,
        Coverage:      float64(covered)/float64(len(matrix)),
        Uncovered:     filterUncovered(matrix),
    }
}
5. Configuration Reference
Config Key
Default
Description
lock.node_timeout_ms
500
Node-level lock acquire timeout
lock.subgraph_timeout_ms
2000
Subgraph lock acquire timeout
version.retain_recent_deltas
50
Keep last N deltas before compaction
version.compact_after_days
7
Compact deltas older than N days
version.compact_threshold
100
Compact when delta count exceeds N
search.semantic_candidates
100
Top-K semantic candidates before rerank
search.text_candidates
100
Top-K BM25 candidates before rerank
search.default_alpha
0.5
Default semantic/text blend (0=text, 1=semantic)
overlay.default_ttl
1h
Auto-discard overlay if not committed
batch.max_entities
1000
Max entities per batch upsert call
batch.max_edges
5000
Max edges per batch create call