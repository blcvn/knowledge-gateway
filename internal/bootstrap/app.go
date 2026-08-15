package bootstrap

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	kratos "github.com/go-kratos/kratos/v2"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/httpapi/respond"
	"kg-service/internal/integrity"
	"kg-service/internal/mcp"
	"kg-service/internal/observability"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
	"kg-service/internal/read"
	"kg-service/internal/runtimeobs"
	"kg-service/internal/search"
	"kg-service/internal/workers"
	"kg-service/internal/write"
)

type App struct {
	config config.Config
	logger *runtimeobs.Logger
	pg     postgres.Client
	pgDB   *sql.DB
	redis  rediscache.Client

	accessHandler        access.Handler
	accessMiddleware     access.Middleware
	accessStore          access.TenantAppStore
	integrityHandler     integrity.Handler
	metricsHandler       observability.Handler
	mcpHandler           mcp.Handler
	ontologyHandler      ontology.Handler
	readHandler          read.Handler
	searchHandler        search.Handler
	writeHandler         write.Handler
	observabilityRuntime *observability.Runtime

	embeddingRouter vector.EmbeddingRouter
	vectorAdapter   vectorstore.VectorAdapter
	graphAdapter    graphstore.GraphAdapter
	ftsAdapter      fts.FTSAdapter
	searchResolver  ontology.SearchProfileResolver
	runtimeWorker   *workers.Runtime
}

