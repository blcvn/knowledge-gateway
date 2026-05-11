package config

type Config struct {
	DatabaseURL string
	GRPCPort    string
	NatsURL     string
	LLMEndpoint string
}

func LoadConfig() *Config {
	return &Config{
		DatabaseURL: "postgres://user:pass@localhost:5432/vnp_memory?sslmode=disable",
		GRPCPort:    "9053",
		NatsURL:     "nats://localhost:4222",
		LLMEndpoint: "localhost:9090",
	}
}
