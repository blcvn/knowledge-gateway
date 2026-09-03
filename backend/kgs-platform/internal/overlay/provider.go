package overlay

import (
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewRedisStore,
	NewManager,
	newEventPublisher,
	NewSessionCloseListener,
	wire.Bind(new(OverlayManager), new(*Manager)),
)

func newEventPublisher(client *data.NATSClient) EventPublisher {
	return client
}
