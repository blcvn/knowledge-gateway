package read

import (
	"fmt"
	"strings"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/write"
)

type ProjectionStore interface {
	ListNodes() []write.NodeRecord
	ListRelationships() []write.RelationshipRecord
}

type GraphIndex interface {
	ExecuteTemplate(actor access.Identity, domainID string, compiled CompiledTemplate, bound map[string]any, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig, maxRows int, queryTimeout time.Duration, now func() time.Time) ([]map[string]any, error)
	GetNode(actor access.Identity, nodeID string, visibility map[string]struct{}) (NodeResponse, bool)
}

type ProjectionGraphIndex struct {
	store ProjectionStore
}

func NewProjectionGraphIndex(store ProjectionStore) ProjectionGraphIndex {
	return ProjectionGraphIndex{store: store}
}

func (i ProjectionGraphIndex) ExecuteTemplate(actor access.Identity, domainID string, compiled CompiledTemplate, bound map[string]any, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig, maxRows int, queryTimeout time.Duration, now func() time.Time) ([]map[string]any, error) {
	nodes := i.store.ListNodes()
	relationships := i.store.ListRelationships()
	nodeByID := map[string]write.NodeRecord{}
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}

	results := []map[string]any{}
	started := now()
	for _, node := range nodes {
		if queryTimeout > 0 && now().Sub(started) > queryTimeout {
			return nil, ErrTimeout
		}
		if maxRows > 0 && len(results) >= maxRows {
			break
		}
		if node.IsDeleted || node.DomainID != domainID || node.NodeType != compiled.StartType {
			continue
		}
		if !isNodeVisible(node, visibility) {
			continue
		}
		if !matchesNode(node, compiled.StartMatch, bound) {
			continue
		}
		if !passesLifecycle(node, statusCfg, nil, nil) {
			continue
		}

		result, ok := i.walkTemplate(node, compiled, bound, nodeByID, relationships, visibility, statusCfg)
		if !ok {
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func (i ProjectionGraphIndex) GetNode(actor access.Identity, nodeID string, visibility map[string]struct{}) (NodeResponse, bool) {
	for _, node := range i.store.ListNodes() {
		if node.ID != nodeID || node.IsDeleted {
			continue
		}
		if _, ok := visibility[nodeKey(node.OwnerTenantID, node.OwnerAppID)]; !ok {
			continue
		}
		relationships := []string{}
		for _, rel := range i.store.ListRelationships() {
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
			Properties:    node.Properties,
			Relationships: relationships,
			CreatedAt:     node.CreatedAt,
			UpdatedAt:     node.UpdatedAt,
		}, true
	}
	_ = actor
	return NodeResponse{}, false
}

func (i ProjectionGraphIndex) walkTemplate(start write.NodeRecord, compiled CompiledTemplate, bound map[string]any, nodeByID map[string]write.NodeRecord, relationships []write.RelationshipRecord, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig) (map[string]any, bool) {
	payload := cloneMap(start.Properties)
	payload["id"] = start.ID
	payload["node_type"] = start.NodeType
	payload["domain_id"] = start.DomainID

	current := start
	for idx, hop := range compiled.Hops {
		nextNode, ok := nextVisibleHop(current, hop.RelType, hop.ToNodeType, hop.Direction, hop.Filter, bound, nodeByID, relationships, visibility, statusCfg, hop.FilterStatus)
		if !ok {
			return nil, false
		}
		current = nextNode
		payload[fmt.Sprintf("hop_%d", idx+1)] = current.ID
	}

	selected := map[string]any{}
	for _, field := range compiled.ReturnFields {
		selected[field] = resolveFieldValue(start, current, field)
	}
	for k, v := range payload {
		selected[k] = v
	}
	return selected, true
}

func nextVisibleHop(current write.NodeRecord, relType, toType, direction string, filter map[string]any, bound map[string]any, nodeByID map[string]write.NodeRecord, relationships []write.RelationshipRecord, visibility map[string]struct{}, statusCfg *ontology.StatusFieldConfig, filterStatus string) (write.NodeRecord, bool) {
	for _, rel := range relationships {
		if rel.RelType != relType {
			continue
		}
		var candidateID string
		switch strings.ToLower(direction) {
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
		if !ok || candidate.IsDeleted || candidate.NodeType != toType {
			continue
		}
		if _, ok := visibility[nodeKey(candidate.OwnerTenantID, candidate.OwnerAppID)]; !ok {
			continue
		}
		if !matchesNode(candidate, filter, bound) {
			continue
		}
		if filterStatus == "valid_only" && !passesLifecycle(candidate, statusCfg, map[string]any{}, map[string]any{}) {
			continue
		}
		return candidate, true
	}
	return write.NodeRecord{}, false
}

func visibleOwnerSet(owners []access.VisibleOwner) map[string]struct{} {
	result := map[string]struct{}{}
	for _, owner := range owners {
		result[nodeKey(owner.TenantID, owner.AppID)] = struct{}{}
	}
	return result
}
