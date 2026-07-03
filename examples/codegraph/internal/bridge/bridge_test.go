package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReconcileNodesAndRelationshipsUseDeterministicRefs(t *testing.T) {
	client := newFakeClient()
	cfg := Config{
		KGDomainID:   "code-graph",
		KGVisibility: "private",
	}
	state := State{
		Nodes: map[string]StateNode{
			"proj:old-node": {ID: "n-old", NodeType: "Function"},
		},
		Relationships: map[string]StateRelationship{},
	}
	report := SyncReport{}
	graph := Graph{
		Nodes: []NodeSpec{
			{
				ExternalRef: "proj:node-1",
				NodeType:    "Function",
				Visibility:  "private",
				Properties: map[string]any{
					"name":       "Node One",
					"kind":       "function",
					"file":       "cmd/main.go",
					"line":       10,
					"language":   "go",
					"project_id": "proj",
					"commit_sha": "abc12345",
				},
			},
			{
				ExternalRef: "proj:node-2",
				NodeType:    "Function",
				Visibility:  "private",
				Properties: map[string]any{
					"name":       "Node Two",
					"kind":       "function",
					"file":       "cmd/main.go",
					"line":       20,
					"language":   "go",
					"project_id": "proj",
					"commit_sha": "abc12345",
				},
			},
		},
		Edges: []EdgeSpec{
			{
				Key:             "proj:proj:node-1:CALLS:proj:node-2",
				RelType:         "CALLS",
				FromExternalRef: "proj:node-1",
				ToExternalRef:   "proj:node-2",
				Properties:      map[string]any{"codegraph_edge_kind": "calls"},
			},
		},
	}

	if err := reconcileNodes(context.Background(), client, cfg, graph.Nodes, &state, &report, "session-1"); err != nil {
		t.Fatalf("reconcileNodes() error = %v", err)
	}
	if report.CreatedNodes != 2 || report.UpdatedNodes != 0 || report.DeletedNodes != 1 {
		t.Fatalf("report = %#v", report)
	}
	if _, ok := state.Nodes["proj:node-1"]; !ok {
		t.Fatalf("state.Nodes missing proj:node-1: %#v", state.Nodes)
	}
	if _, ok := state.Nodes["proj:node-2"]; !ok {
		t.Fatalf("state.Nodes missing proj:node-2: %#v", state.Nodes)
	}

	// Simulate a second run with the same graph.
	report2 := SyncReport{}
	if err := reconcileNodes(context.Background(), client, cfg, graph.Nodes, &state, &report2, "session-1"); err != nil {
		t.Fatalf("second reconcileNodes() error = %v", err)
	}
	if report2.CreatedNodes != 0 || report2.UpdatedNodes != 2 {
		t.Fatalf("second report = %#v", report2)
	}

	if err := reconcileRelationships(context.Background(), client, cfg, graph.Edges, &state, &report, "session-1"); err != nil {
		t.Fatalf("reconcileRelationships() error = %v", err)
	}
	if report.CreatedRelationships != 1 || report.SkippedRelationships != 0 {
		t.Fatalf("relationship report = %#v", report)
	}
	if len(state.Relationships) != 1 {
		t.Fatalf("state.Relationships = %#v", state.Relationships)
	}
}

func TestMCPTemplateAndSearchToolsValidateAndParse(t *testing.T) {
	client := newFakeClient()
	client.templateResult = TemplateExecutionResponse{
		Results: []map[string]any{{"node_id": "node-1", "kind": "Function"}},
	}
	client.semanticResult = SemanticSearchResponse{
		Results: []SearchResult{{NodeID: "node-1", NodeType: "Function", Score: 0.91}},
	}
	client.fullTextResult = FullTextSearchResponse{
		Results: []SearchResult{{NodeID: "node-2", NodeType: "Function", Score: 0.71}},
	}
	server := &mcpServer{
		cfg: Config{
			KGDomainID:       "code-graph",
			TemplateDomainID: "code-graph",
			DefaultTopK:      5,
		},
		client: client,
	}

	validation := server.handle(context.Background(), rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "kg_code_template_query",
			"arguments": map[string]any{"template_name": ""},
		},
	})
	if validation.Error == nil || validation.Error.Code != -32602 {
		t.Fatalf("validation response = %#v", validation)
	}

	resp := server.handle(context.Background(), rpcRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_code_template_query",
			"arguments": map[string]any{
				"template_name": "code_callers",
				"params":        map[string]any{"node_id": "node-1"},
			},
		},
	})
	if resp.Error != nil {
		t.Fatalf("template response error = %#v", resp.Error)
	}
	payload, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("template response result = %#v", resp.Result)
	}
	parsed := payload["structuredContent"].(map[string]any)
	raw, err := json.Marshal(parsed["results"])
	if err != nil {
		t.Fatalf("json.Marshal() parsed results error = %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("json.Unmarshal() parsed results error = %v", err)
	}
	if len(rows) != 1 || rows[0]["node_id"] != "node-1" {
		t.Fatalf("rows = %#v", rows)
	}

	semantic := server.handle(context.Background(), rpcRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_semantic_search",
			"arguments": map[string]any{
				"query": "find function",
			},
		},
	})
	if semantic.Error != nil {
		t.Fatalf("semantic response error = %#v", semantic.Error)
	}
	if len(client.semanticCalls) != 1 {
		t.Fatalf("semanticCalls = %#v", client.semanticCalls)
	}
}

