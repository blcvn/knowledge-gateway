package index

import "context"

// Rebuilder performs a full index rebuild from stored observations.
type Rebuilder struct {
    manager *Manager
}

func NewRebuilder(manager *Manager) *Rebuilder {
    return &Rebuilder{manager: manager}
}

// Rebuild rebuilds the full index from all observations in the store.
func (r *Rebuilder) Rebuild(_ context.Context) error {
    // TODO: iterate observation store and re-add all items
    return nil
}
