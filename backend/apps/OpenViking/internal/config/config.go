package config

import "time"

type Config struct {
	Server struct {
		RESTPort        int
		MCPPort         int
		ManagementPort  int
		ShutdownTimeout time.Duration
	}
	Services struct {
		AdminPort    int
		CryptoPort   int
		FSPort       int
		ResourcePort int
		SearchPort   int
		SessionPort  int
	}
}

func Load() *Config {
	cfg := &Config{}
	cfg.Server.RESTPort = 8080
	cfg.Server.MCPPort = 8082
	cfg.Server.ManagementPort = 8000
	cfg.Server.ShutdownTimeout = 10 * time.Duration(time.Second)

	cfg.Services.AdminPort = 9030
	cfg.Services.CryptoPort = 9015
	cfg.Services.FSPort = 9011
	cfg.Services.ResourcePort = 9014
	cfg.Services.SearchPort = 9012
	cfg.Services.SessionPort = 9013

	return cfg
}

func (c *Config) GatewayServicesMap() map[string]string {
	return map[string]string{
		"ov-admin":    ":9030",
		"ov-crypto":   ":9015",
		"ov-fs":       ":9011",
		"ov-resource": ":9014",
		"ov-search":   ":9012",
		"ov-session":  ":9013",
	}
}
