// Package neo4j implements the domain.GraphDriver interface using Neo4j 5.x.
//
// This is the primary graph database backend for the graphiti-store service.
// It handles all CRUD operations, search primitives, bulk operations,
// and index management via the Neo4j Go driver.
package neo4j

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"vnp-memory/services/graphiti-store/domain"
)

// Driver implements domain.GraphDriver backed by Neo4j.
type Driver struct {
	driver   neo4j.DriverWithContext
	database string
	logger   *slog.Logger
}

// NewDriver creates a new Neo4j driver with the given configuration.
func NewDriver(uri, username, password, database string, logger *slog.Logger) (*Driver, error) {
	auth := neo4j.BasicAuth(username, password, "")
	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		return nil, fmt.Errorf("neo4j: new driver: %w", err)
	}

	// Verify connectivity at startup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("neo4j: verify connectivity: %w", err)
	}

	return &Driver{
		driver:   driver,
		database: database,
		logger:   logger.With("adapter", "neo4j"),
	}, nil
}

// Close releases all Neo4j driver resources.
func (d *Driver) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.driver.Close(ctx)
}

// session returns a new session for the configured database.
func (d *Driver) session(ctx context.Context) neo4j.SessionWithContext {
	return d.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: d.database})
}

// --- NodeRepository ---

// SaveNode creates or updates an EntityNode using MERGE by UUID.
func (d *Driver) SaveNode(ctx context.Context, node domain.EntityNode) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MERGE (n:Entity {uuid: $uuid})
		SET n.name = $name,
		    n.group_id = $group_id,
		    n.summary = $summary,
		    n.name_embedding = $name_embedding,
		    n.labels = $labels,
		    n.created_at = coalesce(n.created_at, datetime()),
		    n.updated_at = datetime()
	`
	params := map[string]any{
		"uuid":           node.UUID,
		"name":           node.Name,
		"group_id":       node.GroupID,
		"summary":        node.Summary,
		"name_embedding": node.NameEmbedding,
		"labels":         node.Labels,
	}

	_, err := session.Run(ctx, cypher, params)
	if err != nil {
		return fmt.Errorf("neo4j: save node: %w", err)
	}
	return nil
}

// GetNode retrieves an EntityNode by UUID scoped to group_id.
func (d *Driver) GetNode(ctx context.Context, groupID, uuid string) (*domain.EntityNode, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (n:Entity {uuid: $uuid, group_id: $group_id})
		RETURN n
	`
	result, err := session.Run(ctx, cypher, map[string]any{
		"uuid":     uuid,
		"group_id": groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: get node: %w", err)
	}

	if result.Next(ctx) {
		return d.recordToEntityNode(result.Record())
	}
	return nil, domain.ErrNodeNotFound
}

// GetNodeByName retrieves an EntityNode by exact name within a group.
func (d *Driver) GetNodeByName(ctx context.Context, groupID, name string) (*domain.EntityNode, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (n:Entity {name: $name, group_id: $group_id})
		RETURN n LIMIT 1
	`
	result, err := session.Run(ctx, cypher, map[string]any{
		"name":     name,
		"group_id": groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: get node by name: %w", err)
	}

	if result.Next(ctx) {
		return d.recordToEntityNode(result.Record())
	}
	return nil, domain.ErrNodeNotFound
}

// DeleteNode removes an entity node and all its relationships.
func (d *Driver) DeleteNode(ctx context.Context, groupID, uuid string) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (n:Entity {uuid: $uuid, group_id: $group_id})
		DETACH DELETE n
	`
	_, err := session.Run(ctx, cypher, map[string]any{
		"uuid":     uuid,
		"group_id": groupID,
	})
	if err != nil {
		return fmt.Errorf("neo4j: delete node: %w", err)
	}
	return nil
}

