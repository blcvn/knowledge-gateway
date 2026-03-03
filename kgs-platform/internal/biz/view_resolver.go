package biz

import "context"

// ViewResolver shapes raw data from Neo4j into App-defined schemas.
// App-defined schemas (views) could be stored in Postgres or cached in Redis.
type ViewResolver struct {
}

// NewViewResolver builds a view resolver instance.
func NewViewResolver() *ViewResolver {
	return &ViewResolver{}
}

// Resolve shapes a raw Neo4j record into a structured JSON representation
// governed by a whitelist of allowed fields. If no view is provided, it returns all properties.
func (vr *ViewResolver) Resolve(ctx context.Context, appID string, queryResult map[string]any, viewName string) (map[string]any, error) {
	// Dummy implementation for shaping based on simple whitelists
	// In production, this would fetch the `view` definition from Postgres and traverse
	// the `queryResult` map to prune keys not in the allowed properties list.

	if viewName == "" {
		// No shaping required
		return queryResult, nil
	}

	// Mocking shaped data
	shapedData := make(map[string]any)
	for key, val := range queryResult {
		// Mock condition: include only "data" list
		if key == "data" {
			shapedData[key] = val
		}
	}

	return shapedData, nil
}
