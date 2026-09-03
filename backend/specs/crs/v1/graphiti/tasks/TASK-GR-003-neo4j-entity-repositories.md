# TASK-GR-003 — Neo4j Entity & Episode Repositories

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-003 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/graphiti-store/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §4.1 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-002 |
| **Estimated** | 4h |

---

## Context

Implement các Neo4j repository cho EntityNode, EntityEdge (với bi-temporal `Invalidate`), EpisodicNode, CommunityNode, SagaNode, và các edge repos (EpisodicEdge, CommunityEdge, HasEpisodeEdge, NextEpisodeEdge).

**Key invariant:** `EntityEdge.Invalidate()` KHÔNG bao giờ xóa — chỉ set `invalid_at` và `expired_at`.

---

## Goal

- `entityNodeRepo` — Save, SaveBulk, GetByUUID, Delete
- `entityEdgeRepo` — Save, SaveBulk, GetByUUID, GetBetweenNodes, **Invalidate** (temporal)
- `episodeNodeRepo` — Save, GetByUUID, RetrieveEpisodes (N most recent)
- `communityNodeRepo`, `sagaNodeRepo` — Save, Get
- Edge repos: `episodicEdgeRepo`, `communityEdgeRepo`, `hasEpisodeEdgeRepo`, `nextEpisodeEdgeRepo`

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/entity_node_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/entity_edge_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/episode_node_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/community_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/saga_repo.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/edge_repos.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/mappers.go` |

---

## Implementation

### File 1: `services/graphiti-store/internal/adapter/driver/neo4j/entity_node_repo.go`

```go
package neo4j

