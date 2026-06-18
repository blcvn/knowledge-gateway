package bootstrap

import (
	"testing"
	"time"

	"kg-service/internal/config"
)

func TestEmbeddingWiringChain(t *testing.T) {
	cfg := config.Config{
		Embedding: config.EmbeddingConfig{
			Provider: "http",
			URL:      "http://embedding.local",
			ProxyURL: "http://proxy.local",
			CacheTTL: time.Minute,
		},
	}
	labels := embeddingChain(cfg)
	want := []string{"cache", "retry", "proxy", "http"}
	if len(labels) != len(want) {
		t.Fatalf("labels len = %d, want %d", len(labels), len(want))
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("labels[%d] = %q, want %q", i, labels[i], want[i])
		}
	}
	router, err := buildEmbeddingRouter(cfg)
	if err != nil {
		t.Fatalf("buildEmbeddingRouter() error = %v", err)
	}
	if router == nil || router.ModelID() != "" {
		t.Fatalf("router = %#v, want configured router", router)
	}
}

func TestAdapterSelectionDefaults(t *testing.T) {
	baseCfg := config.Config{
		Vector: config.AdapterConfig{
			Endpoint:   "http://vector.local:6333",
			Collection: "kg_vectors",
		},
		Graph: config.AdapterConfig{
			Endpoint: "bolt://graph.local:7687",
		},
	}

	vectorAdapter, err := buildVectorAdapter(baseCfg, nil)
	if err != nil {
		t.Fatalf("buildVectorAdapter(memory) error = %v", err)
	}
	if vectorAdapter == nil {
		t.Fatal("vector adapter is nil")
	}
	pgCfg := baseCfg
	pgCfg.Vector.Kind = "pgvector"
	pgVectorAdapter, err := buildVectorAdapter(pgCfg, nil)
	if err != nil {
		t.Fatalf("buildVectorAdapter(pgvector) error = %v", err)
	}
	if pgVectorAdapter == nil {
		t.Fatal("pgvector adapter is nil")
	}
	qdrantCfg := baseCfg
	qdrantCfg.Vector.Kind = "qdrant"
	qdrantAdapter, err := buildVectorAdapter(qdrantCfg, nil)
	if err != nil {
		t.Fatalf("buildVectorAdapter(qdrant) error = %v", err)
	}
	if qdrantAdapter == nil {
		t.Fatal("qdrant adapter is nil")
	}
	milvusCfg := baseCfg
	milvusCfg.Vector.Kind = "milvus"
	milvusAdapter, err := buildVectorAdapter(milvusCfg, nil)
	if err != nil {
		t.Fatalf("buildVectorAdapter(milvus) error = %v", err)
	}
	if milvusAdapter == nil {
		t.Fatal("milvus adapter is nil")
	}
	graphAdapter, err := buildGraphAdapter(baseCfg)
	if err != nil {
		t.Fatalf("buildGraphAdapter(memory) error = %v", err)
	}
	if graphAdapter == nil {
		t.Fatal("graph adapter is nil")
	}
	for _, kind := range []string{"neo4j", "memgraph", "nebula"} {
		graphCfg := baseCfg
		graphCfg.Graph.Kind = kind
		adapter, err := buildGraphAdapter(graphCfg)
		if err != nil {
			t.Fatalf("buildGraphAdapter(%s) error = %v", kind, err)
		}
		if adapter == nil {
			t.Fatalf("buildGraphAdapter(%s) returned nil", kind)
		}
	}
	ftsAdapter, err := buildFTSAdapter("memory", nil)
	if err != nil {
		t.Fatalf("buildFTSAdapter(memory) error = %v", err)
	}
	if ftsAdapter == nil {
		t.Fatal("fts adapter is nil")
	}
}
