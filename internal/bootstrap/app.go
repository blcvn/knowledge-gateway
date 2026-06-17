package bootstrap

import (
	"net/http"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/httpapi/respond"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
)

type App struct {
	config  config.Config
	pg      postgres.Client
	redis   rediscache.Client
	httpMux *http.ServeMux

	accessHandler    access.Handler
	accessMiddleware access.Middleware
	accessStore      *access.MemoryStore
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

	a.accessMiddleware = access.NewMiddleware(identityResolver)
	a.accessHandler = access.NewHandler(accessResolver, service)
}
