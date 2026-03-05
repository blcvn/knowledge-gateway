package overlay

import (
	"kgs-platform/internal/data"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewRedisStore,
	NewManager,
	NewSessionCloseListener,
	wire.Bind(new(OverlayManager), new(*Manager)),
	wire.Bind(new(EventPublisher), new(*data.NATSClient)),
)
