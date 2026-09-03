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
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
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
	if err != nil {
		return err
	}

	// Split on semicolons (simple; doesn't handle semicolons in strings)
	statements := splitStatements(string(content))

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "//") {
			continue
		}

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
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
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
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
