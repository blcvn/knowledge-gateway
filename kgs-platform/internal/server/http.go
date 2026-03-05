package server

import (
	policy "kgs-platform/api/accesscontrol/v1"
	graph "kgs-platform/api/graph/v1"
	v1 "kgs-platform/api/helloworld/v1"
	ontology "kgs-platform/api/ontology/v1"
	pb "kgs-platform/api/registry/v1"
	rules "kgs-platform/api/rules/v1"
	"kgs-platform/internal/biz"
	"kgs-platform/internal/conf"
	"kgs-platform/internal/server/middleware"
	"kgs-platform/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, registry *service.RegistryService, ont *service.OntologyService, g *service.GraphService, ruleSrv *service.RulesService, policySrv *service.PolicyService, healthSrv *service.HealthService, registryUC *biz.RegistryUsecase, redisCli *redis.Client, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			middleware.Tracing(),
			middleware.Metrics(),
			recovery.Recovery(),
			middleware.Auth(registryUC, redisCli),
			middleware.Namespace(),
			middleware.RateLimiter(registryUC, middleware.NewRedisRateLimitStore(redisCli)),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterGreeterHTTPServer(srv, greeter)
	pb.RegisterRegistryHTTPServer(srv, registry)
	ontology.RegisterOntologyHTTPServer(srv, ont)
	graph.RegisterGraphHTTPServer(srv, g)
	rules.RegisterRulesHTTPServer(srv, ruleSrv)
	policy.RegisterAccessControlHTTPServer(srv, policySrv)
	srv.Handle("/metrics", promhttp.Handler())
	if healthSrv != nil {
		srv.HandleFunc("/healthz", healthSrv.Liveness)
		srv.HandleFunc("/readyz", healthSrv.Readiness)
	}
	return srv
}
