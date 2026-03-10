//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/analytics"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/batch"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/lock"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/overlay"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/projection"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/search"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/server"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/service"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func newBizOntologyRepo(repo *data.OntologyRepo) biz.OntologyRepo {
	return repo
}

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, lock.ProviderSet, batch.ProviderSet, search.ProviderSet, version.ProviderSet, overlay.ProviderSet, analytics.ProviderSet, projection.ProviderSet, biz.ProviderSet, service.ProviderSet, newOntologyValidatorConfig, newBizOntologyRepo, newBatchEntityValidator, newApp))
}
