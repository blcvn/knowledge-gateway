package bootstrap

import (
	"log/slog"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/config"
)

type Infra struct {
	// Add real connections later:
	// PG      *pgxpool.Pool
	// Neo4j   neo4j.DriverWithContext
	// Qdrant  *qdrant.Client
	// Redis   *redis.Client
	// MinIO   *minio.Client
	// Bifrost llm.LLMClient
}

func NewInfra(cfg *config.Config, logger *slog.Logger) (*Infra, error) {
	logger.Info("Initializing shared infrastructure...")
	return &Infra{}, nil
}

func (i *Infra) Close() {
	// Close all connections here
}
