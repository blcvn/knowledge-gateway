package read

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/write"
)

type ProjectionStore interface {
	GetNodeByID(id string) (write.NodeRecord, bool)
	GetRelationshipByID(id string) (write.RelationshipRecord, bool)
	ListNodes() []write.NodeRecord
	ListRelationships() []write.RelationshipRecord
}

// ScopeReader pages through a graph scope in the write source of truth.
//
// It is a separate interface from ProjectionStore, not an extension of it, for the same reason the
// read service keeps sourceStore and index apart: a scope read must never be served from the
// asynchronous projections. A caller that writes a graph and immediately reads it back — which is
// what a single unit of agent work does — would otherwise see the pre-write state.
type ScopeReader interface {
	ListNodesByScope(ctx context.Context, query write.ScopeQuery) ([]write.NodeRecord, string, error)
	ListRelationshipsByScope(ctx context.Context, query write.ScopeQuery) ([]write.RelationshipRecord, string, error)
}

type GraphIndex interface {
	ExecuteTemplate(actor access.Identity, domainID string, compiled CompiledTemplate, bound map[string]any, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig, maxRows int, queryTimeout time.Duration, now func() time.Time) ([]map[string]any, error)
	GetNode(actor access.Identity, nodeID string, visibility map[string]struct{}) (NodeResponse, bool)
	ReadSyncVersion(nodeID string) (int64, error)
}

type ProjectionGraphIndex struct {
	adapter graphstore.GraphAdapter
}

func NewProjectionGraphIndex(store ProjectionStore) ProjectionGraphIndex {
	return ProjectionGraphIndex{adapter: newProjectionGraphAdapter(store)}
}

