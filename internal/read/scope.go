package read

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"kg-service/internal/access"
	"kg-service/internal/write"
)

// ErrScopeReadUnsupported is returned when the configured source store cannot page a graph scope.
// It is a real capability guard, not a placeholder: every store wired in production implements
// ScopeReader, and a deployment that somehow does not must fail loudly rather than return an empty
// graph that a caller would mistake for "this scope is empty".
var ErrScopeReadUnsupported = errors.New("configured store does not support graph scope reads")

// ReadGraphByScope returns one page of the live nodes and relationships in a graph scope.
//
// It reads the write source of truth, never the graph/vector/full-text projections. Those are
// updated asynchronously by the outbox workers, so a caller that persists a graph and reloads it
// within the same unit of work would otherwise read back the previous state.
//
// Node and relationship pages advance independently; see GraphScopeReadCursor.
func (s *Service) ReadGraphByScope(ctx context.Context, actor access.Identity, req GraphScopeReadRequest) (GraphScopeReadResponse, error) {
	if err := validateGraphScopeReadRequest(req); err != nil {
		return GraphScopeReadResponse{}, errors.Join(ErrValidation, err)
	}

	scopeReader, ok := s.sourceStore.(ScopeReader)
	if !ok {
		return GraphScopeReadResponse{}, ErrScopeReadUnsupported
	}

	owners, err := s.accessResolver.ResolveVisibleOwners(actor)
	if err != nil {
		return GraphScopeReadResponse{}, err
	}
	visibility := visibleOwnerSet(owners)

	query := write.ScopeQuery{
		ScopeFilter: write.ScopeFilter{
			DomainID:   strings.TrimSpace(req.DomainID),
			GraphScope: strings.TrimSpace(req.GraphScope),
			Levels:     req.Levels,
		},
		RefsOnly: req.RefsOnly,
		Limit:    req.Limit,
	}

	// A collection that has already been exhausted is not queried again. Re-issuing it with an
	// empty position would restart it from the beginning, because an empty position means "first
	// page" — that is exactly what the Done flags guard against.
	var (
		nodes     []write.NodeRecord
		nextNodes string
		nodesDone = req.Cursor.NodesDone
	)
	if !nodesDone {
		nodeQuery := query
		nodeQuery.Cursor = strings.TrimSpace(req.Cursor.Nodes)
		nodes, nextNodes, err = scopeReader.ListNodesByScope(ctx, nodeQuery)
		if err != nil {
			s.recordAudit(actor, "read", "kg_graph_scope", req.GraphScope, "deny", "scope_node_read_failed", nil)
			return GraphScopeReadResponse{}, err
		}
		nodesDone = nextNodes == ""
	}

	var (
		rels     []write.RelationshipRecord
		nextRels string
		relsDone = req.Cursor.RelationshipsDone
	)
	if !relsDone {
		relQuery := query
		relQuery.Cursor = strings.TrimSpace(req.Cursor.Relationships)
		rels, nextRels, err = scopeReader.ListRelationshipsByScope(ctx, relQuery)
		if err != nil {
			s.recordAudit(actor, "read", "kg_graph_scope", req.GraphScope, "deny", "scope_relationship_read_failed", nil)
			return GraphScopeReadResponse{}, err
		}
		relsDone = nextRels == ""
	}

	// Visibility is applied here rather than pushed into SQL because the owner set is resolved from
	// access grants, which the store layer knows nothing about. Filtering after paging can shorten a
	// page, so the cursor deliberately comes from the store's own page boundary, not from the
	// filtered slice: dropping an invisible row must not make the caller skip the rows behind it.
	visibleNodes := make([]GraphScopeNode, 0, len(nodes))
	hiddenNodes := 0
	for _, node := range nodes {
		if !isNodeVisible(node, visibility) {
			hiddenNodes++
			continue
		}
		visibleNodes = append(visibleNodes, toGraphScopeNode(node))
	}

	// Relationships are filtered on their own owner pair rather than on their endpoints being
	// visible. The two are equivalent here — a relationship and both its endpoints always share an
	// owner and a graph scope, which createRelationshipInScope enforces — and checking the owner
	// directly avoids making relationship visibility depend on which page its endpoints landed on.
	visibleRels := make([]GraphScopeRelationship, 0, len(rels))
	hiddenRels := 0
	for _, rel := range rels {
		if !isRelationshipVisible(rel, visibility) {
			hiddenRels++
			continue
		}
		visibleRels = append(visibleRels, toGraphScopeRelationship(rel))
	}

	outcome := "allow"
	reason := ""
	if len(visibleNodes) == 0 && len(visibleRels) == 0 && (hiddenNodes > 0 || hiddenRels > 0) {
		// Every row in this page belonged to an owner the caller cannot see. Recording it as a
		// denial is what makes an access misconfiguration visible in the audit trail instead of
		// looking like an empty scope.
		outcome = "deny"
		reason = "scope_not_visible"
	}
	s.recordAudit(actor, "read", "kg_graph_scope", req.GraphScope, outcome, reason, map[string]any{
		"domain_id":     req.DomainID,
		"graph_scope":   req.GraphScope,
		"levels":        levelsAudit(req.Levels),
		"refs_only":     req.RefsOnly,
		"nodes":         len(visibleNodes),
		"relationships": len(visibleRels),
		"hidden_nodes":  hiddenNodes,
		"hidden_rels":   hiddenRels,
	})

	cursor := GraphScopeReadCursor{
		Nodes:             nextNodes,
		Relationships:     nextRels,
		NodesDone:         nodesDone,
		RelationshipsDone: relsDone,
	}
	return GraphScopeReadResponse{
		Nodes:         visibleNodes,
		Relationships: visibleRels,
		NextCursor:    cursor,
		HasMore:       !cursor.IsZero(),
	}, nil
}

