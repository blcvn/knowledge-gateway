package read

import (
	"testing"

	"kg-service/internal/ontology"
)

func TestQueryTemplateCompilerCompilesDSL(t *testing.T) {
	compiler := NewQueryTemplateCompiler()
	compiled, err := compiler.Compile("noi_bo_hop_dong", ontology.QueryTemplate{
		TemplateName: "contract_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "HopDongMau",
				"match": map[string]any{
					"ten": "$ten_hop_dong",
				},
			},
			"hops": []any{
				map[string]any{
					"rel_type":     "THAM_CHIEU",
					"to_node_type": "KhoanMau",
				},
			},
		},
		ReturnFields: []string{"HopDongMau.ten", "KhoanMau.ten"},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if compiled.StartType != "HopDongMau" {
		t.Fatalf("StartType = %q", compiled.StartType)
	}
	if compiled.Query == "" {
		t.Fatal("Query is empty")
	}
}

func TestQueryTemplateCompilerRejectsDeepTemplates(t *testing.T) {
	compiler := NewQueryTemplateCompiler()
	hops := []any{}
	for i := 0; i < 6; i++ {
		hops = append(hops, map[string]any{
			"rel_type":     "THAM_CHIEU",
			"to_node_type": "KhoanMau",
		})
	}
	_, err := compiler.Compile("noi_bo_hop_dong", ontology.QueryTemplate{
		TemplateName: "too_deep",
		PatternSpec: map[string]any{
			"start": map[string]any{"node_type": "HopDongMau"},
			"hops":  hops,
		},
		ReturnFields: []string{"HopDongMau.ten"},
	})
	if err == nil {
		t.Fatal("Compile() error = nil, want validation failure")
	}
}
