package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	GRPCPort             int    `mapstructure:"GRPC_PORT"`
	HealthPort           int    `mapstructure:"HEALTH_PORT"`
	LogLevel             string `mapstructure:"LOG_LEVEL"`
	OTelEndpoint         string `mapstructure:"OTEL_ENDPOINT"`
	DBDSN                string `mapstructure:"DB_DSN"`
	DBMaxConnections     int    `mapstructure:"DB_MAX_CONNECTIONS"`
	AuthMode             string `mapstructure:"AUTH_MODE"`
	DevAccountID         string `mapstructure:"DEV_ACCOUNT_ID"`
	DevUserID            string `mapstructure:"DEV_USER_ID"`
	OVCryptoAddr         string `mapstructure:"OV_CRYPTO_ADDR"`
	HealthCheckTimeoutMs int    `mapstructure:"HEALTH_CHECK_TIMEOUT_MS"`
	OVFSAddr             string `mapstructure:"OV_FS_ADDR"`
	OVSearchAddr         string `mapstructure:"OV_SEARCH_ADDR"`
	OVSessionAddr        string `mapstructure:"OV_SESSION_ADDR"`
	OVResourceAddr       string `mapstructure:"OV_RESOURCE_ADDR"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("GRPC_PORT", 9056)
	viper.SetDefault("HEALTH_PORT", 9109)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("DB_MAX_CONNECTIONS", 10)
	viper.SetDefault("AUTH_MODE", "api_key")
	viper.SetDefault("HEALTH_CHECK_TIMEOUT_MS", 5000)

	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
		return nil, err
	}

	return &cfg, nil
}