func validateGraphScopeReadRequest(req GraphScopeReadRequest) error {
	if strings.TrimSpace(req.DomainID) == "" {
		return errors.New("domain_id is required")
	}
	if strings.TrimSpace(req.GraphScope) == "" {
		return errors.New("graph_scope is required")
	}
	for i, level := range req.Levels {
		if strings.TrimSpace(level.Level) == "" {
			return errors.New("levels[" + strconv.Itoa(i) + "].level is required")
		}
	}
	if req.Limit < 0 {
		return errors.New("limit must not be negative")
	}
	return nil
}

// isRelationshipVisible mirrors isNodeVisible. Relationships carry the same owner pair as their
// endpoints, so the same grant set governs them.
func isRelationshipVisible(rel write.RelationshipRecord, visibility map[string]struct{}) bool {
	_, ok := visibility[nodeKey(rel.OwnerTenantID, rel.OwnerAppID)]
	return ok
}

func toGraphScopeNode(node write.NodeRecord) GraphScopeNode {
	return GraphScopeNode{
		ID:            node.ID,
		NodeType:      node.NodeType,
		DomainID:      node.DomainID,
		OwnerTenantID: node.OwnerTenantID,
		OwnerAppID:    node.OwnerAppID,
		Visibility:    node.Visibility,
		ExternalRef:   node.ExternalRef,
		Properties:    cloneMap(node.Properties),
		DomainVersion: node.DomainVersion,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     node.UpdatedAt,
	}
}

func toGraphScopeRelationship(rel write.RelationshipRecord) GraphScopeRelationship {
	return GraphScopeRelationship{
		ID:            rel.ID,
		RelType:       rel.RelType,
		FromNodeID:    rel.FromNodeID,
		ToNodeID:      rel.ToNodeID,
		DomainID:      rel.DomainID,
		OwnerTenantID: rel.OwnerTenantID,
		OwnerAppID:    rel.OwnerAppID,
		ExternalRef:   rel.ExternalRef,
		Properties:    cloneMap(rel.Properties),
		DomainVersion: rel.DomainVersion,
		CreatedAt:     rel.CreatedAt,
	}
}

func levelsAudit(levels []write.ScopeLevel) []string {
	if len(levels) == 0 {
		return []string{"*"}
	}
	out := make([]string, 0, len(levels))
	for _, level := range levels {
		if strings.TrimSpace(level.FeatureRef) == "" {
			out = append(out, level.Level)
			continue
		}
		out = append(out, level.Level+":"+level.FeatureRef)
	}
	return out
}
