package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"kg-service/internal/write"
)

type Repository struct {
	db *sql.DB
	tx *sql.Tx
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithTx(tx *sql.Tx) write.Repository {
	return &Repository{db: r.db, tx: tx}
}

func (r *Repository) CreateNodeWithOutbox(ctx context.Context, node write.NodeRecord, event write.OutboxEvent) error {
	if err := r.insertNode(ctx, node); err != nil {
		return err
	}
	return r.insertOutbox(ctx, event)
}

func (r *Repository) CreateNodeBundle(ctx context.Context, node write.NodeRecord, rels []write.RelationshipRecord, event write.OutboxEvent) error {
	if err := r.insertNode(ctx, node); err != nil {
		return err
	}
	for _, rel := range rels {
		if err := r.insertRelationship(ctx, rel); err != nil {
			return err
		}
	}
	return r.insertOutbox(ctx, event)
}

func (r *Repository) UpdateNodeWithOutbox(ctx context.Context, node write.NodeRecord, event write.OutboxEvent) error {
	res, err := r.exec(ctx, `
		UPDATE kg_nodes
		SET node_type = $2,
			domain_id = $3,
			owner_tenant_id = $4,
			owner_app_id = $5,
			visibility = $6,
			properties = $7,
			domain_version = $8,
			external_ref = $9,
			status_value = $10,
			is_deleted = $11,
			updated_at = $12
		WHERE id = $1
	`, node.ID, node.NodeType, node.DomainID, node.OwnerTenantID, nullString(node.OwnerAppID), node.Visibility, node.Properties, node.DomainVersion, nullString(node.ExternalRef), nullString(node.StatusValue), node.IsDeleted, node.UpdatedAt)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return write.ErrNodeNotFound
	}
	return r.insertOutbox(ctx, event)
}

func (r *Repository) SoftDeleteNodeWithOutbox(ctx context.Context, node write.NodeRecord, event write.OutboxEvent) error {
	res, err := r.exec(ctx, `
		UPDATE kg_nodes
		SET is_deleted = $2,
			updated_at = $3,
			status_value = $4
		WHERE id = $1
	`, node.ID, node.IsDeleted, node.UpdatedAt, nullString(node.StatusValue))
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return write.ErrNodeNotFound
	}
	return r.insertOutbox(ctx, event)
}

func (r *Repository) CreateRelationshipWithOutbox(ctx context.Context, rel write.RelationshipRecord, event write.OutboxEvent) error {
	if err := r.insertRelationship(ctx, rel); err != nil {
		return err
	}
	return r.insertOutbox(ctx, event)
}

func (r *Repository) GetNodeByID(id string) (write.NodeRecord, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		FROM kg_nodes
		WHERE id = $1
	`, id)
	node, err := scanNode(row)
	if err != nil {
		return write.NodeRecord{}, false
	}
	return node, true
}

func (r *Repository) GetNodeByExternalRef(externalRef string) (write.NodeRecord, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		FROM kg_nodes
		WHERE external_ref = $1
	`, externalRef)
	node, err := scanNode(row)
	if err != nil {
		return write.NodeRecord{}, false
	}
	return node, true
}

func (r *Repository) ListNodes() []write.NodeRecord {
	rows, err := r.query(context.Background(), `
		SELECT id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		FROM kg_nodes
		ORDER BY created_at, id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []write.NodeRecord
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			continue
		}
		result = append(result, node)
	}
	return result
}

func (r *Repository) GetRelationshipByID(id string) (write.RelationshipRecord, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, created_at
		FROM kg_relationships
		WHERE id = $1
	`, id)
	rel, err := scanRelationship(row)
	if err != nil {
		return write.RelationshipRecord{}, false
	}
	return rel, true
}

