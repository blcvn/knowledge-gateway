package neo4j

import (
	"context"
	"fmt"
	"regexp"

	"vnp-memory/services/cognee-cognify/internal/domain"
)

// GraphRepo implements Neo4j persistence for the knowledge graph.
type GraphRepo struct {
	// session neo4j.SessionWithContext  — injected via constructor in production
}

// NewGraphRepo creates a new GraphRepo.
func NewGraphRepo() *GraphRepo { return &GraphRepo{} }

// UpsertNodeWithLabels merges a node into Neo4j and SET each NodeSet tag as a Cypher label.
//
// Result: (:Concept:customer_123:preferences {id: "...", dataset_id: "...", tenant_id: "..."})
//
// Strategy:
//  1. MERGE base node by id
//  2. SET n.type label
//  3. For each NodeSet label: MATCH + SET n:label (separate queries due to Cypher limitation with params)
func (r *GraphRepo) UpsertNodeWithLabels(ctx context.Context, datasetID, tenantID string, node domain.GraphNode) error {
	// Step 1: MERGE base node with properties
	mergeQuery := fmt.Sprintf(`
		MERGE (n {id: $id})
		SET n += $props
		SET n:%s`, sanitizeLabel(node.Type))

	props := map[string]any{
		"id":         node.ID,
		"dataset_id": datasetID,
		"tenant_id":  tenantID,
		"name":       node.Name,
	}
	for k, v := range node.Properties {
		props[k] = v
	}

	// In production: run via neo4j session
	_ = mergeQuery
	_ = props

	// Step 2: SET labels for each NodeSet tag (separate queries — Cypher param limitation)
	for _, label := range node.Labels {
		if label == node.Type {
			continue // already set in Step 1
		}
		labelQuery := fmt.Sprintf(`MATCH (n {id: $id}) SET n:%s`, sanitizeLabel(label))
		// In production: r.session.Run(ctx, labelQuery, map[string]any{"id": node.ID})
		_ = labelQuery
	}

	return nil
}

// sanitizeLabel ensures a label string is a valid Cypher identifier.
// Replaces non-alphanumeric-underscore chars, and prefixes digit-starting labels with _.
func sanitizeLabel(tag string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	sanitized := re.ReplaceAllString(tag, "_")
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized // prefix digit-leading labels
	}
	return sanitized
}
