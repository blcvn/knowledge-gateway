package search

import "strings"

func ApplyFilters(in []Result, opts Options) []Result {
	if len(in) == 0 {
		return []Result{}
	}
	entityTypes := buildSet(opts.EntityTypes)
	domains := buildSet(opts.Domains)
	provenanceTypes := buildSet(opts.ProvenanceTypes)

	out := make([]Result, 0, len(in))
	for _, item := range in {
		if len(entityTypes) > 0 {
			entityType := item.Label
			if propType := readString(item.Properties, "entity_type"); propType != "" {
				entityType = propType
			}
			if !containsFold(entityTypes, entityType) {
				continue
			}
		}
		if len(domains) > 0 {
			domain := readString(item.Properties, "domain")
			if !containsFold(domains, domain) {
				continue
			}
		}
		if len(provenanceTypes) > 0 {
			provType := readString(item.Properties, "provenance_type")
			if !containsFold(provenanceTypes, provType) {
				continue
			}
		}
		if opts.MinConfidence > 0 {
			confidence := readFloat(item.Properties, "confidence")
			if confidence < opts.MinConfidence {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func buildSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	return out
}

func containsFold(set map[string]struct{}, value string) bool {
	if len(set) == 0 {
		return true
	}
	_, ok := set[strings.ToLower(strings.TrimSpace(value))]
	return ok
}

func readString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}

func readFloat(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	switch value := m[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}
