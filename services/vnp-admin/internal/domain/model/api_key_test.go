package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/model"
)

func TestGenerateAPIKey_SHA256RoundTrip(t *testing.T) {
	key, plaintext, err := model.GenerateAPIKey(uuid.New(), "test-key", model.KeyScopeReadWrite)
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}

	// Verify prefix
	if plaintext[:4] != "vnp_" {
		t.Errorf("expected vnp_ prefix, got %s", plaintext[:4])
	}

	// Verify key prefix is stored
	if key.KeyPrefix != plaintext[:12] {
		t.Errorf("key prefix mismatch: got %s, want %s", key.KeyPrefix, plaintext[:12])
	}

	// Verify hash validation succeeds with correct key
	if !key.ValidateKey(plaintext) {
		t.Error("ValidateKey should return true for correct plaintext")
	}

	// Verify hash validation fails with wrong key
	if key.ValidateKey("vnp_wrong_key_0000000000000000000000000000000000000000000000000000") {
		t.Error("ValidateKey should return false for wrong plaintext")
	}

	// Verify hash is not empty
	if key.KeyHash == "" {
		t.Error("KeyHash should not be empty")
	}

	// Verify hash length (SHA-256 produces 64 hex chars)
	if len(key.KeyHash) != 64 {
		t.Errorf("KeyHash should be 64 chars, got %d", len(key.KeyHash))
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	tenantID := uuid.New()
	key1, plain1, _ := model.GenerateAPIKey(tenantID, "key-1", model.KeyScopeReadOnly)
	key2, plain2, _ := model.GenerateAPIKey(tenantID, "key-2", model.KeyScopeAdmin)

	if plain1 == plain2 {
		t.Error("two generated keys should have different plaintext")
	}
	if key1.KeyHash == key2.KeyHash {
		t.Error("two generated keys should have different hashes")
	}
	if key1.ID == key2.ID {
		t.Error("two generated keys should have different IDs")
	}
}

func TestDefaultConfig_Plans(t *testing.T) {
	tests := []struct {
		plan          model.Plan
		wantMaxKeys   int
		wantMaxUsers  int
		wantRPM       int
	}{
		{model.PlanFree, 2, 5, 100},
		{model.PlanStarter, 10, 50, 1000},
		{model.PlanEnterprise, 100, 1000, 10000},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			cfg := model.DefaultConfig(tt.plan)
			if cfg.MaxAPIKeys != tt.wantMaxKeys {
				t.Errorf("MaxAPIKeys: got %d, want %d", cfg.MaxAPIKeys, tt.wantMaxKeys)
			}
			if cfg.MaxUsers != tt.wantMaxUsers {
				t.Errorf("MaxUsers: got %d, want %d", cfg.MaxUsers, tt.wantMaxUsers)
			}
			if cfg.RateLimitRPM != tt.wantRPM {
				t.Errorf("RateLimitRPM: got %d, want %d", cfg.RateLimitRPM, tt.wantRPM)
			}
		})
	}
}
