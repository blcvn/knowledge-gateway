package data

import (
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/lock"

	"github.com/go-kratos/kratos/v2/log"
)

// StorageBundle groups all storage adapters needed by the application.
// This allows the DI layer to inject all storage dependencies as a single unit,
// regardless of whether the underlying backend is Specialized or SurrealDB.
type StorageBundle struct {
	// Core graph ports (biz/ interfaces)
	GraphRepo    biz.GraphRepo
	WriteRepo    biz.GraphWriteRepo
	Reader       biz.EntityReader
	RegistryRepo biz.RegistryRepo
	RulesRepo    biz.RulesRepo
	PolicyRepo   biz.PolicyRepo

	// Ontology
	OntologyRepo *OntologyRepo

	// Infrastructure ports
	LockMgr lock.LockManager

	// Sync layer (nil when SurrealDB mode — no CQRS fan-out needed)
	OutboxEnabled bool

	// Cleanup function for graceful shutdown
	Cleanup func()
}

// NewStorageFactory creates the appropriate StorageBundle based on config.
// storage_mode: "specialized" (default) | "surrealdb"
func NewStorageFactory(c *conf.Data, logger log.Logger) (*StorageBundle, func(), error) {
	mode := conf.StorageMode(c)
	l := log.NewHelper(logger)
	l.Infof("[KGS][StorageFactory] Initializing storage mode=%s", mode)

	switch mode {
	case "surrealdb":
		return newSurrealDBBundle(c, logger)
	default:
		return newSpecializedBundle(c, logger)
	}
}

// newSpecializedBundle creates a StorageBundle using the existing PG+Neo4j+Qdrant+Redis stack.
// This wraps the existing NewData() initialization logic.
func newSpecializedBundle(c *conf.Data, logger log.Logger) (*StorageBundle, func(), error) {
	l := log.NewHelper(logger)
	l.Info("[KGS][StorageFactory] Creating Specialized storage bundle (PG+Neo4j+Qdrant+Redis)")

	data, cleanup, err := NewData(c, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("specialized storage init: %w", err)
	}

	bundle := &StorageBundle{
		GraphRepo:     NewGraphRepo(data, logger),
		WriteRepo:     NewPGWriteRepo(data.db),
		Reader:        NewEntityReader(NewGraphRepo(data, logger), data.db),
		RegistryRepo:  NewRegistryRepo(data, logger),
		RulesRepo:     NewRulesRepo(data, logger),
		PolicyRepo:    NewPolicyRepo(data, logger),
		OntologyRepo:  NewOntologyRepo(data, logger),
		LockMgr:       lock.NewRedisLockManager(data.rc),
		OutboxEnabled: true,
		Cleanup:       cleanup,
	}

	return bundle, cleanup, nil
}

// newSurrealDBBundle creates a StorageBundle using SurrealDB as the unified backend.
// Outbox/Reconcile are disabled since there's no CQRS fan-out with a single store.
func newSurrealDBBundle(c *conf.Data, logger log.Logger) (*StorageBundle, func(), error) {
	l := log.NewHelper(logger)
	l.Info("[KGS][StorageFactory] Creating SurrealDB storage bundle")

	// TODO: Initialize SurrealDB client and create adapters
	// surrealCfg := c.GetSurrealdb()
	// client, cleanup, err := surrealdb.NewClient(...)

	return nil, nil, fmt.Errorf("surrealdb storage mode not yet implemented — pending Phase 2 adapter tasks")
}
