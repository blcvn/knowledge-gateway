package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/internal/repository"
)

// Service orchestrates MCP-specific KG read use cases.
type Service struct {
	repo *repository.Repository
}

var ErrFeatureNotFound = errors.New("feature not found")

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListDocuments(ctx context.Context, projectID string) ([]repository.DocumentSummary, error) {
	if strings.TrimSpace(projectID) != "" {
		return s.repo.ListDocumentsByProject(ctx, strings.TrimSpace(projectID))
	}
	return s.repo.ListDocuments(ctx)
}

type DocumentSubgraphOptions struct {
	MaxNodes     int
	NodeTypes    []string
	IncludeEdges bool
}

type DocumentSubgraph struct {
	Nodes []repository.RequirementNode `json:"nodes"`
	Edges []repository.DependencyEdge  `json:"edges"`
	Stats DocumentSubgraphStats        `json:"stats"`
}

type DocumentSubgraphStats struct {
	NodeCount int  `json:"node_count"`
	EdgeCount int  `json:"edge_count"`
	Truncated bool `json:"truncated"`
}

type FeatureSearchResult struct {
	FeatureID   string `json:"feature_id"`
	FeatureName string `json:"feature_name"`
	DocumentID  string `json:"document_id"`
	Score       int    `json:"score"`
	Summary     string `json:"summary"`
}

type FeatureDetail struct {
	FeatureID          string                       `json:"feature_id"`
	FeatureName        string                       `json:"feature_name"`
	DocumentID         string                       `json:"document_id"`
	FeatureDetails     []repository.RequirementNode `json:"feature_details"`
	BusinessRules      []repository.RequirementNode `json:"business_rules"`
	UIScreens          []repository.RequirementNode `json:"ui_screens"`
	UserFlows          []repository.RequirementNode `json:"user_flows"`
	AcceptanceCriteria []repository.RequirementNode `json:"acceptance_criteria"`
	Permissions        []repository.RequirementNode `json:"permissions"`
	NonFunctional      []repository.RequirementNode `json:"non_functional"`
	AdditionalNodes    []repository.RequirementNode `json:"additional_nodes"`
}

