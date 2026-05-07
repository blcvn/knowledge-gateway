package server

import (
	"kgs-platform/internal/kafka"

	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewWorkerServer, kafka.NewConsumer)
