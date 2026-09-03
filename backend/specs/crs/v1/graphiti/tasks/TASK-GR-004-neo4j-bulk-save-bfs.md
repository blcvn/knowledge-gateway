# TASK-GR-004 — Neo4j Bulk Save + BFS Search Repository

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-004 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/graphiti-store/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §4.2, §4.3, §6 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-003 |
| **Estimated** | 4h |

---

## Context

Implement `BulkRepository.SaveBulk` (atomic Neo4j transaction cho toàn bộ ingestion result), `SearchRepository` (fulltext, similarity, BFS, rerankers), và `MaintenanceRepository` (clear data, build indices, community clusters).

---

## Goal

- `bulkRepo.SaveBulk` — atomic transaction: invalidate old edges → save entities → save edges → save episode → save saga (optional)
- `searchRepo` — `NodeBFSSearch` (Cypher variable-length paths), `NodeDistanceReranker` (shortestPath), `EpisodeMentionsReranker`, fulltext + similarity search wrappers
- `maintenanceRepo` — `ClearData`, `BuildIndicesAndConstraints`, `GetCommunityClusters` (BFS components)

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/bulk_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/search_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/maintenance_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/community_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/saga_repo.go` |

---

## Implementation

### File 1: `services/graphiti-store/internal/adapter/driver/neo4j/bulk_repo.go`

```go
package neo4j

