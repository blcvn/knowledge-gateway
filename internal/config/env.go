package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func stringEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

func intEnv(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		panic(fmt.Sprintf("invalid integer for %s: %v", key, err))
	}

	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		panic(fmt.Sprintf("invalid duration for %s: %v", key, err))
	}

	return parsed
}
