package model

import "time"

type HotnessScore struct {
	Path                string    `json:"path"`
	AccountID           string    `json:"account_id"`
	BaseScore           float64   `json:"base_score"`
	AccessCount         int       `json:"access_count"`
	SessionRefCount     int       `json:"session_ref_count"`
	ComputedHotness     float64   `json:"computed_hotness"`
	LastAccessedAt      time.Time `json:"last_accessed_at"`
	LastModifiedAt      time.Time `json:"last_modified_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type DecayConfig struct {
	HalfLifeHours float64
	SessionBoost  float64
}