func TestExtractGraphProducesSupportedRefs(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	cfg := Config{
		ProjectPath:     projectRoot,
		ProjectID:       "kg-service",
		CodeGraphDBPath: filepath.Join(projectRoot, ".codegraph", "codegraph.db"),
		KGVisibility:    "private",
	}
	graph, err := ExtractGraph(context.Background(), cfg, "abc12345")
	if err != nil {
		t.Fatalf("ExtractGraph() error = %v", err)
	}
	if len(graph.Nodes) == 0 {
		t.Fatal("ExtractGraph() returned no nodes")
	}
	for _, node := range graph.Nodes[:minInt(3, len(graph.Nodes))] {
		if node.ExternalRef == "" || node.Properties["project_id"] != "kg-service" {
			t.Fatalf("node = %#v", node)
		}
	}
}

func TestReconcileNodesRecreatesNodeWhenManifestIDIsGone(t *testing.T) {
	client := newFakeClient()
	client.updateErrByNodeID = map[string]error{
		"n-gone": errors.New("kg-service 404 NOT_FOUND: Resource not found"),
	}
	cfg := Config{
		KGDomainID:   "code-graph",
		KGVisibility: "private",
	}
	state := State{
		Nodes: map[string]StateNode{
			"proj:file:cmd/cmd.go": {ID: "n-gone", NodeType: "File"},
		},
		Relationships: map[string]StateRelationship{},
	}
	report := SyncReport{}
	graph := []NodeSpec{
		{
			ExternalRef: "proj:file:cmd/cmd.go",
			NodeType:    "File",
			Visibility:  "private",
			Properties: map[string]any{
				"name":       "cmd/cmd.go",
				"kind":       "file",
				"file":       "cmd/cmd.go",
				"line":       1,
				"language":   "go",
				"project_id": "proj",
				"commit_sha": "abc12345",
			},
		},
	}

	if err := reconcileNodes(context.Background(), client, cfg, graph, &state, &report, "session-1"); err != nil {
		t.Fatalf("reconcileNodes() error = %v", err)
	}
	got := state.Nodes["proj:file:cmd/cmd.go"]
	if got.ID == "" || got.ID == "n-gone" {
		t.Fatalf("state.Nodes recreated id = %#v, want fresh node id", got)
	}
	if report.CreatedNodes != 1 || report.UpdatedNodes != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestReconcileNodesIgnoresNotFoundWhenDeletingStaleManifestNode(t *testing.T) {
	client := newFakeClient()
	client.deleteErrByNodeID = map[string]error{
		"n-gone": errors.New("kg-service 404 NOT_FOUND: Resource not found"),
	}
	cfg := Config{
		KGDomainID:   "code-graph",
		KGVisibility: "private",
	}
	state := State{
		Nodes: map[string]StateNode{
			"proj:obsolete": {ID: "n-gone", NodeType: "File"},
		},
		Relationships: map[string]StateRelationship{},
	}
	report := SyncReport{}

	if err := reconcileNodes(context.Background(), client, cfg, nil, &state, &report, "session-1"); err != nil {
		t.Fatalf("reconcileNodes() error = %v", err)
	}
	if len(state.Nodes) != 0 {
		t.Fatalf("state.Nodes = %#v, want empty", state.Nodes)
	}
	if report.DeletedNodes != 1 {
		t.Fatalf("report = %#v", report)
	}
}

type fakeClient struct {
	nextNodeID       int
	nextRelationship int
	nodes            map[string]NodeCreateResponse
	deletedPrefixes  []string
	updateErrByNodeID map[string]error
	deleteErrByNodeID map[string]error
	semanticResult   SemanticSearchResponse
	fullTextResult   FullTextSearchResponse
	templateResult   TemplateExecutionResponse
	semanticCalls    []SemanticSearchRequest
	fullTextCalls    []FullTextSearchRequest
	templateCalls    []struct {
		domainID string
		name     string
		params   map[string]any
	}
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		nodes: map[string]NodeCreateResponse{},
	}
}

func (f *fakeClient) Ping(context.Context) error { return nil }

