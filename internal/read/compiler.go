package read

import (
	"errors"
	"fmt"
	"strings"

	"kg-service/internal/ontology"
)

var ErrTemplateTooDeep = errors.New("template too deep")

type CompiledTemplate struct {
	DomainID     string
	TemplateName string
	StartType    string
	StartMatch   map[string]any
	Hops         []CompiledHop
	ReturnFields []string
	Query        string
}

type CompiledHop struct {
	RelType      string
	ToNodeType   string
	Direction    string
	Filter       map[string]any
	FilterStatus string
}

type QueryTemplateCompiler struct{}

func NewQueryTemplateCompiler() QueryTemplateCompiler {
	return QueryTemplateCompiler{}
}

func (c QueryTemplateCompiler) Compile(domainID string, template ontology.QueryTemplate) (CompiledTemplate, error) {
	start, _ := template.PatternSpec["start"].(map[string]any)
	startType, _ := start["node_type"].(string)
	startMatch, _ := start["match"].(map[string]any)

	hopsRaw, _ := template.PatternSpec["hops"].([]any)
	if len(hopsRaw) > 5 {
		return CompiledTemplate{}, errors.Join(ErrValidation, fmt.Errorf("pattern_spec has %d hops, exceeds limit 5", len(hopsRaw)))
	}

	hops := make([]CompiledHop, 0, len(hopsRaw))
	var query strings.Builder
	query.WriteString(fmt.Sprintf("MATCH (n0:%s)", startType))
	query.WriteString(" WHERE ANY(tok IN n0.acl_visible_to WHERE tok IN $acl_tokens)")
	for i, hopRaw := range hopsRaw {
		hop, ok := hopRaw.(map[string]any)
		if !ok {
			return CompiledTemplate{}, errors.Join(ErrValidation, errors.New("pattern_spec.hops entries must be objects"))
		}
		relType, _ := hop["rel_type"].(string)
		toNodeType, _ := hop["to_node_type"].(string)
		direction, _ := hop["direction"].(string)
		filter, _ := hop["filter"].(map[string]any)
		filterStatus, _ := hop["filter_status"].(string)
		hops = append(hops, CompiledHop{
			RelType:      relType,
			ToNodeType:   toNodeType,
			Direction:    direction,
			Filter:       filter,
			FilterStatus: filterStatus,
		})
		query.WriteString(fmt.Sprintf(" -[:%s]-> (n%d:%s) WHERE ANY(tok IN n%d.acl_visible_to WHERE tok IN $acl_tokens)", relType, i+1, toNodeType, i+1))
	}

	return CompiledTemplate{
		DomainID:     domainID,
		TemplateName: template.TemplateName,
		StartType:    startType,
		StartMatch:   startMatch,
		Hops:         hops,
		ReturnFields: template.ReturnFields,
		Query:        query.String(),
	}, nil
}
