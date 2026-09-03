package biz

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// QueryPlanner generates safe, namespaced Cypher queries.
type QueryPlanner struct{}

func NewQueryPlanner() *QueryPlanner {
	return &QueryPlanner{}
}

func (qp *QueryPlanner) BuildContextQuery(label string, direction string) string {
	lbl := toLabel(label)
	dirPattern := "-[r]-"
	if direction == "INCOMING" {
		dirPattern = "<-[r]-"
	} else if direction == "OUTGOING" {
		dirPattern = "-[r]->"
	}

	return fmt.Sprintf(`
		MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})%s(m%s {app_id: $app_id, tenant_id: $tenant_id})
		RETURN n, r, m
	`, dirPattern, lbl)
}

func (qp *QueryPlanner) BuildImpactQuery(label string, maxDepth int) string {
	lbl := toLabel(label)
	return fmt.Sprintf(`
		MATCH p=(n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})-[*1..%d]->(m%s {app_id: $app_id, tenant_id: $tenant_id})
		RETURN nodes(p) AS nodes, relationships(p) AS rels
	`, maxDepth, lbl)
}

func (qp *QueryPlanner) BuildCoverageQuery(label string, maxDepth int) string {
	lbl := toLabel(label)
	return fmt.Sprintf(`
		MATCH p=(n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})<-[*1..%d]-(m%s {app_id: $app_id, tenant_id: $tenant_id})
		RETURN nodes(p) AS nodes, relationships(p) AS rels
	`, maxDepth, lbl)
}

func (qp *QueryPlanner) BuildSubgraphQuery() string {
	return `
		MATCH (n {app_id: $app_id, tenant_id: $tenant_id})-[r]->(m {app_id: $app_id, tenant_id: $tenant_id})
		WHERE n.id IN $node_ids AND m.id IN $node_ids
		RETURN n, r, m
	`
}

// BuildBatchedTraversalQueries creates depth-windowed queries for depth > 3 traversal.
func (qp *QueryPlanner) BuildBatchedTraversalQueries(kind, label, direction string, depth, batchWindow int) []string {
	if depth <= 0 {
		return nil
	}
	if batchWindow <= 0 {
		batchWindow = 3
	}
	queries := make([]string, 0, (depth+batchWindow-1)/batchWindow)
	for start := 1; start <= depth; start += batchWindow {
		end := start + batchWindow - 1
		if end > depth {
			end = depth
		}
		switch kind {
		case "impact":
			queries = append(queries, qp.buildImpactDepthRange(label, start, end))
		case "coverage":
			queries = append(queries, qp.buildCoverageDepthRange(label, start, end))
		default:
			queries = append(queries, qp.buildContextDepthRange(label, direction, start, end))
		}
	}
	return queries
}

func (qp *QueryPlanner) buildContextDepthRange(label, direction string, minDepth, maxDepth int) string {
	lbl := toLabel(label)
	pattern := "-[*%d..%d]-"
	if direction == "INCOMING" {
		pattern = "<-[*%d..%d]-"
	} else if direction == "OUTGOING" {
		pattern = "-[*%d..%d]->"
	}
	return fmt.Sprintf(`
		MATCH p=(n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})`+pattern+`(m%s {app_id: $app_id, tenant_id: $tenant_id})
		RETURN nodes(p) AS nodes, relationships(p) AS rels
	`, minDepth, maxDepth, lbl)
}

func (qp *QueryPlanner) buildImpactDepthRange(label string, minDepth, maxDepth int) string {
	lbl := toLabel(label)
	return fmt.Sprintf(`
		MATCH p=(n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})-[*%d..%d]->(m%s {app_id: $app_id, tenant_id: $tenant_id})
		RETURN nodes(p) AS nodes, relationships(p) AS rels
	`, minDepth, maxDepth, lbl)
}

func (qp *QueryPlanner) buildCoverageDepthRange(label string, minDepth, maxDepth int) string {
	lbl := toLabel(label)
	return fmt.Sprintf(`
		MATCH p=(n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})<-[*%d..%d]-(m%s {app_id: $app_id, tenant_id: $tenant_id})
		RETURN nodes(p) AS nodes, relationships(p) AS rels
	`, minDepth, maxDepth, lbl)
}

func EncodePageToken(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func DecodePageToken(token string) (int, error) {
	if strings.TrimSpace(token) == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(decoded))
}

func toLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	return ":" + label
}
