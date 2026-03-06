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
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/redis/go-redis/v9"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, greeter *service.GreeterService, registry *service.RegistryService, ont *service.OntologyService, g *service.GraphService, ruleSrv *service.RulesService, policySrv *service.PolicyService, registryUC *biz.RegistryUsecase, redisCli *redis.Client, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			middleware.Tracing(),
			middleware.Metrics(),
			recovery.Recovery(),
			middleware.Auth(registryUC, redisCli),
			middleware.Namespace(),
			middleware.RateLimiter(registryUC, middleware.NewRedisRateLimitStore(redisCli)),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterGreeterServer(srv, greeter)
	pb.RegisterRegistryServer(srv, registry)
	ontology.RegisterOntologyServer(srv, ont)
	graph.RegisterGraphServer(srv, g)
	rules.RegisterRulesServer(srv, ruleSrv)
	policy.RegisterAccessControlServer(srv, policySrv)
	return srv
}
