package data

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// EnsureConstraints creates Neo4j constraints required by upsert logic.
// Community edition does not support relationship uniqueness constraints.
func EnsureConstraints(ctx context.Context, driver neo4j.DriverWithContext) error {
	if driver == nil {
		return fmt.Errorf("neo4j driver is nil")
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	constraints := []string{
		`CREATE CONSTRAINT kgs_entity_unique_key IF NOT EXISTS
		 FOR (n:Entity)
		 REQUIRE n._unique_key IS UNIQUE`,
	}

	for _, cypher := range constraints {
		if _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, runErr := tx.Run(ctx, cypher, nil)
			if runErr != nil {
				return nil, runErr
			}
			return nil, nil
		}); err != nil {
			return fmt.Errorf("failed to create constraint: %w", err)
		}
	}

	return nil
}
