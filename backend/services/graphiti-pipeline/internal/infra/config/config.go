package config

type PostgresConfig struct {
	URI string `mapstructure:"URI"`
}

type StoreConfig struct {
	Endpoint string `mapstructure:"ENDPOINT"`
}

type LLMConfig struct {
	Endpoint string `mapstructure:"ENDPOINT"`
}

type EmbedderConfig struct {
	Endpoint string `mapstructure:"ENDPOINT"`
}

type NATSConfig struct {
	URL string `mapstructure:"URL"`
}

type OTelConfig struct {
	Endpoint string `mapstructure:"ENDPOINT"`
}

type PipelineConfig struct {
	MaxConcurrent int `mapstructure:"MAX_CONCURRENT"`
}

type Config struct {
	GRPCPort   int    `mapstructure:"GRPC_PORT" validate:"required,min=1024,max=65535"`
	HealthPort int    `mapstructure:"HEALTH_PORT"`
	LogLevel   string `mapstructure:"LOG_LEVEL"`

	Postgres PostgresConfig `mapstructure:",squash"`
	Store    StoreConfig    `mapstructure:",squash"`
	LLM      LLMConfig      `mapstructure:",squash"`
	Embedder EmbedderConfig `mapstructure:",squash"`
	NATS     NATSConfig     `mapstructure:",squash"`
	OTel     OTelConfig     `mapstructure:",squash"`
	Pipeline PipelineConfig `mapstructure:",squash"`
}

func LoadConfig() (Config, error) {
	// Use Viper to load and unmarshal configuration
	return Config{}, nil
}