import (
    "context"
    "fmt"
    "time"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type entityNodeRepo struct{ driver *Neo4jDriver }

func (r *entityNodeRepo) Save(ctx context.Context, node graph.EntityNode, tx port.Transaction) error {
    cypher := `
        MERGE (n:Entity {uuid: $uuid})
        SET n.name           = $name,
            n.labels         = $labels,
            n.summary        = $summary,
            n.group_id       = $group_id,
            n.name_embedding = $name_embedding,
            n.updated_at     = datetime()
        ON CREATE SET n.created_at = datetime()
    `
    params := map[string]any{
        "uuid":           node.UUID,
        "name":           node.Name,
        "labels":         node.Labels,
        "summary":        node.Summary,
        "group_id":       node.GroupID,
        "name_embedding": node.NameEmbedding,
    }
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *entityNodeRepo) SaveBulk(ctx context.Context, nodes []graph.EntityNode, tx port.Transaction, batchSize int) error {
    for i := 0; i < len(nodes); i += batchSize {
        end := i + batchSize
        if end > len(nodes) { end = len(nodes) }
        batch := nodes[i:end]
        for _, node := range batch {
            if err := r.Save(ctx, node, tx); err != nil {
                return fmt.Errorf("save entity node %s: %w", node.UUID, err)
            }
        }
    }
    return nil
}

func (r *entityNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EntityNode, error) {
    cypher := `MATCH (n:Entity {uuid: $uuid}) RETURN n`
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": uuid})
    if err != nil { return nil, err }
    if len(records) == 0 { return nil, nil }
    return mapRecordToEntityNode(records[0])
}

func (r *entityNodeRepo) GetByUUIDs(ctx context.Context, uuids []string) ([]*graph.EntityNode, error) {
    cypher := `MATCH (n:Entity) WHERE n.uuid IN $uuids RETURN n`
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuids": uuids})
    if err != nil { return nil, err }
    nodes := make([]*graph.EntityNode, 0, len(records))
    for _, rec := range records {
        n, err := mapRecordToEntityNode(rec)
        if err == nil && n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

func (r *entityNodeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
    cypher := `
        MATCH (n:Entity {uuid: $uuid})
        DETACH DELETE n
    `
    params := map[string]any{"uuid": uuid}
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *entityNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction, batchSize int) error {
    // Batch delete to avoid large transactions
    for {
        cypher := fmt.Sprintf(`
            MATCH (n:Entity {group_id: $group_id})
            WITH n LIMIT %d
            DETACH DELETE n
            RETURN count(n) as deleted
        `, batchSize)
        records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"group_id": groupID})
        if err != nil { return err }
        if len(records) == 0 { break }
        deleted, _ := records[0].Values[0].(int64)
        if deleted == 0 { break }
    }
    return nil
}
```

### File 2: `services/graphiti-store/internal/adapter/driver/neo4j/entity_edge_repo.go`

```go
package neo4j

import (
    "context"
    "fmt"
    "time"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type entityEdgeRepo struct{ driver *Neo4jDriver }

func (r *entityEdgeRepo) Save(ctx context.Context, edge graph.EntityEdge, tx port.Transaction) error {
    cypher := `
        MATCH (src:Entity {uuid: $src_uuid}), (tgt:Entity {uuid: $tgt_uuid})
        MERGE (src)-[e:RELATES_TO {uuid: $uuid}]->(tgt)
        SET e.name           = $name,
            e.fact           = $fact,
            e.fact_embedding = $fact_embedding,
            e.episodes       = $episodes,
            e.group_id       = $group_id,
            e.valid_at       = $valid_at,
            e.invalid_at     = null,
            e.expired_at     = null,
            e.updated_at     = datetime()
        ON CREATE SET e.created_at = datetime()
    `
    params := map[string]any{
        "uuid":           edge.UUID,
        "src_uuid":       edge.SourceNodeUUID,
        "tgt_uuid":       edge.TargetNodeUUID,
        "name":           edge.Name,
        "fact":           edge.Fact,
        "fact_embedding": edge.FactEmbedding,
        "episodes":       edge.Episodes,
        "group_id":       edge.GroupID,
        "valid_at":       edge.ValidAt,
    }
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *entityEdgeRepo) SaveBulk(ctx context.Context, edges []graph.EntityEdge, tx port.Transaction, batchSize int) error {
    for i := 0; i < len(edges); i += batchSize {
        end := i + batchSize
        if end > len(edges) { end = len(edges) }
        for _, edge := range edges[i:end] {
            if err := r.Save(ctx, edge, tx); err != nil {
                return fmt.Errorf("save entity edge %s: %w", edge.UUID, err)
            }
        }
    }
    return nil
}

// Invalidate marks an EntityEdge as temporally invalid.
// NEVER deletes — preserves historical data for point-in-time queries.
func (r *entityEdgeRepo) Invalidate(ctx context.Context, uuid string, invalidAt time.Time, tx port.Transaction) error {
    cypher := `
        MATCH ()-[e:RELATES_TO {uuid: $uuid}]->()
        SET e.invalid_at = $invalid_at,
            e.expired_at = datetime(),
            e.updated_at = datetime()
    `
    params := map[string]any{
        "uuid":       uuid,
        "invalid_at": invalidAt,
    }
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *entityEdgeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EntityEdge, error) {
    cypher := `
        MATCH (src)-[e:RELATES_TO {uuid: $uuid}]->(tgt)
        RETURN e, src.uuid as src_uuid, tgt.uuid as tgt_uuid
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": uuid})
    if err != nil { return nil, err }
    if len(records) == 0 { return nil, nil }
    return mapRecordToEntityEdge(records[0])
}

// GetBetweenNodes returns ALL edges including invalidated (caller filters temporal)
func (r *entityEdgeRepo) GetBetweenNodes(ctx context.Context, srcUUID, tgtUUID string) ([]*graph.EntityEdge, error) {
    cypher := `
        MATCH (src:Entity {uuid: $src_uuid})-[e:RELATES_TO]->(tgt:Entity {uuid: $tgt_uuid})
        RETURN e, src.uuid as src_uuid, tgt.uuid as tgt_uuid
        ORDER BY e.created_at DESC
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "src_uuid": srcUUID, "tgt_uuid": tgtUUID,
    })
    if err != nil { return nil, err }
    edges := make([]*graph.EntityEdge, 0, len(records))
    for _, rec := range records {
        e, err := mapRecordToEntityEdge(rec)
        if err == nil && e != nil { edges = append(edges, e) }
    }
    return edges, nil
}

