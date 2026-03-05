package data

import (
	"context"
	"kgs-platform/internal/biz"
	"kgs-platform/internal/conf"
	"kgs-platform/internal/version"

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

	if qdrantCfg := c.GetQdrant(); qdrantCfg != nil && qdrantCfg.GetHost() != "" {
		qdrantClient, err := NewQdrantClientFromConfig(qdrantCfg, helper)
		if err != nil {
			helper.Fatalf("failed creating qdrant client: %v", err)
		}
		d.qdrant = qdrantClient
		if qdrantCfg.GetCollection() != "" {
			if err := qdrantClient.EnsureCollection(context.Background(), qdrantCfg.GetCollection(), int(qdrantCfg.GetVectorSize())); err != nil {
				helper.Errorf("failed to ensure qdrant collection %s: %v", qdrantCfg.GetCollection(), err)
			}
		}
	}

	if natsCfg := c.GetNats(); natsCfg != nil && natsCfg.GetUrl() != "" {
		natsClient, err := NewNATSClientFromConfig(natsCfg, helper)
		if err != nil {
			helper.Fatalf("failed creating nats client: %v", err)
		}
		d.nats = natsClient
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
