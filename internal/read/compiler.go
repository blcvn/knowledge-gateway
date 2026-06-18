package read

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"kg-service/internal/ontology"
	"kg-service/internal/platform/graphstore"
)

var ErrTemplateTooDeep = errors.New("template too deep")

type QueryStrategyHandler func(base graphstore.GraphQuery, strategy ontology.QueryStrategy) (graphstore.GraphQuery, error)

var (
	queryStrategyHandlers   = map[string]QueryStrategyHandler{}
	queryStrategyHandlersMu sync.RWMutex
)

func init() {
	RegisterQueryStrategyHandler("default", func(base graphstore.GraphQuery, strategy ontology.QueryStrategy) (graphstore.GraphQuery, error) {
		if strategy.MaxDepth <= 0 {
			strategy.MaxDepth = 5
		}
		base.MaxDepth = strategy.MaxDepth
		base.Strategy = "default"
		return base, nil
	})
	RegisterQueryStrategyHandler("deep_traversal", func(base graphstore.GraphQuery, strategy ontology.QueryStrategy) (graphstore.GraphQuery, error) {
		if strategy.MaxDepth <= 0 {
			strategy.MaxDepth = 10
		}
		base.MaxDepth = strategy.MaxDepth
		base.Strategy = "deep_traversal"
		return base, nil
	})
}

func RegisterQueryStrategyHandler(key string, handler QueryStrategyHandler) {
	queryStrategyHandlersMu.Lock()
	defer queryStrategyHandlersMu.Unlock()
	queryStrategyHandlers[key] = handler
}

type CompiledTemplate struct {
	DomainID     string
	TemplateName string
	StartType    string
	StartMatch   map[string]any
	Hops         []CompiledHop
	ReturnFields []string
	Query        string
	GraphQuery   graphstore.GraphQuery
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

func (c QueryTemplateCompiler) Compile(domainID string, template ontology.QueryTemplate, strategy ontology.QueryStrategy) (CompiledTemplate, error) {
	start, _ := template.PatternSpec["start"].(map[string]any)
	startType, _ := start["node_type"].(string)
	startMatch, _ := start["match"].(map[string]any)

	hopsRaw, _ := template.PatternSpec["hops"].([]any)
	if strategy.MaxDepth <= 0 {
		strategy.MaxDepth = 5
	}
	if len(hopsRaw) > strategy.MaxDepth {
		return CompiledTemplate{}, errors.Join(ErrTemplateTooDeep, fmt.Errorf("pattern_spec has %d hops, exceeds limit %d", len(hopsRaw), strategy.MaxDepth))
	}

	hops := make([]CompiledHop, 0, len(hopsRaw))
	graphQuery := graphstore.GraphQuery{
		StartNodeType:  startType,
		StartMatch:     startMatch,
		ReturnFields:   append([]string(nil), template.ReturnFields...),
		ACLTokensParam: "acl_tokens",
		MaxDepth:       strategy.MaxDepth,
		Strategy:       strategy.Key,
	}
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
		graphQuery.Hops = append(graphQuery.Hops, graphstore.GraphQueryHop{
			RelType:      relType,
			ToNodeType:   toNodeType,
			Direction:    direction,
			Filter:       filter,
			FilterStatus: filterStatus,
		})
		query.WriteString(fmt.Sprintf(" -[:%s]-> (n%d:%s) WHERE ANY(tok IN n%d.acl_visible_to WHERE tok IN $acl_tokens)", relType, i+1, toNodeType, i+1))
	}

	queryStrategyHandlersMu.RLock()
	handler, ok := queryStrategyHandlers[graphQuery.Strategy]
	queryStrategyHandlersMu.RUnlock()
	if !ok {
		queryStrategyHandlersMu.RLock()
		handler = queryStrategyHandlers["default"]
		queryStrategyHandlersMu.RUnlock()
	}
	if handler != nil {
		translated, err := handler(graphQuery, strategy)
		if err != nil {
			return CompiledTemplate{}, err
		}
		graphQuery = translated
	}

	return CompiledTemplate{
		DomainID:     domainID,
		TemplateName: template.TemplateName,
		StartType:    startType,
		StartMatch:   startMatch,
		Hops:         hops,
		ReturnFields: template.ReturnFields,
		Query:        query.String(),
		GraphQuery:   graphQuery,
	}, nil
}
