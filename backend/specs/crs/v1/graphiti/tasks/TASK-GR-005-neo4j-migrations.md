# TASK-GR-005 — Neo4j Schema Migrations (Cypher)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-005 |
| **Wave** | 1 (Foundation) |
| **Component** | `db/migrations/graphiti/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §5 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-002 (parallel) |
| **Estimated** | 1h |

---

## Context

Tạo Cypher migration scripts để khởi tạo Neo4j schema cho graphiti. Bao gồm uniqueness constraints, vector indices (1536-dim cosine), và fulltext indices. Chạy khi bootstrap `graphiti-store` lần đầu hoặc khi gọi `BuildIndicesAndConstraints`.

---

## Goal

- Uniqueness constraints cho 4 node labels
- Vector indices cho entity name, edge fact, community name
- Fulltext indices cho entity, episode, community, edge
- Property indices cho `group_id`, `valid_at`, `invalid_at` (query performance)
- Go migration runner sử dụng Neo4j driver

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `db/migrations/graphiti/001_constraints.cypher` |
| CREATE | `db/migrations/graphiti/002_vector_indices.cypher` |
| CREATE | `db/migrations/graphiti/003_fulltext_indices.cypher` |
| CREATE | `db/migrations/graphiti/004_property_indices.cypher` |
| CREATE | `services/graphiti-store/internal/infra/migration/runner.go` |

---

## Implementation

### File 1: `db/migrations/graphiti/001_constraints.cypher`

```cypher
// Uniqueness constraints — ensure UUID uniqueness per node label

CREATE CONSTRAINT entity_node_uuid IF NOT EXISTS
    FOR (n:Entity) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT episodic_node_uuid IF NOT EXISTS
    FOR (n:Episodic) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT community_node_uuid IF NOT EXISTS
    FOR (n:Community) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT saga_node_uuid IF NOT EXISTS
    FOR (n:Saga) REQUIRE n.uuid IS UNIQUE;
```

### File 2: `db/migrations/graphiti/002_vector_indices.cypher`

```cypher
// Vector indices for semantic similarity search
// Requires Neo4j 5.11+ Enterprise or AuraDB

CREATE VECTOR INDEX entity_name_embedding IF NOT EXISTS
    FOR (n:Entity) ON (n.name_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};

CREATE VECTOR INDEX entity_edge_fact_embedding IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.fact_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};

CREATE VECTOR INDEX community_name_embedding IF NOT EXISTS
    FOR (n:Community) ON (n.name_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};
```

### File 3: `db/migrations/graphiti/003_fulltext_indices.cypher`

```cypher
// Fulltext indices for BM25 keyword search

CREATE FULLTEXT INDEX entity_fulltext IF NOT EXISTS
    FOR (n:Entity) ON EACH [n.name, n.summary];

CREATE FULLTEXT INDEX episode_fulltext IF NOT EXISTS
    FOR (n:Episodic) ON EACH [n.content, n.source_description];

CREATE FULLTEXT INDEX community_fulltext IF NOT EXISTS
    FOR (n:Community) ON EACH [n.name, n.summary];

CREATE FULLTEXT INDEX entity_edge_fulltext IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON EACH [r.fact, r.name];
```

### File 4: `db/migrations/graphiti/004_property_indices.cypher`

```cypher
// Property indices for filtering by group_id and temporal fields

CREATE INDEX entity_group_id IF NOT EXISTS
    FOR (n:Entity) ON (n.group_id);

CREATE INDEX episode_group_id IF NOT EXISTS
    FOR (n:Episodic) ON (n.group_id);

CREATE INDEX episode_valid_at IF NOT EXISTS
    FOR (n:Episodic) ON (n.valid_at);

CREATE INDEX community_group_id IF NOT EXISTS
    FOR (n:Community) ON (n.group_id);

// RELATES_TO edge property indices (for temporal filtering)
CREATE INDEX edge_group_id IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.group_id);

