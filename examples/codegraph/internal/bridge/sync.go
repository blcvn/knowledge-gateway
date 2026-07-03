package bridge

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func EnsureBuildLayout(cfg Config) error {
	if err := os.MkdirAll(filepath.Join("examples", "codegraph", "bin"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.StateDir, 0o755); err != nil {
		return err
	}
	return nil
}

func SyncProject(ctx context.Context, cfg Config, dryRun bool, fullReindex bool) (SyncReport, error) {
	start := time.Now()
	if err := EnsureBuildLayout(cfg); err != nil {
		return SyncReport{}, err
	}

	commitSHA, _ := gitCommitSHA(ctx, cfg.ProjectPath)
	graph, err := ExtractGraph(ctx, cfg, commitSHA)
	if err != nil {
		return SyncReport{}, err
	}

	report := SyncReport{
		ProjectID:         cfg.ProjectID,
		CommitSHA:         commitSHA,
		NodeCount:         len(graph.Nodes),
		RelationshipCount: len(graph.Edges),
		NodeTypeCounts:    countNodeTypes(graph.Nodes),
		EdgeTypeCounts:    countEdgeTypes(graph.Edges),
		OutputDir:         cfg.StateDir,
	}

	if dryRun {
		printDryRunSummary(cfg, graph, report)
		report.Duration = time.Since(start)
		return report, nil
	}

	client := KGServiceClient(NewClient(cfg))
	if err := client.Ping(ctx); err != nil {
		return SyncReport{}, err
	}
	sessionResp, err := client.OpenSyncSession(ctx, OpenSyncSessionRequest{
		DomainID:   cfg.KGDomainID,
		GraphScope: "project:" + cfg.ProjectID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "SYNC_SCOPE_LOCKED") {
			return SyncReport{}, fmt.Errorf("sync scope locked for %s: %w", cfg.ProjectID, err)
		}
		return SyncReport{}, err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = client.AbandonSyncSession(ctx, sessionResp.SessionID)
	}()

	statePath := cfg.StatePath()
	state, err := LoadState(statePath)
	if err != nil {
		return SyncReport{}, err
	}
	defer func() {
		_ = SaveState(statePath, state)
	}()

	if fullReindex {
		if err := clearProject(ctx, client, state, sessionResp.GraphVersionID); err != nil {
			return SyncReport{}, err
		}
		state = State{
			Nodes:         map[string]StateNode{},
			Relationships: map[string]StateRelationship{},
		}
	}
	if !fullReindex && len(state.Nodes) == 0 && len(state.Relationships) == 0 {
		if err := clearProjectByPrefix(ctx, client, cfg.ProjectID+":", sessionResp.GraphVersionID); err != nil {
			return SyncReport{}, err
		}
	}

	if err := reconcileNodes(ctx, client, cfg, graph.Nodes, &state, &report, sessionResp.GraphVersionID); err != nil {
		return SyncReport{}, err
	}
	if err := reconcileRelationships(ctx, client, cfg, graph.Edges, &state, &report, sessionResp.GraphVersionID); err != nil {
		return SyncReport{}, err
	}
	if err := client.CommitSyncSession(ctx, sessionResp.SessionID); err != nil {
		return SyncReport{}, err
	}
	committed = true
	if err := SaveState(statePath, state); err != nil {
		return SyncReport{}, err
	}

	report.Duration = time.Since(start)
	return report, nil
}

func reconcileNodes(ctx context.Context, client KGServiceClient, cfg Config, nodes []NodeSpec, state *State, report *SyncReport, graphVersionID string) error {
	next := make(map[string]StateNode, len(state.Nodes))
	for key, value := range state.Nodes {
		next[key] = value
	}
	desired := map[string]struct{}{}

	batchSize := cfg.NodeBatchSize
	if batchSize <= 0 {
		batchSize = 200
	}
	createRequests := make([]NodeRequest, 0, len(nodes))
	createSpecs := make([]NodeSpec, 0, len(nodes))
	if len(nodes) > 0 {
		for _, node := range nodes {
			if existing, ok := state.Nodes[node.ExternalRef]; ok {
				updated, err := client.UpdateNode(ctx, existing.ID, NodeUpdateRequest{
					Properties:     node.Properties,
					Visibility:     node.Visibility,
					ExternalRef:    node.ExternalRef,
					GraphVersionID: graphVersionID,
				})
				if err != nil {
					if !isKGNotFound(err) {
						return fmt.Errorf("update node %s: %w", node.ExternalRef, err)
					}
					createRequests = append(createRequests, NodeRequest{
						DomainID:    cfg.KGDomainID,
						NodeType:    node.NodeType,
						Properties:  node.Properties,
						Visibility:  node.Visibility,
						ExternalRef: node.ExternalRef,
					})
					createSpecs = append(createSpecs, node)
					continue
				}
				_ = updated
				next[node.ExternalRef] = StateNode{ID: existing.ID, NodeType: node.NodeType}
				desired[node.ExternalRef] = struct{}{}
				report.UpdatedNodes++
				continue
			}
			createRequests = append(createRequests, NodeRequest{
				DomainID:    cfg.KGDomainID,
				NodeType:    node.NodeType,
				Properties:  node.Properties,
				Visibility:  node.Visibility,
				ExternalRef: node.ExternalRef,
			})
			createSpecs = append(createSpecs, node)
		}
		for start := 0; start < len(createRequests); start += batchSize {
			end := start + batchSize
			if end > len(createRequests) {
				end = len(createRequests)
			}
			batchRequests := createRequests[start:end]
			batchSpecs := createSpecs[start:end]
			resp, err := client.CreateNodesBulk(ctx, NodeBulkCreateRequest{Nodes: batchRequests, GraphVersionID: graphVersionID})
			if err != nil {
				return fmt.Errorf("bulk create nodes: %w", err)
			}
			failed := map[int]BulkItemError{}
			for _, failure := range resp.Failed {
				failed[failure.Index] = failure
			}
			successIndex := 0
			for idx, spec := range batchSpecs {
				if _, ok := failed[idx]; ok {
					continue
				}
				if successIndex >= len(resp.Succeeded) {
					return fmt.Errorf("bulk create nodes: response success count mismatch")
				}
				entry := resp.Succeeded[successIndex]
				successIndex++
				next[spec.ExternalRef] = StateNode{ID: entry.NodeID, NodeType: spec.NodeType}
				desired[spec.ExternalRef] = struct{}{}
				report.CreatedNodes++
			}
		}
	}
	for externalRef, node := range state.Nodes {
		if _, ok := desired[externalRef]; ok {
			continue
		}
		if err := client.DeleteNodeWithVersion(ctx, node.ID, graphVersionID); err != nil {
			if !isKGNotFound(err) {
				return fmt.Errorf("delete stale node %s: %w", node.ID, err)
			}
		}
		delete(next, externalRef)
		report.DeletedNodes++
	}
	state.Nodes = next
	return nil
}

func reconcileRelationships(ctx context.Context, client KGServiceClient, cfg Config, edges []EdgeSpec, state *State, report *SyncReport, graphVersionID string) error {
	next := make(map[string]StateRelationship, len(state.Relationships))
	for key, value := range state.Relationships {
		next[key] = value
	}
	createRequests := make([]RelationshipRequest, 0, len(edges))
	createKeys := make([]string, 0, len(edges))
	desired := map[string]struct{}{}

	for _, edge := range edges {
		fromNode, ok := state.Nodes[edge.FromExternalRef]
		if !ok {
			report.SkippedRelationships++
			continue
		}
		toNode, ok := state.Nodes[edge.ToExternalRef]
		if !ok {
			report.SkippedRelationships++
			continue
		}
		existing, ok := state.Relationships[edge.Key]
		if ok && existing.RelType == edge.RelType && existing.FromID == fromNode.ID && existing.ToID == toNode.ID {
			next[edge.Key] = existing
			desired[edge.Key] = struct{}{}
			continue
		}
		createRequests = append(createRequests, RelationshipRequest{
			RelType:    edge.RelType,
			FromNodeID: fromNode.ID,
			ToNodeID:   toNode.ID,
			DomainID:   cfg.KGDomainID,
			Properties: edge.Properties,
		})
		createKeys = append(createKeys, edge.Key)
		desired[edge.Key] = struct{}{}
	}

	staleIDs := make([]string, 0)
	for key, rel := range state.Relationships {
		if _, ok := desired[key]; ok {
			continue
		}
		staleIDs = append(staleIDs, rel.ID)
	}
	if len(staleIDs) > 0 {
		deleted, err := client.DeleteRelationshipsBulk(ctx, RelationshipBulkDeleteRequest{RelationshipIDs: staleIDs, GraphVersionID: graphVersionID})
		if err != nil {
			return fmt.Errorf("delete stale relationships: %w", err)
		}
		deletedSet := make(map[string]struct{}, len(deleted.RelationshipIDs))
		for _, id := range deleted.RelationshipIDs {
			deletedSet[id] = struct{}{}
		}
		for _, rel := range state.Relationships {
			if _, ok := deletedSet[rel.ID]; ok {
				for key, existing := range next {
					if existing.ID == rel.ID {
						delete(next, key)
					}
				}
				report.DeletedRelationships++
			}
		}
	}

	if len(createRequests) > 0 {
		batchSize := cfg.RelationshipBatchSize
		if batchSize <= 0 {
			batchSize = 200
		}
		for start := 0; start < len(createRequests); start += batchSize {
			end := start + batchSize
			if end > len(createRequests) {
				end = len(createRequests)
			}
			resp, err := client.CreateRelationshipsBulk(ctx, RelationshipBulkCreateRequest{Relationships: createRequests[start:end], GraphVersionID: graphVersionID})
			if err != nil {
				return fmt.Errorf("bulk create relationships: %w", err)
			}
			failed := map[int]BulkItemError{}
			for _, failure := range resp.Failed {
				failed[failure.Index] = failure
			}
			successIndex := 0
			for idx, key := range createKeys[start:end] {
				if _, ok := failed[idx]; ok {
					continue
				}
				if successIndex >= len(resp.Succeeded) {
					return fmt.Errorf("bulk create relationships: response success count mismatch")
				}
				entry := resp.Succeeded[successIndex]
				successIndex++
				req := createRequests[start:end][idx]
				next[key] = StateRelationship{
					ID:      entry.RelationshipID,
					RelType: req.RelType,
					FromID:  req.FromNodeID,
					ToID:    req.ToNodeID,
				}
				report.CreatedRelationships++
			}
		}
	}
	state.Relationships = next
	return nil
}

func toNodeRequests(cfg Config, nodes []NodeSpec) []NodeRequest {
	result := make([]NodeRequest, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, NodeRequest{
			DomainID:    cfg.KGDomainID,
			NodeType:    node.NodeType,
			Properties:  node.Properties,
			Visibility:  node.Visibility,
			ExternalRef: node.ExternalRef,
		})
	}
	return result
}

