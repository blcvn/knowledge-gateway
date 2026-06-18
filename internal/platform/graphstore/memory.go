package graphstore

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

type InMemoryGraphAdapter struct {
	mu    sync.RWMutex
	nodes map[string]GraphNode
	rels  map[string]GraphRelationship
}

func NewInMemoryGraphAdapter() *InMemoryGraphAdapter {
	return &InMemoryGraphAdapter{
		nodes: map[string]GraphNode{},
		rels:  map[string]GraphRelationship{},
	}
}

func (a *InMemoryGraphAdapter) UpsertNode(_ context.Context, node GraphNode) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	node.Properties = clone(node.Properties)
	node.Properties["_kg_sync_version"] = node.SyncVersion
	a.nodes[node.ID] = node
	return nil
}

func (a *InMemoryGraphAdapter) DeleteNode(_ context.Context, nodeID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.nodes, nodeID)
	for id, rel := range a.rels {
		if rel.FromNodeID == nodeID || rel.ToNodeID == nodeID {
			delete(a.rels, id)
		}
	}
	return nil
}

func (a *InMemoryGraphAdapter) UpsertRelationship(_ context.Context, rel GraphRelationship) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	rel.Properties = clone(rel.Properties)
	rel.Properties["_kg_sync_version"] = rel.SyncVersion
	a.rels[rel.ID] = rel
	return nil
}

func (a *InMemoryGraphAdapter) DeleteRelationship(_ context.Context, relID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.rels, relID)
	return nil
}

func (a *InMemoryGraphAdapter) ExecuteQuery(_ context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	nodes := make([]GraphNode, 0, len(a.nodes))
	for _, node := range a.nodes {
		nodes = append(nodes, node)
	}
	rels := make([]GraphRelationship, 0, len(a.rels))
	for _, rel := range a.rels {
		rels = append(rels, rel)
	}

	nodeByID := map[string]GraphNode{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	results := []map[string]any{}
	for _, node := range nodes {
		if node.IsDeleted || node.NodeType != query.StartNodeType {
			continue
		}
		if !matches(node.Properties, query.StartMatch, params) {
			continue
		}
		if !visible(node, params, query.ACLTokensParam) {
			continue
		}
		payload := clone(node.Properties)
		payload["id"] = node.ID
		payload["node_type"] = node.NodeType
		payload["domain_id"] = node.DomainID
		payload["_kg_sync_version"] = node.SyncVersion
		current := node
		ok := true
		for idx, hop := range query.Hops {
			next, found := nextHop(current, hop, nodeByID, rels, params, query.ACLTokensParam)
			if !found {
				ok = false
				break
			}
			current = next
			payload[fmt.Sprintf("hop_%d", idx+1)] = current.ID
		}
		if !ok {
			continue
		}
		result := map[string]any{}
		for _, field := range query.ReturnFields {
			if value, ok := payload[field]; ok {
				result[field] = value
			}
		}
		for k, v := range payload {
			result[k] = v
		}
		results = append(results, result)
	}
	slices.SortFunc(results, func(a, b map[string]any) int {
		return strings.Compare(fmt.Sprint(a["id"]), fmt.Sprint(b["id"]))
	})
	return results, nil
}

func (a *InMemoryGraphAdapter) ListNodes(_ context.Context) ([]GraphNode, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]GraphNode, 0, len(a.nodes))
	for _, node := range a.nodes {
		result = append(result, node)
	}
	return result, nil
}

func (a *InMemoryGraphAdapter) ListRelationships(_ context.Context) ([]GraphRelationship, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]GraphRelationship, 0, len(a.rels))
	for _, rel := range a.rels {
		result = append(result, rel)
	}
	return result, nil
}

func (a *InMemoryGraphAdapter) ReadSyncVersion(_ context.Context, entityID string) (int64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if node, ok := a.nodes[entityID]; ok {
		return node.SyncVersion, nil
	}
	if rel, ok := a.rels[entityID]; ok {
		return rel.SyncVersion, nil
	}
	return 0, nil
}

func clone(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func matches(props map[string]any, match map[string]any, params map[string]any) bool {
	for key, raw := range match {
		want, ok := raw.(string)
		if ok && strings.HasPrefix(want, "$") {
			if value, exists := params[strings.TrimPrefix(want, "$")]; exists {
				want = fmt.Sprint(value)
			}
		}
		got, exists := props[key]
		if !exists || fmt.Sprint(got) != want {
			return false
		}
	}
	return true
}

func visible(node GraphNode, params map[string]any, aclParam string) bool {
	visibleTokens, _ := params[aclParam].([]string)
	if len(visibleTokens) == 0 {
		return true
	}
	for _, candidate := range node.ACLVisibleTo {
		for _, token := range visibleTokens {
			if candidate == token {
				return true
			}
		}
	}
	return false
}

func nextHop(current GraphNode, hop GraphQueryHop, nodeByID map[string]GraphNode, rels []GraphRelationship, params map[string]any, aclParam string) (GraphNode, bool) {
	for _, rel := range rels {
		if rel.RelType != hop.RelType {
			continue
		}
		var candidateID string
		switch strings.ToLower(hop.Direction) {
		case "in":
			if rel.ToNodeID != current.ID {
				continue
			}
			candidateID = rel.FromNodeID
		default:
			if rel.FromNodeID != current.ID {
				continue
			}
			candidateID = rel.ToNodeID
		}
		candidate, ok := nodeByID[candidateID]
		if !ok || candidate.IsDeleted || candidate.NodeType != hop.ToNodeType {
			continue
		}
		if !visible(candidate, params, aclParam) {
			continue
		}
		if !matches(candidate.Properties, hop.Filter, params) {
			continue
		}
		return candidate, true
	}
	return GraphNode{}, false
}
