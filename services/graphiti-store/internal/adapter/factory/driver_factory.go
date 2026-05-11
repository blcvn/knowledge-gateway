package factory

import (
	"fmt"
	"log/slog"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/neo4j"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/infra/config"
)

// NewGraphDriver creates a new GraphDriver based on the configuration.
func NewGraphDriver(cfg config.Config, logger *slog.Logger) (domain.GraphDriver, error) {
	if cfg.DriverProvider != "neo4j" {
		return nil, fmt.Errorf("unsupported driver provider: %s", cfg.DriverProvider)
	}

	driver, err := neo4j.NewDriver(
		cfg.Neo4jURI,
		cfg.Neo4jUser,
		cfg.Neo4jPassword,
		"neo4j",
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize neo4j driver: %w", err)
	}
	return driver, nil
}
