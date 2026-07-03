package config

import (
	"errors"
	"fmt"
)

func (c Config) Validate() error {
	var errs []error

	switch c.Observability.LogFormat {
	case "", "json", "text":
	default:
		errs = append(errs, fmt.Errorf("log format must be json or text: %s", c.Observability.LogFormat))
	}
	switch c.Observability.LogLevel {
	case "", "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("log level must be debug, info, warn, or error: %s", c.Observability.LogLevel))
	}
	if c.Observability.TraceSamplingRatio < 0 || c.Observability.TraceSamplingRatio > 1 {
		errs = append(errs, fmt.Errorf("trace sampling ratio must be between 0 and 1: %f", c.Observability.TraceSamplingRatio))
	}
	if c.Observability.MetricsEnabled && c.Observability.MetricsAddress != "" && c.HTTP.Address() == c.Observability.MetricsAddress {
		errs = append(errs, fmt.Errorf("metrics address must differ from http address when metrics are enabled"))
	}

	if c.HTTP.Host == "" {
		errs = append(errs, errors.New("http host must not be empty"))
	}
	if c.HTTP.Port <= 0 {
		errs = append(errs, fmt.Errorf("http port must be positive: %d", c.HTTP.Port))
	}

	if c.Postgres.Host == "" {
		errs = append(errs, errors.New("postgres host must not be empty"))
	}
	if c.Postgres.Port <= 0 {
		errs = append(errs, fmt.Errorf("postgres port must be positive: %d", c.Postgres.Port))
	}
	if c.Postgres.User == "" {
		errs = append(errs, errors.New("postgres user must not be empty"))
	}
	if c.Postgres.Database == "" {
		errs = append(errs, errors.New("postgres database must not be empty"))
	}

	if c.Redis.Host == "" {
		errs = append(errs, errors.New("redis host must not be empty"))
	}
	if c.Redis.Port <= 0 {
		errs = append(errs, fmt.Errorf("redis port must be positive: %d", c.Redis.Port))
	}

	switch c.Embedding.Provider {
	case "", "deterministic", "http":
	default:
		errs = append(errs, fmt.Errorf("embedding provider must be deterministic or http: %s", c.Embedding.Provider))
	}
	switch c.Vector.Kind {
	case "", "memory", "pgvector", "qdrant", "milvus":
	default:
		errs = append(errs, fmt.Errorf("vector adapter must be memory, pgvector, qdrant, or milvus: %s", c.Vector.Kind))
	}
	switch c.Vector.Kind {
	case "qdrant", "milvus":
		if c.Vector.Endpoint == "" {
			errs = append(errs, fmt.Errorf("vector adapter %s requires KG_VECTOR_ENDPOINT", c.Vector.Kind))
		}
	}
	switch c.Graph.Kind {
	case "", "memory", "neo4j", "memgraph", "nebula":
	default:
		errs = append(errs, fmt.Errorf("graph adapter must be memory, neo4j, memgraph, or nebula: %s", c.Graph.Kind))
	}
	switch c.Graph.Kind {
	case "neo4j", "memgraph", "nebula":
		if c.Graph.Endpoint == "" {
			errs = append(errs, fmt.Errorf("graph adapter %s requires KG_GRAPH_ENDPOINT", c.Graph.Kind))
		}
		if c.Graph.Kind == "nebula" && c.Graph.Database == "" {
			errs = append(errs, fmt.Errorf("graph adapter %s requires KG_GRAPH_DATABASE", c.Graph.Kind))
		}
	}
	switch c.FTS.Kind {
	case "", "memory", "postgres":
	default:
		errs = append(errs, fmt.Errorf("fts adapter must be memory or postgres: %s", c.FTS.Kind))
	}

	return errors.Join(errs...)
}
