package neo4j

import (
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

func mapRecordToEntityNode(rec port.Record) (*graph.EntityNode, error) {
	nodeVal, ok := rec.Values[0].(neo4j.Node)
	if !ok {
		return nil, nil
	}
	props := nodeVal.Props

	node := &graph.EntityNode{
		UUID:    getString(props, "uuid"),
		Name:    getString(props, "name"),
		Summary: getString(props, "summary"),
		GroupID: getString(props, "group_id"),
	}
	if labels, ok := props["labels"].([]any); ok {
		for _, l := range labels {
			if s, ok := l.(string); ok {
				node.Labels = append(node.Labels, s)
			}
		}
	}
	if emb, ok := props["name_embedding"].([]any); ok {
		node.NameEmbedding = toFloat32Slice(emb)
	}
	node.CreatedAt = parseTime(props["created_at"])
	node.UpdatedAt = parseTime(props["updated_at"])
	return node, nil
}

func mapRecordToEntityEdge(rec port.Record) (*graph.EntityEdge, error) {
	relVal, ok := rec.Values[0].(neo4j.Relationship)
	if !ok {
		return nil, nil
	}
	props := relVal.Props

	srcUUID := ""
	tgtUUID := ""
	if len(rec.Values) > 1 {
		srcUUID, _ = rec.Values[1].(string)
	}
	if len(rec.Values) > 2 {
		tgtUUID, _ = rec.Values[2].(string)
	}

	edge := &graph.EntityEdge{
		UUID:           getString(props, "uuid"),
		Name:           getString(props, "name"),
		Fact:           getString(props, "fact"),
		GroupID:        getString(props, "group_id"),
		SourceNodeUUID: srcUUID,
		TargetNodeUUID: tgtUUID,
	}
	if emb, ok := props["fact_embedding"].([]any); ok {
		edge.FactEmbedding = toFloat32Slice(emb)
	}
	if eps, ok := props["episodes"].([]any); ok {
		for _, e := range eps {
			if s, ok := e.(string); ok {
				edge.Episodes = append(edge.Episodes, s)
			}
		}
	}
	if v := parseTimePtr(props["valid_at"]); v != nil {
		edge.ValidAt = v
	}
	if v := parseTimePtr(props["invalid_at"]); v != nil {
		edge.InvalidAt = v
	}
	if v := parseTimePtr(props["expired_at"]); v != nil {
		edge.ExpiredAt = v
	}
	edge.CreatedAt = parseTime(props["created_at"])
	edge.UpdatedAt = parseTime(props["updated_at"])
	return edge, nil
}

func mapRecordToEpisodicNode(rec port.Record) (*graph.EpisodicNode, error) {
	nodeVal, ok := rec.Values[0].(neo4j.Node)
	if !ok {
		return nil, nil
	}
	props := nodeVal.Props
	return &graph.EpisodicNode{
		UUID:              getString(props, "uuid"),
		Name:              getString(props, "name"),
		Content:           getString(props, "content"),
		Source:            graph.EpisodeType(getString(props, "source")),
		SourceDescription: getString(props, "source_description"),
		GroupID:           getString(props, "group_id"),
		ValidAt:           parseTime(props["valid_at"]),
		CreatedAt:         parseTime(props["created_at"]),
	}, nil
}

func mapRecordToCommunityNode(rec port.Record) (*graph.CommunityNode, error) {
	nodeVal, ok := rec.Values[0].(neo4j.Node)
	if !ok {
		return nil, nil
	}
	props := nodeVal.Props
	node := &graph.CommunityNode{
		UUID:      getString(props, "uuid"),
		Name:      getString(props, "name"),
		Summary:   getString(props, "summary"),
		GroupID:   getString(props, "group_id"),
		CreatedAt: parseTime(props["created_at"]),
	}
	if emb, ok := props["name_embedding"].([]any); ok {
		node.NameEmbedding = toFloat32Slice(emb)
	}
	return node, nil
}

// Helper functions
func getString(v any, def string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

func toFloat32Slice(vals []any) []float32 {
	result := make([]float32, 0, len(vals))
	for _, v := range vals {
		switch f := v.(type) {
		case float32:
			result = append(result, f)
		case float64:
			result = append(result, float32(f))
		}
	}
	return result
}

func parseTime(v any) time.Time {
	if t, ok := v.(time.Time); ok {
		return t
	}
	if neo4jTime, ok := v.(neo4j.LocalDateTime); ok {
		return neo4jTime.Time()
	}
	return time.Time{}
}

func parseTimePtr(v any) *time.Time {
	if v == nil {
		return nil
	}
	t := parseTime(v)
	if t.IsZero() {
		return nil
	}
	return &t
}