func (r *Repository) ListRelationships() []write.RelationshipRecord {
	rows, err := r.query(context.Background(), `
		SELECT id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, created_at
		FROM kg_relationships
		ORDER BY created_at, id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []write.RelationshipRecord
	for rows.Next() {
		rel, err := scanRelationship(rows)
		if err != nil {
			continue
		}
		result = append(result, rel)
	}
	return result
}

func (r *Repository) ListOutboxEvents() []write.OutboxEvent {
	rows, err := r.query(context.Background(), `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at, processed_at
		FROM kg_outbox_events
		ORDER BY created_at, id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []write.OutboxEvent
	for rows.Next() {
		event, err := scanOutboxEvent(rows)
		if err != nil {
			continue
		}
		result = append(result, event)
	}
	return result
}

func (r *Repository) GetOutboxEventByID(id string) (write.OutboxEvent, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at, processed_at
		FROM kg_outbox_events
		WHERE id = $1
	`, id)
	event, err := scanOutboxEvent(row)
	if err != nil {
		return write.OutboxEvent{}, false
	}
	return event, true
}

func (r *Repository) insertNode(ctx context.Context, node write.NodeRecord) error {
	_, err := r.exec(ctx, `
		INSERT INTO kg_nodes (
			id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), NULLIF($10, ''), $11, $12, $13)
	`, node.ID, node.NodeType, node.DomainID, node.OwnerTenantID, nullString(node.OwnerAppID), node.Visibility, node.Properties, node.DomainVersion, node.ExternalRef, node.StatusValue, node.IsDeleted, node.CreatedAt, node.UpdatedAt)
	return normalizeWriteError(err)
}

func (r *Repository) insertRelationship(ctx context.Context, rel write.RelationshipRecord) error {
	_, err := r.exec(ctx, `
		INSERT INTO kg_relationships (
			id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, is_deleted, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10)
	`, rel.ID, rel.RelType, rel.FromNodeID, rel.ToNodeID, rel.DomainID, rel.OwnerTenantID, nullString(rel.OwnerAppID), rel.DomainVersion, rel.Properties, rel.CreatedAt)
	return err
}

func (r *Repository) insertOutbox(ctx context.Context, event write.OutboxEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	_, err = r.exec(ctx, `
		INSERT INTO kg_outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.ID, event.AggregateType, event.AggregateID, event.EventType, payload, event.Status, event.RetryCount, event.CreatedAt, event.ProcessedAt)
	return err
}

func (r *Repository) UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error {
	_, err := r.exec(ctx, `
		UPDATE kg_outbox_events
		SET status = $2,
			retry_count = $3,
			processed_at = $4
		WHERE id = $1
	`, eventID, status, retryCount, processedAt)
	return err
}

func (r *Repository) UpsertProjectionVersion(ctx context.Context, record write.ProjectionVersionRecord) error {
	_, err := r.exec(ctx, `
		INSERT INTO kg_projection_versions (
			entity_id, entity_kind, source_version, source_event_id, source_updated_at,
			graph_backend, graph_version, graph_synced_at,
			vector_backend, vector_version, vector_synced_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11
		)
		ON CONFLICT (entity_id, entity_kind) DO UPDATE SET
			source_version = EXCLUDED.source_version,
			source_event_id = EXCLUDED.source_event_id,
			source_updated_at = EXCLUDED.source_updated_at,
			graph_backend = EXCLUDED.graph_backend,
			graph_version = EXCLUDED.graph_version,
			graph_synced_at = EXCLUDED.graph_synced_at,
			vector_backend = EXCLUDED.vector_backend,
			vector_version = EXCLUDED.vector_version,
			vector_synced_at = EXCLUDED.vector_synced_at
	`, record.EntityID, record.EntityKind, record.SourceVersion, record.SourceEventID, record.SourceUpdatedAt, record.GraphBackend, nullInt64(record.GraphVersion), nullTime(record.LastGraphSyncedAt), record.VectorBackend, nullInt64(record.VectorVersion), nullTime(record.LastVectorSyncedAt))
	return err
}

func (r *Repository) GetProjectionVersion(entityID, entityKind string) (write.ProjectionVersionRecord, bool) {
	row := r.queryRow(context.Background(), `
		SELECT entity_id, entity_kind, source_version, source_event_id, source_updated_at,
		       COALESCE(graph_backend, ''), COALESCE(graph_version, 0), graph_synced_at,
		       COALESCE(vector_backend, ''), COALESCE(vector_version, 0), vector_synced_at
		FROM kg_projection_versions
		WHERE entity_id = $1 AND entity_kind = $2
	`, entityID, entityKind)
	var record write.ProjectionVersionRecord
	var graphSyncedAt, vectorSyncedAt sql.NullTime
	if err := row.Scan(
		&record.EntityID,
		&record.EntityKind,
		&record.SourceVersion,
		&record.SourceEventID,
		&record.SourceUpdatedAt,
		&record.GraphBackend,
		&record.GraphVersion,
		&graphSyncedAt,
		&record.VectorBackend,
		&record.VectorVersion,
		&vectorSyncedAt,
	); err != nil {
		return write.ProjectionVersionRecord{}, false
	}
	if graphSyncedAt.Valid {
		record.LastGraphSyncedAt = graphSyncedAt.Time
	}
	if vectorSyncedAt.Valid {
		record.LastVectorSyncedAt = vectorSyncedAt.Time
	}
	return record, true
}