type UpsertNodePayload struct {
	ID          string         `json:"id"`
	ReferenceID string         `json:"reference_id,omitempty"`
	Type        string         `json:"type"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	SourceID    string         `json:"source_id,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

type UpsertEdgePayload struct {
	ID         string         `json:"id,omitempty"`
	Source     string         `json:"source"`
	Target     string         `json:"target"`
	Type       string         `json:"type"`
	Reason     string         `json:"reason,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

type GraphSnapshot struct {
	Nodes    []UpsertNodePayload `json:"nodes"`
	Edges    []UpsertEdgePayload `json:"edges"`
	Metadata map[string]any      `json:"metadata,omitempty"`
}

type SaveDocumentResult struct {
	DocumentID  string `json:"document_id"`
	DocKind     string `json:"doc_kind"`
	Saved       bool   `json:"saved"`
	ContentType string `json:"content_type"`
}

var allowedDocumentKinds = map[string]struct{}{
	"product_index":    {},
	"project_overview": {},
	"urd_feature":      {},
}

func (s *Service) GetDocumentSubgraph(ctx context.Context, documentID string, opts DocumentSubgraphOptions) (*DocumentSubgraph, error) {
	documentID = strings.TrimSpace(documentID)
	if opts.MaxNodes <= 0 {
		opts.MaxNodes = 200
	}
	if !opts.IncludeEdges {
		opts.IncludeEdges = false
	}

	totalNodes, err := s.repo.CountNodesByDocumentID(ctx, documentID, opts.NodeTypes)
	if err != nil {
		return nil, err
	}

	nodes, err := s.repo.ListNodesByDocumentID(ctx, documentID, opts.NodeTypes, opts.MaxNodes)
	if err != nil {
		return nil, err
	}

	edges := make([]repository.DependencyEdge, 0)
	if opts.IncludeEdges && len(nodes) > 0 {
		nodeIDs := make([]string, 0, len(nodes))
		for _, node := range nodes {
			nodeIDs = append(nodeIDs, node.ID)
		}

		edges, err = s.repo.ListEdgesByDocumentID(ctx, documentID, nodeIDs)
		if err != nil {
			return nil, err
		}
	}

	return &DocumentSubgraph{
		Nodes: nodes,
		Edges: edges,
		Stats: DocumentSubgraphStats{
			NodeCount: len(nodes),
			EdgeCount: len(edges),
			Truncated: totalNodes > int64(len(nodes)),
		},
	}, nil
}

func (s *Service) GetFeatureSubgraph(ctx context.Context, documentID, featureID string) (*DocumentSubgraph, error) {
	graph, err := s.loadDocumentGraph(ctx, documentID)
	if err != nil {
		return nil, err
	}

	featureNode, err := findFeatureNode(graph.nodes, featureID)
	if err != nil {
		return nil, err
	}

	selectedNodeIDs := collectReachableNodeIDs(featureNode.ID, graph.edges)
	selectedNodes := make([]repository.RequirementNode, 0, len(selectedNodeIDs))
	for _, node := range graph.nodes {
		if selectedNodeIDs[node.ID] {
			selectedNodes = append(selectedNodes, node)
		}
	}

	selectedEdges := filterEdgesByNodeIDs(graph.edges, selectedNodeIDs)
	return &DocumentSubgraph{
		Nodes: selectedNodes,
		Edges: selectedEdges,
		Stats: DocumentSubgraphStats{
			NodeCount: len(selectedNodes),
			EdgeCount: len(selectedEdges),
			Truncated: false,
		},
	}, nil
}

func (s *Service) SearchFeatures(ctx context.Context, query, documentID string, limit int) ([]FeatureSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	nodes, err := s.repo.SearchFeatureNodes(ctx, query, documentID, limit)
	if err != nil {
		return nil, err
	}

	results := make([]FeatureSearchResult, 0, len(nodes))
	for _, node := range nodes {
		featureID := strings.TrimSpace(node.ReferenceID)
		if featureID == "" {
			featureID = node.ID
		}
		results = append(results, FeatureSearchResult{
			FeatureID:   featureID,
			FeatureName: node.Summary,
			DocumentID:  node.DocumentID,
			Score:       scoreFeatureMatch(node, query),
			Summary:     node.Description,
		})
	}
	return results, nil
}

func (s *Service) GetFeatureDetail(ctx context.Context, documentID, featureID string) (*FeatureDetail, error) {
	graph, err := s.loadDocumentGraph(ctx, documentID)
	if err != nil {
		return nil, err
	}

	featureNode, err := findFeatureNode(graph.nodes, featureID)
	if err != nil {
		return nil, err
	}

	selectedNodeIDs := collectReachableNodeIDs(featureNode.ID, graph.edges)
	detail := &FeatureDetail{
		FeatureID:   firstNonEmpty(featureNode.ReferenceID, featureID, featureNode.ID),
		FeatureName: featureNode.Summary,
		DocumentID:  featureNode.DocumentID,
	}

	for _, node := range graph.nodes {
		if !selectedNodeIDs[node.ID] || node.ID == featureNode.ID {
			continue
		}

		switch node.Type {
		case "FEATURE_DETAIL":
			detail.FeatureDetails = append(detail.FeatureDetails, node)
		case "BUSINESS_RULE":
			detail.BusinessRules = append(detail.BusinessRules, node)
		case "UI_SCREEN":
			detail.UIScreens = append(detail.UIScreens, node)
		case "USER_FLOW", "UI_FLOW", "FLOW_STEP", "PROCESS", "PROCESS_STEP":
			detail.UserFlows = append(detail.UserFlows, node)
		case "ACCEPTANCE_CRITERIA":
			detail.AcceptanceCriteria = append(detail.AcceptanceCriteria, node)
		case "PERMISSION":
			detail.Permissions = append(detail.Permissions, node)
		case "NON_FUNCTIONAL":
			detail.NonFunctional = append(detail.NonFunctional, node)
		default:
			detail.AdditionalNodes = append(detail.AdditionalNodes, node)
		}
	}

	return detail, nil
}

func (s *Service) UpsertNodes(ctx context.Context, documentID string, nodes []UpsertNodePayload, edges []UpsertEdgePayload) (int, int, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return 0, 0, fmt.Errorf("document_id is required")
	}

	repoNodes, repoEdges, err := normalizeGraphPatch(documentID, nodes, edges)
	if err != nil {
		return 0, 0, err
	}
	if err := s.repo.UpsertDocumentPatch(ctx, documentID, repoNodes, repoEdges); err != nil {
		return 0, 0, err
	}
	return len(repoNodes), len(repoEdges), nil
}

func (s *Service) SaveGraph(ctx context.Context, documentID string, graph GraphSnapshot) error {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return fmt.Errorf("document_id is required")
	}

	repoNodes, repoEdges, err := normalizeGraphPatch(documentID, graph.Nodes, graph.Edges)
	if err != nil {
		return err
	}
	return s.repo.SaveGraphSnapshot(ctx, documentID, repoNodes, repoEdges)
}

func (s *Service) SaveDocument(ctx context.Context, documentID, docKind, content string) (*SaveDocumentResult, error) {
	documentID = strings.TrimSpace(documentID)
	docKind = strings.TrimSpace(docKind)
	content = strings.TrimSpace(content)
	if documentID == "" {
		return nil, fmt.Errorf("document_id is required")
	}
	if _, ok := allowedDocumentKinds[docKind]; !ok {
		return nil, fmt.Errorf("unsupported doc_kind %q", docKind)
	}
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	if err := s.repo.SaveDocumentArtifact(ctx, repository.DocumentArtifactModel{
		DocumentID:  documentID,
		DocKind:     docKind,
		Content:     content,
		ContentType: "text/markdown",
	}); err != nil {
		return nil, err
	}
	return &SaveDocumentResult{
		DocumentID:  documentID,
		DocKind:     docKind,
		Saved:       true,
		ContentType: "text/markdown",
	}, nil
}

type documentGraph struct {
	nodes []repository.RequirementNode
	edges []repository.DependencyEdge
}

func (s *Service) loadDocumentGraph(ctx context.Context, documentID string) (*documentGraph, error) {
	documentID = strings.TrimSpace(documentID)
	nodes, err := s.repo.ListNodesByDocumentID(ctx, documentID, nil, 0)
	if err != nil {
		return nil, err
	}
	edges, err := s.repo.ListEdgesByDocumentID(ctx, documentID, nil)
	if err != nil {
		return nil, err
	}
	return &documentGraph{nodes: nodes, edges: edges}, nil
}

func findFeatureNode(nodes []repository.RequirementNode, featureID string) (*repository.RequirementNode, error) {
	needle := strings.TrimSpace(featureID)
	for i := range nodes {
		node := &nodes[i]
		if !isFeatureType(node.Type) {
			continue
		}
		if matchesFeatureIdentifier(*node, needle) {
			return node, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrFeatureNotFound, featureID)
}

func collectReachableNodeIDs(rootID string, edges []repository.DependencyEdge) map[string]bool {
	selected := map[string]bool{rootID: true}
	queue := []string{rootID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range edges {
			if edge.SourceID != current || !strings.EqualFold(edge.Type, "CONTAINS") {
				continue
			}
			if selected[edge.TargetID] {
				continue
			}
			selected[edge.TargetID] = true
			queue = append(queue, edge.TargetID)
		}
	}
	return selected
}

func normalizeGraphPatch(documentID string, nodes []UpsertNodePayload, edges []UpsertEdgePayload) ([]repository.RequirementNode, []repository.DependencyEdge, error) {
	if len(nodes) == 0 && len(edges) == 0 {
		return nil, nil, fmt.Errorf("at least one node or edge is required")
	}

	repoNodes := make([]repository.RequirementNode, 0, len(nodes))
	seenNodes := make(map[string]struct{}, len(nodes))
	for i, node := range nodes {
		id := strings.TrimSpace(node.ID)
		if id == "" {
			return nil, nil, fmt.Errorf("nodes[%d].id is required", i)
		}
		if _, ok := seenNodes[id]; ok {
			continue
		}
		nodeType := strings.TrimSpace(node.Type)
		if nodeType == "" {
			return nil, nil, fmt.Errorf("nodes[%d].type is required", i)
		}
		seenNodes[id] = struct{}{}
		repoNodes = append(repoNodes, repository.RequirementNode{
			ID:          id,
			DocumentID:  documentID,
			ReferenceID: strings.TrimSpace(node.ReferenceID),
			Type:        nodeType,
			Summary:     node.Summary,
			Description: node.Description,
			SourceID:    strings.TrimSpace(node.SourceID),
			Metadata:    cloneMap(node.Properties),
		})
	}

	repoEdges := make([]repository.DependencyEdge, 0, len(edges))
	seenEdges := make(map[string]struct{}, len(edges))
	for i, edge := range edges {
		source := strings.TrimSpace(edge.Source)
		target := strings.TrimSpace(edge.Target)
		edgeType := strings.TrimSpace(edge.Type)
		if source == "" {
			return nil, nil, fmt.Errorf("edges[%d].source is required", i)
		}
		if target == "" {
			return nil, nil, fmt.Errorf("edges[%d].target is required", i)
		}
		if edgeType == "" {
			return nil, nil, fmt.Errorf("edges[%d].type is required", i)
		}
		id := strings.TrimSpace(edge.ID)
		if id == "" {
			id = fmt.Sprintf("%s:%s:%s", source, edgeType, target)
		}
		if _, ok := seenEdges[id]; ok {
			continue
		}
		seenEdges[id] = struct{}{}
		reason := edge.Reason
		if reason == "" && len(edge.Properties) > 0 {
			if value, ok := edge.Properties["reason"].(string); ok {
				reason = value
			}
		}
		repoEdges = append(repoEdges, repository.DependencyEdge{
			ID:         id,
			DocumentID: documentID,
			SourceID:   source,
			TargetID:   target,
			Type:       edgeType,
			Reason:     reason,
		})
	}

	return repoNodes, repoEdges, nil
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func filterEdgesByNodeIDs(edges []repository.DependencyEdge, selected map[string]bool) []repository.DependencyEdge {
	out := make([]repository.DependencyEdge, 0, len(edges))
	for _, edge := range edges {
		if selected[edge.SourceID] && selected[edge.TargetID] {
			out = append(out, edge)
		}
	}
	return out
}

func isFeatureType(nodeType string) bool {
	return strings.EqualFold(nodeType, "FEATURE") || strings.EqualFold(nodeType, "FUNCTIONAL")
}

func matchesFeatureIdentifier(node repository.RequirementNode, needle string) bool {
	if strings.EqualFold(strings.TrimSpace(node.ID), needle) || strings.EqualFold(strings.TrimSpace(node.ReferenceID), needle) {
		return true
	}
	for _, key := range []string{"feature_id", "preferred_feature_id", "feature_parent"} {
		if strings.EqualFold(metadataString(node.Metadata, key), needle) {
			return true
		}
	}
	return false
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func scoreFeatureMatch(node repository.RequirementNode, query string) int {
	query = strings.ToLower(strings.TrimSpace(query))
	summary := strings.ToLower(node.Summary)
	description := strings.ToLower(node.Description)
	score := 0
	if strings.Contains(summary, query) {
		score += 2
	}
	if strings.Contains(description, query) {
		score++
	}
	return score
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
