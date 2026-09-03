package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	GRPCPort               int     `mapstructure:"GRPC_PORT"`
	HealthPort             int     `mapstructure:"HEALTH_PORT"`
	LogLevel               string  `mapstructure:"LOG_LEVEL"`
	OTelEndpoint           string  `mapstructure:"OTEL_ENDPOINT"`
	NatsURL                string  `mapstructure:"NATS_URL"`
	DBDSN                  string  `mapstructure:"DB_DSN"`
	RedisURL               string  `mapstructure:"REDIS_URL"`
	ProfileCacheTTL        int     `mapstructure:"PROFILE_CACHE_TTL"`
	DefaultMaxTokenSize    int32   `mapstructure:"DEFAULT_MAX_TOKEN_SIZE"`
	ProfileEventRatio      float32 `mapstructure:"PROFILE_EVENT_RATIO"`
	EventSearchThreshold   float32 `mapstructure:"EVENT_SEARCH_THRESHOLD"`
	EventSearchWindowDays  int     `mapstructure:"EVENT_SEARCH_WINDOW_DAYS"`
	EventSearchTopK        int     `mapstructure:"EVENT_SEARCH_TOPK"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("GRPC_PORT", 9033)
	viper.SetDefault("HEALTH_PORT", 9100)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("PROFILE_CACHE_TTL", 1200)
	viper.SetDefault("DEFAULT_MAX_TOKEN_SIZE", 500)
	viper.SetDefault("PROFILE_EVENT_RATIO", 0.7)
	viper.SetDefault("EVENT_SEARCH_THRESHOLD", 0.2)
	viper.SetDefault("EVENT_SEARCH_WINDOW_DAYS", 21)
	viper.SetDefault("EVENT_SEARCH_TOPK", 10)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
