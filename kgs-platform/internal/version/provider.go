package version

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewManager,
	NewGC,
	wire.Bind(new(VersionManager), new(*Manager)),
)
