package analytics

import (
	"context"
	"errors"
	"sort"
)

const clusterQuery = `
CALL gds.louvain.stream($graph_name)
YIELD nodeId, communityId
WITH gds.util.asNode(nodeId) AS n, communityId
WHERE n.app_id = $app_id
  AND n.tenant_id = $tenant_id
  AND ($entity_type = '' OR $entity_type IN labels(n))
RETURN communityId AS community_id,
       collect(n.id) AS node_ids,
       count(*) AS size
ORDER BY size DESC
`

func (e *Engine) ClusterAnalysis(ctx context.Context, namespace, entityType string) (*ClusterReport, error) {
	if e == nil || e.query == nil {
		return nil, errors.New("analytics engine is not configured")
	}

	params := map[string]any{"entity_type": entityType}
	var cached ClusterReport
	if e.cache != nil {
		if ok, err := e.cache.Get(ctx, "cluster", namespace, params, &cached); err == nil && ok {
			return &cached, nil
		}
	}

	appID, tenantID, err := splitNamespace(namespace)
	if err != nil {
		return nil, err
	}

	result, err := e.query.ExecuteQuery(ctx, clusterQuery, map[string]any{
		"graph_name":  "kgs-graph-" + namespace,
		"app_id":      appID,
		"tenant_id":   tenantID,
		"entity_type": entityType,
	})
	if err != nil {
		return nil, err
	}

	report := &ClusterReport{EntityType: entityType, Clusters: make([]Cluster, 0)}
	for _, row := range toRows(result) {
		cluster := Cluster{
			CommunityID: asInt64(row["community_id"]),
			NodeIDs:     asStringSlice(row["node_ids"]),
			Size:        asInt(row["size"]),
		}
		sort.Strings(cluster.NodeIDs)
		report.Clusters = append(report.Clusters, cluster)
	}

	if e.cache != nil {
		_ = e.cache.Set(ctx, "cluster", namespace, params, report)
	}
	return report, nil
}