func (r *entityEdgeRepo) GetByNodeUUID(ctx context.Context, nodeUUID string) ([]*graph.EntityEdge, error) {
    cypher := `
        MATCH (n:Entity {uuid: $uuid})-[e:RELATES_TO]-(other:Entity)
        RETURN e, 
               CASE WHEN startNode(e) = n THEN n.uuid ELSE other.uuid END as src_uuid,
               CASE WHEN endNode(e) = n THEN n.uuid ELSE other.uuid END as tgt_uuid
        ORDER BY e.created_at DESC
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": nodeUUID})
    if err != nil { return nil, err }
    edges := make([]*graph.EntityEdge, 0)
    for _, rec := range records {
        e, err := mapRecordToEntityEdge(rec)
        if err == nil && e != nil { edges = append(edges, e) }
    }
    return edges, nil
}

func (r *entityEdgeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
    cypher := `MATCH ()-[e:RELATES_TO {uuid: $uuid}]->() DELETE e`
    params := map[string]any{"uuid": uuid}
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}
```

### File 3: `services/graphiti-store/internal/adapter/driver/neo4j/episode_node_repo.go`

```go
package neo4j

import (
    "context"
    "fmt"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type episodeNodeRepo struct{ driver *Neo4jDriver }

func (r *episodeNodeRepo) Save(ctx context.Context, node graph.EpisodicNode, tx port.Transaction) error {
    cypher := `
        MERGE (ep:Episodic {uuid: $uuid})
        SET ep.name               = $name,
            ep.content            = $content,
            ep.source             = $source,
            ep.source_description = $source_description,
            ep.valid_at           = $valid_at,
            ep.group_id           = $group_id,
            ep.updated_at         = datetime()
        ON CREATE SET ep.created_at = datetime()
    `
    params := map[string]any{
        "uuid":               node.UUID,
        "name":               node.Name,
        "content":            node.Content,
        "source":             string(node.Source),
        "source_description": node.SourceDescription,
        "valid_at":           node.ValidAt,
        "group_id":           node.GroupID,
    }
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *episodeNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EpisodicNode, error) {
    records, err := r.driver.ExecuteQuery(ctx,
        `MATCH (ep:Episodic {uuid: $uuid}) RETURN ep`,
        map[string]any{"uuid": uuid},
    )
    if err != nil { return nil, err }
    if len(records) == 0 { return nil, nil }
    return mapRecordToEpisodicNode(records[0])
}

// RetrieveEpisodes returns the N most recent episodes for given group_ids
func (r *episodeNodeRepo) RetrieveEpisodes(ctx context.Context, req port.RetrieveEpisodesReq) ([]*graph.EpisodicNode, error) {
    cypher := `
        MATCH (ep:Episodic)
        WHERE ep.group_id IN $group_ids
    `
    params := map[string]any{"group_ids": req.GroupIDs, "limit": req.LastN}

    if req.Source != nil {
        cypher += " AND ep.source = $source"
        params["source"] = string(*req.Source)
    }
    if req.SagaID != "" {
        cypher += " AND EXISTS((s:Saga {uuid: $saga_id})-[:HAS_EPISODE]->(ep))"
        params["saga_id"] = req.SagaID
    }

    cypher += " RETURN ep ORDER BY ep.valid_at DESC LIMIT $limit"

    records, err := r.driver.ExecuteQuery(ctx, cypher, params)
    if err != nil { return nil, err }
    nodes := make([]*graph.EpisodicNode, 0, len(records))
    for _, rec := range records {
        n, err := mapRecordToEpisodicNode(rec)
        if err == nil && n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

func (r *episodeNodeRepo) GetByEntityNodeUUID(ctx context.Context, entityNodeUUID string) ([]*graph.EpisodicNode, error) {
    cypher := `
        MATCH (ep:Episodic)-[:MENTIONS]->(entity:Entity {uuid: $uuid})
        RETURN ep ORDER BY ep.valid_at DESC
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": entityNodeUUID})
    if err != nil { return nil, err }
    nodes := make([]*graph.EpisodicNode, 0)
    for _, rec := range records {
        n, err := mapRecordToEpisodicNode(rec)
        if err == nil && n != nil { nodes = append(nodes, n) }
    }
    return nodes, nil
}

func (r *episodeNodeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
    cypher := `MATCH (ep:Episodic {uuid: $uuid}) DETACH DELETE ep`
    params := map[string]any{"uuid": uuid}
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *episodeNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction, batchSize int) error {
    for {
        cypher := fmt.Sprintf(`
            MATCH (ep:Episodic {group_id: $group_id})
            WITH ep LIMIT %d DETACH DELETE ep
            RETURN count(ep) as deleted
        `, batchSize)
        records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"group_id": groupID})
        if err != nil { return err }
        if len(records) == 0 { break }
        deleted, _ := records[0].Values[0].(int64)
        if deleted == 0 { break }
    }
    return nil
}
```

### File 4: `services/graphiti-store/internal/adapter/driver/neo4j/edge_repos.go`

```go
package neo4j

