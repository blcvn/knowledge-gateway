package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/vnp-memory/pkg/config"
)

// testConfig is the struct used in loader tests.
type testConfig struct {
	Language string `mapstructure:"language"`
	LLM      struct {
		APIKey string `mapstructure:"api_key"`
	} `mapstructure:"llm"`
	MaxChatBlobBufferTokenSize int `mapstructure:"max_chat_blob_buffer_token_size"`
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_ValidYAML(t *testing.T) {
	path := writeTempYAML(t, `language: en
llm:
  api_key: test-key
max_chat_blob_buffer_token_size: 1024
`)
	cfg, err := config.Load[testConfig](path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Language != "en" {
		t.Errorf("Language = %q, want \"en\"", cfg.Language)
	}
	if cfg.LLM.APIKey != "test-key" {
		t.Errorf("LLM.APIKey = %q, want \"test-key\"", cfg.LLM.APIKey)
	}
	if cfg.MaxChatBlobBufferTokenSize != 1024 {
		t.Errorf("MaxChatBlobBufferTokenSize = %d, want 1024", cfg.MaxChatBlobBufferTokenSize)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load[testConfig]("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_ENVOverride(t *testing.T) {
	path := writeTempYAML(t, `language: en`)
	t.Setenv("MEMOBASE_LANGUAGE", "zh")

	cfg, err := config.Load[testConfig](path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Language != "zh" {
		t.Errorf("Language = %q, want \"zh\" (ENV override)", cfg.Language)
	}
}

func TestLoad_ENVOverride_Integer(t *testing.T) {
	path := writeTempYAML(t, `max_chat_blob_buffer_token_size: 512`)
	t.Setenv("MEMOBASE_MAX_CHAT_BLOB_BUFFER_TOKEN_SIZE", "2048")

	cfg, err := config.Load[testConfig](path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.MaxChatBlobBufferTokenSize != 2048 {
		t.Errorf("MaxChatBlobBufferTokenSize = %d, want 2048 (ENV override)", cfg.MaxChatBlobBufferTokenSize)
	}
}

func TestValidate_AllPass(t *testing.T) {
	v := viper.New()
	v.Set("api_key", "secret")
	v.Set("port", 8080)
	v.Set("url", "http://localhost:8080")

	rules := []config.ValidationRule{
		config.RequireNonEmpty("api_key"),
		config.RequirePositiveInt("port"),
		config.RequireValidURL("url"),
	}
	if err := config.Validate(v, rules); err != nil {
		t.Errorf("Validate error: %v", err)
	}
}

func TestValidate_OneFailure(t *testing.T) {
	v := viper.New()
	v.Set("api_key", "")

	rules := []config.ValidationRule{
		config.RequireNonEmpty("api_key"),
	}
	err := config.Validate(v, rules)
	if err == nil {
		t.Error("expected error for empty api_key")
	}
}

func TestValidate_MultipleFailures(t *testing.T) {
	v := viper.New()
	// No values set → both fields fail

	rules := []config.ValidationRule{
		config.RequireNonEmpty("api_key"),
		config.RequirePositiveInt("port"),
	}
	err := config.Validate(v, rules)
	if err == nil {
		t.Error("expected error")
		return
	}
	errStr := err.Error()
	if errStr == "" {
		t.Error("error message is empty")
	}
	// Should mention both fields
	if !containsAll(errStr, "api_key", "port") {
		t.Errorf("error does not mention both fields: %s", errStr)
	}
}

func TestRequireNonEmpty_EmptyString(t *testing.T) {
	v := viper.New()
	v.Set("field", "")
	rule := config.RequireNonEmpty("field")
	if rule.Check(v) {
		t.Error("empty string should fail RequireNonEmpty")
	}
}

func TestRequirePositiveInt_Zero(t *testing.T) {
	v := viper.New()
	v.Set("field", 0)
	rule := config.RequirePositiveInt("field")
	if rule.Check(v) {
		t.Error("zero should fail RequirePositiveInt")
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