func (f *fakeClient) CreateNode(_ context.Context, req NodeRequest) (NodeCreateResponse, error) {
	f.nextNodeID++
	resp := NodeCreateResponse{NodeID: "n-" + req.ExternalRef, Status: "created"}
	f.nodes[req.ExternalRef] = resp
	return resp, nil
}

func (f *fakeClient) CreateNodesBulk(_ context.Context, req NodeBulkCreateRequest) (NodeBulkCreateResponse, error) {
	resp := NodeBulkCreateResponse{Succeeded: make([]NodeCreateResponse, 0, len(req.Nodes))}
	for _, node := range req.Nodes {
		created, err := f.CreateNode(context.Background(), node)
		if err != nil {
			resp.Failed = append(resp.Failed, BulkItemError{Error: err.Error()})
			continue
		}
		resp.Succeeded = append(resp.Succeeded, created)
	}
	return resp, nil
}

func (f *fakeClient) UpdateNode(_ context.Context, nodeID string, req NodeUpdateRequest) (NodeUpdateResponse, error) {
	if err := f.updateErrByNodeID[nodeID]; err != nil {
		return NodeUpdateResponse{}, err
	}
	_ = req
	return NodeUpdateResponse{NodeID: nodeID, Status: "updated"}, nil
}

func (f *fakeClient) DeleteNode(_ context.Context, nodeID string) error {
	_ = nodeID
	return nil
}

func (f *fakeClient) DeleteNodeWithVersion(_ context.Context, nodeID, graphVersionID string) error {
	if err := f.deleteErrByNodeID[nodeID]; err != nil {
		return err
	}
	_ = nodeID
	_ = graphVersionID
	return nil
}

func (f *fakeClient) DeleteNodesByExternalRefPrefix(_ context.Context, prefix string) error {
	f.deletedPrefixes = append(f.deletedPrefixes, prefix)
	return nil
}

func (f *fakeClient) DeleteNodesByExternalRefPrefixWithVersion(_ context.Context, prefix, graphVersionID string) error {
	f.deletedPrefixes = append(f.deletedPrefixes, prefix+":"+graphVersionID)
	return nil
}

func (f *fakeClient) CreateRelationship(_ context.Context, req RelationshipRequest) (RelationshipCreateResponse, error) {
	f.nextRelationship++
	return RelationshipCreateResponse{RelationshipID: "r-1", Status: "created"}, nil
}

func (f *fakeClient) CreateRelationshipsBulk(_ context.Context, req RelationshipBulkCreateRequest) (RelationshipBulkCreateResponse, error) {
	resp := RelationshipBulkCreateResponse{Succeeded: make([]RelationshipCreateResponse, 0, len(req.Relationships))}
	for range req.Relationships {
		created, err := f.CreateRelationship(context.Background(), RelationshipRequest{})
		if err != nil {
			resp.Failed = append(resp.Failed, BulkItemError{Error: err.Error()})
			continue
		}
		resp.Succeeded = append(resp.Succeeded, created)
	}
	return resp, nil
}

func (f *fakeClient) DeleteRelationshipsBulk(_ context.Context, req RelationshipBulkDeleteRequest) (RelationshipBulkDeleteResponse, error) {
	return RelationshipBulkDeleteResponse{RelationshipIDs: append([]string(nil), req.RelationshipIDs...), Count: len(req.RelationshipIDs)}, nil
}

func (f *fakeClient) OpenSyncSession(_ context.Context, req OpenSyncSessionRequest) (SyncSessionResponse, error) {
	return SyncSessionResponse{
		SessionID:          "session-1",
		GraphVersionID:     "session-1",
		GraphIdentifierID:  "graph-1",
		GraphVersionNumber: 1,
	}, nil
}

func (f *fakeClient) CommitSyncSession(context.Context, string) error {
	return nil
}

func (f *fakeClient) AbandonSyncSession(context.Context, string) error {
	return nil
}

func (f *fakeClient) SemanticSearch(_ context.Context, req SemanticSearchRequest) (SemanticSearchResponse, error) {
	f.semanticCalls = append(f.semanticCalls, req)
	return f.semanticResult, nil
}

func (f *fakeClient) FullTextSearch(_ context.Context, req FullTextSearchRequest) (FullTextSearchResponse, error) {
	f.fullTextCalls = append(f.fullTextCalls, req)
	return f.fullTextResult, nil
}

func (f *fakeClient) TemplateQuery(_ context.Context, domainID, templateName string, params map[string]any) (TemplateExecutionResponse, error) {
	f.templateCalls = append(f.templateCalls, struct {
		domainID string
		name     string
		params   map[string]any
	}{domainID: domainID, name: templateName, params: params})
	return f.templateResult, nil
}
