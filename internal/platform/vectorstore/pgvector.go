package vectorstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type PgVectorAdapter struct {
	db *sql.DB
}

func NewPgVectorAdapter(db *sql.DB) *PgVectorAdapter {
	return &PgVectorAdapter{db: db}
}

func (a *PgVectorAdapter) Upsert(ctx context.Context, doc VectorDocument) error {
	if a == nil || a.db == nil {
		return errors.New("pgvector adapter is not configured")
	}
	payload, err := json.Marshal(doc.DomainProps)
	if err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, `
INSERT INTO kg_vector_documents (
    node_id,
    node_type,
    domain_id,
    owner_tenant_id,
    owner_app_id,
    acl_visible_to,
    is_deleted,
    status_value,
    authority_score,
    domain_props,
    embedding,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, NULLIF($5, ''), $6, $7, NULLIF($8, ''), $9, $10::jsonb, $11::vector, NOW(), NOW()
)
ON CONFLICT (node_id) DO UPDATE SET
    node_type = EXCLUDED.node_type,
    domain_id = EXCLUDED.domain_id,
    owner_tenant_id = EXCLUDED.owner_tenant_id,
    owner_app_id = EXCLUDED.owner_app_id,
    acl_visible_to = EXCLUDED.acl_visible_to,
    is_deleted = EXCLUDED.is_deleted,
    status_value = EXCLUDED.status_value,
    authority_score = EXCLUDED.authority_score,
    domain_props = EXCLUDED.domain_props,
    embedding = EXCLUDED.embedding,
    updated_at = NOW()
`, doc.NodeID, doc.NodeType, doc.DomainID, doc.OwnerTenantID, doc.OwnerAppID, doc.ACLVisibleTo, doc.IsDeleted, doc.StatusValue, doc.AuthorityScore, payload, vectorLiteral(doc.Embedding))
	return err
}

func (a *PgVectorAdapter) Delete(ctx context.Context, nodeID string) error {
	if a == nil || a.db == nil {
		return errors.New("pgvector adapter is not configured")
	}
	_, err := a.db.ExecContext(ctx, `DELETE FROM kg_vector_documents WHERE node_id = $1`, nodeID)
	return err
}

func (a *PgVectorAdapter) ANN(ctx context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error) {
	if a == nil || a.db == nil {
		return nil, errors.New("pgvector adapter is not configured")
	}
	stmt, args := a.buildANNStatement(query, filter, opts)
	rows, err := a.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]VectorResult, 0)
	for rows.Next() {
		var doc VectorDocument
		var propsRaw []byte
		var aclRaw []byte
		var updatedAt time.Time
		var score float64
		if err := rows.Scan(
			&doc.NodeID,
			&doc.NodeType,
			&doc.DomainID,
			&doc.OwnerTenantID,
			&doc.OwnerAppID,
			&aclRaw,
			&doc.IsDeleted,
			&doc.StatusValue,
			&doc.AuthorityScore,
			&propsRaw,
			&updatedAt,
			&score,
		); err != nil {
			return nil, err
		}
		if len(aclRaw) > 0 {
			_ = json.Unmarshal(aclRaw, &doc.ACLVisibleTo)
		}
		if len(propsRaw) > 0 {
			_ = json.Unmarshal(propsRaw, &doc.DomainProps)
		}
		results = append(results, VectorResult{Document: doc, Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (a *PgVectorAdapter) buildANNStatement(query []float64, filter VectorFilter, opts ANNOptions) (string, []any) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	args := []any{vectorLiteral(query)}
	clauses := make([]string, 0, 8)
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
	if len(filter.ACLVisibleTo) > 0 {
		args = append(args, filter.ACLVisibleTo)
		clauses = append(clauses, fmt.Sprintf("acl_visible_to && $%d::text[]", len(args)))
	}

	setStmts := make([]string, 0, 2)
	if strings.TrimSpace(opts.IndexHint) != "" {
		setStmts = append(setStmts, "SET LOCAL enable_seqscan = off")
	}
	if opts.EfSearch > 0 {
		setStmts = append(setStmts, fmt.Sprintf("SET LOCAL hnsw.ef_search = %d", opts.EfSearch))
	}

	distanceExpr := "embedding <=> $1::vector"
	filterMode := strings.ToLower(strings.TrimSpace(opts.FilterMode))
	if filterMode == "post" {
		stmt := buildSQLPreamble(setStmts)
		stmt += fmt.Sprintf(`
WITH ranked AS (
    SELECT
        node_id,
        node_type,
        domain_id,
        owner_tenant_id,
        COALESCE(owner_app_id::text, '') AS owner_app_id,
        COALESCE(to_jsonb(acl_visible_to), '[]'::jsonb) AS acl_visible_to,
        is_deleted,
        COALESCE(status_value, '') AS status_value,
        authority_score,
        domain_props,
        updated_at,
        1 - (%s) AS score
    FROM kg_vector_documents
    WHERE NOT is_deleted
    ORDER BY %s ASC, updated_at DESC, node_id ASC
    LIMIT $%d
)
SELECT
    node_id,
    node_type,
    domain_id,
    owner_tenant_id,
    owner_app_id,
    acl_visible_to,
    is_deleted,
    status_value,
    authority_score,
    domain_props,
    updated_at,
    score
FROM ranked`, distanceExpr, distanceExpr, len(args)+1)
		args = append(args, opts.TopK)
		if len(clauses) > 0 {
			stmt += "\nWHERE " + strings.Join(clauses, "\n  AND ")
		}
		stmt += "\nORDER BY score DESC, updated_at DESC, node_id ASC"
		return stmt, args
	}

	stmt := buildSQLPreamble(setStmts)
	stmt += `
SELECT
    node_id,
    node_type,
    domain_id,
    owner_tenant_id,
    COALESCE(owner_app_id::text, '') AS owner_app_id,
    COALESCE(to_jsonb(acl_visible_to), '[]'::jsonb) AS acl_visible_to,
    is_deleted,
    COALESCE(status_value, '') AS status_value,
    authority_score,
    domain_props,
    updated_at,
    1 - (embedding <=> $1::vector) AS score
FROM kg_vector_documents
WHERE NOT is_deleted`
	if len(clauses) > 0 {
		stmt += "\n  AND " + strings.Join(clauses, "\n  AND ")
	}
	stmt += fmt.Sprintf(`
ORDER BY %s ASC, updated_at DESC, node_id ASC
LIMIT $%d`, distanceExpr, len(args)+1)
	args = append(args, opts.TopK)
	return stmt, args
}

func buildSQLPreamble(statements []string) string {
	if len(statements) == 0 {
		return ""
	}
	return strings.Join(statements, ";\n") + ";\n"
}

func vectorLiteral(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatFloat(value, 'f', -1, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
