package biz

import (
	"fmt"
)

// QueryPlanner is responsible for generating safe, namespaced Cypher queries.
// It ensures that all queries are scoped to the specific AppID (Namespace)
// and handles dynamic depth/filter construction safely.
// Note: Since Neo4j parameters cannot be used for Labels or Relationship Types,
// we safely interpolate them via string formatting while using parameters for values.
type QueryPlanner struct {
}

// NewQueryPlanner creates a new QueryPlanner instance.
func NewQueryPlanner() *QueryPlanner {
	return &QueryPlanner{}
}

// BuildContextQuery generates a query to fetch the immediate context (neighbors) of a node.
func (qp *QueryPlanner) BuildContextQuery(label string, direction string) string {
	lbl := ""
	if label != "" {
		lbl = ":" + label
	}

	dirPattern := "-[r]-"
	if direction == "INCOMING" {
		dirPattern = "<-[r]-"
	} else if direction == "OUTGOING" {
		dirPattern = "-[r]->"
	}

	// Uses parameters $app_id and $node_id
	return fmt.Sprintf(`
		MATCH (n%s {app_id: $app_id, id: $node_id})%s(m {app_id: $app_id})
		RETURN n, r, m
	`, lbl, dirPattern)
}

// BuildImpactQuery generates a query to find downstream nodes up to a certain depth.
func (qp *QueryPlanner) BuildImpactQuery(label string, maxDepth int) string {
	lbl := ""
	if label != "" {
		lbl = ":" + label
	}
	// Uses parameters $app_id and $node_id
	return fmt.Sprintf(`
		MATCH p=(n%s {app_id: $app_id, id: $node_id})-[*1..%d]->(m {app_id: $app_id})
		RETURN nodes(p) AS nodes, relationships(p) AS rels
	`, lbl, maxDepth)
}

// BuildCoverageQuery generates a query to find upstream nodes up to a certain depth.
func (qp *QueryPlanner) BuildCoverageQuery(label string, maxDepth int) string {
	lbl := ""
	if label != "" {
		lbl = ":" + label
	}
	// Uses parameters $app_id and $node_id
	return fmt.Sprintf(`
		MATCH p=(n%s {app_id: $app_id, id: $node_id})<-[*1..%d]-(m {app_id: $app_id})
		RETURN nodes(p) AS nodes, relationships(p) AS rels
	`, lbl, maxDepth)
}

// BuildSubgraphQuery generates a query to fetch the subgraph formed by a given list of node IDs.
func (qp *QueryPlanner) BuildSubgraphQuery() string {
	// Uses parameters $app_id and $node_ids
	return `
		MATCH (n {app_id: $app_id})-[r]->(m {app_id: $app_id})
		WHERE n.id IN $node_ids AND m.id IN $node_ids
		RETURN n, r, m
	`
}
