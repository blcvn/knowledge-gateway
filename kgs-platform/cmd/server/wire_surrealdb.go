package main

import (
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/analytics"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/batch"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	sdb "github.com/blcvn/knowledge-gateway/kgs-platform/internal/data/surrealdb"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/projection"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/search"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/server"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/service"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
)

// wireAppSurrealDB creates the Kratos application using SurrealDB as the unified storage backend.
// Key differences from specialized mode:
// - No Neo4j, Qdrant, Redis, NATS connections
// - No OutboxWorker or ReconcileJob (no CQRS fan-out)
// - QueryTranslator converts Cypher→SurrealQL
// - EnqueueOutbox is a no-op
func wireAppSurrealDB(confServer *conf.Server, confData *conf.Data, logger log.Logger) (*kratos.App, func(), error) {
	l := log.NewHelper(logger)
	l.Info("[KGS] Initializing SurrealDB mode...")

	// 1. SurrealDB client
	surrealCfg := conf.GetSurrealDBFromEnv()
	client, cleanup, err := sdb.NewClient(
		surrealCfg.URL,
		surrealCfg.Namespace,
		surrealCfg.Database,
		surrealCfg.User,
		surrealCfg.Password,
		logger,
	)
	if err != nil {
		return nil, nil, err
	}

	// 2. Schema init
	vectorSize := int(confData.GetEmbedding().GetVectorSize())
	if err := sdb.InitSchema(nil, client, vectorSize, logger); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("surrealdb schema init: %w", err)
	}

	// 3. Core adapters
	graphRepo := sdb.NewSurrealGraphRepo(client, logger)
	writeRepo := sdb.NewSurrealGraphWriteRepo(client, logger)
	entityReader := sdb.NewSurrealEntityReader(client, logger)
	registryRepo := sdb.NewSurrealRegistryRepo(client, logger)
	ontologyRepoSurreal := sdb.NewSurrealOntologyRepo(client, logger)
	rulesRepo := sdb.NewSurrealRulesRepo(client, logger)
	policyRepo := sdb.NewSurrealPolicyRepo(client, logger)
	lockMgr := sdb.NewSurrealLockManager(client, logger)
	_ = sdb.NewSurrealOverlayStore(client, logger)

	// 4. Search adapters
	vectorRetriever := sdb.NewSurrealVectorRetriever(client, logger)
	textRetriever := sdb.NewSurrealTextRetriever(client, logger)
	centralityScorer := sdb.NewSurrealCentralityScorer(client, logger)
	_ = vectorRetriever
	_ = textRetriever
	_ = centralityScorer

	// 5. Analytics
	analyticsCache := analytics.NewCache(nil) // no Redis in SurrealDB mode
	analyticsExecutor := sdb.NewSurrealAnalyticsExecutor(client, logger)
	analyticsEngine := analytics.NewEngine(analyticsExecutor, analyticsCache)

	// 6. Biz layer — same interfaces, different adapters
	queryPlanner := biz.NewQueryPlanner()
	opaClient := biz.NewOPAClient(logger)

	// OntologyValidator wraps the SurrealDB ontology repo
	ontologyValidatorConfig := newOntologyValidatorConfig(confData)
	ontologyValidator := biz.NewOntologyValidator(ontologyRepoSurreal, graphRepo, ontologyValidatorConfig, logger)

	// RegistryUsecase
	registryUsecase := biz.NewRegistryUsecase(registryRepo, logger)

	// GraphUsecase — lockMgr replaces Redis lock, overlayManager is nil for now
	graphUsecase := biz.NewGraphUsecaseWithStorage(
		graphRepo, writeRepo, entityReader, queryPlanner,
		opaClient, ontologyValidator,
		nil, // no Redis client in SurrealDB mode
		lockMgr,
		nil, // overlay manager TBD
		logger,
	)

	// RulesUsecase, PolicyUsecase
	rulesUsecase := biz.NewRulesUsecase(rulesRepo, logger)
	policyUsecase := biz.NewPolicyUsecase(policyRepo, logger)

	// RuleRunner, EventRunner, PolicySyncRunner
	ruleRunner := biz.NewRuleRunner(rulesRepo, graphRepo, logger)
	eventRunner := biz.NewEventRunner(rulesRepo, graphRepo, nil, logger) // no Redis
	policySyncRunner := biz.NewPolicySyncRunner(policyRepo, opaClient, logger)

	// 7. Service layer
	greeterUsecase := biz.NewGreeterUsecase(data.NewGreeterRepo(nil, logger))
	greeterService := service.NewGreeterService(greeterUsecase)
	registryService := service.NewRegistryService(registryUsecase)

	// OntologyService needs *data.OntologyRepo — pass nil for SurrealDB mode (TBD: adapter)
	ontologyService := service.NewOntologyService(nil, nil, nil)

	// ViewResolver
	projEngine := projection.NewEngine(nil, logger)
	viewResolver := biz.NewViewResolver(projEngine)

	// VersionManager (nil DB — version tracking via SurrealDB TBD)
	versionManager := version.NewManager(nil, logger)

	// Search engine using SurrealDB adapters
	_ = search.NewEngine(nil, nil, nil) // placeholder — need adapter wiring

	// Batch handler — simplified for SurrealDB
	batchUsecase := batch.NewUsecaseWithIndexer(nil, nil, nil, newBatchEntityValidator(ontologyValidator))

	graphService := service.NewGraphServiceWithGraphBatchAndReader(
		graphUsecase, batchUsecase, nil, nil, // entityReader: nil — SurrealDB reader TBD
		nil, // search engine TBD
		nil, // overlay manager TBD
		versionManager,
		analyticsEngine,
		viewResolver,
		projEngine,
	)

	rulesService := service.NewRulesService(rulesUsecase)
	policyService := service.NewPolicyService(policyUsecase)
	healthService := service.NewHealthService(nil, nil, nil, nil, nil, logger) // no infra clients

	// 8. Servers
	grpcServer := server.NewGRPCServer(confServer, greeterService, registryService, ontologyService, graphService, rulesService, policyService, registryUsecase, nil, logger)
	httpServer := server.NewHTTPServer(confServer, greeterService, registryService, ontologyService, graphService, rulesService, policyService, healthService, registryUsecase, nil, logger)

	// WorkerServer — NO outboxWorker, NO reconcileJob (SurrealDB = no CQRS fan-out)
	workerServer := server.NewWorkerServer(
		confData,
		ruleRunner, eventRunner, policySyncRunner,
		nil, // no overlay listener (no NATS)
		nil, // outboxWorker = nil → skipped
		nil, // reconcileJob = nil → skipped
		logger,
	)

	app := newApp(logger, grpcServer, httpServer, workerServer)
	return app, cleanup, nil
}
