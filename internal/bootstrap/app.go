package bootstrap

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/httpapi/respond"
	"kg-service/internal/integrity"
	"kg-service/internal/mcp"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
	"kg-service/internal/read"
	"kg-service/internal/search"
	"kg-service/internal/write"
)

type App struct {
	config  config.Config
	pg      postgres.Client
	pgDB    *sql.DB
	redis   rediscache.Client
	httpMux *http.ServeMux

	accessHandler    access.Handler
	accessMiddleware access.Middleware
	accessStore      *access.MemoryStore
	integrityHandler integrity.Handler
	mcpHandler       mcp.Handler
	ontologyHandler  ontology.Handler
	readHandler      read.Handler
	searchHandler    search.Handler
	writeHandler     write.Handler

	embeddingRouter vector.EmbeddingRouter
	vectorAdapter   vectorstore.VectorAdapter
	graphAdapter    graphstore.GraphAdapter
	ftsAdapter      fts.FTSAdapter
	searchResolver  ontology.SearchProfileResolver
}

func New(cfg config.Config) (*App, error) {
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
		config:  cfg,
		pg:      pg,
		pgDB:    db,
		redis:   redisClient,
		httpMux: http.NewServeMux(),
	}
	if err := app.initAccess(); err != nil {
		_ = db.Close()
		return nil, err
	}
	app.routes()

	return app, nil
}

func (a *App) Handler() http.Handler {
	return a.httpMux
}

func (a *App) routes() {
	a.httpMux.HandleFunc("GET /healthz", a.handleHealthz)
	a.httpMux.Handle("POST /v1/tenants", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.CreateTenant)))
	a.httpMux.Handle("GET /v1/tenants/{tenant_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.GetTenant)))
	a.httpMux.Handle("PUT /v1/tenants/{tenant_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.UpdateTenant)))
	a.httpMux.Handle("DELETE /v1/tenants/{tenant_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.DeleteTenant)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/apps", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.CreateApp)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.RotateAppKey)))
	a.httpMux.Handle("GET /v1/tenants/{tenant_id}/apps", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.ListApps)))
	a.httpMux.Handle("DELETE /v1/tenants/{tenant_id}/apps/{app_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.DeleteApp)))
	a.httpMux.Handle("POST /v1/access/grants", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.CreateGrant)))
	a.httpMux.Handle("GET /v1/access/grants", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.ListGrants)))
	a.httpMux.Handle("DELETE /v1/access/grants/{id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.DeleteGrant)))
	a.httpMux.Handle("GET /v1/access/audit", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.ListAudit)))
	a.httpMux.Handle("GET /v1/kg/integrity/tenant/{tenant_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.integrityHandler.TenantIntegrity)))
	a.httpMux.Handle("GET /v1/kg/integrity/missing-bridges", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.integrityHandler.MissingBridges)))
	a.httpMux.Handle("GET /v1/mcp/connect", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.mcpHandler.Connect)))
	a.httpMux.Handle("POST /v1/mcp/messages/{session_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.mcpHandler.Message)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateDomain)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateNodeType)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateRelType)))
	a.httpMux.Handle("GET /v1/tenants/{tenant_id}/ontology/effective", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetEffective)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateQueryTemplate)))
	a.httpMux.Handle("PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.ActivateQueryTemplate)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.UpsertStatusFieldConfig)))
	a.httpMux.Handle("PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/search-profile", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.UpsertSearchProfile)))
	a.httpMux.Handle("GET /v1/ontology/domains/{domain_id}/search-profile", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetSearchProfile)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/query-strategies", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateQueryStrategy)))
	a.httpMux.Handle("PUT /v1/tenants/{tenant_id}/ontology/query-strategies/{key}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.UpdateQueryStrategy)))
	a.httpMux.Handle("GET /v1/ontology/query-strategies", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.ListQueryStrategies)))
	a.httpMux.Handle("GET /v1/ontology/domains/{domain_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetDomain)))
	a.httpMux.Handle("GET /v1/kg/read/templates", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.ListTemplates)))
	a.httpMux.Handle("POST /v1/kg/read/template/{domain_id}/{template_name}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.ExecuteTemplate)))
	a.httpMux.Handle("GET /v1/kg/read/nodes/{id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.GetNode)))
	a.httpMux.Handle("POST /v1/kg/search/semantic", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.SemanticSearch)))
	a.httpMux.Handle("POST /v1/kg/search/rag", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.RagSearch)))
	a.httpMux.Handle("POST /v1/kg/search/fulltext", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.FullTextSearch)))
	a.httpMux.Handle("POST /v1/kg/search/hybrid", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.searchHandler.HybridSearch)))
	a.httpMux.Handle("POST /v1/kg/write/nodes", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.CreateNode)))
	a.httpMux.Handle("PUT /v1/kg/write/nodes/{id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.UpdateNode)))
	a.httpMux.Handle("DELETE /v1/kg/write/nodes/{id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.DeleteNode)))
	a.httpMux.Handle("POST /v1/kg/write/relationships", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.CreateRelationship)))
	a.httpMux.Handle("POST /v1/kg/write/ingest/document", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.IngestDocument)))
	a.httpMux.Handle("GET /v1/kg/write/ingest/jobs/{job_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.GetIngestJob)))
	a.httpMux.Handle("GET /v1/access/resolve", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.accessHandler.GetResolve)))
}

