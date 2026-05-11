package neo4j

import (
	"context"
	"fmt"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

// --- IndexRepository ---

// BuildIndices creates all required indexes (idempotent — uses IF NOT EXISTS).
func (d *Driver) BuildIndices(ctx context.Context, groupID string, defs []domain.IndexDefinition) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	for _, def := range defs {
		var cypher string

		switch def.Type {
		case domain.IndexVector:
			cypher = fmt.Sprintf(
				"CREATE VECTOR INDEX %s IF NOT EXISTS FOR (n:%s) ON (n.%s) OPTIONS {indexConfig: {"+
					"`vector.dimensions`: %d, "+
					"`vector.similarity_function`: '%s'}}",
				def.Name, def.TargetLabel, def.Properties[0],
				def.VectorDimension, def.SimilarityFunc,
			)

		case domain.IndexFulltext:
			propsList := ""
			for i, p := range def.Properties {
				if i > 0 {
					propsList += ", "
				}
				propsList += fmt.Sprintf("n.%s", p)
			}
			cypher = fmt.Sprintf(
				`CREATE FULLTEXT INDEX %s IF NOT EXISTS FOR (n:%s) ON EACH [%s]`,
				def.Name, def.TargetLabel, propsList,
			)

		case domain.IndexRange:
			cypher = fmt.Sprintf(
				`CREATE INDEX %s IF NOT EXISTS FOR (n:%s) ON (n.%s)`,
				def.Name, def.TargetLabel, def.Properties[0],
			)

		case domain.IndexComposite:
			propsList := ""
			for i, p := range def.Properties {
				if i > 0 {
					propsList += ", "
				}
				propsList += fmt.Sprintf("n.%s", p)
			}
			cypher = fmt.Sprintf(
				`CREATE INDEX %s IF NOT EXISTS FOR (n:%s) ON (%s)`,
				def.Name, def.TargetLabel, propsList,
			)

		default:
			d.logger.Warn("skip unknown index type", "type", def.Type, "name", def.Name)
			continue
		}

		_, err := session.Run(ctx, cypher, nil)
		if err != nil {
			return fmt.Errorf("neo4j: create index %s: %w", def.Name, err)
		}
		d.logger.Info("index created", "name", def.Name, "type", def.Type)
	}

	return nil
}

// DropIndices removes all custom indexes.
func (d *Driver) DropIndices(ctx context.Context, groupID string) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	defs := domain.DefaultIndexes(0)
	for _, def := range defs {
		cypher := fmt.Sprintf(`DROP INDEX %s IF EXISTS`, def.Name)
		_, err := session.Run(ctx, cypher, nil)
		if err != nil {
			d.logger.Warn("drop index failed", "name", def.Name, "error", err)
		}
	}
	return nil
}

// ListIndices returns current index definitions from Neo4j.
func (d *Driver) ListIndices(ctx context.Context) ([]domain.IndexDefinition, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	result, err := session.Run(ctx, `SHOW INDEXES YIELD name, type, labelsOrTypes, properties`, nil)
	if err != nil {
		return nil, fmt.Errorf("neo4j: list indexes: %w", err)
	}

	var indexes []domain.IndexDefinition
	for result.Next(ctx) {
		record := result.Record()
		idx := domain.IndexDefinition{
			Name: getRecordString(record, "name"),
		}

		if typeVal, ok := record.Get("type"); ok {
			if t, ok := typeVal.(string); ok {
				switch t {
				case "VECTOR":
					idx.Type = domain.IndexVector
				case "FULLTEXT":
					idx.Type = domain.IndexFulltext
				case "RANGE":
					idx.Type = domain.IndexRange
				default:
					idx.Type = domain.IndexType(t)
				}
			}
		}

		indexes = append(indexes, idx)
	}
	return indexes, nil
}
