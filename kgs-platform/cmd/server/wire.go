//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"kgs-platform/internal/batch"
	"kgs-platform/internal/biz"
	"kgs-platform/internal/conf"
	"kgs-platform/internal/data"
	"kgs-platform/internal/lock"
	"kgs-platform/internal/search"
	"kgs-platform/internal/server"
	"kgs-platform/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, lock.ProviderSet, batch.ProviderSet, search.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
