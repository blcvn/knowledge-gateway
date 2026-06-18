package read

import (
	"errors"
	"testing"

	"kg-service/internal/ontology"
	"kg-service/internal/platform/graphstore"
)

func TestQueryTemplateCompilerUsesStrategyHandlers(t *testing.T) {
	compiler := NewQueryTemplateCompiler()
	template := ontology.QueryTemplate{
		TemplateName: "sample",
		PatternSpec: map[string]any{
			"start": map[string]any{"node_type": "Doc", "match": map[string]any{"title": "alpha"}},
			"hops": []any{
				map[string]any{"rel_type": "LINKS", "to_node_type": "Doc", "direction": "out"},
			},
		},
		ReturnFields: []string{"id"},
	}

	compiled, err := compiler.Compile("domain-1", template, ontology.QueryStrategy{Key: "default", MaxDepth: 5})
	if err != nil {
		t.Fatalf("Compile(default) error = %v", err)
	}
	if compiled.GraphQuery.MaxDepth != 5 || compiled.GraphQuery.Strategy != "default" {
		t.Fatalf("default GraphQuery = %#v", compiled.GraphQuery)
	}

	compiled, err = compiler.Compile("domain-1", template, ontology.QueryStrategy{Key: "deep_traversal", MaxDepth: 10})
	if err != nil {
		t.Fatalf("Compile(deep_traversal) error = %v", err)
	}
	if compiled.GraphQuery.MaxDepth != 10 || compiled.GraphQuery.Strategy != "deep_traversal" {
		t.Fatalf("deep_traversal GraphQuery = %#v", compiled.GraphQuery)
	}

	RegisterQueryStrategyHandler("finance_deep", func(base graphstore.GraphQuery, strategy ontology.QueryStrategy) (graphstore.GraphQuery, error) {
		base.MaxDepth = strategy.MaxDepth
		base.Strategy = strategy.Key
		return base, nil
	})
	compiled, err = compiler.Compile("domain-1", template, ontology.QueryStrategy{Key: "finance_deep", MaxDepth: 8})
	if err != nil {
		t.Fatalf("Compile(custom) error = %v", err)
	}
	if compiled.GraphQuery.MaxDepth != 8 || compiled.GraphQuery.Strategy != "finance_deep" {
		t.Fatalf("custom GraphQuery = %#v", compiled.GraphQuery)
	}
}

func TestQueryTemplateCompilerRejectsTooDeepTemplates(t *testing.T) {
	compiler := NewQueryTemplateCompiler()
	template := ontology.QueryTemplate{
		TemplateName: "too-deep",
		PatternSpec: map[string]any{
			"start": map[string]any{"node_type": "Doc"},
			"hops": []any{
				map[string]any{"rel_type": "LINKS", "to_node_type": "Doc"},
				map[string]any{"rel_type": "LINKS", "to_node_type": "Doc"},
			},
		},
	}

	_, err := compiler.Compile("domain-1", template, ontology.QueryStrategy{Key: "default", MaxDepth: 1})
	if err == nil {
		t.Fatal("Compile() error = nil, want too-deep error")
	}
	if !errors.Is(err, ErrTemplateTooDeep) {
		t.Fatalf("error = %v, want ErrTemplateTooDeep", err)
	}
}
