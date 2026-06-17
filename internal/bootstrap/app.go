package bootstrap

import (
	"net/http"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/httpapi/respond"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/read"
	"kg-service/internal/write"
)

type App struct {
	config  config.Config
	pg      postgres.Client
	redis   rediscache.Client
	httpMux *http.ServeMux

	accessHandler    access.Handler
	accessMiddleware access.Middleware
	accessStore      *access.MemoryStore
	ontologyHandler  ontology.Handler
	readHandler      read.Handler
	writeHandler     write.Handler
}

func New(cfg config.Config) (*App, error) {
	pg, err := postgres.New(cfg.Postgres)
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
		redis:   redisClient,
		httpMux: http.NewServeMux(),
	}
	app.initAccess()
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
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateDomain)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateNodeType)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateRelType)))
	a.httpMux.Handle("GET /v1/tenants/{tenant_id}/ontology/effective", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetEffective)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.CreateQueryTemplate)))
	a.httpMux.Handle("PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.ActivateQueryTemplate)))
	a.httpMux.Handle("POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.UpsertStatusFieldConfig)))
	a.httpMux.Handle("GET /v1/ontology/domains/{domain_id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.ontologyHandler.GetDomain)))
	a.httpMux.Handle("GET /v1/kg/read/templates", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.ListTemplates)))
	a.httpMux.Handle("POST /v1/kg/read/template/{domain_id}/{template_name}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.ExecuteTemplate)))
	a.httpMux.Handle("GET /v1/kg/read/nodes/{id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.readHandler.GetNode)))
	a.httpMux.Handle("POST /v1/kg/write/nodes", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.CreateNode)))
	a.httpMux.Handle("PUT /v1/kg/write/nodes/{id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.UpdateNode)))
	a.httpMux.Handle("DELETE /v1/kg/write/nodes/{id}", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.DeleteNode)))
	a.httpMux.Handle("POST /v1/kg/write/relationships", a.accessMiddleware.RequireIdentity(http.HandlerFunc(a.writeHandler.CreateRelationship)))
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

func (a *App) initAccess() {
	store := access.NewMemoryStore()
	store.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	a.accessStore = store

	identityResolver := access.NewIdentityResolver(store, &a.redis)
	accessResolver := access.NewAccessResolver(store, store, &a.redis)
	service := access.NewService(store, &a.redis)
	ontologyStore := ontology.NewMemoryStore()
	ontologyService := ontology.NewService(ontologyStore, accessResolver)
	bootstrapIdentity := access.Identity{
		TenantID: access.PlatformTenantID,
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	bootstrapLegalOntology(ontologyService, bootstrapIdentity)
	ontologyStore.Seed(nil, nil, nil, nil, ontology.SeedCrossDomainRules(), nil, nil)
	writeStore := write.NewMemoryStore()
	sessionManager := &postgres.SessionManager{}
	writeService := write.NewService(writeStore, ontologyService, accessResolver, sessionManager, service)
	readService := read.NewService(writeStore, ontologyService, accessResolver, service)

	a.accessMiddleware = access.NewMiddleware(identityResolver)
	a.accessHandler = access.NewHandler(accessResolver, service)
	a.ontologyHandler = ontology.NewHandler(ontologyService)
	a.readHandler = read.NewHandler(readService)
	a.writeHandler = write.NewHandler(writeService)
}