import (
    "context"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

// ─── Episodic Edges (MENTIONS: episode → entity) ─────────────────────────────

type episodicEdgeRepo struct{ driver *Neo4jDriver }

func (r *episodicEdgeRepo) Save(ctx context.Context, edge graph.EpisodicEdge, tx port.Transaction) error {
    cypher := `
        MATCH (ep:Episodic {uuid: $src}), (entity:Entity {uuid: $tgt})
        MERGE (ep)-[e:MENTIONS {uuid: $uuid}]->(entity)
        ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
    `
    params := map[string]any{
        "uuid": edge.UUID, "src": edge.SourceUUID,
        "tgt": edge.TargetUUID, "group_id": edge.GroupID,
    }
    if tx != nil { _, err := tx.Run(ctx, cypher, params); return err }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *episodicEdgeRepo) SaveBulk(ctx context.Context, edges []graph.EpisodicEdge, tx port.Transaction) error {
    for _, e := range edges {
        if err := r.Save(ctx, e, tx); err != nil { return err }
    }
    return nil
}

func (r *episodicEdgeRepo) DeleteByEpisodeUUID(ctx context.Context, episodeUUID string, tx port.Transaction) error {
    cypher := `MATCH (ep:Episodic {uuid: $uuid})-[e:MENTIONS]->() DELETE e`
    params := map[string]any{"uuid": episodeUUID}
    if tx != nil { _, err := tx.Run(ctx, cypher, params); return err }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

// ─── Community Edges (HAS_MEMBER: community → entity) ────────────────────────

type communityEdgeRepo struct{ driver *Neo4jDriver }

func (r *communityEdgeRepo) Save(ctx context.Context, edge graph.CommunityEdge, tx port.Transaction) error {
    cypher := `
        MATCH (c:Community {uuid: $src}), (n:Entity {uuid: $tgt})
        MERGE (c)-[e:HAS_MEMBER {uuid: $uuid}]->(n)
        ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
    `
    params := map[string]any{
        "uuid": edge.UUID, "src": edge.SourceUUID,
        "tgt": edge.TargetUUID, "group_id": edge.GroupID,
    }
    if tx != nil { _, err := tx.Run(ctx, cypher, params); return err }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

func (r *communityEdgeRepo) DeleteByCommunityUUID(ctx context.Context, communityUUID string, tx port.Transaction) error {
    cypher := `MATCH (c:Community {uuid: $uuid})-[e:HAS_MEMBER]->() DELETE e`
    params := map[string]any{"uuid": communityUUID}
    if tx != nil { _, err := tx.Run(ctx, cypher, params); return err }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

// ─── Saga Edges ───────────────────────────────────────────────────────────────

type hasEpisodeEdgeRepo struct{ driver *Neo4jDriver }

func (r *hasEpisodeEdgeRepo) Save(ctx context.Context, edge graph.HasEpisodeEdge, tx port.Transaction) error {
    cypher := `
        MATCH (s:Saga {uuid: $src}), (ep:Episodic {uuid: $tgt})
        MERGE (s)-[e:HAS_EPISODE {uuid: $uuid}]->(ep)
        ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
    `
    params := map[string]any{
        "uuid": edge.UUID, "src": edge.SourceUUID,
        "tgt": edge.TargetUUID, "group_id": edge.GroupID,
    }
    if tx != nil { _, err := tx.Run(ctx, cypher, params); return err }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

type nextEpisodeEdgeRepo struct{ driver *Neo4jDriver }

func (r *nextEpisodeEdgeRepo) Save(ctx context.Context, edge graph.NextEpisodeEdge, tx port.Transaction) error {
    cypher := `
        MATCH (src:Episodic {uuid: $src}), (tgt:Episodic {uuid: $tgt})
        MERGE (src)-[e:NEXT_EPISODE {uuid: $uuid}]->(tgt)
        ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
    `
    params := map[string]any{
        "uuid": edge.UUID, "src": edge.SourceUUID,
        "tgt": edge.TargetUUID, "group_id": edge.GroupID,
    }
    if tx != nil { _, err := tx.Run(ctx, cypher, params); return err }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}
```

### File 5: `services/graphiti-store/internal/adapter/driver/neo4j/mappers.go`

```go
package neo4j

import (
    "time"

    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

func mapRecordToEntityNode(rec port.Record) (*graph.EntityNode, error) {
    nodeVal, ok := rec.Values[0].(neo4j.Node)
    if !ok { return nil, nil }
    props := nodeVal.Props

    node := &graph.EntityNode{
        UUID:    getString(props, "uuid"),
        Name:    getString(props, "name"),
        Summary: getString(props, "summary"),
        GroupID: getString(props, "group_id"),
    }
    if labels, ok := props["labels"].([]any); ok {
        for _, l := range labels {
            if s, ok := l.(string); ok { node.Labels = append(node.Labels, s) }
        }
    }
    if emb, ok := props["name_embedding"].([]any); ok {
        node.NameEmbedding = toFloat32Slice(emb)
    }
    node.CreatedAt = parseTime(props["created_at"])
    node.UpdatedAt = parseTime(props["updated_at"])
    return node, nil
}

func mapRecordToEntityEdge(rec port.Record) (*graph.EntityEdge, error) {
    relVal, ok := rec.Values[0].(neo4j.Relationship)
    if !ok { return nil, nil }
    props := relVal.Props

    edge := &graph.EntityEdge{
        UUID:           getString(props, "uuid"),
        Name:           getString(props, "name"),
        Fact:           getString(props, "fact"),
        GroupID:        getString(props, "group_id"),
        SourceNodeUUID: getString(rec.Values[1], ""),
        TargetNodeUUID: getString(rec.Values[2], ""),
    }
    if emb, ok := props["fact_embedding"].([]any); ok {
        edge.FactEmbedding = toFloat32Slice(emb)
    }
    if eps, ok := props["episodes"].([]any); ok {
        for _, e := range eps {
            if s, ok := e.(string); ok { edge.Episodes = append(edge.Episodes, s) }
        }
    }
    if v := parseTimePtr(props["valid_at"]);   v != nil { edge.ValidAt = v }
    if v := parseTimePtr(props["invalid_at"]); v != nil { edge.InvalidAt = v }
    if v := parseTimePtr(props["expired_at"]); v != nil { edge.ExpiredAt = v }
    edge.CreatedAt = parseTime(props["created_at"])
    edge.UpdatedAt = parseTime(props["updated_at"])
    return edge, nil
}

func mapRecordToEpisodicNode(rec port.Record) (*graph.EpisodicNode, error) {
    nodeVal, ok := rec.Values[0].(neo4j.Node)
    if !ok { return nil, nil }
    props := nodeVal.Props
    return &graph.EpisodicNode{
        UUID:              getString(props, "uuid"),
        Name:              getString(props, "name"),
        Content:           getString(props, "content"),
        Source:            graph.EpisodeType(getString(props, "source")),
        SourceDescription: getString(props, "source_description"),
        GroupID:           getString(props, "group_id"),
        ValidAt:           parseTime(props["valid_at"]),
        CreatedAt:         parseTime(props["created_at"]),
    }, nil
}

// Helper functions
func getString(v any, def string) string {
    if s, ok := v.(string); ok { return s }
    if m, ok := v.(map[string]any); ok {
        if s, ok := m[def].(string); ok { return s }
    }
    return def
}

func toFloat32Slice(vals []any) []float32 {
    result := make([]float32, 0, len(vals))
    for _, v := range vals {
        switch f := v.(type) {
        case float32: result = append(result, f)
        case float64: result = append(result, float32(f))
        }
    }
    return result
}

func parseTime(v any) time.Time {
    if t, ok := v.(time.Time); ok { return t }
    if neo4jTime, ok := v.(neo4j.LocalDateTime); ok {
        return neo4jTime.Time()
    }
    return time.Time{}
}

func parseTimePtr(v any) *time.Time {
    if v == nil { return nil }
    t := parseTime(v)
    if t.IsZero() { return nil }
    return &t
}
```

---

## Verification

```bash
cd services/graphiti-store
go build ./internal/adapter/driver/neo4j/...
```

**Expected:** No compilation errors. All repository interfaces satisfied.
