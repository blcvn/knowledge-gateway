package prompt

// JSON schemas for structured LLM output — used with response_format: json_schema

var ExtractedNodeListSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"entities": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":    map[string]interface{}{"type": "string"},
					"label":   map[string]interface{}{"type": "string"},
					"summary": map[string]interface{}{"type": "string"},
				},
				"required": []string{"name", "label"},
			},
		},
	},
	"required": []string{"entities"},
}

var ExtractedEdgeListSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"edges": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source_entity": map[string]interface{}{"type": "string"},
					"target_entity": map[string]interface{}{"type": "string"},
					"relation_type": map[string]interface{}{"type": "string"},
					"fact":          map[string]interface{}{"type": "string"},
					"valid_at":      map[string]interface{}{"type": []string{"string", "null"}},
					"invalid_at":    map[string]interface{}{"type": []string{"string", "null"}},
				},
				"required": []string{"source_entity", "target_entity", "relation_type", "fact"},
			},
		},
	},
	"required": []string{"edges"},
}

var EntityResolutionSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"decision":      map[string]interface{}{"type": "string", "enum": []string{"merge", "new"}},
		"existing_uuid": map[string]interface{}{"type": "string"},
		"reasoning":     map[string]interface{}{"type": "string"},
	},
	"required": []string{"decision"},
}

var EdgeResolutionSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"resolution": map[string]interface{}{
			"type": "string",
			"enum": []string{"DUPLICATE", "NEW", "CONTRADICTION", "UPDATE"},
		},
		"invalidated_edge_uuids": map[string]interface{}{
			"type":  "array",
			"items": map[string]interface{}{"type": "string"},
		},
		"reasoning": map[string]interface{}{"type": "string"},
	},
	"required": []string{"resolution"},
}

var NodeSummarySchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"summary": map[string]interface{}{"type": "string"},
	},
	"required": []string{"summary"},
}

var SagaSummarySchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"summary": map[string]interface{}{"type": "string"},
		"title":   map[string]interface{}{"type": "string"},
	},
	"required": []string{"summary"},
}
