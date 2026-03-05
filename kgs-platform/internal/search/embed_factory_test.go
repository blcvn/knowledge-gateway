package search

import (
	"testing"

	"kgs-platform/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestNewEmbeddingClientDefaultDeterministic(t *testing.T) {
	cfg := &conf.Data{
		Qdrant: &conf.Data_Qdrant{VectorSize: 1536},
	}
	client, err := NewEmbeddingClient(cfg, log.DefaultLogger)
	if err != nil {
		t.Fatalf("NewEmbeddingClient error: %v", err)
	}
	if _, ok := client.(*DeterministicEmbeddingClient); !ok {
		t.Fatalf("expected deterministic client, got %T", client)
	}
}

func TestNewEmbeddingClientOpenAIRequiresKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	cfg := &conf.Data{
		Embedding: &conf.Data_Embedding{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Timeout:  durationpb.New(5),
		},
	}
	if _, err := NewEmbeddingClient(cfg, log.DefaultLogger); err == nil {
		t.Fatalf("expected error when OpenAI key is missing")
	}
}
