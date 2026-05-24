package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	surreal "github.com/surrealdb/surrealdb.go"
)

// Client wraps the SurrealDB connection and provides lifecycle management.
type Client struct {
	db  *surreal.DB
	log *log.Helper
}

// NewClient creates a new SurrealDB client and connects to the server.
// Returns the client, a cleanup function, and any error.
func NewClient(url, namespace, database, user, password string, logger log.Logger) (*Client, func(), error) {
	l := log.NewHelper(logger)
	ctx := context.Background()

	l.Infof("[KGS][SurrealDB] Connecting to %s ns=%s db=%s", url, namespace, database)
	db, err := surreal.FromEndpointURLString(ctx, url)
	if err != nil {
		return nil, nil, fmt.Errorf("surrealdb connect %s: %w", url, err)
	}

	// Authenticate — v1.4.0: SignIn returns (token string, error)
	if _, err := db.SignIn(ctx, map[string]interface{}{
		"username": user,
		"password": password,
	}); err != nil {
		db.Close(ctx)
		return nil, nil, fmt.Errorf("surrealdb signin: %w", err)
	}

	// Select namespace and database
	if err := db.Use(ctx, namespace, database); err != nil {
		db.Close(ctx)
		return nil, nil, fmt.Errorf("surrealdb use ns=%s db=%s: %w", namespace, database, err)
	}

	client := &Client{db: db, log: l}

	// Verify connectivity
	if err := client.Ping(context.Background()); err != nil {
		db.Close(ctx)
		return nil, nil, fmt.Errorf("surrealdb ping: %w", err)
	}

	l.Infof("[KGS][SurrealDB] Connected successfully to %s ns=%s db=%s", url, namespace, database)

	cleanup := func() {
		l.Info("[KGS][SurrealDB] Closing connection")
		db.Close(context.Background())
	}

	return client, cleanup, nil
}

// DB returns the underlying SurrealDB instance for direct queries.
func (c *Client) DB() *surreal.DB {
	return c.db
}

// Ping checks the SurrealDB connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	// SurrealDB doesn't have a native ping — use a lightweight query
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// v1.4.0: Query is a package-level generic function
	_, err := surreal.Query[any](ctx, c.db, "RETURN true", nil)
	if err != nil {
		return fmt.Errorf("surrealdb health check failed: %w", err)
	}
	return nil
}

// Query executes a SurrealQL query and returns the result.
func (c *Client) Query(ctx context.Context, sql string, vars map[string]any) (any, error) {
	// v1.4.0: Query is a package-level generic function
	result, err := surreal.Query[any](ctx, c.db, sql, vars)
	if err != nil {
		c.log.Errorf("[KGS][SurrealDB] Query failed sql=%q err=%v", truncate(sql, 200), err)
		return nil, err
	}
	return result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
