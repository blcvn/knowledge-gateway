package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// ProjectStatus represents the lifecycle state of a project.
type ProjectStatus string

const (
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusSuspended ProjectStatus = "suspended"
)

// Project is the core tenant/project entity.
type Project struct {
	ProjectID     string
	ProjectSecret string // bcrypt hash stored in DB
	ProfileConfig string // YAML string
	Status        ProjectStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProjectProfileConfig holds per-project memory configuration.
type ProjectProfileConfig struct {
	MaxSubtopics     int      `yaml:"max_subtopics"`
	MaxSlotTokenSize int      `yaml:"max_slot_token_size"`
	StrictMode       bool     `yaml:"strict_mode"`
	ValidateMode     bool     `yaml:"validate_mode"`
	Language         string   `yaml:"language"`
	AdditionalTopics []string `yaml:"additional_topics"`
}

// DefaultProfileConfig returns safe defaults for a new project.
func DefaultProfileConfig() ProjectProfileConfig {
	return ProjectProfileConfig{
		MaxSubtopics:     20,
		MaxSlotTokenSize: 2048,
		Language:         "en",
	}
}

// Validate checks that required fields are within acceptable ranges.
func (c *ProjectProfileConfig) Validate() error {
	if c.MaxSubtopics <= 0 {
		return fmt.Errorf("max_subtopics must be > 0, got %d", c.MaxSubtopics)
	}
	if c.MaxSlotTokenSize < 0 {
		return fmt.Errorf("max_slot_token_size must be >= 0, got %d", c.MaxSlotTokenSize)
	}
	validLangs := map[string]bool{"en": true, "zh": true, "": true}
	if !validLangs[c.Language] {
		return fmt.Errorf("language must be 'en' or 'zh', got %q", c.Language)
	}
	return nil
}

// ParseProfileConfig deserializes a YAML string into ProjectProfileConfig.
func ParseProfileConfig(yamlStr string) (*ProjectProfileConfig, error) {
	cfg := DefaultProfileConfig()
	if err := yaml.Unmarshal([]byte(yamlStr), &cfg); err != nil {
		return nil, fmt.Errorf("parse profile config: %w", err)
	}
	return &cfg, nil
}

// MarshalProfileConfig serializes ProjectProfileConfig to a YAML string.
func MarshalProfileConfig(cfg ProjectProfileConfig) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

const tokenPrefix = "sk-proj-"

// GenerateProjectToken generates a new project API token.
// Returns the plaintext token (returned once to caller) and the bcrypt hash to store in DB.
func GenerateProjectToken(projectID string) (token, hashedSecret string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	secret := base64.URLEncoding.EncodeToString(raw)
	token = tokenPrefix + projectID + "-" + secret

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), 12)
	if err != nil {
		return "", "", fmt.Errorf("hash token: %w", err)
	}
	return token, string(hash), nil
}

// ParseProjectToken extracts the projectID and plaintext secret from a token.
func ParseProjectToken(token string) (projectID, secret string, err error) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return "", "", ErrInvalidTokenFormat
	}
	rest := token[len(tokenPrefix):]
	// Find second "-" after the projectID section
	// Format: projectID-secret (where projectID has no "-" prefix of the format sk-proj-)
	// We split on the LAST separator that starts the base64 secret
	// Actually: split on first "-" after prefix to get projectID, rest is secret
	parts := strings.SplitN(rest, "-", 2)
	if len(parts) != 2 {
		return "", "", ErrInvalidTokenFormat
	}
	projectID = parts[0]
	secret = parts[1]
	if projectID == "" || secret == "" {
		return "", "", ErrInvalidTokenFormat
	}
	return projectID, secret, nil
}

// User is a memory user within a project.
type User struct {
	ID        string
	ProjectID string
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserStatus wraps per-user flexible status data.
type UserStatus struct {
	ID         string
	ProjectID  string
	UserID     string
	StatusData map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ProjectContext is the resolved caller identity after token validation.
type ProjectContext struct {
	ProjectID string
}