import (
    "context"
    "fmt"
    "time"

    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type bulkRepo struct {
    driver           *Neo4jDriver
    entityNodes      *entityNodeRepo
    entityEdges      *entityEdgeRepo
    episodeNodes     *episodeNodeRepo
    sagaNodes        *sagaNodeRepo
    episodicEdges    *episodicEdgeRepo
    hasEpisodeEdges  *hasEpisodeEdgeRepo
    nextEpisodeEdges *nextEpisodeEdgeRepo
}

// SaveBulk persists all objects for a single ingestion atomically.
// Order: invalidate old edges → save nodes → save edges → save episode → save saga
func (r *bulkRepo) SaveBulk(ctx context.Context, req port.SaveBulkReq) error {
    tx, err := r.driver.BeginTransaction(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }

    defer func() {
        if err != nil {
            _ = tx.Rollback(ctx)
        }
    }()

    // Step 1: Invalidate old contradicted/updated edges (temporal)
    now := time.Now()
    for _, edgeID := range req.InvalidatedEdgeIDs {
        if err = r.entityEdges.Invalidate(ctx, edgeID, now, tx); err != nil {
            return fmt.Errorf("invalidate edge %s: %w", edgeID, err)
        }
    }

    // Step 2: Save entity nodes (MERGE — upsert)
    if err = r.entityNodes.SaveBulk(ctx, req.EntityNodes, tx, 100); err != nil {
        return fmt.Errorf("save entity nodes: %w", err)
    }

    // Step 3: Save new entity edges
    if err = r.entityEdges.SaveBulk(ctx, req.EntityEdges, tx, 100); err != nil {
        return fmt.Errorf("save entity edges: %w", err)
    }

    // Step 4: Save episodic node
    if err = r.episodeNodes.Save(ctx, req.Episode, tx); err != nil {
        return fmt.Errorf("save episode node: %w", err)
    }

    // Step 5: Save MENTIONS edges (episode → entity)
    if err = r.episodicEdges.SaveBulk(ctx, req.EpisodicEdges, tx); err != nil {
        return fmt.Errorf("save episodic edges: %w", err)
    }

    // Step 6: Save saga (optional)
    if req.SagaNode != nil {
        if err = r.sagaNodes.Save(ctx, *req.SagaNode, tx); err != nil {
            return fmt.Errorf("save saga node: %w", err)
        }
        for _, e := range req.HasEpisodeEdges {
            if err = r.hasEpisodeEdges.Save(ctx, e, tx); err != nil {
                return fmt.Errorf("save has_episode edge: %w", err)
            }
        }
        for _, e := range req.NextEpisodeEdges {
            if err = r.nextEpisodeEdges.Save(ctx, e, tx); err != nil {
                return fmt.Errorf("save next_episode edge: %w", err)
            }
        }
    }

    return tx.Commit(ctx)
}
```

### File 2: `services/graphiti-store/internal/adapter/driver/neo4j/search_repo.go`

```go
package neo4j

import (
    "context"
    "fmt"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type searchRepo struct{ driver *Neo4jDriver }

// NodeFulltextSearch — BM25 fulltext search using Neo4j fulltext index
func (r *searchRepo) NodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, labels []string) ([]*graph.EntityNode, error) {
    cypher := `
        CALL db.index.fulltext.queryNodes("entity_fulltext", $query)
        YIELD node, score
        WHERE node.group_id IN $group_ids
    `
    params := map[string]any{"query": query, "group_ids": groupIDs, "limit": limit}

    if len(labels) > 0 {
        cypher += " AND any(l IN node.labels WHERE l IN $labels)"
        params["labels"] = labels
    }
    cypher += " RETURN node ORDER BY score DESC LIMIT $limit"

    records, err := r.driver.ExecuteQuery(ctx, cypher, params)
    if err != nil { return nil, err }
    nodes := make([]*graph.EntityNode, 0, len(records))
    for _, rec := range records {
        n, _ := mapRecordToEntityNode(rec)
        if n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

// NodeSimilaritySearch — vector cosine similarity using Neo4j vector index
func (r *searchRepo) NodeSimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.EntityNode, error) {
    cypher := `
        CALL db.index.vector.queryNodes("entity_name_embedding", $limit, $vector)
        YIELD node, score
        WHERE node.group_id IN $group_ids AND score >= $min_score
        RETURN node
        ORDER BY score DESC
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "vector": vector, "group_ids": groupIDs,
        "limit": limit * 2, "min_score": minScore,
    })
    if err != nil { return nil, err }
    nodes := make([]*graph.EntityNode, 0, len(records))
    for _, rec := range records {
        n, _ := mapRecordToEntityNode(rec)
        if n != nil { nodes = append(nodes, n) }
    }
    if len(nodes) > limit { nodes = nodes[:limit] }
    return nodes, nil
}

// NodeBFSSearch — breadth-first traversal from origin nodes up to maxDepth hops
func (r *searchRepo) NodeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityNode, error) {
    cypher := fmt.Sprintf(`
        MATCH path = (origin:Entity)-[:RELATES_TO*1..%d]-(n:Entity)
        WHERE origin.uuid IN $origin_uuids
          AND n.group_id IN $group_ids
          AND n.invalid_at IS NULL
          AND NOT n.uuid IN $origin_uuids
        WITH DISTINCT n, min(length(path)) as distance
        ORDER BY distance ASC
        LIMIT $limit
        RETURN n
    `, maxDepth)

    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "origin_uuids": originUUIDs,
        "group_ids":    groupIDs,
        "limit":        limit,
    })
    if err != nil { return nil, err }
    nodes := make([]*graph.EntityNode, 0, len(records))
    for _, rec := range records {
        n, _ := mapRecordToEntityNode(rec)
        if n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

// EdgeFulltextSearch — BM25 fulltext search on edge facts
func (r *searchRepo) EdgeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, filters port.EdgeSearchFilters) ([]*graph.EntityEdge, error) {
    cypher := `
        CALL db.index.fulltext.queryRelationships("entity_edge_fulltext", $query)
        YIELD relationship as e, score
        WHERE e.group_id IN $group_ids
    `
    params := map[string]any{"query": query, "group_ids": groupIDs, "limit": limit}
    cypher += buildTemporalFilter("e", filters, params)
    cypher += " RETURN e, startNode(e).uuid as src_uuid, endNode(e).uuid as tgt_uuid ORDER BY score DESC LIMIT $limit"

    records, err := r.driver.ExecuteQuery(ctx, cypher, params)
    if err != nil { return nil, err }
    return mapRecordsToEntityEdges(records), nil
}

// EdgeSimilaritySearch — vector cosine similarity on edge fact embeddings
func (r *searchRepo) EdgeSimilaritySearch(ctx context.Context, req port.EdgeSimilarityReq) ([]*graph.EntityEdge, error) {
    cypher := `
        CALL db.index.vector.queryRelationships("entity_edge_fact_embedding", $limit, $vector)
        YIELD relationship as e, score
        WHERE e.group_id IN $group_ids AND score >= $min_score
    `
    params := map[string]any{
        "vector": req.Vector, "group_ids": req.GroupIDs,
        "limit": req.Limit * 2, "min_score": req.MinScore,
    }
    if req.SourceUUID != "" {
        cypher += " AND startNode(e).uuid = $src_uuid"
        params["src_uuid"] = req.SourceUUID
    }
    cypher += buildTemporalFilter("e", req.Filters, params)
    cypher += " RETURN e, startNode(e).uuid as src_uuid, endNode(e).uuid as tgt_uuid ORDER BY score DESC LIMIT $limit"

    records, err := r.driver.ExecuteQuery(ctx, cypher, params)
    if err != nil { return nil, err }
    edges := mapRecordsToEntityEdges(records)
    if len(edges) > req.Limit { edges = edges[:req.Limit] }
    return edges, nil
}

// EdgeBFSSearch — BFS traversal returning edges (not nodes)
func (r *searchRepo) EdgeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityEdge, error) {
    cypher := fmt.Sprintf(`
        MATCH (origin:Entity)-[e:RELATES_TO*1..%d]-(:Entity)
        WHERE origin.uuid IN $origin_uuids
          AND e.group_id IN $group_ids
          AND e.invalid_at IS NULL
        UNWIND e as rel
        WITH DISTINCT rel
        LIMIT $limit
        RETURN rel as e, startNode(rel).uuid as src_uuid, endNode(rel).uuid as tgt_uuid
    `, maxDepth)

    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "origin_uuids": originUUIDs, "group_ids": groupIDs, "limit": limit,
    })
    if err != nil { return nil, err }
    return mapRecordsToEntityEdges(records), nil
}

// EpisodeFulltextSearch — fulltext search on episode content
func (r *searchRepo) EpisodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.EpisodicNode, error) {
    cypher := `
        CALL db.index.fulltext.queryNodes("episode_fulltext", $query)
        YIELD node, score
        WHERE node.group_id IN $group_ids
        RETURN node ORDER BY score DESC LIMIT $limit
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "query": query, "group_ids": groupIDs, "limit": limit,
    })
    if err != nil { return nil, err }
    nodes := make([]*graph.EpisodicNode, 0)
    for _, rec := range records {
        n, _ := mapRecordToEpisodicNode(rec)
        if n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

// CommunityFulltextSearch — fulltext search on community names/summaries
func (r *searchRepo) CommunityFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.CommunityNode, error) {
    cypher := `
        CALL db.index.fulltext.queryNodes("community_fulltext", $query)
        YIELD node, score
        WHERE node.group_id IN $group_ids
        RETURN node ORDER BY score DESC LIMIT $limit
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "query": query, "group_ids": groupIDs, "limit": limit,
    })
    if err != nil { return nil, err }
    nodes := make([]*graph.CommunityNode, 0)
    for _, rec := range records {
        n, _ := mapRecordToCommunityNode(rec)
        if n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

func (r *searchRepo) CommunitySimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.CommunityNode, error) {
    cypher := `
        CALL db.index.vector.queryNodes("community_name_embedding", $limit, $vector)
        YIELD node, score
        WHERE node.group_id IN $group_ids AND score >= $min_score
        RETURN node ORDER BY score DESC
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "vector": vector, "group_ids": groupIDs, "limit": limit, "min_score": minScore,
    })
    if err != nil { return nil, err }
    nodes := make([]*graph.CommunityNode, 0)
    for _, rec := range records {
        n, _ := mapRecordToCommunityNode(rec)
        if n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

// NodeDistanceReranker — returns hop distance from center node using shortestPath
func (r *searchRepo) NodeDistanceReranker(ctx context.Context, nodeUUIDs []string, centerUUID string) (map[string]float64, error) {
    cypher := `
        MATCH (center:Entity {uuid: $center_uuid})
        UNWIND $node_uuids AS targetUUID
        MATCH (target:Entity {uuid: targetUUID})
        MATCH path = shortestPath((center)-[:RELATES_TO*1..5]-(target))
        RETURN targetUUID, length(path) as distance
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "center_uuid": centerUUID, "node_uuids": nodeUUIDs,
    })
    if err != nil { return nil, err }

    scores := make(map[string]float64, len(records))
    for _, rec := range records {
        if len(rec.Values) < 2 { continue }
        uuid, _ := rec.Values[0].(string)
        dist, _ := rec.Values[1].(int64)
        // Score inversely proportional to distance
        scores[uuid] = 1.0 / (float64(dist) + 1.0)
    }
    return scores, nil
}

// EpisodeMentionsReranker — returns count of episodes mentioning each node
func (r *searchRepo) EpisodeMentionsReranker(ctx context.Context, nodeUUIDs []string) (map[string]int, error) {
    cypher := `
        MATCH (ep:Episodic)-[:MENTIONS]->(n:Entity)
        WHERE n.uuid IN $node_uuids
        RETURN n.uuid as node_uuid, count(ep) as mention_count
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"node_uuids": nodeUUIDs})
    if err != nil { return nil, err }

    counts := make(map[string]int, len(records))
    for _, rec := range records {
        if len(rec.Values) < 2 { continue }
        uuid, _ := rec.Values[0].(string)
        count, _ := rec.Values[1].(int64)
        counts[uuid] = int(count)
    }
    return counts, nil
}

