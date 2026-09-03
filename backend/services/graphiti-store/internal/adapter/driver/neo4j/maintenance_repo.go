package neo4j

import (
	"context"
	"fmt"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
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
			if err != nil {
				return fmt.Errorf("clear %s: %w", label, err)
			}
			if len(records) == 0 {
				break
			}
			deleted, _ := records[0].Values[0].(int64)
			if deleted == 0 {
				break
			}
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
		r.driver.ExecuteQuery(ctx, cypher, nil) //nolint:errcheck
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
	if err != nil {
		return nil, err
	}

	adj := make(map[string][]string)
	allNodes := make(map[string]bool)
	for _, rec := range records {
		if len(rec.Values) < 2 {
			continue
		}
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
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &port.GroupStats{GroupID: groupID}, nil
	}

	rec := records[0]
	stats := &port.GroupStats{GroupID: groupID}
	if len(rec.Values) >= 4 {
		stats.EntityCount, _ = rec.Values[0].(int64)
		stats.EpisodeCount, _ = rec.Values[1].(int64)
		stats.CommunityCount, _ = rec.Values[2].(int64)
		stats.EdgeCount, _ = rec.Values[3].(int64)
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
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EntityNode, 0)
	for _, rec := range records {
		n, _ := mapRecordToEntityNode(rec)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// bfsComponents finds connected components using BFS
func bfsComponents(adj map[string][]string, nodes map[string]bool) [][]string {
	visited := make(map[string]bool)
	var clusters [][]string

	for node := range nodes {
		if visited[node] {
			continue
		}
		// BFS from this node
		cluster := []string{}
		queue := []string{node}
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			if visited[curr] {
				continue
			}
			visited[curr] = true
			cluster = append(cluster, curr)
			for _, neighbor := range adj[curr] {
				if !visited[neighbor] {
					queue = append(queue, neighbor)
				}
			}
		}
		if len(cluster) > 1 {
			clusters = append(clusters, cluster)
		}
	}
	return clusters
}
