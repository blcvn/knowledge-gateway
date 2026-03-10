package lock

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewRedisLockManager)
