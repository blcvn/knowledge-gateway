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

func intEnv(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return parsed, nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback, fmt.Errorf("%s must be a duration: %w", key, err)
	}

	return parsed, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback, fmt.Errorf("%s must be a boolean: %w", key, err)
	}

	return parsed, nil
}

func floatEnv(key string, fallback float64) (float64, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback, fmt.Errorf("%s must be a float: %w", key, err)
	}

	return parsed, nil
}

func jsonMapEnv[T any](key string) (map[string]T, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return nil, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	var parsed map[string]T
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return nil, fmt.Errorf("%s must be a valid JSON object: %w", key, err)
	}

	return parsed, nil
}
