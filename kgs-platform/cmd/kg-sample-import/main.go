package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	defaultNodesCSV = "documents/ai-orchestrator/kg-sample-data/query_1-2026-04-07_62330-nodes.csv"
	defaultEdgesCSV = "documents/ai-orchestrator/kg-sample-data/query_1-2026-04-07_62346-edges.csv"
	defaultAppID    = "ba-agent-system"
	defaultTenantID = "default"
)

var nonIdentifierPattern = regexp.MustCompile(`[^A-Za-z0-9_]`)

type config struct {
	Neo4jURI     string
	Neo4jUser    string
	Neo4jPass    string
	Neo4jDB      string
	NodesCSV     string
	EdgesCSV     string
	AppID        string
	TenantID     string
	DocumentID   string
	BatchSize    int
	ClearScope   bool
	EnsureSchema bool
}

type nodeRow struct {
	ID          string
	DocumentID  string
	ReferenceID string
	Type        string
	Summary     string
	Description string
	SourceID    string
	MetadataRaw string
	Label       string
}

type edgeRow struct {
	ID         string
	DocumentID string
	SourceID   string
	TargetID   string
	Type       string
	Reason     string
	RelType    string
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("kg sample import failed: %v", err)
	}
}

func run() error {
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		return err
	}

	nodesCSV, err := resolvePath(cfg.NodesCSV)
	if err != nil {
		return err
	}
	edgesCSV, err := resolvePath(cfg.EdgesCSV)
	if err != nil {
		return err
	}

	nodes, nodeIDs, docIDs, badMetadataCount, err := readNodesCSV(nodesCSV, cfg.DocumentID)
	if err != nil {
		return err
	}
	edges, skippedMissingNode, err := readEdgesCSV(edgesCSV, cfg.DocumentID, nodeIDs)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return fmt.Errorf("nodes csv is empty: %s", nodesCSV)
	}

	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4jURI,
		neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPass, ""),
	)
	if err != nil {
		return fmt.Errorf("create neo4j driver: %w", err)
	}
	defer driver.Close(ctx)

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("neo4j connectivity check failed: %w", err)
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: cfg.Neo4jDB,
	})
	defer session.Close(ctx)

	if cfg.EnsureSchema {
		if err := ensureConstraints(ctx, session); err != nil {
			return err
		}
	}

	if cfg.ClearScope {
		if err := clearImportScope(ctx, session, cfg.AppID, cfg.TenantID, docIDs); err != nil {
			return err
		}
	}

	started := time.Now()
	importedNodes, err := importNodes(ctx, session, cfg.AppID, cfg.TenantID, nodes, cfg.BatchSize)
	if err != nil {
		return err
	}
	importedEdges, err := importEdges(ctx, session, cfg.AppID, cfg.TenantID, edges, cfg.BatchSize)
	if err != nil {
		return err
	}

	log.Printf(
		"import completed: nodes=%d edges=%d metadata_parse_errors=%d skipped_edges_missing_node=%d duration=%s",
		importedNodes,
		importedEdges,
		badMetadataCount,
		skippedMissingNode,
		time.Since(started),
	)
	return nil
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.Neo4jURI, "neo4j-uri", envOrDefault("NEO4J_URI", "bolt://localhost:7687"), "Neo4j URI")
	flag.StringVar(&cfg.Neo4jUser, "neo4j-user", envOrDefault("NEO4J_USER", "neo4j"), "Neo4j username")
	flag.StringVar(&cfg.Neo4jPass, "neo4j-pass", envOrDefault("NEO4J_PASSWORD", ""), "Neo4j password")
	flag.StringVar(&cfg.Neo4jDB, "neo4j-db", envOrDefault("NEO4J_DB", ""), "Neo4j database name (optional)")
	flag.StringVar(&cfg.NodesCSV, "nodes-csv", defaultNodesCSV, "Path to KG sample nodes CSV")
	flag.StringVar(&cfg.EdgesCSV, "edges-csv", defaultEdgesCSV, "Path to KG sample edges CSV")
	flag.StringVar(&cfg.AppID, "app-id", defaultAppID, "Namespace app_id in Neo4j")
	flag.StringVar(&cfg.TenantID, "tenant-id", defaultTenantID, "Namespace tenant_id in Neo4j")
	flag.StringVar(&cfg.DocumentID, "document-id", "", "Override document_id for all rows (optional)")
	flag.IntVar(&cfg.BatchSize, "batch-size", 500, "Batch size per write query")
	flag.BoolVar(&cfg.ClearScope, "clear-scope", false, "Delete existing namespace/doc scope before import")
	flag.BoolVar(&cfg.EnsureSchema, "ensure-schema", false, "Create Neo4j constraints/indexes required by namespaced writes")
	flag.Parse()
	return cfg
}

