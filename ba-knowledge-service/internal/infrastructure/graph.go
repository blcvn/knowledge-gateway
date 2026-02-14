package infrastructure

import (
	"context"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jConfig struct {
	Uri      string
	Username string
	Password string
}

func ConnectNeo4j(ctx context.Context, cfg Neo4jConfig) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.Uri,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify neo4j connectivity: %w", err)
	}

	return driver, nil
}

func SetupGraphSchema(ctx context.Context, driver neo4j.DriverWithContext) error {
	log.Println("Setting up Neo4j Graph Schema constraints...")
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	constraints := []string{
		"CREATE CONSTRAINT IF NOT EXISTS FOR (n:PRD) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT IF NOT EXISTS FOR (n:URD_Index) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT IF NOT EXISTS FOR (n:UseCase) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT IF NOT EXISTS FOR (n:Actor) REQUIRE n.id IS UNIQUE",
	}

	for _, query := range constraints {
		_, err := session.Run(ctx, query, nil)
		if err != nil {
			return fmt.Errorf("failed to create constraint %s: %w", query, err)
		}
	}
	log.Println("Graph Schema setup complete.")
	return nil
}
