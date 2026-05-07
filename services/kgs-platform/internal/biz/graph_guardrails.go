package biz

import "errors"

const (
	MaxAllowedDepth = 10
	MaxAllowedNodes = 1000
)

var (
	ErrDepthExceeded = errors.New("requested query depth exceeds the maximum allowed limit")
	ErrNodesExceeded = errors.New("requested query node count exceeds the maximum allowed limit")
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
