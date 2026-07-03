package bootstrap

import (
	"database/sql"
	"testing"

	"kg-service/internal/config"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/vectorstore"
)

func TestSupportedRuntimeProfilesBuildAdapters(t *testing.T) {
	base := config.Config{
		HTTP: config.HTTPConfig{Host: "0.0.0.0", Port: 8082},
		Postgres: config.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "postgres",
			Database: "kg_service",
		},
		Redis: config.RedisConfig{Host: "127.0.0.1", Port: 6379},
		FTS:   config.AdapterConfig{Kind: "postgres"},
	}

	cases := []struct {
		name           string
		graphKind      string
		graphEndpoint  string
		graphDatabase  string
		vectorKind     string
		vectorEndpoint string
	}{
		{name: "pgvector-memgraph", graphKind: "memgraph", graphEndpoint: "bolt://memgraph:7687", graphDatabase: "kg", vectorKind: "pgvector"},
		{name: "pgvector-neo4j", graphKind: "neo4j", graphEndpoint: "neo4j://neo4j:7687", graphDatabase: "neo4j", vectorKind: "pgvector"},
		{name: "qdrant-memgraph", graphKind: "memgraph", graphEndpoint: "bolt://memgraph:7687", graphDatabase: "kg", vectorKind: "qdrant", vectorEndpoint: "http://qdrant:6333"},
		{name: "qdrant-neo4j", graphKind: "neo4j", graphEndpoint: "neo4j://neo4j:7687", graphDatabase: "neo4j", vectorKind: "qdrant", vectorEndpoint: "http://qdrant:6333"},
		{name: "milvus-neo4j", graphKind: "neo4j", graphEndpoint: "neo4j://neo4j:7687", graphDatabase: "neo4j", vectorKind: "milvus", vectorEndpoint: "http://milvus:19530"},
		{name: "qdrant-nebula", graphKind: "nebula", graphEndpoint: "nebula://nebula:9669", graphDatabase: "kg", vectorKind: "qdrant", vectorEndpoint: "http://qdrant:6333"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			cfg.Graph.Kind = tc.graphKind
			cfg.Graph.Endpoint = tc.graphEndpoint
			cfg.Graph.Database = tc.graphDatabase
			cfg.Vector.Kind = tc.vectorKind
			cfg.Vector.Endpoint = tc.vectorEndpoint
			cfg.Vector.Collection = "kg_vectors"

			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			vectorAdapter, err := buildVectorAdapter(cfg, (*sql.DB)(nil))
			if err != nil {
				t.Fatalf("buildVectorAdapter() error = %v", err)
			}
			if vectorAdapter == nil {
				t.Fatal("buildVectorAdapter() returned nil")
			}

			graphAdapter, err := buildGraphAdapter(cfg)
			if err != nil {
				t.Fatalf("buildGraphAdapter() error = %v", err)
			}
			if graphAdapter == nil {
				t.Fatal("buildGraphAdapter() returned nil")
			}

			switch got := vectorAdapter.(type) {
			case *vectorstore.QdrantVectorAdapter:
				if tc.vectorKind != "qdrant" {
					t.Fatalf("vector adapter type = %T, want non-qdrant type", got)
				}
				if got.Endpoint != tc.vectorEndpoint {
					t.Fatalf("qdrant endpoint = %q, want %q", got.Endpoint, tc.vectorEndpoint)
				}
				if got.Collection != "kg_vectors" {
					t.Fatalf("qdrant collection = %q, want kg_vectors", got.Collection)
				}
			case *vectorstore.MilvusVectorAdapter:
				if tc.vectorKind != "milvus" {
					t.Fatalf("vector adapter type = %T, want non-milvus type", got)
				}
				if got.Endpoint != tc.vectorEndpoint {
					t.Fatalf("milvus endpoint = %q, want %q", got.Endpoint, tc.vectorEndpoint)
				}
				if got.Collection != "kg_vectors" {
					t.Fatalf("milvus collection = %q, want kg_vectors", got.Collection)
				}
			case *vectorstore.PgVectorAdapter:
				if tc.vectorKind != "pgvector" {
					t.Fatalf("vector adapter type = %T, want non-pgvector type", got)
				}
			default:
				t.Fatalf("unexpected vector adapter type = %T", got)
			}

			switch got := graphAdapter.(type) {
			case *graphstore.Neo4jGraphAdapter:
				if tc.graphKind != "neo4j" {
					t.Fatalf("graph adapter type = %T, want non-neo4j type", got)
				}
				if got.Endpoint != tc.graphEndpoint {
					t.Fatalf("neo4j endpoint = %q, want %q", got.Endpoint, tc.graphEndpoint)
				}
			case *graphstore.MemgraphGraphAdapter:
				if tc.graphKind != "memgraph" {
					t.Fatalf("graph adapter type = %T, want non-memgraph type", got)
				}
				if got.Endpoint != tc.graphEndpoint {
					t.Fatalf("memgraph endpoint = %q, want %q", got.Endpoint, tc.graphEndpoint)
				}
			case *graphstore.NebulaGraphAdapter:
				if tc.graphKind != "nebula" {
					t.Fatalf("graph adapter type = %T, want non-nebula type", got)
				}
				if got.Endpoint != tc.graphEndpoint {
					t.Fatalf("nebula endpoint = %q, want %q", got.Endpoint, tc.graphEndpoint)
				}
			default:
				t.Fatalf("unexpected graph adapter type = %T", got)
			}
		})
	}
}