CREATE INDEX edge_valid_at IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.valid_at);

CREATE INDEX edge_invalid_at IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.invalid_at);

CREATE INDEX edge_created_at IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.created_at);
```

### File 5: `services/graphiti-store/internal/infra/migration/runner.go`

```go
// Package migration runs Cypher migration scripts against Neo4j.
// Called on graphiti-store startup if GRAPHITI_AUTO_MIGRATE=true (default: true in dev)
package migration

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Runner struct {
    driver neo4j.DriverWithContext
    db     string
}

func NewRunner(driver neo4j.DriverWithContext, database string) *Runner {
    return &Runner{driver: driver, db: database}
}

// RunAll executes all .cypher files in the given directory in lexicographic order.
// Each file is executed statement by statement (separated by semicolons).
func (r *Runner) RunAll(ctx context.Context, migrationDir string) error {
    files, err := filepath.Glob(filepath.Join(migrationDir, "*.cypher"))
    if err != nil { return fmt.Errorf("glob migrations: %w", err) }
    sort.Strings(files)

    for _, file := range files {
        if err := r.runFile(ctx, file); err != nil {
            return fmt.Errorf("run migration %s: %w", filepath.Base(file), err)
        }
        fmt.Printf("✓ graphiti migration: %s\n", filepath.Base(file))
    }
    return nil
}

func (r *Runner) runFile(ctx context.Context, path string) error {
    content, err := os.ReadFile(path)
    if err != nil { return err }

    // Split on semicolons (simple; doesn't handle semicolons in strings)
    statements := splitStatements(string(content))

    for _, stmt := range statements {
        stmt = strings.TrimSpace(stmt)
        if stmt == "" || strings.HasPrefix(stmt, "//") { continue }

        _, err := neo4j.ExecuteQuery(ctx, r.driver, stmt, nil,
            neo4j.EagerResultTransformer,
            neo4j.ExecuteQueryWithDatabase(r.db),
        )
        if err != nil {
            // Log but don't fail on "already exists" type errors
            if isIdempotentError(err) {
                fmt.Printf("  skip (already exists): %s\n", truncate(stmt, 60))
                continue
            }
            return fmt.Errorf("execute statement: %w\nStatement: %s", err, truncate(stmt, 200))
        }
    }
    return nil
}

func splitStatements(content string) []string {
    var stmts []string
    scanner := bufio.NewScanner(strings.NewReader(content))
    var current strings.Builder

    for scanner.Scan() {
        line := scanner.Text()
        // Skip comment lines
        if strings.HasPrefix(strings.TrimSpace(line), "//") { continue }
        current.WriteString(line)
        current.WriteString("\n")
        if strings.HasSuffix(strings.TrimSpace(line), ";") {
            stmt := strings.TrimSuffix(strings.TrimSpace(current.String()), ";")
            stmts = append(stmts, stmt)
            current.Reset()
        }
    }
    if remaining := strings.TrimSpace(current.String()); remaining != "" {
        stmts = append(stmts, remaining)
    }
    return stmts
}

func isIdempotentError(err error) bool {
    msg := err.Error()
    return strings.Contains(msg, "already exists") ||
        strings.Contains(msg, "EquivalentSchemaRuleAlreadyExists")
}

func truncate(s string, n int) string {
    if len(s) <= n { return s }
    return s[:n] + "..."
}
```

---

## Verification

```bash
# Verify Cypher files are syntactically valid (requires cypher-shell)
cypher-shell -u neo4j -p password < db/migrations/graphiti/001_constraints.cypher

# Or verify runner compiles
cd services/graphiti-store
go build ./internal/infra/migration/...
```

**Manual verification in Neo4j Browser:**
```cypher
SHOW INDEXES YIELD name, type, state
WHERE name STARTS WITH "entity" OR name STARTS WITH "episode" OR name STARTS WITH "community"
RETURN name, type, state;
```

**Expected:** All indices show state `ONLINE`.
