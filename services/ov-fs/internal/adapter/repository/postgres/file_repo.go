package postgres

import (
\t"context"
\t"database/sql"
\t"errors"

\t"vnp-memory/services/ov-fs/internal/domain/model"
)

// FileRepositoryImpl implements the storage port for FSNode using PostgreSQL.
type FileRepositoryImpl struct {
\tdb *sql.DB
}

// NewFileRepository creates a new FileRepositoryImpl.
func NewFileRepository(db *sql.DB) *FileRepositoryImpl {
\treturn &FileRepositoryImpl{
\t\tdb: db,
\t}
}

// Save inserts or updates a file system node, enforcing optimistic concurrency.
func (r *FileRepositoryImpl) Save(ctx context.Context, node *model.FSNode) error {
\tquery := `
\t\tINSERT INTO fs_nodes (id, tenant_id, parent_id, name, type, size, mime_type, checksum_sha256, version, created_at, updated_at)
\t\tVALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
\t\tON CONFLICT (id) DO UPDATE SET
\t\t\tparent_id = EXCLUDED.parent_id,
\t\t\tname = EXCLUDED.name,
\t\t\tsize = EXCLUDED.size,
\t\t\tmime_type = EXCLUDED.mime_type,
\t\t\tchecksum_sha256 = EXCLUDED.checksum_sha256,
\t\t\tversion = fs_nodes.version + 1,
\t\t\tupdated_at = EXCLUDED.updated_at
\t\tWHERE fs_nodes.version = $9 - 1; -- Enforce OCC: only update if version matches expected previous version
\t`
\t
\t// For new items version is usually 1, so the condition version = 0 matches nothing on conflict.
\t// Wait, the UPSERT semantic in Postgres with OCC is slightly tricky.
\t// If it's an insert, ON CONFLICT will trigger. 
\tres, err := r.db.ExecContext(ctx, query,
\t\tnode.ID, node.TenantID, node.ParentID, node.Name, node.Type,
\t\tnode.Size, node.MimeType, node.ChecksumSHA256, node.Version,
\t\tnode.CreatedAt, node.UpdatedAt,
\t)
\tif err != nil {
\t\treturn err
\t}

\taffected, err := res.RowsAffected()
\tif err != nil {
\t\treturn err
\t}
\tif affected == 0 {
\t\treturn errors.New("optimistic concurrency conflict: node was modified by another transaction")
\t}

\treturn nil
}

// FindByID retrieves an FSNode by its ID and TenantID to enforce data isolation.
func (r *FileRepositoryImpl) FindByID(ctx context.Context, id, tenantID string) (*model.FSNode, error) {
\tquery := `
\t\tSELECT id, tenant_id, parent_id, name, type, size, mime_type, checksum_sha256, version, created_at, updated_at
\t\tFROM fs_nodes
\t\tWHERE id = $1 AND tenant_id = $2
\t`
\t
\tvar node model.FSNode
\terr := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
\t\t&node.ID, &node.TenantID, &node.ParentID, &node.Name, &node.Type,
\t\t&node.Size, &node.MimeType, &node.ChecksumSHA256, &node.Version,
\t\t&node.CreatedAt, &node.UpdatedAt,
\t)
\tif err != nil {
\t\tif errors.Is(err, sql.ErrNoRows) {
\t\t\treturn nil, nil // Not found
\t\t}
\t\treturn nil, err
\t}

\treturn &node, nil
}

// ListByParent retrieves all children of a specific directory.
func (r *FileRepositoryImpl) ListByParent(ctx context.Context, parentID, tenantID string) ([]*model.FSNode, error) {
\tquery := `
\t\tSELECT id, tenant_id, parent_id, name, type, size, mime_type, checksum_sha256, version, created_at, updated_at
\t\tFROM fs_nodes
\t\tWHERE parent_id = $1 AND tenant_id = $2
\t\tORDER BY type ASC, name ASC
\t` // Order directories first, then by name
\t
\trows, err := r.db.QueryContext(ctx, query, parentID, tenantID)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()

\tvar nodes []*model.FSNode
\tfor rows.Next() {
\t\tvar node model.FSNode
\t\tif err := rows.Scan(
\t\t\t&node.ID, &node.TenantID, &node.ParentID, &node.Name, &node.Type,
\t\t\t&node.Size, &node.MimeType, &node.ChecksumSHA256, &node.Version,
\t\t\t&node.CreatedAt, &node.UpdatedAt,
\t\t); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tnodes = append(nodes, &node)
\t}

\tif err = rows.Err(); err != nil {
\t\treturn nil, err
\t}

\treturn nodes, nil
}