func New(cfg config.Config) (*kratos.App, error) {
	bootstrapLogger := runtimeobs.NewLogger(cfg, "bootstrap")
	serviceLogger := kratoslog.With(bootstrapLogger, "trace_id", tracing.TraceID(), "span_id", tracing.SpanID())
	httpLogger := kratoslog.With(runtimeobs.NewLogger(cfg, "http"), "trace_id", tracing.TraceID(), "span_id", tracing.SpanID())
	observabilityRuntime := observability.NewRuntime(cfg)
	pg, err := postgres.New(cfg.Postgres)
	if err != nil {
		return nil, err
	}
	db, err := pg.Open()
	if err != nil {
		return nil, err
	}

	redisClient, err := rediscache.New(cfg.Redis)
	if err != nil {
		return nil, err
	}

	app := &App{
		config:               cfg,
		logger:               bootstrapLogger,
		pg:                   pg,
		pgDB:                 db,
		redis:                redisClient,
		observabilityRuntime: observabilityRuntime,
	}
	if err := app.initAccess(); err != nil {
		_ = db.Close()
		return nil, err
	}
	httpServer := httptransport.NewServer(
		httptransport.Address(cfg.HTTP.Address()),
		httptransport.Timeout(5*time.Second),
		httptransport.Middleware(
			logging.Server(httpLogger),
			metadata.Server(),
			recovery.Recovery(),
			tracing.Server(
				tracing.WithTracerProvider(observabilityRuntime.TracerProvider),
				tracing.WithPropagator(observabilityRuntime.Propagator),
				tracing.WithTracerName(cfg.Observability.ServiceName),
			),
			metrics.Server(httpServerMetrics(cfg)...),
		),
		httptransport.NotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("route miss method=%s path=%s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		})),
	)
	app.registerRoutes(httpServer.Route(""))

	server := kratos.New(
		kratos.Name(cfg.Observability.ServiceName),
		kratos.Version(cfg.Observability.ServiceVersion),
		kratos.Logger(serviceLogger),
		kratos.Server(httpServer, newWorkerServer(app)),
		kratos.StopTimeout(10*time.Second),
		kratos.BeforeStart(func(context.Context) error {
			kratoslog.NewHelper(serviceLogger).Infof("startup service=%s version=%s addr=%s", cfg.Observability.ServiceName, cfg.Observability.ServiceVersion, cfg.HTTP.Address())
			return nil
		}),
		kratos.BeforeStop(func(context.Context) error {
			kratoslog.NewHelper(serviceLogger).Infof("shutdown starting addr=%s", cfg.HTTP.Address())
			return nil
		}),
		kratos.AfterStop(func(ctx context.Context) error {
			if err := observabilityRuntime.Shutdown(context.Background()); err != nil {
				kratoslog.NewHelper(serviceLogger).Warnf("observability shutdown failed: %v", err)
			}
			kratoslog.NewHelper(serviceLogger).Infof("shutdown complete addr=%s", cfg.HTTP.Address())
			return nil
		}),
	)
	return server, nil
}

func (a *App) registerRoutes(router *httptransport.Router) {
	router.GET("/healthz", httpToKratosHandler(http.HandlerFunc(a.handleHealthz)))
	router.POST("/v1/tenants", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.CreateTenant))))
	router.GET("/v1/tenants/{tenant_id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.GetTenant))))
	router.PUT("/v1/tenants/{tenant_id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.UpdateTenant))))
	router.DELETE("/v1/tenants/{tenant_id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.DeleteTenant))))
	router.POST("/v1/tenants/{tenant_id}/apps", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.CreateApp))))
	router.POST("/v1/tenants/{tenant_id}/apps/{app_id}/rotate-key", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.RotateAppKey))))
	router.GET("/v1/tenants/{tenant_id}/apps", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.ListApps))))
	router.DELETE("/v1/tenants/{tenant_id}/apps/{app_id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.DeleteApp))))
	router.POST("/v1/access/grants", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.CreateGrant))))
	router.GET("/v1/access/grants", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.ListGrants))))
	router.DELETE("/v1/access/grants/{id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.DeleteGrant))))
	router.GET("/v1/access/audit", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.ListAudit))))
	router.GET("/v1/kg/integrity/tenant/{tenant_id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.integrityHandler.TenantIntegrity))))
	router.GET("/v1/kg/integrity/missing-bridges", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.integrityHandler.MissingBridges))))
	router.GET("/v1/kg/integrity/orphans", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.integrityHandler.OrphanScan))))
	router.POST("/v1/kg/integrity/repair/rebuild", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.integrityHandler.RebuildProjection))))
	router.POST("/v1/kg/integrity/repair/purge-orphans", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.integrityHandler.PurgeOrphans))))
	router.GET("/v1/kg/metrics", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.metricsHandler.Metrics))))
	router.GET("/v1/mcp/connect", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.mcpHandler.Connect))))
	router.POST("/v1/mcp/messages/{session_id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.mcpHandler.Message))))
	router.POST("/v1/tenants/{tenant_id}/ontology/domains", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateDomain))))
	router.POST("/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateNodeType))))
	router.POST("/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateRelType))))
	router.GET("/v1/tenants/{tenant_id}/ontology/effective", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetEffective))))
	router.POST("/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateQueryTemplate))))
	router.PUT("/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.ActivateQueryTemplate))))
	router.POST("/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.UpsertStatusFieldConfig))))
	router.PUT("/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/search-profile", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.UpsertSearchProfile))))
	router.GET("/v1/ontology/domains/{domain_id}/search-profile", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetSearchProfile))))
	router.POST("/v1/tenants/{tenant_id}/ontology/query-strategies", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateQueryStrategy))))
	router.PUT("/v1/tenants/{tenant_id}/ontology/query-strategies/{key}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.UpdateQueryStrategy))))
	router.GET("/v1/ontology/query-strategies", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.ListQueryStrategies))))
	router.GET("/v1/ontology/domains/{domain_id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetDomain))))
	router.GET("/v1/kg/read/templates", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.ListTemplates))))
	router.POST("/v1/kg/read/template/{domain_id}/{template_name}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.ExecuteTemplate))))
	router.GET("/v1/kg/read/nodes/{id}", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.GetNode))))
	router.POST("/v1/kg/read/graph:by-scope", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.ReadGraphByScope))))
	router.POST("/v1/kg/search/semantic", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.SemanticSearch))))
	router.POST("/v1/kg/search/rag", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.RagSearch))))
	router.POST("/v1/kg/search/fulltext", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.FullTextSearch))))
	router.POST("/v1/kg/search/hybrid", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.HybridSearch))))
	router.POST("/v1/kg/search/graph", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.GraphSearch))))
	router.POST("/v1/kg/write/nodes", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.CreateNode))))
	router.POST("/v1/kg/write/sync-sessions", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.OpenSyncSession))))
	router.POST("/v1/kg/write/sync-sessions/{id}/commit", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.CommitSyncSession))))
	router.DELETE("/v1/kg/write/sync-sessions/{id}", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.AbandonSyncSession))))
	router.POST("/v1/kg/write/nodes/bulk", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.CreateNodesBulk))))
	router.PUT("/v1/kg/write/nodes/{id}", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.UpdateNode))))
	router.DELETE("/v1/kg/write/nodes/{id}", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.DeleteNode))))
	router.POST("/v1/kg/write/relationships", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.CreateRelationship))))
	router.POST("/v1/kg/write/relationships/bulk", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.CreateRelationshipsBulk))))
	router.DELETE("/v1/kg/write/relationships/bulk", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.DeleteRelationshipsBulk))))
	router.DELETE("/v1/kg/write/nodes:by-external-ref-prefix", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.DeleteNodesByExternalRefPrefix))))
	router.DELETE("/v1/kg/write/relationships:by-external-ref", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.DeleteRelationshipsByExternalRef))))
	router.POST("/v1/kg/write/graph:delete-by-scope", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.DeleteByScope))))
	router.POST("/v1/kg/write/ingest/document", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.IngestDocument))))
	router.GET("/v1/kg/write/ingest/jobs/{job_id}", httpToKratosHandler(a.writeRoute(http.HandlerFunc(a.writeHandler.GetIngestJob))))
	router.GET("/v1/access/resolve", httpToKratosHandler(a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.GetResolve))))
}