// ListNodes returns paginated EntityNodes for a group.
func (d *Driver) ListNodes(ctx context.Context, groupID string, opts domain.PaginationOpts) ([]domain.EntityNode, string, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (n:Entity {group_id: $group_id})
		WHERE $cursor = '' OR n.uuid > $cursor
		RETURN n
		ORDER BY n.uuid
		LIMIT $limit
	`
	result, err := session.Run(ctx, cypher, map[string]any{
		"group_id": groupID,
		"cursor":   opts.Cursor,
		"limit":    opts.Limit,
	})
	if err != nil {
		return nil, "", fmt.Errorf("neo4j: list nodes: %w", err)
	}

	var nodes []domain.EntityNode
	var lastUUID string
	for result.Next(ctx) {
		node, err := d.recordToEntityNode(result.Record())
		if err != nil {
			d.logger.Warn("skip node", "error", err)
			continue
		}
		nodes = append(nodes, *node)
		lastUUID = node.UUID
	}

	return nodes, lastUUID, nil
}

// SaveEpisodicNode creates or updates an EpisodicNode.
func (d *Driver) SaveEpisodicNode(ctx context.Context, node domain.EpisodicNode) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MERGE (n:Episodic {uuid: $uuid})
		SET n.name = $name,
		    n.group_id = $group_id,
		    n.content = $content,
		    n.source = $source,
		    n.valid_at = datetime($valid_at),
		    n.created_at = coalesce(n.created_at, datetime())
	`
	_, err := session.Run(ctx, cypher, map[string]any{
		"uuid":     node.UUID,
		"name":     node.Name,
		"group_id": node.GroupID,
		"content":  node.Content,
		"source":   node.Source,
		"valid_at": node.ValidAt.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("neo4j: save episodic node: %w", err)
	}
	return nil
}

// SaveCommunityNode creates or updates a CommunityNode.
func (d *Driver) SaveCommunityNode(ctx context.Context, node domain.CommunityNode) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MERGE (n:Community {uuid: $uuid})
		SET n.name = $name,
		    n.group_id = $group_id,
		    n.summary = $summary,
		    n.name_embedding = $name_embedding,
		    n.level = $level,
		    n.created_at = coalesce(n.created_at, datetime()),
		    n.updated_at = datetime()
	`
	_, err := session.Run(ctx, cypher, map[string]any{
		"uuid":           node.UUID,
		"name":           node.Name,
		"group_id":       node.GroupID,
		"summary":        node.Summary,
		"name_embedding": node.NameEmbedding,
		"level":          node.Level,
	})
	if err != nil {
		return fmt.Errorf("neo4j: save community node: %w", err)
	}
	return nil
}

// SaveSagaNode creates or updates a SagaNode.
func (d *Driver) SaveSagaNode(ctx context.Context, node domain.SagaNode) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MERGE (n:Saga {uuid: $uuid})
		SET n.name = $name,
		    n.group_id = $group_id,
		    n.summary = $summary,
		    n.first_episode_uuid = $first_episode_uuid,
		    n.last_episode_uuid = $last_episode_uuid,
		    n.created_at = coalesce(n.created_at, datetime())
	`
	_, err := session.Run(ctx, cypher, map[string]any{
		"uuid":               node.UUID,
		"name":               node.Name,
		"group_id":           node.GroupID,
		"summary":            node.Summary,
		"first_episode_uuid": node.FirstEpisodeUUID,
		"last_episode_uuid":  node.LastEpisodeUUID,
	})
	if err != nil {
		return fmt.Errorf("neo4j: save saga node: %w", err)
	}
	return nil
}

// recordToEntityNode maps a Neo4j record to a domain.EntityNode.
func (d *Driver) recordToEntityNode(record *neo4j.Record) (*domain.EntityNode, error) {
	nodeVal, ok := record.Get("n")
	if !ok {
		return nil, fmt.Errorf("neo4j: missing 'n' in record")
	}

	dbNode, ok := nodeVal.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("neo4j: unexpected type for node: %T", nodeVal)
	}

	props := dbNode.Props
	node := &domain.EntityNode{
		UUID:    getStringProp(props, "uuid"),
		Name:    getStringProp(props, "name"),
		GroupID: getStringProp(props, "group_id"),
		Summary: getStringProp(props, "summary"),
	}

	if labels, ok := props["labels"].([]any); ok {
		for _, l := range labels {
			if s, ok := l.(string); ok {
				node.Labels = append(node.Labels, s)
			}
		}
	}

	if emb, ok := props["name_embedding"].([]any); ok {
		vec := make([]float32, len(emb))
		for i, v := range emb {
			if f, ok := v.(float64); ok {
				vec[i] = float32(f)
			}
		}
		node.NameEmbedding = vec
	}

	return node, nil
}

// getStringProp safely extracts a string property from a Neo4j props map.
func getStringProp(props map[string]any, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
