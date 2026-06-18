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
	vectorAdapter, err := buildVectorAdapter("memory", nil)
	if err != nil {
		t.Fatalf("buildVectorAdapter(memory) error = %v", err)
	}
	if vectorAdapter == nil {
		t.Fatal("vector adapter is nil")
	}
	pgVectorAdapter, err := buildVectorAdapter("pgvector", nil)
	if err != nil {
		t.Fatalf("buildVectorAdapter(pgvector) error = %v", err)
	}
	if pgVectorAdapter == nil {
		t.Fatal("pgvector adapter is nil")
	}
	graphAdapter, err := buildGraphAdapter("memory")
	if err != nil {
		t.Fatalf("buildGraphAdapter(memory) error = %v", err)
	}
	if graphAdapter == nil {
		t.Fatal("graph adapter is nil")
	}
	ftsAdapter, err := buildFTSAdapter("memory", nil)
	if err != nil {
		t.Fatalf("buildFTSAdapter(memory) error = %v", err)
	}
	if ftsAdapter == nil {
		t.Fatal("fts adapter is nil")
	}
}