func validateConfig(cfg config) error {
	if strings.TrimSpace(cfg.Neo4jURI) == "" {
		return fmt.Errorf("neo4j-uri is required")
	}
	if strings.TrimSpace(cfg.Neo4jUser) == "" {
		return fmt.Errorf("neo4j-user is required")
	}
	if strings.TrimSpace(cfg.Neo4jPass) == "" {
		return fmt.Errorf("neo4j-pass is required (or set NEO4J_PASSWORD)")
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return fmt.Errorf("app-id is required")
	}
	if strings.TrimSpace(cfg.TenantID) == "" {
		return fmt.Errorf("tenant-id is required")
	}
	if cfg.BatchSize <= 0 {
		return fmt.Errorf("batch-size must be > 0")
	}
	return nil
}

func resolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if exists(path) {
		return path, nil
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("path not found: %s", path)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(cur, path)
		if exists(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return "", fmt.Errorf("path not found from cwd hierarchy: %s", path)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readNodesCSV(path string, docOverride string) ([]nodeRow, map[string]struct{}, map[string]struct{}, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("open nodes csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("read nodes header: %w", err)
	}
	idx := indexColumns(header)
	required := []string{"id", "document_id", "reference_id", "type", "summary", "description", "source_id", "metadata"}
	for _, col := range required {
		if _, ok := idx[col]; !ok {
			return nil, nil, nil, 0, fmt.Errorf("nodes csv missing required column %q", col)
		}
	}

	nodes := make([]nodeRow, 0, 1024)
	nodeIDs := make(map[string]struct{}, 1024)
	docIDs := make(map[string]struct{}, 8)
	badMetadata := 0

	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, nil, 0, fmt.Errorf("read nodes row: %w", err)
		}

		id := strings.TrimSpace(getColumn(row, idx, "id"))
		if id == "" {
			continue
		}
		docID := strings.TrimSpace(getColumn(row, idx, "document_id"))
		if strings.TrimSpace(docOverride) != "" {
			docID = strings.TrimSpace(docOverride)
		}
		if docID != "" {
			docIDs[docID] = struct{}{}
		}

		nodeType := strings.TrimSpace(getColumn(row, idx, "type"))
		node := nodeRow{
			ID:          id,
			DocumentID:  docID,
			ReferenceID: strings.TrimSpace(getColumn(row, idx, "reference_id")),
			Type:        nodeType,
			Summary:     strings.TrimSpace(getColumn(row, idx, "summary")),
			Description: strings.TrimSpace(getColumn(row, idx, "description")),
			SourceID:    strings.TrimSpace(getColumn(row, idx, "source_id")),
			MetadataRaw: normalizeMetadataJSON(getColumn(row, idx, "metadata"), &badMetadata),
			Label:       sanitizeIdentifier(nodeType, "TYPE"),
		}
		nodes = append(nodes, node)
		nodeIDs[id] = struct{}{}
	}

	return nodes, nodeIDs, docIDs, badMetadata, nil
}

