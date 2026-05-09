package surrealdb

import (
	"context"
	"fmt"
	"regexp"

	"github.com/go-kratos/kratos/v2/log"
)

// surrealAnalyticsExecutor implements analytics.QueryExecutor using SurrealDB.
// Translates analytics-specific Cypher queries to SurrealQL equivalents.
type surrealAnalyticsExecutor struct {
	client *Client
	log    *log.Helper
}

func NewSurrealAnalyticsExecutor(client *Client, logger log.Logger) *surrealAnalyticsExecutor {
	return &surrealAnalyticsExecutor{
		client: client,
		log:    log.NewHelper(logger),
	}
}

func (r *surrealAnalyticsExecutor) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	surql, err := translateAnalyticsCypher(cypher, params)
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] Analytics translate failed cypher=%q err=%v", truncate(cypher, 100), err)
		return nil, fmt.Errorf("analytics cypher translation: %w", err)
	}

	result, err := r.client.Query(ctx, surql, params)
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] Analytics query failed surql=%q err=%v", truncate(surql, 100), err)
		return nil, err
	}

	rows, err := unmarshalSlice[map[string]any](result)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": rows}, nil
}

// ── Analytics Cypher → SurrealQL Translators ──────────────────

var (
	coveragePattern_   = regexp.MustCompile(`(?i)MATCH\s+\(n\s*\{app_id.*\}\).*OPTIONAL MATCH.*count\(r\)`)
	tracePattern_      = regexp.MustCompile(`(?i)MATCH\s+p=.*\[\*1\.\.(\d+)\].*source_types`)
	clusterPattern_    = regexp.MustCompile(`(?i)CALL\s+gds\.louvain`)
)

func translateAnalyticsCypher(cypher string, params map[string]any) (string, error) {
	// Pattern 1: Coverage report — count entities by type, check outgoing edges
	if coveragePattern_.MatchString(cypher) {
		return buildCoverageAnalyticsSurrealQL(params), nil
	}

	// Pattern 2: Traceability matrix — multi-hop path between source→target types
	if matches := tracePattern_.FindStringSubmatch(cypher); len(matches) >= 2 {
		var maxHops int
		fmt.Sscanf(matches[1], "%d", &maxHops)
		if maxHops <= 0 {
			maxHops = 5
		}
		return buildTraceabilityAnalyticsSurrealQL(params, maxHops), nil
	}

	// Pattern 3: Cluster analysis — Louvain community detection
	// SurrealDB doesn't have GDS. Use connected-component approximation.
	if clusterPattern_.MatchString(cypher) {
		return buildClusterAnalyticsSurrealQL(params), nil
	}

	return "", fmt.Errorf("unsupported analytics cypher pattern: %s", truncate(cypher, 200))
}

// ── Coverage: count entities grouped by type, track which have outgoing edges ──

func buildCoverageAnalyticsSurrealQL(params map[string]any) string {
	return `
LET $entities = (SELECT entity_type, entity_id FROM kg_entities
	WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
	AND ($domain = '' OR $domain IN domains OR domain = $domain));
LET $entity_ids_with_edges = (SELECT VALUE from_entity_id FROM kg_edges
	WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
	GROUP from_entity_id);
SELECT
	entity_type,
	count() AS total_entities,
	count(entity_id IN $entity_ids_with_edges) AS covered_entities
FROM $entities
GROUP BY entity_type
ORDER BY entity_type ASC;`
}

// ── Traceability: multi-hop path between source-type nodes and target-type nodes ──

func buildTraceabilityAnalyticsSurrealQL(params map[string]any, maxHops int) string {
	return fmt.Sprintf(`
LET $sources = (SELECT entity_id, name, entity_type FROM kg_entities
	WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
	AND entity_type IN $source_types);
LET $results = [];
FOR $src IN $sources {
	LET $visited = [$src.entity_id];
	LET $frontier = [$src.entity_id];
	LET $hop = 0;
	LET $paths = [];

	REPEAT {
		LET $hop = $hop + 1;
		LET $new_edges = (SELECT from_entity_id, to_entity_id, relation_type FROM kg_edges
			WHERE from_entity_id IN $frontier AND app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
			AND to_entity_id NOT IN $visited);
		LET $new_ids = (SELECT VALUE to_entity_id FROM $new_edges);
		LET $targets_found = (SELECT entity_id, name, entity_type FROM kg_entities
			WHERE entity_id IN $new_ids AND entity_type IN $target_types AND is_deleted = false);

		FOR $t IN $targets_found {
			LET $paths = array::push($paths, {
				source_id: $src.entity_id,
				source_name: $src.name,
				source_type: $src.entity_type,
				target_id: $t.entity_id,
				target_name: $t.name,
				target_type: $t.entity_type,
				hops: $hop,
				path: []
			});
		};

		LET $visited = array::union($visited, $new_ids);
		LET $frontier = $new_ids;
	} WHILE $hop < %d AND array::len($frontier) > 0;

	LET $results = array::union($results, $paths);
};
RETURN $results;`, maxHops)
}

// ── Cluster analysis: approximate connected components ──
// SurrealDB doesn't have GDS Louvain. Use edge-based grouping as an approximation.

func buildClusterAnalyticsSurrealQL(params map[string]any) string {
	return `
LET $nodes = (SELECT entity_id FROM kg_entities
	WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
	AND ($entity_type = '' OR entity_type = $entity_type));
LET $edges = (SELECT from_entity_id, to_entity_id FROM kg_edges
	WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false);

-- Approximate clustering via shared-edge grouping
-- Group nodes that share at least one edge
SELECT
	math::min([from_entity_id, to_entity_id]) AS community_id,
	array::distinct(array::union(
		(SELECT VALUE from_entity_id FROM kg_edges WHERE to_entity_id = $parent.to_entity_id AND is_deleted = false),
		(SELECT VALUE to_entity_id FROM kg_edges WHERE from_entity_id = $parent.from_entity_id AND is_deleted = false)
	)) AS node_ids,
	count() AS size
FROM $edges
GROUP BY community_id
ORDER BY size DESC;`
}
