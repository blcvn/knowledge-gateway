package config

import (
	"encoding/json"
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

func boolEnv(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		panic(fmt.Sprintf("invalid boolean for %s: %v", key, err))
	}

	return parsed
}

// jsonMapEnv parses a JSON object env var into map[string]T.
// Returns nil when the variable is unset or empty.
// Panics on invalid JSON to catch misconfiguration at startup.
func jsonMapEnv[T any](key string) map[string]T {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}
	var result map[string]T
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &result); err != nil {
		panic(fmt.Sprintf("invalid JSON for %s: %v", key, err))
	}
	return result
}
