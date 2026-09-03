package surrealdb

import (
	"encoding/json"
	"fmt"
)

// unmarshalSlice attempts to extract a slice of T from SurrealDB query results.
// SurrealDB returns results as []interface{} containing map[string]interface{}.
func unmarshalSlice[T any](raw any) ([]T, error) {
	if raw == nil {
		return nil, nil
	}

	// SurrealDB query results come as []interface{} of result sets
	resultSets, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", raw)
	}

	var out []T
	for _, rs := range resultSets {
		rsMap, ok := rs.(map[string]interface{})
		if !ok {
			continue
		}
		status, _ := rsMap["status"].(string)
		if status != "OK" {
			continue
		}
		result, ok := rsMap["result"]
		if !ok || result == nil {
			continue
		}

		// Marshal back to JSON then unmarshal to target type
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal result: %w", err)
		}

		var items []T
		if err := json.Unmarshal(data, &items); err != nil {
			// Try single item
			var item T
			if err2 := json.Unmarshal(data, &item); err2 != nil {
				return nil, fmt.Errorf("unmarshal result: %w", err)
			}
			items = []T{item}
		}
		out = append(out, items...)
	}
	return out, nil
}

// unmarshalOne extracts a single T from SurrealDB query results.
func unmarshalOne[T any](raw any) (*T, error) {
	items, err := unmarshalSlice[T](raw)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}
