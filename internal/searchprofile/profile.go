package searchprofile

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"kg-service/internal/ontology"
	"kg-service/internal/write"
)

func BuildEmbeddingText(node write.NodeRecord, profile ontology.ResolvedSearchProfile) string {
	fields := profile.SemanticFields
	if len(fields) == 0 {
		fields = defaultFields()
	}

	parts := make([]string, 0, len(fields)*2)
	for _, field := range fields {
		switch field.FieldName {
		case "id":
			appendWeighted(&parts, field, node.ID)
		case "node_type":
			appendWeighted(&parts, field, node.NodeType)
		case "domain_id":
			appendWeighted(&parts, field, node.DomainID)
		case "external_ref":
			appendWeighted(&parts, field, node.ExternalRef)
		case "status_value":
			appendWeighted(&parts, field, node.StatusValue)
		case "*properties":
			keys := make([]string, 0, len(node.Properties))
			for k := range node.Properties {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				v := node.Properties[k]
				appendWeightedRaw(&parts, field, k)
				appendWeightedRaw(&parts, field, fmt.Sprintf("%v", v))
			}
		default:
			if value, ok := node.Properties[field.FieldName]; ok {
				appendWeightedRaw(&parts, field, fmt.Sprintf("%v", value))
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func defaultFields() []ontology.IndexedField {
	return []ontology.IndexedField{
		{FieldName: "id", Weight: 1.0},
		{FieldName: "node_type", Weight: 1.0},
		{FieldName: "domain_id", Weight: 1.0},
		{FieldName: "external_ref", Weight: 1.0},
		{FieldName: "status_value", Weight: 1.0},
		{FieldName: "*properties", Weight: 1.0},
	}
}

func appendWeighted(parts *[]string, field ontology.IndexedField, value string) {
	appendWeightedRaw(parts, field, value)
}

func appendWeightedRaw(parts *[]string, field ontology.IndexedField, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	repeat := int(math.Ceil(field.Weight))
	if repeat < 1 {
		repeat = 1
	}
	prefix := strings.TrimSpace(field.Prefix)
	for i := 0; i < repeat; i++ {
		if prefix != "" {
			*parts = append(*parts, prefix+":")
		}
		*parts = append(*parts, value)
	}
}

func ParseFloatOrDefault(raw string, fallback float64) float64 {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}
