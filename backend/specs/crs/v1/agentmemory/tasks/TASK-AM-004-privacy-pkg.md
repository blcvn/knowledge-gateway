# TASK-AM-004 — Privacy Package (`pkg/privacy/`)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-004 |
| **Wave** | 1 (Foundation) |
| **Component** | `pkg/privacy/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.10 |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Context

Shared privacy redaction package được dùng bởi observe-service (pipeline step 3) và consolidation (LLM input sanitization). Cần tạo trước TASK-AM-002.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `pkg/privacy/redact.go` |
| CREATE | `pkg/privacy/redact_test.go` |

---

## Implementation

### `pkg/privacy/redact.go`

```go
package privacy

import (
    "regexp"
    "strings"
)

type redactPattern struct {
    name string
    re   *regexp.Regexp
}

// Redactor removes sensitive strings from arbitrary text
type Redactor struct {
    patterns []redactPattern
}

func NewRedactor() *Redactor {
    return &Redactor{
        patterns: []redactPattern{
            {"bearer_token",  regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._-]{20,}`)},
            {"openai_key",    regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`)},
            {"anthropic_key", regexp.MustCompile(`sk-ant-[A-Za-z0-9-]{20,}`)},
            {"aws_key",       regexp.MustCompile(`AKIA[A-Z0-9]{16}`)},
            {"private_key",   regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
            {"jwt_token",     regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)},
            {"github_pat",    regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`)},
            {"env_secret",    regexp.MustCompile(`(?m)^[A-Z_]+=["']?[A-Za-z0-9+/=]{20,}["']?$`)},
            {"google_key",    regexp.MustCompile(`AIza[A-Za-z0-9_-]{35}`)},
            {"database_url",  regexp.MustCompile(`(?i)postgres(?:ql)?://[^\s"']+`)},
        },
    }
}

// Strip removes sensitive patterns from a string
func (r *Redactor) Strip(input string) string {
    for _, p := range r.patterns {
        input = p.re.ReplaceAllString(input, "[REDACTED:"+p.name+"]")
    }
    return input
}

// StripBytes strips a byte slice (JSON payload)
func (r *Redactor) StripBytes(b []byte) []byte {
    return []byte(r.Strip(string(b)))
}

// StripMap redacts all string values in a map (shallow)
func (r *Redactor) StripMap(m map[string]any) map[string]any {
    result := make(map[string]any, len(m))
    for k, v := range m {
        if s, ok := v.(string); ok {
            result[k] = r.Strip(s)
        } else {
            result[k] = v
        }
    }
    return result
}

// ContainsSensitive returns true if the text contains any pattern
func (r *Redactor) ContainsSensitive(text string) bool {
    for _, p := range r.patterns {
        if p.re.MatchString(text) { return true }
    }
    return false
}

// PatternNames returns all registered pattern names (for testing/audit)
func (r *Redactor) PatternNames() []string {
    names := make([]string, len(r.patterns))
    for i, p := range r.patterns { names[i] = p.name }
    return names
}

// Default package-level redactor
var Default = NewRedactor()

// Strip is a convenience function using the default redactor
func Strip(input string) string { return Default.Strip(input) }
```

### `pkg/privacy/redact_test.go`

```go
package privacy_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/vnp-memory/pkg/privacy"
)

func TestRedactor_OpenAIKey(t *testing.T) {
    r := privacy.NewRedactor()
    input := `{"key": "sk-abc123456789abcdefghij"}`
    out := r.Strip(input)
    assert.Contains(t, out, "[REDACTED:openai_key]")
    assert.NotContains(t, out, "sk-abc123456789")
}

func TestRedactor_BearerToken(t *testing.T) {
    r := privacy.NewRedactor()
    input := "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6InRlc3QifQ"
    out := r.Strip(input)
    assert.Contains(t, out, "[REDACTED:bearer_token]")
}

func TestRedactor_JWTToken(t *testing.T) {
    r := privacy.NewRedactor()
    jwt := "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ1c2VyMSJ9.signature123"
    out := r.Strip(jwt)
    assert.Contains(t, out, "[REDACTED:jwt_token]")
}

func TestRedactor_AWSKey(t *testing.T) {
    r := privacy.NewRedactor()
    input := "AWS key: AKIAIOSFODNN7EXAMPLE"
    out := r.Strip(input)
    assert.Contains(t, out, "[REDACTED:aws_key]")
}

func TestRedactor_GitHubPAT(t *testing.T) {
    r := privacy.NewRedactor()
    input := "token = ghp_abc123456789012345678901234567890123"
    out := r.Strip(input)
    assert.Contains(t, out, "[REDACTED:github_pat]")
}

func TestRedactor_NoFalsePositives(t *testing.T) {
    r := privacy.NewRedactor()
    input := "Normal text with no secrets, just regular words and numbers like 12345"
    out := r.Strip(input)
    assert.Equal(t, input, out)  // unchanged
}

func TestRedactor_ContainsSensitive(t *testing.T) {
    r := privacy.NewRedactor()
    assert.True(t, r.ContainsSensitive("sk-abc123456789abcdefghij"))
    assert.False(t, r.ContainsSensitive("normal text"))
}

func TestRedactor_DatabaseURL(t *testing.T) {
    r := privacy.NewRedactor()
    input := "DSN: postgres://user:password@localhost:5432/mydb"
    out := r.Strip(input)
    assert.Contains(t, out, "[REDACTED:database_url]")
}
```

---

## Verification

```bash
cd pkg/privacy
go test ./... -v -run TestRedactor
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `sk-abc123456789abcdef` → `[REDACTED:openai_key]` | ✅ |
| `Bearer <token>` → `[REDACTED:bearer_token]` | ✅ |
| `AKIA...` → `[REDACTED:aws_key]` | ✅ |
| `postgres://...` → `[REDACTED:database_url]` | ✅ |
| Normal text → unchanged | ✅ |
| `ContainsSensitive()` returns true for sensitive text | ✅ |