// buildTemporalFilter adds WHERE clauses for temporal filters
func buildTemporalFilter(alias string, f port.EdgeSearchFilters, params map[string]any) string {
    clause := ""
    if f.ValidAt != nil {
        clause += fmt.Sprintf(" AND (%s.valid_at IS NULL OR %s.valid_at <= $valid_at)", alias, alias)
        clause += fmt.Sprintf(" AND (%s.invalid_at IS NULL OR %s.invalid_at > $valid_at)", alias, alias)
        params["valid_at"] = f.ValidAt
    }
    if f.CreatedAtStart != nil {
        clause += fmt.Sprintf(" AND %s.created_at >= $created_at_start", alias)
        params["created_at_start"] = f.CreatedAtStart
    }
    if f.CreatedAtEnd != nil {
        clause += fmt.Sprintf(" AND %s.created_at <= $created_at_end", alias)
        params["created_at_end"] = f.CreatedAtEnd
    }
    return clause
}

func mapRecordsToEntityEdges(records []port.Record) []*graph.EntityEdge {
    edges := make([]*graph.EntityEdge, 0, len(records))
    for _, rec := range records {
        e, _ := mapRecordToEntityEdge(rec)
        if e != nil { edges = append(edges, e) }
    }
    return edges
}
```

### File 3: `services/graphiti-store/internal/adapter/driver/neo4j/maintenance_repo.go`

```go
package neo4j

