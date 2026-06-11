package data

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/projection"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo, NewRegistryRepo, NewOntologyRepo, NewGraphRepo, NewRulesRepo, NewPolicyRepo, NewRedisClient, NewNeo4jDriver, NewQdrantClient, NewNATSClient, NewGormDB)

// NewRedisClient exposes the redis client to wire
func NewRedisClient(data *Data) *redis.Client {
	return data.rc
}

func NewNeo4jDriver(data *Data) neo4j.DriverWithContext {
	return data.neo4j
}

func NewQdrantClient(data *Data) *QdrantClient {
	return data.qdrant
}

func NewNATSClient(data *Data) *NATSClient {
	return data.nats
}

func NewGormDB(data *Data) *gorm.DB {
	return data.db
}

// Data .
type Data struct {
	db     *gorm.DB
	neo4j  neo4j.DriverWithContext
	rc     *redis.Client
	qdrant *QdrantClient
	nats   *NATSClient
	opa    string
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(logger)

	// Postgres Setup
	db, err := gorm.Open(postgres.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		helper.Fatalf("failed opening connection to postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		helper.Fatalf("failed to get postgres sql DB handle: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		helper.Fatalf("failed pinging postgres: %v", err)
	}

	// Auto-Migrate Schemas
	if err := db.AutoMigrate(
		&biz.App{},
		&biz.APIKey{},
		&biz.Quota{},
		&biz.AuditLog{},
		&biz.EntityType{},
		&biz.RelationType{},
		&biz.Rule{},
		&biz.RuleExecution{},
		&biz.Policy{},
		&version.GraphVersion{},
		&projection.ViewDefinitionRecord{},
	); err != nil {
		helper.Errorf("failed to auto-migrate postgres schemas: %v", err)
	}
	if err := ensureOntologyUniqueIndexes(context.Background(), db, helper); err != nil {
		helper.Fatalf("failed ensuring ontology unique indexes: %v", err)
	}

	// Neo4j Setup
	driver, err := neo4j.NewDriverWithContext(
		c.Neo4J.Uri,
		neo4j.BasicAuth(c.Neo4J.User, c.Neo4J.Password, ""),
	)
	if err != nil {
		helper.Fatalf("failed creating neo4j driver: %v", err)
	}
	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		helper.Fatalf("failed verifying neo4j connectivity: %v", err)
	}
	if err := EnsureConstraints(context.Background(), driver); err != nil {
		// Do not crash service startup on dirty legacy data; log and continue.
		helper.Warnf("failed ensuring neo4j constraints: %v", err)
	}

	// Redis Setup
	readTimeout := c.Redis.ReadTimeout.AsDuration()
	writeTimeout := c.Redis.WriteTimeout.AsDuration()

	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		helper.Fatalf("failed connecting to redis: %v", err)
	}

	d := &Data{
		db:    db,
		neo4j: driver,
		rc:    rdb,
		opa:   c.Opa.Url,
	}

	if qdrantCfg := c.GetQdrant(); qdrantCfg != nil && qdrantCfg.GetHost() != "" {
		qdrantClient, err := NewQdrantClientFromConfig(qdrantCfg, helper)
		if err != nil {
			helper.Fatalf("failed creating qdrant client: %v", err)
		}
		d.qdrant = qdrantClient
		if qdrantCfg.GetCollection() != "" {
			if err := qdrantClient.EnsureCollection(context.Background(), qdrantCfg.GetCollection(), int(qdrantCfg.GetVectorSize())); err != nil {
				helper.Fatalf("failed to ensure qdrant collection %s: %v", qdrantCfg.GetCollection(), err)
			}
		}
		if err := qdrantClient.Ping(context.Background()); err != nil {
			helper.Fatalf("failed verifying qdrant connectivity: %v", err)
		}
	}

	if natsCfg := c.GetNats(); natsCfg != nil && natsCfg.GetUrl() != "" {
		natsClient, err := NewNATSClientFromConfig(natsCfg, helper)
		if err != nil {
			helper.Fatalf("failed creating nats client: %v", err)
		}
		d.nats = natsClient
		if err := natsClient.Ping(context.Background()); err != nil {
			helper.Fatalf("failed verifying nats connectivity: %v", err)
		}
	}

	opaURL := ""
	if c.GetOpa() != nil {
		opaURL = c.GetOpa().GetUrl()
	}
	if err := verifyOPAConnectivity(context.Background(), opaURL); err != nil {
		helper.Fatalf("failed verifying OPA connectivity: %v", err)
	}

	cleanup := func() {
		helper.Info("closing the data resources")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		_ = driver.Close(context.Background())
		_ = rdb.Close()
		if d.nats != nil {
			if err := d.nats.Close(); err != nil {
				helper.Warnf("failed closing nats client: %v", err)
			}
		}
	}

	return d, cleanup, nil
}

