package config

import "time"

func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Host: stringEnv("KG_HTTP_HOST", "0.0.0.0"),
			Port: intEnv("KG_HTTP_PORT", 8082),
		},
		Postgres: PostgresConfig{
			Host:            stringEnv("KG_POSTGRES_HOST", "127.0.0.1"),
			Port:            intEnv("KG_POSTGRES_PORT", 5432),
			User:            stringEnv("KG_POSTGRES_USER", "postgres"),
			Password:        stringEnv("KG_POSTGRES_PASSWORD", "postgres"),
			Database:        stringEnv("KG_POSTGRES_DATABASE", "kg_service"),
			SSLMode:         stringEnv("KG_POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    intEnv("KG_POSTGRES_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    intEnv("KG_POSTGRES_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: durationEnv("KG_POSTGRES_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Host:     stringEnv("KG_REDIS_HOST", "127.0.0.1"),
			Port:     intEnv("KG_REDIS_PORT", 6379),
			Username: stringEnv("KG_REDIS_USERNAME", ""),
			Password: stringEnv("KG_REDIS_PASSWORD", ""),
			DB:       intEnv("KG_REDIS_DB", 0),
		},
	}

	return cfg, cfg.Validate()
}
