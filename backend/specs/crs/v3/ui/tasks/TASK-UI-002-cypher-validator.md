# TASK-UI-002 — Cypher Query Validator

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-002 |
| **Wave** | 1 (Backend API) |
| **Solution** | [SOL-UI-001](../solutions/SOL-UI-001-Graph-Studio.md) §2.2 |
| **Component** | `gateway/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Mục tiêu

Implement Cypher query validator: whitelist read-only clauses, block write operations. Used by `POST /v1/console/graph/query` endpoint.

---

## Công việc cụ thể

### 1. Tạo `gateway/internal/usecase/cypher_validator.go` [NEW]

```go
package usecase

import (
    "fmt"
    "regexp"
    "strings"
)

// CypherValidator validates Cypher queries are read-only
type CypherValidator struct {
    forbiddenPatterns []*regexp.Regexp
}

// NewCypherValidator creates a validator with CR-UI-001 §4 security rules
func NewCypherValidator() *CypherValidator {
    return &CypherValidator{
        forbiddenPatterns: []*regexp.Regexp{
            regexp.MustCompile(`(?i)\bCREATE\b`),
            regexp.MustCompile(`(?i)\bMERGE\b`),
            regexp.MustCompile(`(?i)\bDELETE\b`),
            regexp.MustCompile(`(?i)\bDETACH\s+DELETE\b`),
            regexp.MustCompile(`(?i)\bSET\b`),
            regexp.MustCompile(`(?i)\bREMOVE\b`),
            regexp.MustCompile(`(?i)\bDROP\b`),
            regexp.MustCompile(`(?i)\bCALL\s+\{`),              // block CALL subqueries (may write)
            regexp.MustCompile(`(?i)\bLOAD\s+CSV\b`),           // block CSV import
            regexp.MustCompile(`(?i)\bAPOC\.LOAD\b`),           // block APOC load procedures
            regexp.MustCompile(`(?i)\bAPOC\.PERIODIC\b`),       // block APOC periodic procedures
        },
    }
}

// Validate returns an error if query contains write operations
func (v *CypherValidator) Validate(query string) error {
    query = strings.TrimSpace(query)
    if query == "" {
        return fmt.Errorf("query cannot be empty")
    }

    // Check max length to prevent DoS
    if len(query) > 10000 {
        return fmt.Errorf("query exceeds maximum length of 10000 characters")
    }

    for _, pattern := range v.forbiddenPatterns {
        if pattern.MatchString(query) {
            return fmt.Errorf("write operations are not allowed in Graph Studio (blocked: %s)",
                pattern.String())
        }
    }
    return nil
}

// ValidClauses returns the list of allowed Cypher clauses (for documentation)
func ValidClauses() []string {
    return []string{
        "MATCH", "OPTIONAL MATCH", "RETURN", "WITH",
        "WHERE", "ORDER BY", "LIMIT", "SKIP",
        "UNWIND", "CALL",   // CALL for read-only procedures only
        "EXISTS", "CASE", "FOREACH", // read contexts
    }
}
```

### 2. Unit tests `gateway/internal/usecase/cypher_validator_test.go` [NEW]

```go
package usecase_test

func TestCypherValidator_AllowsReadOnlyQueries(t *testing.T) {
    v := NewCypherValidator()
    validQueries := []string{
        "MATCH (n) RETURN n LIMIT 10",
        "MATCH (n:Person)-[r:KNOWS]->(m) WHERE n.name = 'Alice' RETURN n, r, m",
        "MATCH (n) WHERE n.tenant_id = 'x' RETURN count(n)",
        "MATCH (n:Event) RETURN n ORDER BY n.created_at DESC LIMIT 50",
        "OPTIONAL MATCH (n:Entity {id: $id})-[r]->(m) RETURN n, r, m",
        "WITH 1 AS x MATCH (n) RETURN n",
    }
    for _, q := range validQueries {
        assert.NoError(t, v.Validate(q), "should allow: %s", q)
    }
}

func TestCypherValidator_BlocksWriteOperations(t *testing.T) {
    v := NewCypherValidator()
    writeQueries := []string{
        "CREATE (n:Person {name: 'Bob'})",
        "MERGE (n:Person {name: 'Bob'}) ON CREATE SET n.created = timestamp()",
        "MATCH (n) DELETE n",
        "MATCH (n) DETACH DELETE n",
        "MATCH (n) SET n.updated = timestamp()",
        "MATCH (n) REMOVE n:Label",
        "LOAD CSV FROM 'file:///data.csv' AS row CREATE (:Row {data: row})",
        "CALL { CREATE (n) }",
    }
    for _, q := range writeQueries {
        assert.Error(t, v.Validate(q), "should block: %s", q)
    }
}

func TestCypherValidator_EmptyQuery_Error(t *testing.T) {
    v := NewCypherValidator()
    assert.Error(t, v.Validate(""))
    assert.Error(t, v.Validate("   "))
}

func TestCypherValidator_TooLong_Error(t *testing.T) {
    v := NewCypherValidator()
    longQuery := "MATCH (n) RETURN n " + strings.Repeat("x", 10000)
    assert.Error(t, v.Validate(longQuery))
}
```

---

## Acceptance Criteria

- [ ] `MATCH (n) RETURN n` → no error (allowed)
- [ ] `CREATE (n:Person)` → error (blocked)
- [ ] `MATCH (n) DELETE n` → error (blocked)
- [ ] `MATCH (n) SET n.x = 1` → error (blocked)
- [ ] `CALL { CREATE ... }` → error (blocked subquery write)
- [ ] Empty query → error
- [ ] Query > 10000 chars → error
- [ ] `go test ./gateway/internal/usecase/...` passes

## Files

```
gateway/internal/usecase/cypher_validator.go       [NEW]
gateway/internal/usecase/cypher_validator_test.go  [NEW]
```