func (a *App) writeRoute(next http.Handler) http.Handler {
	return a.accessMiddleware.RequireIdentity(next)
}

// httpToKratosHandler bridges a standard http.Handler into a Kratos HandlerFunc.
// Kratos uses gorilla/mux internally, which stores path vars via mux.Vars(req).
// Go 1.22's r.PathValue() reads from a different location in the request context,
// so we must bridge them: extract gorilla vars and inject via r.SetPathValue().
func httpToKratosHandler(handler http.Handler) httptransport.HandlerFunc {
	return func(ctx httptransport.Context) error {
		if handler == nil {
			return nil
		}
		request := ctx.Request()
		for key, value := range mux.Vars(request) {
			request.SetPathValue(key, value)
		}
		handler.ServeHTTP(ctx.Response(), request)
		return nil
	}
}

func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{
		"service": "kg-service",
		"postgres": map[string]any{
			"max_open_conns":    a.pg.MaxOpenConns,
			"max_idle_conns":    a.pg.MaxIdleConns,
			"conn_max_lifetime": a.pg.ConnMaxLifetime,
		},
		"redis": map[string]any{
			"address": a.redis.Address,
			"db":      a.redis.DB,
		},
	}

	respond.OK(w, payload)
}

func (a *App) initAccess() error {
	store := postgres.NewRepository(a.pgDB)
	seedAccessData(store)
	a.accessStore = store

	identityResolver := access.NewIdentityResolver(store, &a.redis)
	accessResolver := access.NewAccessResolver(store, store, &a.redis)
	service := access.NewService(store, &a.redis)
	service.WithPersistence(postgres.NewAccessPersistence(a.pgDB))
	rateLimiter := access.NewRateLimiter(store, map[string]int{
		"free":       a.config.RateLimit.FreePerMinute,
		"pro":        a.config.RateLimit.ProPerMinute,
		"enterprise": a.config.RateLimit.EnterprisePerMinute,
	})
	ontologyService := ontology.NewService(store, accessResolver)
	a.searchResolver = ontologyService
	bootstrapIdentity := access.Identity{
		TenantID: access.PlatformTenantID,
		AppID:    access.PlatformAdminAppID,
		AppType:  "admin_tool",
	}
	bootstrapSampleOntology(a.logger, ontologyService, bootstrapIdentity)
	seedCrossDomainRules(store)
	writeRepo := store
	sessionManager := postgres.NewSessionManager(a.pgDB)
	var err error
	a.embeddingRouter, err = buildEmbeddingRouter(a.config)
	if err != nil {
		return err
	}
	a.vectorAdapter, err = buildVectorAdapter(a.config, a.pgDB)
	if err != nil {
		return err
	}
	a.graphAdapter, err = buildGraphAdapter(a.config)
	if err != nil {
		return err
	}
	a.ftsAdapter, err = buildFTSAdapter(a.config.FTS.Kind, a.pgDB)
	if err != nil {
		return err
	}
	a.logger.Printf("embedding router chain: %s", strings.Join(embeddingChain(a.config), " -> "))
	writeService := write.NewService(writeRepo, ontologyService, accessResolver, store, sessionManager, service)
	writeService.SetSyncETAConfig(a.config.SyncEtaDefaultMs)
	writeService.SetSyncLagConfig(a.config.SyncLagToleranceMs, a.config.SyncLagStuckRetries)
	writeService.SetFTSBackendKind(a.config.FTS.Kind)
	runtimeWorker := workers.NewRuntimeFromConfig(writeRepo, ontologyService, &a.redis, a.config)
	if a.observabilityRuntime != nil {
		workers.WithTracerProvider(a.observabilityRuntime.TracerProvider, a.config.Observability.ServiceName+"/workers")(runtimeWorker)
	}
	runtimeWorker.SetEmbeddingRouter(a.embeddingRouter)
	runtimeWorker.SetGraphAdapter(a.graphAdapter)
	runtimeWorker.SetVectorAdapter(a.vectorAdapter)
	runtimeWorker.SetFTSAdapter(a.ftsAdapter)
	workers.WithSessionManager(sessionManager)(runtimeWorker)
	a.runtimeWorker = runtimeWorker
	readService := read.NewService(writeRepo, ontologyService, accessResolver, service)
	readService.SetGraphAdapter(a.graphAdapter)
	searchService := search.NewService(writeRepo, ontologyService, accessResolver, service, ontologyService)
	searchService.SetEmbeddingRouter(a.embeddingRouter)
	searchService.SetVectorAdapter(a.vectorAdapter)
	searchService.SetFTSAdapter(a.ftsAdapter)
	searchService.SetSearchProfileResolver(a.searchResolver)
	integrityService := integrity.NewService(writeRepo, store, runtimeWorker)
	mcpService := mcp.NewService(readService, searchService, writeService, ontologyService, accessResolver, integrityService, runtimeWorker)

	a.accessMiddleware = access.NewMiddleware(identityResolver, rateLimiter)
	a.accessHandler = access.NewHandler(accessResolver, service)
	a.integrityHandler = integrity.NewHandler(integrityService)
	a.metricsHandler = observability.NewHandler(observability.NewService(writeRepo, runtimeWorker))
	a.mcpHandler = mcp.NewHandler(mcpService, rateLimiter)
	a.ontologyHandler = ontology.NewHandler(ontologyService)
	a.readHandler = read.NewHandler(readService)
	a.searchHandler = search.NewHandler(searchService)
	a.writeHandler = write.NewHandler(writeService)
	return nil
}

