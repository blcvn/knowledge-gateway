package neo4j_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/neo4j"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

// Tests are skipped if NEO4J_TEST_URI is not set, meaning they can run locally with a real neo4j instance or testcontainers.
func getTestDriver(t *testing.T) (*neo4j.Driver, func()) {
	t.Helper()
	uri := os.Getenv("NEO4J_TEST_URI")
	if uri == "" {
		t.Skip("Skipping integration test: NEO4J_TEST_URI not set")
	}

	user := os.Getenv("NEO4J_TEST_USER")
	pass := os.Getenv("NEO4J_TEST_PASS")

	// Assuming we have a basic logger.
	logger := domain.NewTestLogger() // Hypothetical or use a dummy logger.

	driver, err := neo4j.NewDriver(uri, user, pass, "neo4j", logger)
	if err != nil {
		t.Fatalf("Failed to create neo4j driver: %v", err)
	}

	return driver, func() {
		driver.Close()
	}
}

func TestNeo4jNodeRepo_Integration(t *testing.T) {
	driver, cleanup := getTestDriver(t)
	defer cleanup()

	ctx := context.Background()
	groupID := "test-group-1"
	nodeID := "node-1"

	t.Run("Save and Get Node", func(t *testing.T) {
		node := domain.EntityNode{
			UUID:    nodeID,
			Name:    "Test Node",
			GroupID: groupID,
		}

		err := driver.SaveNode(ctx, node)
		if err != nil {
			t.Fatalf("SaveNode failed: %v", err)
		}

		got, err := driver.GetNode(ctx, groupID, nodeID)
		if err != nil {
			t.Fatalf("GetNode failed: %v", err)
		}

		if got.Name != "Test Node" {
			t.Errorf("Expected name 'Test Node', got %s", got.Name)
		}
	})

	t.Run("Delete Node", func(t *testing.T) {
		err := driver.DeleteNode(ctx, groupID, nodeID)
		if err != nil {
			t.Fatalf("DeleteNode failed: %v", err)
		}

		_, err = driver.GetNode(ctx, groupID, nodeID)
		if err != domain.ErrNodeNotFound {
			t.Errorf("Expected ErrNodeNotFound, got %v", err)
		}
	})
}

// Similarly, we would implement TestNeo4jEdgeRepo_Integration, TestNeo4jSearchRepo_Integration, etc.
// that follow the same pattern of setting up the driver and verifying Neo4j interactions.
