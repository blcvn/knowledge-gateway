package privacy

import (
	"regexp"
	"strings"
)

// Redactor performs security-focused redaction of sensitive values.
// It targets API keys, tokens, secrets, and other credentials that
// should never be persisted to storage or logs.
type Redactor struct {
	patterns []*redactPattern
}

type redactPattern struct {
	re          *regexp.Regexp
	replacement string
}

// NewRedactor creates a Redactor with all built-in secret patterns.
func NewRedactor() *Redactor {
	return &Redactor{
		patterns: []*redactPattern{
			// OpenAI / Anthropic API keys
			{regexp.MustCompile(`sk-[A-Za-z0-9\-_]{20,}`), "[REDACTED:openai_key]"},
			// Bearer tokens
			{regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._\-]+`), "[REDACTED:bearer_token]"},
			// JWT tokens (3 base64 parts separated by dots)
			{regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), "[REDACTED:jwt_token]"},
			// AWS Access Keys
			{regexp.MustCompile(`AKIA[A-Z0-9]{16}`), "[REDACTED:aws_key]"},
			// AWS Secret Access Keys
			{regexp.MustCompile(`(?i)aws[_\-]?secret[_\-]?access[_\-]?key[^\w]*[A-Za-z0-9/+]{40}`), "[REDACTED:aws_secret]"},
			// PEM private keys
			{regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[^-]*-----END [A-Z ]*PRIVATE KEY-----`), "[REDACTED:private_key]"},
			// Generic high-entropy env var assignments (key=value where value is long)
			{regexp.MustCompile(`(?i)[A-Z_]{4,}[_]?(TOKEN|KEY|SECRET|PASSWORD|PASS|PWD|APIKEY)[^\w]*[=:]\s*['""]?([A-Za-z0-9+/=]{20,})['""]?`), "[REDACTED:env_secret]"},
			// GitHub tokens
			{regexp.MustCompile(`gh[ps]_[A-Za-z0-9]{36}`), "[REDACTED:github_pat]"},
			// Slack tokens
			{regexp.MustCompile(`xox[baprs]-[A-Za-z0-9\-]{10,48}`), "[REDACTED:slack_token]"},
			// Anthropic Claude API key
			{regexp.MustCompile(`sk-ant-[A-Za-z0-9\-_]{40,}`), "[REDACTED:anthropic_key]"},
			// Database URLs with embedded credentials
			{regexp.MustCompile(`(?i)(postgres|postgresql|mysql|mongodb|redis)://[^:@]+:[^@]+@[^\s"']+`), "[REDACTED:database_url]"},
		},
	}
}

// Redact replaces all recognized sensitive patterns in text with redaction markers.
func (r *Redactor) Redact(text string) string {
	for _, p := range r.patterns {
		text = p.re.ReplaceAllString(text, p.replacement)
	}
	return text
}

// RedactBytes performs redaction on a byte slice.
func (r *Redactor) RedactBytes(data []byte) []byte {
	return []byte(r.Redact(string(data)))
}

// RedactMap redacts all string values in a map (shallow, not recursive).
func (r *Redactor) RedactMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = r.Redact(s)
		} else {
			out[k] = v
		}
	}
	return out
}

// ContainsSecret reports whether text likely contains a sensitive value.
func (r *Redactor) ContainsSecret(text string) bool {
	for _, p := range r.patterns {
		if p.re.MatchString(text) {
			return true
		}
	}
	return false
}

// DefaultRedactor is a package-level singleton for convenience.
var DefaultRedactor = NewRedactor()

// Redact is a package-level convenience function.
func Redact(text string) string {
	return DefaultRedactor.Redact(text)
}

// isHighEntropy checks if a string has high entropy (likely a secret).
func isHighEntropy(s string) bool {
	if len(s) < 20 {
		return false
	}
	charSet := make(map[rune]bool)
	for _, c := range s {
		charSet[c] = true
	}
	return len(charSet) > 10
}

// SanitizeKey strips sensitive key names.
func SanitizeKey(key string) string {
	lower := strings.ToLower(key)
	sensitiveKeywords := []string{"password", "secret", "token", "apikey", "api_key", "auth", "credential"}
	for _, kw := range sensitiveKeywords {
		if strings.Contains(lower, kw) {
			return "[REDACTED:sensitive_key]"
		}
	}
	return key
}

// Strip is an alias for Redact, provided for backward compatibility with tests.
func (r *Redactor) Strip(text string) string {
	return r.Redact(text)
}

// ContainsSensitive is an alias for ContainsSecret.
func (r *Redactor) ContainsSensitive(text string) bool {
	return r.ContainsSecret(text)
}

// init extends the DefaultRedactor with additional patterns for tests.
func init() {
	DefaultRedactor.patterns = append(DefaultRedactor.patterns,
		// Database URLs with passwords
		&redactPattern{
			regexp.MustCompile(`(?i)(postgres|postgresql|mysql|mongodb|redis)://[^:@]+:[^@]+@[^\s"']+`),
			"[REDACTED:database_url]",
		},
	)
}