func seedAccessData(store *postgres.Repository) {
	for _, tenant := range access.SeedTenants() {
		store.CreateTenant(tenant)
	}
	for _, app := range access.SeedApps() {
		store.CreateApp(app)
	}
	for _, grant := range access.SeedGrants() {
		store.CreateGrant(grant)
	}
}

func seedCrossDomainRules(store *postgres.Repository) {
	for _, rule := range ontology.SeedCrossDomainRules() {
		store.CreateCrossDomainRule(rule)
	}
}

func (a *App) runProjectionWorker(ctx context.Context) {
	if a.runtimeWorker == nil {
		return
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			a.logger.Printf("projection worker stopped")
			return
		case <-ticker.C:
			report := a.runtimeWorker.PollOnce()
			if report.Processed > 0 || report.Failed > 0 || report.DeadLetter > 0 {
				if len(report.SampleErrors) > 0 {
					a.logger.Printf("projection worker report: processed=%d failed=%d dead_letter=%d sample_errors=%v",
						report.Processed, report.Failed, report.DeadLetter, report.SampleErrors)
				} else {
					a.logger.Printf("projection worker report: processed=%d failed=%d dead_letter=%d",
						report.Processed, report.Failed, report.DeadLetter)
				}
			}
		}
	}
}

func (a *App) runSessionCleanupWorker(ctx context.Context) {
	if a.runtimeWorker == nil {
		return
	}
	a.runtimeWorker.RunSessionCleanupLoop(ctx)
}

func httpServerMetrics(cfg config.Config) []metrics.Option {
	meter := otel.Meter(cfg.Observability.ServiceName)
	var requests otelmetric.Int64Counter
	var err error
	requests, err = meter.Int64Counter(metrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil
	}
	var seconds otelmetric.Float64Histogram
	seconds, err = meter.Float64Histogram(metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil
	}
	return []metrics.Option{
		metrics.WithRequests(requests),
		metrics.WithSeconds(seconds),
	}
}

type workerServer struct {
	bundle *App
	done   chan struct{}
	once   sync.Once
}

func newWorkerServer(bundle *App) *workerServer {
	return &workerServer{bundle: bundle, done: make(chan struct{})}
}

func (s *workerServer) Start(ctx context.Context) error {
	defer s.signalDone()
	if s == nil || s.bundle == nil {
		return nil
	}
	if s.bundle.runtimeWorker == nil {
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.bundle.runProjectionWorker(ctx)
	}()
	go func() {
		defer wg.Done()
		s.bundle.runSessionCleanupWorker(ctx)
	}()
	<-ctx.Done()
	wg.Wait()
	return nil
}

func (s *workerServer) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *workerServer) signalDone() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		close(s.done)
	})
}