import (
    "context"
    "fmt"

    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type maintenanceRepo struct{ driver *Neo4jDriver }

// ClearData deletes ALL data for given group_ids
func (r *maintenanceRepo) ClearData(ctx context.Context, groupIDs []string) error {
    labels := []string{"Entity", "Episodic", "Community", "Saga"}
    for _, label := range labels {
        for {
            cypher := fmt.Sprintf(`
                MATCH (n:%s) WHERE n.group_id IN $group_ids
                WITH n LIMIT 1000 DETACH DELETE n
                RETURN count(n) as deleted
            `, label)
            records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"group_ids": groupIDs})
            if err != nil { return fmt.Errorf("clear %s: %w", label, err) }
            if len(records) == 0 { break }
            deleted, _ := records[0].Values[0].(int64)
            if deleted == 0 { break }
        }
    }
    return nil
}

// BuildIndicesAndConstraints creates all Neo4j indices and uniqueness constraints
func (r *maintenanceRepo) BuildIndicesAndConstraints(ctx context.Context, deleteExisting bool) error {
    if deleteExisting {
        if err := r.DeleteAllIndexes(ctx); err != nil {
            return fmt.Errorf("delete existing indices: %w", err)
        }
    }

    statements := []string{
        // Uniqueness constraints
        `CREATE CONSTRAINT entity_node_uuid IF NOT EXISTS FOR (n:Entity) REQUIRE n.uuid IS UNIQUE`,
        `CREATE CONSTRAINT episodic_node_uuid IF NOT EXISTS FOR (n:Episodic) REQUIRE n.uuid IS UNIQUE`,
        `CREATE CONSTRAINT community_node_uuid IF NOT EXISTS FOR (n:Community) REQUIRE n.uuid IS UNIQUE`,
        `CREATE CONSTRAINT saga_node_uuid IF NOT EXISTS FOR (n:Saga) REQUIRE n.uuid IS UNIQUE`,

        // Vector indices (requires Neo4j 5.11+)
        `CREATE VECTOR INDEX entity_name_embedding IF NOT EXISTS FOR (n:Entity) ON (n.name_embedding) OPTIONS {indexConfig: {"vector.dimensions": 1536, "vector.similarity_function": "cosine"}}`,
        `CREATE VECTOR INDEX entity_edge_fact_embedding IF NOT EXISTS FOR ()-[r:RELATES_TO]-() ON (r.fact_embedding) OPTIONS {indexConfig: {"vector.dimensions": 1536, "vector.similarity_function": "cosine"}}`,
        `CREATE VECTOR INDEX community_name_embedding IF NOT EXISTS FOR (n:Community) ON (n.name_embedding) OPTIONS {indexConfig: {"vector.dimensions": 1536, "vector.similarity_function": "cosine"}}`,

        // Fulltext indices
        `CREATE FULLTEXT INDEX entity_fulltext IF NOT EXISTS FOR (n:Entity) ON EACH [n.name, n.summary]`,
        `CREATE FULLTEXT INDEX episode_fulltext IF NOT EXISTS FOR (n:Episodic) ON EACH [n.content, n.source_description]`,
        `CREATE FULLTEXT INDEX community_fulltext IF NOT EXISTS FOR (n:Community) ON EACH [n.name, n.summary]`,
        `CREATE FULLTEXT INDEX entity_edge_fulltext IF NOT EXISTS FOR ()-[r:RELATES_TO]-() ON EACH [r.fact, r.name]`,

        // Standard property indices for filtering
        `CREATE INDEX entity_group_id IF NOT EXISTS FOR (n:Entity) ON (n.group_id)`,
        `CREATE INDEX episode_group_id IF NOT EXISTS FOR (n:Episodic) ON (n.group_id)`,
        `CREATE INDEX episode_valid_at IF NOT EXISTS FOR (n:Episodic) ON (n.valid_at)`,
        `CREATE INDEX edge_group_id IF NOT EXISTS FOR ()-[r:RELATES_TO]-() ON (r.group_id)`,
        `CREATE INDEX edge_valid_at IF NOT EXISTS FOR ()-[r:RELATES_TO]-() ON (r.valid_at)`,
        `CREATE INDEX edge_invalid_at IF NOT EXISTS FOR ()-[r:RELATES_TO]-() ON (r.invalid_at)`,
    }

    for _, stmt := range statements {
        if _, err := r.driver.ExecuteQuery(ctx, stmt, nil); err != nil {
            // Log but continue — some indices may already exist
            fmt.Printf("warn: build index: %v\n", err)
        }
    }
    return nil
}