func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{
		"service": "kg-service",
		"postgres": map[string]any{
			"dsn": a.pg.DSN,
		},
		"redis": map[string]any{
			"address": a.redis.Address,
			"db":      a.redis.DB,
		},
	}

	respond.OK(w, payload)
}

func (a *App) initAccess() error {
	store := access.NewMemoryStore()
	store.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	a.accessStore = store

	identityResolver := access.NewIdentityResolver(store, &a.redis)
	accessResolver := access.NewAccessResolver(store, store, &a.redis)
	service := access.NewService(store, &a.redis)
	rateLimiter := access.NewRateLimiter(store)
	ontologyStore := ontology.NewMemoryStore()
	ontologyService := ontology.NewService(ontologyStore, accessResolver)
	a.searchResolver = ontologyService
	bootstrapIdentity := access.Identity{
		TenantID: access.PlatformTenantID,
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	bootstrapSampleOntology(ontologyService, bootstrapIdentity)
	ontologyStore.Seed(nil, nil, nil, nil, ontology.SeedCrossDomainRules(), nil, nil)
	writeRepo := postgres.NewRepository(a.pgDB)
	sessionManager := postgres.NewSessionManager(a.pgDB)
	var err error
	a.embeddingRouter, err = buildEmbeddingRouter(a.config)
	if err != nil {
		return err
	}
	a.vectorAdapter, err = buildVectorAdapter(a.config.Vector.Kind, a.pgDB)
	if err != nil {
		return err
	}
	a.graphAdapter, err = buildGraphAdapter(a.config.Graph.Kind)
	if err != nil {
		return err
	}
	a.ftsAdapter, err = buildFTSAdapter(a.config.FTS.Kind, a.pgDB)
	if err != nil {
		return err
	}
	log.Printf("embedding router chain: %s", strings.Join(embeddingChain(a.config), " -> "))
	writeService := write.NewService(writeRepo, ontologyService, accessResolver, sessionManager, service)
	readService := read.NewService(writeRepo, ontologyService, accessResolver, service)
	searchService := search.NewService(writeRepo, ontologyService, accessResolver, service, ontologyService)
	searchService.SetEmbeddingRouter(a.embeddingRouter)
	searchService.SetVectorAdapter(a.vectorAdapter)
	searchService.SetFTSAdapter(a.ftsAdapter)
	searchService.SetSearchProfileResolver(a.searchResolver)
	integrityService := integrity.NewService(writeRepo, ontologyStore)
	mcpService := mcp.NewService(readService, searchService, writeService, ontologyService, accessResolver, integrityService)

	a.accessMiddleware = access.NewMiddleware(identityResolver, rateLimiter)
	a.accessHandler = access.NewHandler(accessResolver, service)
	a.integrityHandler = integrity.NewHandler(integrityService)
	a.mcpHandler = mcp.NewHandler(mcpService, rateLimiter)
	a.ontologyHandler = ontology.NewHandler(ontologyService)
	a.readHandler = read.NewHandler(readService)
	a.searchHandler = search.NewHandler(searchService)
	a.writeHandler = write.NewHandler(writeService)
	return nil
}