func verifyOPAConnectivity(ctx context.Context, raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid OPA url %q: %w", raw, err)
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid OPA url %q", raw)
	}

	healthURL := parsed.Scheme + "://" + parsed.Host + "/health"
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("opa health check returned status %d", resp.StatusCode)
	}
	return nil
}

func ensureOntologyUniqueIndexes(ctx context.Context, db *gorm.DB, logger *log.Helper) error {
	plans := []struct {
		table string
		index string
	}{
		{table: "kgs_entity_types", index: "idx_app_tenant_entity"},
		{table: "kgs_relation_types", index: "idx_app_tenant_relation"},
	}
	for _, plan := range plans {
		if err := ensureOntologyTableUniqueIndex(ctx, db, logger, plan.table, plan.index); err != nil {
			return err
		}
	}
	return nil
}

func ensureOntologyTableUniqueIndex(ctx context.Context, db *gorm.DB, logger *log.Helper, table, index string) error {
	if err := normalizeOntologyTenantID(ctx, db, table); err != nil {
		return fmt.Errorf("normalize tenant_id for %s: %w", table, err)
	}

	exists, unique, err := queryIndexDefinition(ctx, db, table, index)
	if err != nil {
		return fmt.Errorf("query index %s on %s: %w", index, table, err)
	}
	if exists && unique {
		return nil
	}
	if exists && !unique {
		logger.Warnf("index %s on %s exists but is not unique; recreating as unique", index, table)
		dropSQL := fmt.Sprintf(`DROP INDEX IF EXISTS %s`, quoteIdent(index))
		if err := db.WithContext(ctx).Exec(dropSQL).Error; err != nil {
			return fmt.Errorf("drop non-unique index %s on %s: %w", index, table, err)
		}
	}

	if err := createOntologyUniqueIndex(ctx, db, table, index); err != nil {
		if !isDuplicateConstraintDataError(err) {
			return fmt.Errorf("create unique index %s on %s: %w", index, table, err)
		}
		removed, dedupeErr := dedupeOntologyRows(ctx, db, table)
		if dedupeErr != nil {
			return fmt.Errorf("dedupe rows for %s: %w", table, dedupeErr)
		}
		logger.Warnf("removed %d duplicate rows from %s before creating unique index %s", removed, table, index)
		if err := createOntologyUniqueIndex(ctx, db, table, index); err != nil {
			return fmt.Errorf("recreate unique index %s on %s after dedupe: %w", index, table, err)
		}
	}
	return nil
}

func normalizeOntologyTenantID(ctx context.Context, db *gorm.DB, table string) error {
	sql := fmt.Sprintf(`UPDATE %s SET tenant_id = 'default' WHERE tenant_id IS NULL OR tenant_id = ''`, quoteIdent(table))
	return db.WithContext(ctx).Exec(sql).Error
}

func queryIndexDefinition(ctx context.Context, db *gorm.DB, table, index string) (exists bool, unique bool, err error) {
	var indexDef string
	tx := db.WithContext(ctx).Raw(
		`SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND tablename = ? AND indexname = ? LIMIT 1`,
		table,
		index,
	).Scan(&indexDef)
	if tx.Error != nil {
		return false, false, tx.Error
	}
	if tx.RowsAffected == 0 {
		return false, false, nil
	}
	def := strings.ToUpper(strings.TrimSpace(indexDef))
	return true, strings.Contains(def, "UNIQUE INDEX"), nil
}

func createOntologyUniqueIndex(ctx context.Context, db *gorm.DB, table, index string) error {
	sql := fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (app_id, tenant_id, name)`,
		quoteIdent(index),
		quoteIdent(table),
	)
	return db.WithContext(ctx).Exec(sql).Error
}

func dedupeOntologyRows(ctx context.Context, db *gorm.DB, table string) (int64, error) {
	sql := fmt.Sprintf(`
WITH ranked AS (
	SELECT id,
	       ROW_NUMBER() OVER (PARTITION BY app_id, tenant_id, name ORDER BY id DESC) AS rn
	FROM %s
)
DELETE FROM %s AS t
USING ranked r
WHERE t.id = r.id
  AND r.rn > 1
`, quoteIdent(table), quoteIdent(table))
	tx := db.WithContext(ctx).Exec(sql)
	return tx.RowsAffected, tx.Error
}

func isDuplicateConstraintDataError(err error) bool {
	if err == nil {
		return false
	}
	raw := strings.ToLower(err.Error())
	return strings.Contains(raw, "duplicate key value violates unique constraint") ||
		strings.Contains(raw, "sqlstate 23505")
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
