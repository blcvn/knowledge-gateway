package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load parses a YAML config file into type T.
// Environment variables with prefix MEMOBASE_ override file values.
// Example: MEMOBASE_LANGUAGE=zh overrides config.language
func Load[T any](configPath string) (*T, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("MEMOBASE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// LoadWithDefaults is like Load but applies default values before reading the file.
func LoadWithDefaults[T any](configPath string, defaults map[string]any) (*T, error) {
	v := viper.New()
	for k, val := range defaults {
		v.SetDefault(k, val)
	}
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("MEMOBASE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// LoadFromViper unmarshals an already-configured Viper instance.
// Useful for testing or embedding config in larger structs.
func LoadFromViper[T any](v *viper.Viper) (*T, error) {
	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// MustLoad calls Load and panics on error. Use only in main().
func MustLoad[T any](configPath string) *T {
	cfg, err := Load[T](configPath)
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}
	return cfg
}