func printDryRunSummary(cfg Config, graph Graph, report SyncReport) {
	fmt.Printf("[sync:dry] project=%s commit=%s nodes=%d relationships=%d\n", cfg.ProjectID, report.CommitSHA, report.NodeCount, report.RelationshipCount)
	fmt.Printf("[sync:dry] node_types=%s\n", formatCounts(report.NodeTypeCounts))
	fmt.Printf("[sync:dry] edge_types=%s\n", formatCounts(report.EdgeTypeCounts))
	limit := minInt(10, len(graph.Nodes))
	if limit > 0 {
		fmt.Println("[sync:dry] sample_nodes:")
		for i := 0; i < limit; i++ {
			node := graph.Nodes[i]
			fmt.Printf("  - %s %s external_ref=%s\n", node.NodeType, node.Properties["name"], node.ExternalRef)
		}
	}
	limit = minInt(10, len(graph.Edges))
	if limit > 0 {
		fmt.Println("[sync:dry] sample_edges:")
		for i := 0; i < limit; i++ {
			edge := graph.Edges[i]
			fmt.Printf("  - %s %s -> %s\n", edge.RelType, edge.FromExternalRef, edge.ToExternalRef)
		}
	}
}

func countNodeTypes(nodes []NodeSpec) map[string]int {
	out := map[string]int{}
	for _, node := range nodes {
		out[node.NodeType]++
	}
	return out
}

func countEdgeTypes(edges []EdgeSpec) map[string]int {
	out := map[string]int{}
	for _, edge := range edges {
		out[edge.RelType]++
	}
	return out
}

func formatCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clearProject(ctx context.Context, client KGServiceClient, state State, graphVersionID string) error {
	for _, node := range state.Nodes {
		_ = client.DeleteNodeWithVersion(ctx, node.ID, graphVersionID)
	}
	return nil
}

func clearProjectByPrefix(ctx context.Context, client KGServiceClient, prefix, graphVersionID string) error {
	return client.DeleteNodesByExternalRefPrefixWithVersion(ctx, prefix, graphVersionID)
}

func isKGNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, " 404 ") || strings.Contains(message, " NOT_FOUND:")
}

func gitCommitSHA(ctx context.Context, projectPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", projectPath, "rev-parse", "--short=8", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
