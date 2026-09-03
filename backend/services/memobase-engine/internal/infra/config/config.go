package config

// Config represents the service configuration mapped from viper/.env
type Config struct {
	GRPCPort               int    `mapstructure:"GRPC_PORT"`
	HealthPort             int    `mapstructure:"HEALTH_PORT"`
	LogLevel               string `mapstructure:"LOG_LEVEL"`
	OTelEndpoint           string `mapstructure:"OTEL_ENDPOINT"`
	NatsURL                string `mapstructure:"NATS_URL"`
	DBDSN                  string `mapstructure:"DB_DSN"`
	BifrostURL             string `mapstructure:"BIFROST_URL"`
	BestLLMModel           string `mapstructure:"BEST_LLM_MODEL"`
	ThinkingLLMModel       string `mapstructure:"THINKING_LLM_MODEL"`
	SummaryLLMModel        string `mapstructure:"SUMMARY_LLM_MODEL"`
	LLMMaxTokens           int    `mapstructure:"LLM_MAX_TOKENS"`
	EmbeddingProvider      string `mapstructure:"EMBEDDING_PROVIDER"`
	EmbeddingModel         string `mapstructure:"EMBEDDING_MODEL"`
	EmbeddingDim           int    `mapstructure:"EMBEDDING_DIM"`
	MaxProcessTokenSize    int    `mapstructure:"MAX_PROCESS_TOKEN_SIZE"`
	MaxProfileSubtopics    int    `mapstructure:"MAX_PROFILE_SUBTOPICS"`
	MaxPreProfileTokenSize int    `mapstructure:"MAX_PRE_PROFILE_TOKEN_SIZE"`
	PromptLanguage         string `mapstructure:"PROMPT_LANGUAGE"`
}

// LoadConfig loads the configuration using viper
func LoadConfig() (*Config, error) {
	// Stub implementation
	return &Config{}, nil
}