func (r *maintenanceRepo) DeleteAllIndexes(ctx context.Context) error {
    // Drop vector and fulltext indices
    indices := []string{
        "entity_name_embedding", "entity_edge_fact_embedding", "community_name_embedding",
        "entity_fulltext", "episode_fulltext", "community_fulltext", "entity_edge_fulltext",
    }
    for _, idx := range indices {
        cypher := fmt.Sprintf("DROP INDEX %s IF EXISTS", idx)
        r.driver.ExecuteQuery(ctx, cypher, nil)
    }
    return nil
}

// GetCommunityClusters returns node groups using BFS connected components
func (r *maintenanceRepo) GetCommunityClusters(ctx context.Context, groupIDs []string) ([][]string, error) {
    cypher := `
        MATCH (n:Entity)-[:RELATES_TO]->(m:Entity)
        WHERE n.group_id IN $group_ids AND m.group_id IN $group_ids
          AND n.invalid_at IS NULL
        RETURN n.uuid as source, collect(DISTINCT m.uuid) as neighbors
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"group_ids": groupIDs})
    if err != nil { return nil, err }

    adj := make(map[string][]string)
    allNodes := make(map[string]bool)
    for _, rec := range records {
        if len(rec.Values) < 2 { continue }
        src, _ := rec.Values[0].(string)
        allNodes[src] = true
        if neighbors, ok := rec.Values[1].([]any); ok {
            for _, n := range neighbors {
                if uuid, ok := n.(string); ok {
                    adj[src] = append(adj[src], uuid)
                    allNodes[uuid] = true
                }
            }
        }
    }
    return bfsComponents(adj, allNodes), nil
}

func (r *maintenanceRepo) RemoveCommunities(ctx context.Context, groupID string) error {
    _, err := r.driver.ExecuteQuery(ctx,
        `MATCH (c:Community {group_id: $group_id}) DETACH DELETE c`,
        map[string]any{"group_id": groupID},
    )
    return err
}

func (r *maintenanceRepo) GetGroupStats(ctx context.Context, groupID string) (*port.GroupStats, error) {
    cypher := `
        MATCH (n:Entity {group_id: $gid}) WITH count(n) as entities
        MATCH (ep:Episodic {group_id: $gid}) WITH entities, count(ep) as episodes
        MATCH (c:Community {group_id: $gid}) WITH entities, episodes, count(c) as communities
        MATCH ()-[e:RELATES_TO {group_id: $gid}]->() WITH entities, episodes, communities, count(e) as edges
        RETURN entities, episodes, communities, edges
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"gid": groupID})
    if err != nil { return nil, err }
    if len(records) == 0 { return &port.GroupStats{GroupID: groupID}, nil }

    rec := records[0]
    stats := &port.GroupStats{GroupID: groupID}
    if len(rec.Values) >= 4 {
        stats.EntityCount, _    = rec.Values[0].(int64)
        stats.EpisodeCount, _   = rec.Values[1].(int64)
        stats.CommunityCount, _ = rec.Values[2].(int64)
        stats.EdgeCount, _      = rec.Values[3].(int64)
    }
    return stats, nil
}