func readEdgesCSV(path string, docOverride string, knownNodeIDs map[string]struct{}) ([]edgeRow, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open edges csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, 0, fmt.Errorf("read edges header: %w", err)
	}
	idx := indexColumns(header)
	required := []string{"id", "document_id", "source_id", "target_id", "type", "reason"}
	for _, col := range required {
		if _, ok := idx[col]; !ok {
			return nil, 0, fmt.Errorf("edges csv missing required column %q", col)
		}
	}

	edges := make([]edgeRow, 0, 1024)
	skippedMissingNode := 0

	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read edges row: %w", err)
		}

		sourceID := strings.TrimSpace(getColumn(row, idx, "source_id"))
		targetID := strings.TrimSpace(getColumn(row, idx, "target_id"))
		if sourceID == "" || targetID == "" {
			continue
		}
		if _, ok := knownNodeIDs[sourceID]; !ok {
			skippedMissingNode++
			continue
		}
		if _, ok := knownNodeIDs[targetID]; !ok {
			skippedMissingNode++
			continue
		}

		edgeType := strings.TrimSpace(getColumn(row, idx, "type"))
		edgeID := strings.TrimSpace(getColumn(row, idx, "id"))
		if edgeID == "" {
			edgeID = fmt.Sprintf("edge_%s_%s_%s", sourceID, sanitizeIdentifier(edgeType, "REL"), targetID)
		}
		docID := strings.TrimSpace(getColumn(row, idx, "document_id"))
		if strings.TrimSpace(docOverride) != "" {
			docID = strings.TrimSpace(docOverride)
		}

		edges = append(edges, edgeRow{
			ID:         edgeID,
			DocumentID: docID,
			SourceID:   sourceID,
			TargetID:   targetID,
			Type:       edgeType,
			Reason:     strings.TrimSpace(getColumn(row, idx, "reason")),
			RelType:    sanitizeIdentifier(edgeType, "REL"),
		})
	}
	return edges, skippedMissingNode, nil
}

func importNodes(
	ctx context.Context,
	session neo4j.SessionWithContext,
	appID string,
	tenantID string,
	nodes []nodeRow,
	batchSize int,
) (int, error) {
	grouped := make(map[string][]nodeRow)
	for _, n := range nodes {
		grouped[n.Label] = append(grouped[n.Label], n)
	}
	labels := make([]string, 0, len(grouped))
	for label := range grouped {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	total := 0
	for _, label := range labels {
		rows := grouped[label]
		query := fmt.Sprintf(`
UNWIND $rows AS row
MERGE (n:Entity:%s {app_id: $app_id, tenant_id: $tenant_id, id: row.id})
ON CREATE SET n.created_at = datetime()
SET n += row.props,
    n._unique_key = $app_id + "|" + $tenant_id + "|" + row.id,
    n.updated_at = datetime()
RETURN count(n) AS affected
`, label)

		for start := 0; start < len(rows); start += batchSize {
			end := start + batchSize
			if end > len(rows) {
				end = len(rows)
			}
			chunk := rows[start:end]
			payload := make([]map[string]any, 0, len(chunk))
			for _, n := range chunk {
				props := map[string]any{
					"id":           n.ID,
					"document_id":  n.DocumentID,
					"reference_id": n.ReferenceID,
					"type":         n.Type,
					"summary":      n.Summary,
					"description":  n.Description,
					"source_id":    n.SourceID,
				}
				if n.MetadataRaw != "" {
					props["metadata"] = n.MetadataRaw
				}
				payload = append(payload, map[string]any{
					"id":    n.ID,
					"props": props,
				})
			}

			count, err := executeCountWrite(ctx, session, query, map[string]any{
				"app_id":    appID,
				"tenant_id": tenantID,
				"rows":      payload,
			})
			if err != nil {
				return 0, fmt.Errorf("import nodes label=%s: %w", label, err)
			}
			total += count
		}
	}
	return total, nil
}

func importEdges(
	ctx context.Context,
	session neo4j.SessionWithContext,
	appID string,
	tenantID string,
	edges []edgeRow,
	batchSize int,
) (int, error) {
	grouped := make(map[string][]edgeRow)
	for _, e := range edges {
		grouped[e.RelType] = append(grouped[e.RelType], e)
	}
	relTypes := make([]string, 0, len(grouped))
	for relType := range grouped {
		relTypes = append(relTypes, relType)
	}
	sort.Strings(relTypes)

	total := 0
	for _, relType := range relTypes {
		rows := grouped[relType]
		query := fmt.Sprintf(`
UNWIND $rows AS row
MATCH (s:Entity {app_id: $app_id, tenant_id: $tenant_id, id: row.source_id})
MATCH (t:Entity {app_id: $app_id, tenant_id: $tenant_id, id: row.target_id})
MERGE (s)-[r:%s {app_id: $app_id, tenant_id: $tenant_id, id: row.id}]->(t)
ON CREATE SET r.created_at = datetime()
SET r += row.props,
    r.updated_at = datetime()
RETURN count(r) AS affected
`, relType)

		for start := 0; start < len(rows); start += batchSize {
			end := start + batchSize
			if end > len(rows) {
				end = len(rows)
			}
			chunk := rows[start:end]
			payload := make([]map[string]any, 0, len(chunk))
			for _, e := range chunk {
				props := map[string]any{
					"id":          e.ID,
					"document_id": e.DocumentID,
					"source_id":   e.SourceID,
					"target_id":   e.TargetID,
					"type":        e.Type,
					"reason":      e.Reason,
				}
				payload = append(payload, map[string]any{
					"id":        e.ID,
					"source_id": e.SourceID,
					"target_id": e.TargetID,
					"props":     props,
				})
			}

			count, err := executeCountWrite(ctx, session, query, map[string]any{
				"app_id":    appID,
				"tenant_id": tenantID,
				"rows":      payload,
			})
			if err != nil {
				return 0, fmt.Errorf("import edges rel_type=%s: %w", relType, err)
			}
			total += count
		}
	}
	return total, nil
}

func executeCountWrite(
	ctx context.Context,
	session neo4j.SessionWithContext,
	query string,
	params map[string]any,
) (int, error) {
	res, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return 0, err
		}
		if result.Next(ctx) {
			raw, ok := result.Record().Get("affected")
			if !ok {
				return 0, nil
			}
			return toInt(raw), result.Err()
		}
		return 0, result.Err()
	})
	if err != nil {
		return 0, err
	}
	count, _ := res.(int)
	return count, nil
}

