package fts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type PgFTSAdapter struct {
	db       *sql.DB
	language string
}

func NewPgFTSAdapter(db *sql.DB, language string) *PgFTSAdapter {
	if strings.TrimSpace(language) == "" {
		language = "simple"
	}
	return &PgFTSAdapter{db: db, language: language}
}

func (a *PgFTSAdapter) Index(context.Context, FTSDocument) error {
	return nil
}

func (a *PgFTSAdapter) Delete(context.Context, string) error {
	return nil
}

func (a *PgFTSAdapter) Search(ctx context.Context, query FTSQuery, filter FTSFilter, opts SearchOptions) ([]FTSResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("fts postgres adapter is not configured")
	}
	stmt, args := a.buildSearchStatement(query, filter, opts)
	rows, err := a.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]FTSResult, 0)
	for rows.Next() {
		var (
			doc      FTSDocument
			propsRaw []byte
			score    float64
		)
		if err := rows.Scan(
			&doc.NodeID,
			&doc.NodeType,
			&doc.DomainID,
			&doc.OwnerTenantID,
			&doc.OwnerAppID,
			&doc.StatusValue,
			&doc.IsDeleted,
			&propsRaw,
			&score,
			&doc.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(propsRaw) > 0 {
			_ = json.Unmarshal(propsRaw, &doc.DomainProps)
		}
		results = append(results, FTSResult{Document: doc, Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (a *PgFTSAdapter) buildSearchStatement(query FTSQuery, filter FTSFilter, opts SearchOptions) (string, []any) {
	args := make([]any, 0, 8)
	clauses := make([]string, 0, 8)

	tsQueryExpr, tsQueryArg := buildTsQueryExpr(a.language, query)
	args = append(args, tsQueryArg)
	vectorExpr := buildDocumentVectorExpr(a.language, query.Fields)

	selectSQL := fmt.Sprintf(`
SELECT
    id,
    node_type,
    domain_id,
    owner_tenant_id,
    COALESCE(owner_app_id::text, ''),
    COALESCE(status_value, ''),
    is_deleted,
    properties,
    ts_rank_cd(%s, %s) AS score,
    created_at
FROM kg_nodes
WHERE NOT is_deleted
  AND %s @@ %s
`, vectorExpr, tsQueryExpr, vectorExpr, tsQueryExpr)

	if len(filter.DomainIDs) > 0 {
		args = append(args, filter.DomainIDs)
		clauses = append(clauses, fmt.Sprintf("domain_id = ANY($%d)", len(args)))
	}
	if len(filter.NodeTypes) > 0 {
		args = append(args, filter.NodeTypes)
		clauses = append(clauses, fmt.Sprintf("node_type = ANY($%d)", len(args)))
	}
	if len(filter.OwnerTenantIDs) > 0 {
		args = append(args, filter.OwnerTenantIDs)
		clauses = append(clauses, fmt.Sprintf("owner_tenant_id = ANY($%d)", len(args)))
	}
	if len(filter.OwnerAppIDs) > 0 {
		args = append(args, filter.OwnerAppIDs)
		clauses = append(clauses, fmt.Sprintf("COALESCE(owner_app_id::text, '') = ANY($%d)", len(args)))
	}

	if len(clauses) > 0 {
		selectSQL += "  AND " + strings.Join(clauses, "\n  AND ") + "\n"
	}

	if opts.TopK > 0 {
		args = append(args, opts.TopK)
		selectSQL += fmt.Sprintf("ORDER BY score DESC, created_at ASC, id ASC\nLIMIT $%d", len(args))
	} else {
		selectSQL += "ORDER BY score DESC, created_at ASC, id ASC"
	}

	return selectSQL, args
}

func buildTsQueryExpr(language string, query FTSQuery) (string, string) {
	text := strings.TrimSpace(query.Text)
	mode := normalizeMode(query.Mode)
	switch mode {
	case "any_token":
		return fmt.Sprintf("to_tsquery('%s', $1)", escapeTSConfig(language)), joinOrQuery(text)
	case "phrase":
		return fmt.Sprintf("phraseto_tsquery('%s', $1)", escapeTSConfig(language)), text
	default:
		return fmt.Sprintf("plainto_tsquery('%s', $1)", escapeTSConfig(language)), text
	}
}

func buildDocumentVectorExpr(language string, fields []string) string {
	cfg := fmt.Sprintf("'%s'", escapeTSConfig(language))
	if len(fields) == 0 {
		return "fts_vector"
	}
	parts := make([]string, 0, len(fields))
	for idx, field := range fields {
		weight := fieldWeight(idx)
		switch field {
		case "id", "node_type", "external_ref", "status_value", "domain_id":
			parts = append(parts, fmt.Sprintf("setweight(to_tsvector(%s, COALESCE(%s::text, '')), '%s')", cfg, fieldExpr(field), weight))
		default:
			parts = append(parts, fmt.Sprintf("setweight(to_tsvector(%s, COALESCE(properties ->> '%s', '')), '%s')", cfg, escapeSQLString(field), weight))
		}
	}
	return strings.Join(parts, " || ")
}

func fieldExpr(field string) string {
	switch field {
	case "domain_id":
		return "domain_id"
	case "status_value":
		return "status_value"
	case "external_ref":
		return "external_ref"
	case "node_type":
		return "node_type"
	case "id":
		return "id"
	default:
		return field
	}
}

func fieldWeight(index int) string {
	switch {
	case index == 0:
		return "A"
	case index == 1:
		return "B"
	case index == 2:
		return "C"
	default:
		return "D"
	}
}

func joinOrQuery(text string) string {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return text
	}
	return strings.Join(tokens, " | ")
}

func escapeTSConfig(language string) string {
	return strings.ReplaceAll(strings.TrimSpace(language), "'", "''")
}

func escapeSQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
