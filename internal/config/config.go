package config

import (
	"fmt"
	"time"
)

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Redis    RedisConfig
}

type HTTPConfig struct {
	Host string
	Port int
}

func (c HTTPConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DB       int
}

func (c RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
