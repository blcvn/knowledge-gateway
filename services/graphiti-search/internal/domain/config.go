package domain

import "time"

type SearchConfig struct {
	DefaultLimit int
	MaxLimit     int
}

type RerankerConfig struct {
	RRFKValue        int
	MMRLambda        float64
	NodeDistanceWeight float64
}

type CacheConfig struct {
	Enabled bool
	TTL     time.Duration
}
