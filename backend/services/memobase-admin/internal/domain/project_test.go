package domain_test

import (
	"strings"
	"testing"

	"github.com/vnp-memory/services/memobase-admin/internal/domain"
)

func TestGenerateProjectToken_Format(t *testing.T) {
	projectID := "proj-123"
	token, hash, err := domain.GenerateProjectToken(projectID)
	if err != nil {
		t.Fatalf("GenerateProjectToken error: %v", err)
	}
	if !strings.HasPrefix(token, "sk-proj-"+projectID+"-") {
		t.Errorf("token %q does not have expected prefix", token)
	}
	if hash == "" {
		t.Error("hash is empty")
	}
}

func TestGenerateProjectToken_Unique(t *testing.T) {
	token1, _, _ := domain.GenerateProjectToken("proj-1")
	token2, _, _ := domain.GenerateProjectToken("proj-1")
	if token1 == token2 {
		t.Error("two tokens should be different")
	}
}

func TestParseProjectToken_Valid(t *testing.T) {
	projectID := "myproject"
	token, _, _ := domain.GenerateProjectToken(projectID)
	gotProject, gotSecret, err := domain.ParseProjectToken(token)
	if err != nil {
		t.Fatalf("ParseProjectToken error: %v", err)
	}
	if gotProject != projectID {
		t.Errorf("projectID = %q, want %q", gotProject, projectID)
	}
	if gotSecret == "" {
		t.Error("secret is empty")
	}
}

func TestParseProjectToken_BadPrefix(t *testing.T) {
	_, _, err := domain.ParseProjectToken("Bearer sometoken")
	if err == nil {
		t.Error("expected error for bad prefix")
	}
}

func TestProjectProfileConfig_Validate_Valid(t *testing.T) {
	cfg := domain.DefaultProfileConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate error: %v", err)
	}
}

func TestProjectProfileConfig_Validate_Max0(t *testing.T) {
	cfg := domain.DefaultProfileConfig()
	cfg.MaxSubtopics = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for max_subtopics=0")
	}
}

func TestParseProfileConfig_ValidYAML(t *testing.T) {
	yaml := `
max_subtopics: 30
language: zh
`
	cfg, err := domain.ParseProfileConfig(yaml)
	if err != nil {
		t.Fatalf("ParseProfileConfig error: %v", err)
	}
	if cfg.MaxSubtopics != 30 {
		t.Errorf("MaxSubtopics = %d, want 30", cfg.MaxSubtopics)
	}
	if cfg.Language != "zh" {
		t.Errorf("Language = %q, want \"zh\"", cfg.Language)
	}
}
