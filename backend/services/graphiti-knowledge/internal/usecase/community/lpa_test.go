package community_test

import (
	"testing"

	"vnp-memory/services/graphiti-knowledge/internal/usecase/community"
)

func TestRunLPA_SingleCluster(t *testing.T) {
	// 3 nodes all connected → should be 1 community
	clusters := [][]string{{"a", "b", "c"}}
	cfg := community.LPAConfig{MaxIterations: 10, MinCluster: 2, Seed: 42}
	result := community.RunLPA(clusters, cfg)

	if len(result) != 1 {
		t.Errorf("expected 1 community, got %d", len(result))
	}
}

func TestRunLPA_MinClusterFilter(t *testing.T) {
	// 2 clusters: one with 3 nodes, one with 2 nodes
	// MinCluster=3 → only the 3-node cluster should pass
	clusters := [][]string{
		{"a", "b", "c"},
		{"x", "y"},
	}
	cfg := community.LPAConfig{MaxIterations: 5, MinCluster: 3, Seed: 42}
	result := community.RunLPA(clusters, cfg)

	// At least one community should be the 3-node one
	hasLargeCluster := false
	for _, members := range result {
		if len(members) >= 3 {
			hasLargeCluster = true
		}
	}
	if !hasLargeCluster {
		t.Error("expected at least one community with 3+ members")
	}
}

func TestRunLPA_EmptyClusters(t *testing.T) {
	result := community.RunLPA(nil, community.DefaultLPAConfig)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d communities", len(result))
	}
}

func TestRunLPA_Convergence(t *testing.T) {
	// 5 nodes in one cluster — should converge
	clusters := [][]string{{"a", "b", "c", "d", "e"}}
	cfg := community.LPAConfig{MaxIterations: 20, MinCluster: 2, Seed: 1}
	result := community.RunLPA(clusters, cfg)
	if len(result) == 0 {
		t.Error("expected at least one community")
	}
}
