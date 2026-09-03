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