func (r *Repository) ListProjectionVersions() []write.ProjectionVersionRecord {
	rows, err := r.query(context.Background(), `
		SELECT entity_id, entity_kind, source_version, source_event_id, source_updated_at,
		       COALESCE(graph_backend, ''), COALESCE(graph_version, 0), graph_synced_at,
		       COALESCE(vector_backend, ''), COALESCE(vector_version, 0), vector_synced_at
		FROM kg_projection_versions
		ORDER BY entity_kind, entity_id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]write.ProjectionVersionRecord, 0)
	for rows.Next() {
		var record write.ProjectionVersionRecord
		var graphSyncedAt, vectorSyncedAt sql.NullTime
		if err := rows.Scan(
			&record.EntityID,
			&record.EntityKind,
			&record.SourceVersion,
			&record.SourceEventID,
			&record.SourceUpdatedAt,
			&record.GraphBackend,
			&record.GraphVersion,
			&graphSyncedAt,
			&record.VectorBackend,
			&record.VectorVersion,
			&vectorSyncedAt,
		); err != nil {
			continue
		}
		if graphSyncedAt.Valid {
			record.LastGraphSyncedAt = graphSyncedAt.Time
		}
		if vectorSyncedAt.Valid {
			record.LastVectorSyncedAt = vectorSyncedAt.Time
		}
		result = append(result, record)
	}
	return result
}

func (r *Repository) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if r.tx != nil {
		return r.tx.ExecContext(ctx, query, args...)
	}
	return r.db.ExecContext(ctx, query, args...)
}

func (r *Repository) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if r.tx != nil {
		return r.tx.QueryContext(ctx, query, args...)
	}
	return r.db.QueryContext(ctx, query, args...)
}

func (r *Repository) queryRow(ctx context.Context, query string, args ...any) rowScanner {
	if r.tx != nil {
		return r.tx.QueryRowContext(ctx, query, args...)
	}
	return r.db.QueryRowContext(ctx, query, args...)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNode(row rowScanner) (write.NodeRecord, error) {
	var node write.NodeRecord
	var properties []byte
	var externalRef, statusValue sql.NullString
	var ownerApp sql.NullString
	if err := row.Scan(
		&node.ID,
		&node.NodeType,
		&node.DomainID,
		&node.OwnerTenantID,
		&ownerApp,
		&node.Visibility,
		&properties,
		&node.DomainVersion,
		&externalRef,
		&statusValue,
		&node.IsDeleted,
		&node.CreatedAt,
		&node.UpdatedAt,
	); err != nil {
		return write.NodeRecord{}, err
	}
	node.OwnerAppID = ownerApp.String
	node.ExternalRef = externalRef.String
	node.StatusValue = statusValue.String
	node.ACLVisibleTo = []string{fmt.Sprintf("%s:%s", node.OwnerTenantID, node.OwnerAppID)}
	if len(properties) > 0 {
		if err := json.Unmarshal(properties, &node.Properties); err != nil {
			return write.NodeRecord{}, err
		}
	}
	if node.Properties == nil {
		node.Properties = map[string]any{}
	}
	return node, nil
}

func scanRelationship(row rowScanner) (write.RelationshipRecord, error) {
	var rel write.RelationshipRecord
	var properties []byte
	var ownerApp sql.NullString
	if err := row.Scan(
		&rel.ID,
		&rel.RelType,
		&rel.FromNodeID,
		&rel.ToNodeID,
		&rel.DomainID,
		&rel.OwnerTenantID,
		&ownerApp,
		&rel.DomainVersion,
		&properties,
		&rel.CreatedAt,
	); err != nil {
		return write.RelationshipRecord{}, err
	}
	rel.OwnerAppID = ownerApp.String
	if len(properties) > 0 {
		if err := json.Unmarshal(properties, &rel.Properties); err != nil {
			return write.RelationshipRecord{}, err
		}
	}
	if rel.Properties == nil {
		rel.Properties = map[string]any{}
	}
	return rel, nil
}

func scanOutboxEvent(row rowScanner) (write.OutboxEvent, error) {
	var event write.OutboxEvent
	var payload []byte
	var processedAt sql.NullTime
	if err := row.Scan(
		&event.ID,
		&event.AggregateType,
		&event.AggregateID,
		&event.EventType,
		&payload,
		&event.Status,
		&event.RetryCount,
		&event.CreatedAt,
		&processedAt,
	); err != nil {
		return write.OutboxEvent{}, err
	}
	if processedAt.Valid {
		event.ProcessedAt = &processedAt.Time
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return write.OutboxEvent{}, err
		}
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	return event, nil
}

func normalizeWriteError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "external_ref") || strings.Contains(msg, "kg_nodes_external_ref_key") || strings.Contains(msg, "duplicate key value") {
		return errors.Join(write.ErrDuplicateExternalRef, err)
	}
	return err
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
