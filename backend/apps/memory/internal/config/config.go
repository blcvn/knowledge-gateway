package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig   `mapstructure:"server"`
	Auth        AuthConfig     `mapstructure:"auth"`
	Postgres    PostgresConfig `mapstructure:"postgres"`
	Neo4j       Neo4jConfig    `mapstructure:"neo4j"`
	Qdrant      QdrantConfig   `mapstructure:"qdrant"`
	Redis       RedisConfig    `mapstructure:"redis"`
	NATS        NATSConfig     `mapstructure:"nats"`
	MinIO       MinIOConfig    `mapstructure:"minio"`
	Bifrost     BifrostConfig  `mapstructure:"bifrost"`
	Cognee      CogneeConfig   `mapstructure:"cognee"`
	Graphiti    GraphitiConfig `mapstructure:"graphiti"`
	Memobase    MemobaseConfig `mapstructure:"memobase"`
	OpenViking  OVConfig       `mapstructure:"openviking"`
	Zep         ZepConfig      `mapstructure:"zep"`
	Supermemory SMConfig       `mapstructure:"supermemory"`
	Platform    PlatformConfig `mapstructure:"platform"`
}

type ServerConfig struct {
	RESTPort   int    `mapstructure:"rest_port"`
	GRPCPort   int    `mapstructure:"grpc_port"`
	MCPPort    int    `mapstructure:"mcp_port"`
	HealthPort int    `mapstructure:"health_port"`
	LogLevel   string `mapstructure:"log_level"`
}

type AuthConfig struct {
	DevMode      bool   `mapstructure:"dev_mode"`
	JWTPublicKey string `mapstructure:"jwt_public_key"`
	JWTIssuer    string `mapstructure:"jwt_issuer"`
	JWTAudience  string `mapstructure:"jwt_audience"`
}

type PostgresConfig struct {
	DSN      string `mapstructure:"dsn"`
	MaxConns int32  `mapstructure:"max_conns"`
	MinConns int32  `mapstructure:"min_conns"`
}

type Neo4jConfig struct {
	URI      string `mapstructure:"uri"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type QdrantConfig struct {
	Addr string `mapstructure:"addr"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type NATSConfig struct {
	Mode     string `mapstructure:"mode"`
	URL      string `mapstructure:"url"`
	StoreDir string `mapstructure:"store_dir"`
}

type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
}

type BifrostConfig struct {
	URL string `mapstructure:"url"`
}

type CogneeConfig struct {
	ChunkSize         int `mapstructure:"chunk_size"`
	MaxConcurrentJobs int `mapstructure:"max_concurrent_jobs"`
}

type GraphitiConfig struct {
	CommunityDetection bool `mapstructure:"community_detection"`
	MaxBFSDepth        int  `mapstructure:"max_bfs_depth"`
}

type MemobaseConfig struct {
	BufferFlushThreshold int    `mapstructure:"buffer_flush_threshold"`
	BufferIdleTimeout    string `mapstructure:"buffer_idle_timeout"`
	MaxLLMCallsPerFlush  int    `mapstructure:"max_llm_calls_per_flush"`
}

type OVConfig struct {
	DataDir           string `mapstructure:"data_dir"`
	EncryptionEnabled bool   `mapstructure:"encryption_enabled"`
}

type ZepConfig struct {
	ContextAssemblyTimeout string `mapstructure:"context_assembly_timeout"`
	MaxMessagesPerThread   int    `mapstructure:"max_messages_per_thread"`
}

type SMConfig struct {
	ForgettingCurveEnabled bool   `mapstructure:"forgetting_curve_enabled"`
	DecayHalfLife          string `mapstructure:"decay_half_life"`
}

type PlatformConfig struct {
	SearchHubTimeout string `mapstructure:"search_hub_timeout"`
	MaxFanOut        int    `mapstructure:"max_fan_out"`
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs")
	viper.AddConfigPath("/etc/vnp-memory")

	viper.SetEnvPrefix("VNP_MEMORY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Warning: failed to read config file: %v\n", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("unable to decode into config struct: %w", err))
	}

	return &cfg
}

func (c *Config) Validate() error {
	// Add validation logic here
	return nil
}
