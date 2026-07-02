package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func ExtractGraph(ctx context.Context, cfg Config, commitSHA string) (Graph, error) {
	nodes, err := readRawNodes(ctx, cfg.CodeGraphDBPath)
	if err != nil {
		return Graph{}, err
	}
	edges, err := readRawEdges(ctx, cfg.CodeGraphDBPath)
	if err != nil {
		return Graph{}, err
	}

	supported := map[string]string{
		"function":  "Function",
		"method":    "Method",
		"struct":    "Struct",
		"interface": "Interface",
		"file":      "File",
		"import":    "Package",
		"route":     "Function",
	}
	edgeKinds := map[string]string{
		"calls":        "CALLS",
		"implements":   "IMPLEMENTS",
		"contains":     "CONTAINS",
		"references":   "REFERENCES",
		"imports":      "IMPORTS",
		"instantiates": "REFERENCES",
		"extends":      "REFERENCES",
	}

	nodeByID := make(map[string]NodeSpec, len(nodes))
	orderedIDs := make([]string, 0, len(nodes))
	for _, raw := range nodes {
		nodeType, ok := supported[strings.ToLower(strings.TrimSpace(raw.Kind))]
		if !ok {
			continue
		}
		spec := NodeSpec{
			ExternalRef: externalRef(cfg.ProjectID, raw.ID),
			NodeType:    nodeType,
			Visibility:  cfg.KGVisibility,
			SourceID:    raw.ID,
			SourceKind:  raw.Kind,
			Properties:  nodeProperties(raw, cfg.ProjectID, commitSHA),
		}
		nodeByID[raw.ID] = spec
		orderedIDs = append(orderedIDs, raw.ID)
	}

	sort.Strings(orderedIDs)
	nodeList := make([]NodeSpec, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		nodeList = append(nodeList, nodeByID[id])
	}

	edgeList := make([]EdgeSpec, 0, len(edges))
	for _, raw := range edges {
		relType, ok := edgeKinds[strings.ToLower(strings.TrimSpace(raw.Kind))]
		if !ok {
			continue
		}
		fromNode, ok := nodeByID[raw.Source]
		if !ok {
			continue
		}
		toNode, ok := nodeByID[raw.Target]
		if !ok {
			continue
		}
		edgeList = append(edgeList, EdgeSpec{
			Key:              edgeKey(cfg.ProjectID, fromNode.ExternalRef, relType, toNode.ExternalRef),
			RelType:          relType,
			FromExternalRef:  fromNode.ExternalRef,
			ToExternalRef:    toNode.ExternalRef,
			SourceKind:       raw.Kind,
			SourceProvenance: raw.Provenance,
			Properties: map[string]any{
				"codegraph_edge_kind": raw.Kind,
				"line":                raw.Line,
				"col":                 raw.Col,
				"provenance":          raw.Provenance,
			},
		})
	}

	sort.Slice(edgeList, func(i, j int) bool { return edgeList[i].Key < edgeList[j].Key })

	return Graph{Nodes: nodeList, Edges: edgeList}, nil
}

func readRawNodes(ctx context.Context, dbPath string) ([]RawNode, error) {
	sql := `
.mode list
select json_object(
  'id', id,
  'kind', kind,
  'name', name,
  'qualified_name', qualified_name,
  'file_path', file_path,
  'language', language,
  'start_line', start_line,
  'end_line', end_line,
  'start_column', start_column,
  'end_column', end_column,
  'docstring', coalesce(docstring, ''),
  'signature', coalesce(signature, ''),
  'visibility', coalesce(visibility, ''),
  'is_exported', is_exported,
  'is_async', is_async,
  'is_static', is_static,
  'is_abstract', is_abstract,
  'decorators', coalesce(decorators, '[]'),
  'type_parameters', coalesce(type_parameters, '[]'),
  'return_type', coalesce(return_type, '')
) as row
from nodes
order by file_path, start_line, id;
`
	rows, err := runSQLite(ctx, dbPath, sql)
	if err != nil {
		return nil, err
	}
	out := make([]RawNode, 0, len(rows))
	for _, row := range rows {
		var item RawNode
		if err := json.Unmarshal([]byte(row), &item); err != nil {
			return nil, fmt.Errorf("decode node row: %w", err)
		}
		out = append(out, item)
	}
	return out, nil
}

func readRawEdges(ctx context.Context, dbPath string) ([]RawEdge, error) {
	sql := `
.mode list
select json_object(
  'source', source,
  'target', target,
  'kind', kind,
  'metadata', coalesce(metadata, '{}'),
  'line', coalesce(line, -1),
  'col', coalesce(col, -1),
  'provenance', coalesce(provenance, '')
) as row
from edges
order by id;
`
	rows, err := runSQLite(ctx, dbPath, sql)
	if err != nil {
		return nil, err
	}
	out := make([]RawEdge, 0, len(rows))
	for _, row := range rows {
		var item RawEdge
		if err := json.Unmarshal([]byte(row), &item); err != nil {
			return nil, fmt.Errorf("decode edge row: %w", err)
		}
		out = append(out, item)
	}
	return out, nil
}

func runSQLite(ctx context.Context, dbPath, sql string) ([]string, error) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return nil, errors.New("sqlite3 is required to read the CodeGraph index")
	}
	cmd := exec.CommandContext(ctx, "sqlite3", "-readonly", dbPath)
	cmd.Stdin = strings.NewReader(sql)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 query failed: %w", err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	rows := make([]string, 0, 128)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			rows = append(rows, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func nodeProperties(node RawNode, projectID, commitSHA string) map[string]any {
	props := map[string]any{
		"name":       node.Name,
		"kind":       node.Kind,
		"file":       node.FilePath,
		"line":       maxInt(node.StartLine, 1),
		"language":   node.Language,
		"project_id": projectID,
		"commit_sha": commitSHA,
	}
	if trimmed := strings.TrimSpace(node.Signature); trimmed != "" {
		props["signature"] = trimmed
	}
	if trimmed := strings.TrimSpace(node.Docstring); trimmed != "" {
		props["docstring"] = trimmed
	}
	if pkg := filepath.Dir(node.FilePath); pkg != "." && pkg != "" {
		props["package"] = filepath.ToSlash(pkg)
	} else {
		props["package"] = projectID
	}
	return props
}

func externalRef(projectID, sourceID string) string {
	return projectID + ":" + sourceID
}

func edgeKey(projectID, fromExternalRef, relType, toExternalRef string) string {
	return projectID + ":" + fromExternalRef + ":" + relType + ":" + toExternalRef
}

func maxInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
