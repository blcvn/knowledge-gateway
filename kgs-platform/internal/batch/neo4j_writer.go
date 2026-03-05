package batch

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jWriter struct {
	driver    neo4j.DriverWithContext
	batchSize int
}

func NewNeo4jWriter(driver neo4j.DriverWithContext) *Neo4jWriter {
	return &Neo4jWriter{
		driver:    driver,
		batchSize: 200,
	}
}

func (w *Neo4jWriter) BulkCreate(ctx context.Context, appID, tenantID string, entities []Entity) (int, error) {
	if len(entities) == 0 {
		return 0, nil
	}
	total := 0
	for i := 0; i < len(entities); i += w.batchSize {
		end := i + w.batchSize
		if end > len(entities) {
			end = len(entities)
		}
		created, err := w.writeChunk(ctx, appID, tenantID, entities[i:end])
		if err != nil {
			return total, err
		}
		total += created
	}
	return total, nil
}

func (w *Neo4jWriter) writeChunk(ctx context.Context, appID, tenantID string, entities []Entity) (int, error) {
	byLabel := make(map[string][]map[string]any)
	for _, entity := range entities {
		label, err := sanitizeCypherIdentifier(entity.Label)
		if err != nil {
			return 0, err
		}
		props := cloneMap(entity.Properties)
		if _, ok := props["id"].(string); !ok {
			props["id"] = uuid.NewString()
		}
		byLabel[label] = append(byLabel[label], props)
	}

	session := w.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	total := 0
	for label, propsList := range byLabel {
		countAny, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := fmt.Sprintf(`
				UNWIND $entities AS e
				CREATE (n:%s {app_id: $app_id, tenant_id: $tenant_id, id: e.id})
				SET n += e
				RETURN count(n) AS created
			`, label)
			res, err := tx.Run(ctx, query, map[string]any{
				"app_id":    appID,
				"tenant_id": tenantID,
				"entities":  propsList,
			})
			if err != nil {
				return nil, err
			}
			if !res.Next(ctx) {
				return 0, res.Err()
			}
			return res.Record().Values[0], nil
		})
		if err != nil {
			return total, err
		}
		switch v := countAny.(type) {
		case int64:
			total += int(v)
		case int:
			total += v
		default:
			total += len(propsList)
		}
	}
	return total, nil
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

var cypherIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func sanitizeCypherIdentifier(input string) (string, error) {
	if !cypherIdentifierPattern.MatchString(input) {
		return "", fmt.Errorf("invalid cypher identifier: %q", input)
	}
	return input, nil
}