func clearImportScope(
	ctx context.Context,
	session neo4j.SessionWithContext,
	appID string,
	tenantID string,
	docIDs map[string]struct{},
) error {
	if len(docIDs) == 0 {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(
				ctx,
				`MATCH (n:Entity {app_id: $app_id, tenant_id: $tenant_id}) DETACH DELETE n`,
				map[string]any{"app_id": appID, "tenant_id": tenantID},
			)
			return nil, err
		})
		if err != nil {
			return fmt.Errorf("clear scope (namespace): %w", err)
		}
		return nil
	}

	docList := make([]string, 0, len(docIDs))
	for docID := range docIDs {
		docList = append(docList, docID)
	}
	sort.Strings(docList)

	for _, docID := range docList {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(
				ctx,
				`MATCH (n:Entity {app_id: $app_id, tenant_id: $tenant_id, document_id: $document_id}) DETACH DELETE n`,
				map[string]any{
					"app_id":      appID,
					"tenant_id":   tenantID,
					"document_id": docID,
				},
			)
			return nil, err
		})
		if err != nil {
			return fmt.Errorf("clear scope (doc_id=%s): %w", docID, err)
		}
	}
	return nil
}

func ensureConstraints(ctx context.Context, session neo4j.SessionWithContext) error {
	queries := []string{
		`CREATE CONSTRAINT kgs_entity_unique_key IF NOT EXISTS
FOR (n:Entity)
REQUIRE n._unique_key IS UNIQUE`,
		`CREATE INDEX kgs_entity_app_tenant IF NOT EXISTS
FOR (n:Entity)
ON (n.app_id, n.tenant_id)`,
		`CREATE INDEX kgs_entity_document_id IF NOT EXISTS
FOR (n:Entity)
ON (n.document_id)`,
	}
	for _, query := range queries {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, query, nil)
			return nil, err
		})
		if err != nil {
			return fmt.Errorf("ensure schema failed: %w", err)
		}
	}
	return nil
}

func indexColumns(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, col := range header {
		idx[strings.TrimSpace(strings.ToLower(col))] = i
	}
	return idx
}

func getColumn(row []string, idx map[string]int, key string) string {
	i, ok := idx[key]
	if !ok || i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

func normalizeMetadataJSON(raw string, badCount *int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		if badCount != nil {
			*badCount++
		}
		return trimmed
	}
	b, err := json.Marshal(parsed)
	if err != nil {
		if badCount != nil {
			*badCount++
		}
		return trimmed
	}
	return string(b)
}

func sanitizeIdentifier(raw string, fallbackPrefix string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return fallbackPrefix + "_UNKNOWN"
	}
	cleaned = strings.ToUpper(cleaned)
	cleaned = nonIdentifierPattern.ReplaceAllString(cleaned, "_")
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return fallbackPrefix + "_UNKNOWN"
	}
	first := cleaned[0]
	if (first >= '0' && first <= '9') || first == '_' {
		cleaned = fallbackPrefix + "_" + cleaned
	}
	return cleaned
}

func envOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case int32:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}
