package data

import (
	"context"
	"kgs-platform/internal/biz"
	"kgs-platform/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo, NewRegistryRepo, NewOntologyRepo, NewGraphRepo, NewRulesRepo, NewPolicyRepo, NewRedisClient)

// NewRedisClient exposes the redis client to wire
func NewRedisClient(data *Data) *redis.Client {
	return data.rc
}

// Data .
type Data struct {
	db    *gorm.DB
	neo4j neo4j.DriverWithContext
	rc    *redis.Client
	opa   string
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(logger)

	// Postgres Setup
	db, err := gorm.Open(postgres.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		helper.Fatalf("failed opening connection to postgres: %v", err)
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
	); err != nil {
		helper.Errorf("failed to auto-migrate postgres schemas: %v", err)
	}

	// Neo4j Setup
	driver, err := neo4j.NewDriverWithContext(
		c.Neo4J.Uri,
		neo4j.BasicAuth(c.Neo4J.User, c.Neo4J.Password, ""),
	)
	if err != nil {
		helper.Fatalf("failed creating neo4j driver: %v", err)
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

	cleanup := func() {
		helper.Info("closing the data resources")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		_ = driver.Close(context.Background())
		_ = rdb.Close()
	}

	return d, cleanup, nil
}