func (i ProjectionGraphIndex) ExecuteTemplate(actor access.Identity, domainID string, compiled CompiledTemplate, bound map[string]any, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig, maxRows int, queryTimeout time.Duration, now func() time.Time) ([]map[string]any, error) {
	params := map[string]any{}
	for k, v := range bound {
		params[k] = v
	}
	params[compiled.GraphQuery.ACLTokensParam] = visibleOwnerTokens(visibility)
	params["__query_timeout"] = queryTimeout
	params["__max_rows"] = maxRows
	params["__now"] = now
	params["__status_cfg"] = statusCfg
	results, err := i.adapter.ExecuteQuery(context.Background(), compiled.GraphQuery, params)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (i ProjectionGraphIndex) GetNode(actor access.Identity, nodeID string, visibility map[string]struct{}) (NodeResponse, bool) {
	nodes, err := i.adapter.ListNodes(context.Background())
	if err != nil {
		return NodeResponse{}, false
	}
	rels, err := i.adapter.ListRelationships(context.Background())
	if err != nil {
		return NodeResponse{}, false
	}
	for _, node := range nodes {
		if node.ID != nodeID || node.IsDeleted {
			continue
		}
		if _, ok := visibility[nodeKey(node.OwnerTenantID, node.OwnerAppID)]; !ok {
			continue
		}
		relationships := []string{}
		for _, rel := range rels {
			if rel.FromNodeID == node.ID || rel.ToNodeID == node.ID {
				relationships = append(relationships, rel.ID)
			}
		}
		return NodeResponse{
			ID:            node.ID,
			NodeType:      node.NodeType,
			DomainID:      node.DomainID,
			OwnerTenantID: node.OwnerTenantID,
			OwnerAppID:    node.OwnerAppID,
			Visibility:    node.Visibility,
			SyncVersion:   node.SyncVersion,
			Properties:    node.Properties,
			Relationships: relationships,
			CreatedAt:     node.CreatedAt,
			UpdatedAt:     node.UpdatedAt,
		}, true
	}
	_ = actor
	return NodeResponse{}, false
}

func (i ProjectionGraphIndex) ReadSyncVersion(nodeID string) (int64, error) {
	return i.adapter.ReadSyncVersion(context.Background(), nodeID)
}

func visibleOwnerSet(owners []access.VisibleOwner) map[string]struct{} {
	result := map[string]struct{}{}
	for _, owner := range owners {
		result[nodeKey(owner.TenantID, owner.AppID)] = struct{}{}
	}
	return result
}

func visibleOwnerTokens(visibility map[string]struct{}) []string {
	tokens := make([]string, 0, len(visibility))
	for token := range visibility {
		tokens = append(tokens, token)
	}
	return tokens
}

type projectionGraphAdapter struct {
	store ProjectionStore
}

func newProjectionGraphAdapter(store ProjectionStore) graphstore.GraphAdapter {
	return projectionGraphAdapter{store: store}
}

func (p projectionGraphAdapter) UpsertNode(context.Context, graphstore.GraphNode) error { return nil }
func (p projectionGraphAdapter) DeleteNode(context.Context, string) error               { return nil }
func (p projectionGraphAdapter) UpsertRelationship(context.Context, graphstore.GraphRelationship) error {
	return nil
}
func (p projectionGraphAdapter) DeleteRelationship(context.Context, string) error { return nil }

func (p projectionGraphAdapter) ExecuteQuery(_ context.Context, query graphstore.GraphQuery, params map[string]any) ([]map[string]any, error) {
	nodes := p.store.ListNodes()
	rels := p.store.ListRelationships()
	nodeByID := map[string]write.NodeRecord{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	results := []map[string]any{}
	queryTimeout, _ := params["__query_timeout"].(time.Duration)
	maxRows, _ := params["__max_rows"].(int)
	nowFn, _ := params["__now"].(func() time.Time)
	if nowFn == nil {
		nowFn = time.Now
	}
	statusCfg, _ := params["__status_cfg"].(*ontology.StatusFieldConfig)
	started := nowFn()
	visibility := visibilityFromParams(params, query.ACLTokensParam)
	for _, node := range nodes {
		if queryTimeout > 0 && nowFn().Sub(started) > queryTimeout {
			return nil, ErrTimeout
		}
		if maxRows > 0 && len(results) >= maxRows {
			break
		}
		if node.IsDeleted || node.NodeType != query.StartNodeType {
			continue
		}
		if !matchesNode(node, query.StartMatch, params) {
			continue
		}
		if !isNodeVisible(node, visibility) {
			continue
		}
		payload := cloneMap(node.Properties)
		payload["id"] = node.ID
		payload["node_type"] = node.NodeType
		payload["domain_id"] = node.DomainID
		payload["_kg_sync_version"] = node.DomainVersion
		current := node
		ok := true
		for idx, hop := range query.Hops {
			next, found := nextVisibleHop(current, hop, nodeByID, rels, visibility, params, statusCfg)
			if !found {
				ok = false
				break
			}
			current = next
			payload[fmt.Sprintf("hop_%d", idx+1)] = current.ID
		}
		if statusCfg != nil && !passesLifecycle(node, statusCfg, nil, nil) {
			continue
		}
		if !ok {
			continue
		}
		selected := map[string]any{}
		for _, field := range query.ReturnFields {
			selected[field] = resolveFieldValue(node, current, field)
		}
		for k, v := range payload {
			selected[k] = v
		}
		results = append(results, selected)
	}
	return results, nil
}

func (p projectionGraphAdapter) ListNodes(context.Context) ([]graphstore.GraphNode, error) {
	nodes := p.store.ListNodes()
	result := make([]graphstore.GraphNode, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, graphstore.GraphNode{
			ID:            node.ID,
			NodeType:      node.NodeType,
			DomainID:      node.DomainID,
			OwnerTenantID: node.OwnerTenantID,
			OwnerAppID:    node.OwnerAppID,
			ACLVisibleTo:  append([]string(nil), node.ACLVisibleTo...),
			Visibility:    node.Visibility,
			StatusValue:   node.StatusValue,
			IsDeleted:     node.IsDeleted,
			SyncVersion:   int64(node.DomainVersion),
			Properties:    cloneMap(node.Properties),
			CreatedAt:     node.CreatedAt,
			UpdatedAt:     node.UpdatedAt,
		})
	}
	return result, nil
}

func nextVisibleHop(current write.NodeRecord, hop graphstore.GraphQueryHop, nodeByID map[string]write.NodeRecord, relationships []write.RelationshipRecord, visibility map[string]struct{}, params map[string]any, statusCfg *ontology.StatusFieldConfig) (write.NodeRecord, bool) {
	for _, rel := range relationships {
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
		if _, ok := visibility[nodeKey(candidate.OwnerTenantID, candidate.OwnerAppID)]; !ok {
			continue
		}
		if !matchesNode(candidate, hop.Filter, params) {
			continue
		}
		if hop.FilterStatus == "valid_only" && !passesLifecycle(candidate, statusCfg, nil, nil) {
			continue
		}
		return candidate, true
	}
	return write.NodeRecord{}, false
}

func (p projectionGraphAdapter) ListRelationships(context.Context) ([]graphstore.GraphRelationship, error) {
	rels := p.store.ListRelationships()
	result := make([]graphstore.GraphRelationship, 0, len(rels))
	for _, rel := range rels {
		result = append(result, graphstore.GraphRelationship{
			ID:          rel.ID,
			RelType:     rel.RelType,
			FromNodeID:  rel.FromNodeID,
			ToNodeID:    rel.ToNodeID,
			DomainID:    rel.DomainID,
			SyncVersion: int64(rel.DomainVersion),
			Properties:  cloneMap(rel.Properties),
		})
	}
	return result, nil
}

func (p projectionGraphAdapter) ReadSyncVersion(_ context.Context, entityID string) (int64, error) {
	if node, ok := p.store.GetNodeByID(entityID); ok {
		return int64(node.DomainVersion), nil
	}
	if rel, ok := p.store.GetRelationshipByID(entityID); ok {
		return int64(rel.DomainVersion), nil
	}
	return 0, nil
}

func visibilityFromParams(params map[string]any, key string) map[string]struct{} {
	tokens := map[string]struct{}{}
	raw, _ := params[key]
	switch value := raw.(type) {
	case []string:
		for _, token := range value {
			tokens[token] = struct{}{}
		}
	case []any:
		for _, item := range value {
			if token, ok := item.(string); ok {
				tokens[token] = struct{}{}
			}
		}
	}
	return tokens
}