func (r *maintenanceRepo) GetMentionedNodes(ctx context.Context, episodeUUIDs []string) ([]*graph.EntityNode, error) {
    cypher := `
        MATCH (ep:Episodic)-[:MENTIONS]->(n:Entity)
        WHERE ep.uuid IN $episode_uuids
        RETURN DISTINCT n
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"episode_uuids": episodeUUIDs})
    if err != nil { return nil, err }
    nodes := make([]*graph.EntityNode, 0)
    for _, rec := range records {
        n, _ := mapRecordToEntityNode(rec)
        if n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

// bfsComponents finds connected components using BFS
func bfsComponents(adj map[string][]string, nodes map[string]bool) [][]string {
    visited := make(map[string]bool)
    var clusters [][]string

    for node := range nodes {
        if visited[node] { continue }
        // BFS from this node
        cluster := []string{}
        queue := []string{node}
        for len(queue) > 0 {
            curr := queue[0]
            queue = queue[1:]
            if visited[curr] { continue }
            visited[curr] = true
            cluster = append(cluster, curr)
            for _, neighbor := range adj[curr] {
                if !visited[neighbor] { queue = append(queue, neighbor) }
            }
        }
        if len(cluster) > 1 { clusters = append(clusters, cluster) }
    }
    return clusters
}
```

---

## Verification

```bash
cd services/graphiti-store
go build ./internal/adapter/driver/neo4j/...
go vet ./internal/adapter/driver/neo4j/...
```

**Expected:** No errors. All port.BulkRepository, port.SearchRepository, port.MaintenanceRepository interfaces satisfied.
