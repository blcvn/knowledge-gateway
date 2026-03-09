package biz

const (
	MaxAllowedDepth = 10
	// Keep in sync with search max top-k so full graph hydration does not fail on subgraph query.
	MaxAllowedNodes = 10000
)

// ValidateDepth limits how deep an impact/coverage/context query can go to prevent recursive blowups.
func ValidateDepth(requestedDepth int) error {
	if requestedDepth > MaxAllowedDepth {
		return ErrDepthExceeded
	}
	return nil
}

// ValidateNodeCount is intended to be used on Subgraph queries or result sets to enforce limits.
func ValidateNodeCount(requestedNodeCount int) error {
	if requestedNodeCount > MaxAllowedNodes {
		return ErrNodesExceeded
	}
	return nil
}
