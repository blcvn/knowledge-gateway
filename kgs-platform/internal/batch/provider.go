package batch

import (
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/overlay"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewPGWriter,
	NewSemanticDeduper,
	newNoopIndexer,
	NewUsecaseWithIndexer,
	newOverlayDeltaAppender,
	NewGraphBatchHandler,
)

func newNoopIndexer() VectorIndexer { return nil }

func newOverlayDeltaAppender(mgr *overlay.Manager) OverlayDeltaAppender { return mgr }
