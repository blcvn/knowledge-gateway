package integration_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	neo4jc "github.com/testcontainers/testcontainers-go/modules/neo4j"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/neo4j"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

func TestNeo4jIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}

	ctx := context.Background()

	// Start Neo4j Testcontainer
	neo4jContainer, err := neo4jc.Run(ctx, "neo4j:5-enterprise",
		neo4jc.WithAdminPassword("password"),
		testcontainers.WithEnv(map[string]string{
			"NEO4J_ACCEPT_LICENSE_AGREEMENT": "yes",
		}),
	)
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	defer func() {
		if err := neo4jContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	uri, err := neo4jContainer.BoltURI(ctx)
	if err != nil {
		t.Fatalf("failed to get bolt URI: %s", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	driver, err := neo4j.NewDriver(uri, "neo4j", "password", "neo4j", logger)
	if err != nil {
		t.Fatalf("failed to create neo4j driver: %v", err)
	}
	defer driver.Close()

	// System Integration Tests (End-to-End)
	t.Run("Save and Retrieve Entity Node", func(t *testing.T) {
		node := domain.EntityNode{
			UUID:      "test-node-uuid",
			Name:      "Integration Node",
			GroupID:   "tenant-1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := driver.SaveNode(ctx, node)
		if err != nil {
			t.Fatalf("failed to save node: %v", err)
		}

		retrieved, err := driver.GetNode(ctx, "tenant-1", "test-node-uuid")
		if err != nil {
			t.Fatalf("failed to get node: %v", err)
		}
		if retrieved.Name != "Integration Node" {
			t.Errorf("expected node name to be Integration Node, got %s", retrieved.Name)
		}
	})
}
