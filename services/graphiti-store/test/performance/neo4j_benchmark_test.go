package performance_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/neo4j"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

func BenchmarkNeo4jQueries(b *testing.T) {
	uri := os.Getenv("NEO4J_TEST_URI")
	if uri == "" {
		b.Skip("NEO4J_TEST_URI not set")
	}
	user := os.Getenv("NEO4J_TEST_USER")
	pass := os.Getenv("NEO4J_TEST_PASS")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	driver, err := neo4j.NewDriver(uri, user, pass, "neo4j", logger)
	if err != nil {
		b.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()
	groupID := "bench-group"

	// Pre-populate data
	for i := 0; i < 100; i++ {
		node := domain.EntityNode{
			UUID:      fmt.Sprintf("bench-node-%d", i),
			Name:      fmt.Sprintf("Benchmark Node %d", i),
			GroupID:   groupID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = driver.SaveNode(ctx, node)
	}

	b.ResetTimer()

	b.Run("GetNode", func(b *testing.T) {
		for i := 0; i < b.N; i++ {
			nodeID := fmt.Sprintf("bench-node-%d", i%100)
			_, _ = driver.GetNode(ctx, groupID, nodeID)
		}
	})
}
