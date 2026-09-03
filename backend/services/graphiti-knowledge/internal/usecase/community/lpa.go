// Package community implements Label Propagation Algorithm (LPA)
// for discovering communities in the entity graph.
package community

import (
	"math/rand"
	"sort"
)

// LPAConfig configures the Label Propagation Algorithm
type LPAConfig struct {
	MaxIterations int     // convergence limit (default: 10)
	MinCluster    int     // minimum nodes per community (default: 3)
	Seed          int64   // random seed for reproducibility (0 = random)
}

var DefaultLPAConfig = LPAConfig{
	MaxIterations: 10,
	MinCluster:    3,
}

// RunLPA runs Label Propagation on the given adjacency data.
// clusters is a list of connected components where each component is a list of node UUIDs.
// Returns community partitions: map[communityLabel][]nodeUUID
func RunLPA(clusters [][]string, cfg LPAConfig) map[string][]string {
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 10
	}
	if cfg.MinCluster <= 0 {
		cfg.MinCluster = 1
	}

	// Assign initial labels = self UUID (each node is its own community)
	labels := make(map[string]string)
	for _, cluster := range clusters {
		for _, node := range cluster {
			labels[node] = node
		}
	}

	// Build adjacency for LPA (within connected components)
	adj := buildAdj(clusters)

	rng := rand.New(rand.NewSource(cfg.Seed))

	for iter := 0; iter < cfg.MaxIterations; iter++ {
		changed := false

		// Process nodes in random order (standard LPA)
		nodeOrder := allNodes(labels)
		rng.Shuffle(len(nodeOrder), func(i, j int) {
			nodeOrder[i], nodeOrder[j] = nodeOrder[j], nodeOrder[i]
		})

		for _, node := range nodeOrder {
			neighbors := adj[node]
			if len(neighbors) == 0 {
				continue
			}

			// Count neighbor labels
			labelCount := make(map[string]int)
			for _, n := range neighbors {
				labelCount[labels[n]]++
			}

			// Find the label with maximum count (tie-break: lexicographic)
			bestLabel := labels[node]
			bestCount := 0
			for l, c := range labelCount {
				if c > bestCount || (c == bestCount && l < bestLabel) {
					bestLabel = l
					bestCount = c
				}
			}

			if labels[node] != bestLabel {
				labels[node] = bestLabel
				changed = true
			}
		}

		if !changed {
			break // converged
		}
	}

	// Group nodes by label into communities
	communities := make(map[string][]string)
	for node, label := range labels {
		communities[label] = append(communities[label], node)
	}

	// Filter communities smaller than MinCluster
	filtered := make(map[string][]string)
	for label, nodes := range communities {
		if len(nodes) >= cfg.MinCluster {
			sort.Strings(nodes) // deterministic order
			filtered[label] = nodes
		}
	}
	return filtered
}

func buildAdj(clusters [][]string) map[string][]string {
	adj := make(map[string][]string)
	for _, cluster := range clusters {
		// Connect all nodes within a cluster (complete graph as proxy)
		// In practice the cluster comes from BFS/connected-components from Neo4j
		for i, node := range cluster {
			for j, other := range cluster {
				if i != j {
					adj[node] = append(adj[node], other)
				}
			}
		}
	}
	return adj
}

func allNodes(labels map[string]string) []string {
	nodes := make([]string, 0, len(labels))
	for n := range labels {
		nodes = append(nodes, n)
	}
	return nodes
}
