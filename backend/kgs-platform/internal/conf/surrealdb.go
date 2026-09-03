package conf

import "os"

// SurrealDBConfig provides SurrealDB connection settings.
// Used as a manual extension until protoc regenerates conf.pb.go.
type SurrealDBConfig struct {
	URL       string `json:"url" yaml:"url"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Database  string `json:"database" yaml:"database"`
	User      string `json:"user" yaml:"user"`
	Password  string `json:"password" yaml:"password"`
}

// StorageMode returns the configured storage mode.
// Reads from environment variable KGS_STORAGE_MODE since the proto
// hasn't been regenerated yet. Falls back to "specialized".
func StorageMode(d *Data) string {
	// Environment variable takes precedence (set by docker-compose)
	if mode := os.Getenv("KGS_STORAGE_MODE"); mode != "" {
		return mode
	}
	return "specialized"
}

// IsSurrealDBMode checks if the current config uses SurrealDB storage mode.
func IsSurrealDBMode(d *Data) bool {
	return StorageMode(d) == "surrealdb"
}

// GetSurrealDBFromEnv builds a SurrealDBConfig from environment variables.
func GetSurrealDBFromEnv() *SurrealDBConfig {
	return &SurrealDBConfig{
		URL:       getEnvDefault("KGS_SURREALDB_URL", "ws://localhost:8000"),
		Namespace: getEnvDefault("KGS_SURREALDB_NAMESPACE", "kgs"),
		Database:  getEnvDefault("KGS_SURREALDB_DATABASE", "production"),
		User:      getEnvDefault("KGS_SURREALDB_USER", "root"),
		Password:  getEnvDefault("KGS_SURREALDB_PASSWORD", "secret"),
	}
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
