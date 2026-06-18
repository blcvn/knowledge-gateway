//go:build !nebula

package graphstore

func newNebulaDelegate(cfg CypherConfig) GraphAdapter {
	return NewInMemoryGraphAdapter()
}
