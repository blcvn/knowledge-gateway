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

const postgresMaxBindParameters = 65535

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

func (r *Repository) UpdateNode(ctx context.Context, node write.NodeRecord) error {
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
	return nil
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

func (r *Repository) SoftDeleteNode(ctx context.Context, node write.NodeRecord) error {
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
	return nil
}

func (r *Repository) SoftDeleteNodesByExternalRefPrefix(ctx context.Context, prefix string, deletedAt time.Time) ([]write.NodeRecord, error) {
	rows, err := r.query(ctx, `
		UPDATE kg_nodes
		SET is_deleted = true,
			updated_at = $2
		WHERE external_ref IS NOT NULL
		  AND external_ref LIKE $1
		  AND is_deleted = false
		RETURNING id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
	`, prefix+"%", deletedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deleted := make([]write.NodeRecord, 0)
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			continue
		}
		deleted = append(deleted, node)
	}
	return deleted, nil
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

func (r *Repository) GetNodesByIDs(ids []string) map[string]write.NodeRecord {
	if len(ids) == 0 {
		return map[string]write.NodeRecord{}
	}
	rows, err := r.query(context.Background(), `
		SELECT id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		FROM kg_nodes
		WHERE id = ANY($1::text[])
	`, arrayLiteral(ids))
	if err != nil {
		return map[string]write.NodeRecord{}
	}
	defer rows.Close()
	result := make(map[string]write.NodeRecord, len(ids))
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			continue
		}
		result[node.ID] = node
	}
	return result
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

func (r *Repository) GetNodesByExternalRefs(externalRefs []string) map[string]write.NodeRecord {
	if len(externalRefs) == 0 {
		return map[string]write.NodeRecord{}
	}
	rows, err := r.query(context.Background(), `
		SELECT id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		FROM kg_nodes
		WHERE external_ref = ANY($1::text[])
	`, arrayLiteral(externalRefs))
	if err != nil {
		return map[string]write.NodeRecord{}
	}
	defer rows.Close()
	result := make(map[string]write.NodeRecord, len(externalRefs))
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			continue
		}
		result[node.ExternalRef] = node
	}
	return result
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

func (r *Repository) ListNodesBatch(afterID string, limit int) []write.NodeRecord {
	query := `
		SELECT id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		FROM kg_nodes
	`
	args := []any{}
	if afterID != "" {
		query += " WHERE id > $1"
		args = append(args, afterID)
	}
	query += " ORDER BY id"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := r.query(context.Background(), query, args...)
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

func (r *Repository) CreateNodesBulkWithOutbox(ctx context.Context, nodes []write.NodeRecord, events []write.OutboxEvent) error {
	for _, chunk := range chunkNodeRecords(nodes) {
		query, args, err := buildNodeBulkUpsertQuery(chunk)
		if err != nil {
			return err
		}
		if _, err := r.exec(ctx, query, args...); err != nil {
			return normalizeWriteError(err)
		}
	}
	return r.CreateOutboxEvents(ctx, events)
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

func (r *Repository) GetRelationshipsByIDs(ids []string) map[string]write.RelationshipRecord {
	if len(ids) == 0 {
		return map[string]write.RelationshipRecord{}
	}
	rows, err := r.query(context.Background(), `
		SELECT id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, created_at
		FROM kg_relationships
		WHERE id = ANY($1::text[])
	`, arrayLiteral(ids))
	if err != nil {
		return map[string]write.RelationshipRecord{}
	}
	defer rows.Close()
	result := make(map[string]write.RelationshipRecord, len(ids))
	for rows.Next() {
		rel, err := scanRelationship(rows)
		if err != nil {
			continue
		}
		result[rel.ID] = rel
	}
	return result
}

func (r *Repository) CreateRelationshipsBulkWithOutbox(ctx context.Context, rels []write.RelationshipRecord, events []write.OutboxEvent) error {
	for _, chunk := range chunkRelationshipRecords(rels) {
		query, args, err := buildRelationshipBulkInsertQuery(chunk)
		if err != nil {
			return err
		}
		if _, err := r.exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return r.CreateOutboxEvents(ctx, events)
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

func (r *Repository) ListRelationshipsBatch(afterID string, limit int) []write.RelationshipRecord {
	query := `
		SELECT id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, created_at
		FROM kg_relationships
	`
	args := []any{}
	if afterID != "" {
		query += " WHERE id > $1"
		args = append(args, afterID)
	}
	query += " ORDER BY id"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := r.query(context.Background(), query, args...)
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

func (r *Repository) ClaimOutboxBatch(ctx context.Context, pageSize int) ([]write.OutboxEvent, error) {
	if pageSize <= 0 {
		pageSize = 100
	}
	rows, err := r.query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM kg_outbox_events
			WHERE status = 'PENDING'
			ORDER BY created_at, id
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE kg_outbox_events o
		SET status = 'PROCESSING'
		FROM claimed
		WHERE o.id = claimed.id
		RETURNING o.id, o.aggregate_type, o.aggregate_id, o.event_type, o.payload, o.status, o.retry_count, o.created_at, o.processed_at
	`, pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]write.OutboxEvent, 0, pageSize)
	for rows.Next() {
		event, err := scanOutboxEvent(rows)
		if err != nil {
			continue
		}
		result = append(result, event)
	}
	return result, nil
}

func (r *Repository) SoftDeleteNodesByExternalRefPrefixWithOutbox(ctx context.Context, prefix string, deletedAt time.Time) ([]write.NodeRecord, error) {
	rows, err := r.query(ctx, `
		UPDATE kg_nodes
		SET is_deleted = true,
			updated_at = $2
		WHERE external_ref IS NOT NULL
		  AND external_ref LIKE $1
		  AND NOT is_deleted
		RETURNING id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
	`, prefix+"%", deletedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deleted := make([]write.NodeRecord, 0)
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			continue
		}
		deleted = append(deleted, node)
	}
	return deleted, nil
}

func (r *Repository) SoftDeleteRelationshipsWithOutbox(ctx context.Context, relationshipIDs []string, deletedAt time.Time) ([]write.RelationshipRecord, error) {
	if len(relationshipIDs) == 0 {
		return []write.RelationshipRecord{}, nil
	}
	rows, err := r.query(ctx, `
		UPDATE kg_relationships
		SET is_deleted = true
		WHERE id = ANY($1::uuid[])
		  AND NOT is_deleted
		RETURNING id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, created_at
	`, arrayLiteral(relationshipIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deleted := make([]write.RelationshipRecord, 0, len(relationshipIDs))
	for rows.Next() {
		rel, err := scanRelationship(rows)
		if err != nil {
			continue
		}
		rel.IsDeleted = true
		deleted = append(deleted, rel)
	}
	_ = deletedAt
	return deleted, nil
}

func (r *Repository) CreateOutboxEvents(ctx context.Context, events []write.OutboxEvent) error {
	for _, chunk := range chunkOutboxEvents(events) {
		query, args, err := buildOutboxBulkInsertQuery(chunk)
		if err != nil {
			return err
		}
		if _, err := r.exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
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
	return normalizeRepositoryError(err)
}

func (r *Repository) insertRelationship(ctx context.Context, rel write.RelationshipRecord) error {
	_, err := r.exec(ctx, `
		INSERT INTO kg_relationships (
			id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, is_deleted, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10)
	`, rel.ID, rel.RelType, rel.FromNodeID, rel.ToNodeID, rel.DomainID, rel.OwnerTenantID, nullString(rel.OwnerAppID), rel.DomainVersion, rel.Properties, rel.CreatedAt)
	return normalizeRepositoryError(err)
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

func (r *Repository) SealGraphVersion(ctx context.Context, request write.GraphVersionSealRequest) (write.GraphIdentityRecord, write.GraphVersionRecord, error) {
	now := time.Now().UTC()
	if request.ReferenceID == "" {
		return write.GraphIdentityRecord{}, write.GraphVersionRecord{}, errors.New("reference_id is required")
	}
	_, err := r.exec(ctx, `
		INSERT INTO kg_graph_identifiers (
			owner_tenant_id, owner_app_id, graph_scope, status, head_version_number, created_at, updated_at
		) VALUES ($1, $2, $3, 'ACTIVE', 0, $4, $4)
		ON CONFLICT (owner_tenant_id, owner_app_id, graph_scope) DO UPDATE SET
			updated_at = EXCLUDED.updated_at
	`, request.OwnerTenantID, nullString(request.OwnerAppID), request.GraphScope, now)
	if err != nil {
		return write.GraphIdentityRecord{}, write.GraphVersionRecord{}, normalizeRepositoryError(err)
	}
	var identity write.GraphIdentityRecord
	var headVersionID sql.NullString
	var ownerAppID sql.NullString
	row := r.queryRow(ctx, `
		SELECT identifier_id, owner_tenant_id, owner_app_id, graph_scope, head_version_number, head_version_id, created_at, updated_at
		FROM kg_graph_identifiers
		WHERE owner_tenant_id = $1
		  AND owner_app_id IS NOT DISTINCT FROM $2
		  AND graph_scope = $3
		FOR UPDATE
	`, request.OwnerTenantID, nullString(request.OwnerAppID), request.GraphScope)
	if err := row.Scan(&identity.IdentifierID, &identity.OwnerTenantID, &ownerAppID, &identity.GraphScope, &identity.HeadVersionNumber, &headVersionID, &identity.CreatedAt, &identity.UpdatedAt); err != nil {
		return write.GraphIdentityRecord{}, write.GraphVersionRecord{}, err
	}
	identity.OwnerAppID = ownerAppID.String
	if headVersionID.Valid {
		identity.HeadVersionID = headVersionID.String
	}
	nextVersion := identity.HeadVersionNumber + 1
	var versionID string
	row = r.queryRow(ctx, `
		INSERT INTO kg_graph_versions (
			identifier_id, version_number, reference_id, storage_class, version_status, change_summary, manifest_locator, created_at, sealed_at
		) VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'ONLINE'), COALESCE(NULLIF($5, ''), 'SEALED'), $6, $7, $8, $8)
		RETURNING version_id
	`, identity.IdentifierID, nextVersion, request.ReferenceID, request.StorageClass, request.VersionStatus, nullString(request.ChangeSummary), nullString(request.ManifestLocator), now)
	if err := row.Scan(&versionID); err != nil {
		return write.GraphIdentityRecord{}, write.GraphVersionRecord{}, err
	}
	_, err = r.exec(ctx, `
		UPDATE kg_graph_identifiers
		SET head_version_number = $2,
			head_version_id = $3,
			updated_at = $4
		WHERE identifier_id = $1
	`, identity.IdentifierID, nextVersion, versionID, now)
	if err != nil {
		return write.GraphIdentityRecord{}, write.GraphVersionRecord{}, normalizeRepositoryError(err)
	}
	if len(request.Entities) > 0 {
		query, args, err := buildGraphVersionEntitiesInsertQuery(versionID, request.Entities)
		if err != nil {
			return write.GraphIdentityRecord{}, write.GraphVersionRecord{}, err
		}
		if _, err := r.exec(ctx, query, args...); err != nil {
			return write.GraphIdentityRecord{}, write.GraphVersionRecord{}, err
		}
	}
	identity.HeadVersionNumber = nextVersion
	identity.HeadVersionID = versionID
	identity.UpdatedAt = now
	version := write.GraphVersionRecord{
		VersionID:       versionID,
		IdentifierID:    identity.IdentifierID,
		VersionNumber:   nextVersion,
		ReferenceID:     request.ReferenceID,
		StorageClass:    request.StorageClass,
		VersionStatus:   request.VersionStatus,
		ChangeSummary:   request.ChangeSummary,
		ManifestLocator: request.ManifestLocator,
		CreatedAt:       now,
		SealedAt:        now,
	}
	if version.StorageClass == "" {
		version.StorageClass = "ONLINE"
	}
	if version.VersionStatus == "" {
		version.VersionStatus = "SEALED"
	}
	return identity, version, nil
}

func (r *Repository) FinalizeGraphVersion(ctx context.Context, versionID string) (int64, error) {
	res, err := r.exec(ctx, `
		UPDATE kg_graph_versions
		SET version_status = 'SEALED',
			sealed_at = COALESCE(sealed_at, now())
		WHERE version_id = $1 AND version_status = 'PENDING_ENTITIES'
	`, versionID)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *Repository) AddGraphVersionEntities(ctx context.Context, versionID string, entities []write.GraphVersionEntityRecord) error {
	if len(entities) == 0 {
		return nil
	}
	rows, err := r.query(ctx, `
		SELECT entity_kind, entity_id
		FROM kg_graph_version_entities
		WHERE version_id = $1
	`, versionID)
	if err != nil {
		return err
	}
	defer rows.Close()
	existing := map[string]struct{}{}
	for rows.Next() {
		var entityKind, entityID string
		if err := rows.Scan(&entityKind, &entityID); err != nil {
			continue
		}
		existing[entityKind+":"+entityID] = struct{}{}
	}
	filtered := make([]write.GraphVersionEntityRecord, 0, len(entities))
	for _, entity := range entities {
		key := entity.EntityKind + ":" + entity.EntityID
		if _, ok := existing[key]; ok {
			if _, err := r.exec(ctx, `
				DELETE FROM kg_graph_version_entities
				WHERE version_id = $1
				  AND entity_kind = $2
				  AND entity_id = $3
			`, versionID, entity.EntityKind, entity.EntityID); err != nil {
				return err
			}
		}
		existing[key] = struct{}{}
		filtered = append(filtered, entity)
	}
	if len(filtered) == 0 {
		return nil
	}
	for _, entity := range filtered {
		query, args, err := buildGraphVersionEntitiesInsertQuery(versionID, []write.GraphVersionEntityRecord{entity})
		if err != nil {
			return err
		}
		if _, err := r.exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) AbandonGraphVersion(ctx context.Context, versionID string) error {
	_, err := r.exec(ctx, `
		UPDATE kg_graph_versions
		SET version_status = 'ABANDONED'
		WHERE version_id = $1 AND version_status = 'PENDING_ENTITIES'
	`, versionID)
	return err
}

func (r *Repository) CleanupExpiredSyncSession(ctx context.Context, versionID string) error {
	if strings.TrimSpace(versionID) == "" {
		return nil
	}
	if r.tx != nil {
		return r.cleanupExpiredSyncSessionInTx(ctx, versionID)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txRepo := &Repository{db: r.db, tx: tx}
	if err := txRepo.cleanupExpiredSyncSessionInTx(ctx, versionID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *Repository) cleanupExpiredSyncSessionInTx(ctx context.Context, versionID string) error {
	var identityID, ownerTenantID, ownerAppID, graphScope string
	row := r.queryRow(ctx, `
		SELECT i.identifier_id, i.owner_tenant_id, COALESCE(i.owner_app_id, ''), i.graph_scope
		FROM kg_graph_versions v
		JOIN kg_graph_identifiers i ON i.identifier_id = v.identifier_id
		WHERE v.version_id = $1
	`, versionID)
	if err := row.Scan(&identityID, &ownerTenantID, &ownerAppID, &graphScope); err != nil {
		return err
	}
	_ = identityID
	if _, err := r.exec(ctx, `
		UPDATE kg_graph_versions
		SET version_status = 'ABANDONED'
		WHERE version_id = $1 AND version_status = 'PENDING_ENTITIES'
	`, versionID); err != nil {
		return err
	}
	_, err := r.exec(ctx, `
		DELETE FROM kg_graph_scope_leases
		WHERE owner_tenant_id = $1
		  AND owner_app_id IS NOT DISTINCT FROM NULLIF($2, '')
		  AND graph_scope = $3
		  AND version_id = $4
	`, ownerTenantID, ownerAppID, graphScope, versionID)
	return err
}

func (r *Repository) GetGraphVersionByID(ctx context.Context, versionID string) (write.GraphVersionRecord, bool) {
	row := r.queryRow(ctx, `
		SELECT version_id, identifier_id, version_number, reference_id, storage_class, version_status, COALESCE(change_summary, ''), COALESCE(manifest_locator, ''), created_at, sealed_at
		FROM kg_graph_versions
		WHERE version_id = $1
	`, versionID)
	var record write.GraphVersionRecord
	if err := row.Scan(&record.VersionID, &record.IdentifierID, &record.VersionNumber, &record.ReferenceID, &record.StorageClass, &record.VersionStatus, &record.ChangeSummary, &record.ManifestLocator, &record.CreatedAt, &record.SealedAt); err != nil {
		return write.GraphVersionRecord{}, false
	}
	return record, true
}

func (r *Repository) ListPendingGraphVersionsBefore(cutoff time.Time) []write.GraphVersionRecord {
	rows, err := r.query(context.Background(), `
		SELECT version_id, identifier_id, version_number, reference_id, storage_class, version_status, COALESCE(change_summary, ''), COALESCE(manifest_locator, ''), created_at, sealed_at
		FROM kg_graph_versions
		WHERE version_status = 'PENDING_ENTITIES'
		  AND created_at < $1
		ORDER BY created_at, version_id
	`, cutoff)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]write.GraphVersionRecord, 0)
	for rows.Next() {
		var record write.GraphVersionRecord
		if err := rows.Scan(&record.VersionID, &record.IdentifierID, &record.VersionNumber, &record.ReferenceID, &record.StorageClass, &record.VersionStatus, &record.ChangeSummary, &record.ManifestLocator, &record.CreatedAt, &record.SealedAt); err != nil {
			continue
		}
		result = append(result, record)
	}
	return result
}

func (r *Repository) GetGraphIdentityByID(ctx context.Context, identifierID string) (write.GraphIdentityRecord, bool) {
	row := r.queryRow(ctx, `
		SELECT identifier_id, owner_tenant_id, owner_app_id, graph_scope, head_version_number, COALESCE(head_version_id::text, ''), created_at, updated_at
		FROM kg_graph_identifiers
		WHERE identifier_id = $1
	`, identifierID)
	var record write.GraphIdentityRecord
	var ownerApp sql.NullString
	if err := row.Scan(&record.IdentifierID, &record.OwnerTenantID, &ownerApp, &record.GraphScope, &record.HeadVersionNumber, &record.HeadVersionID, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return write.GraphIdentityRecord{}, false
	}
	record.OwnerAppID = ownerApp.String
	return record, true
}

func (r *Repository) AcquireScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string, expiresAt time.Time) error {
	now := time.Now().UTC()
	lease, found := r.GetScopeLease(ctx, ownerTenantID, ownerAppID, graphScope)
	if found {
		if lease.VersionID != versionID && lease.ExpiresAt.After(now) {
			return write.ErrScopeLocked
		}
		if _, err := r.exec(ctx, `
			DELETE FROM kg_graph_scope_leases
			WHERE owner_tenant_id = $1
			  AND owner_app_id IS NOT DISTINCT FROM $2
			  AND graph_scope = $3
		`, ownerTenantID, nullString(ownerAppID), graphScope); err != nil {
			return err
		}
	}
	_, err := r.exec(ctx, `
		INSERT INTO kg_graph_scope_leases (
			owner_tenant_id, owner_app_id, graph_scope, version_id, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $6)
	`, ownerTenantID, nullString(ownerAppID), graphScope, versionID, expiresAt, now)
	if err != nil {
		if isUniqueViolation(err) {
			return write.ErrScopeLocked
		}
		return err
	}
	return nil
}

func (r *Repository) ReleaseScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error {
	_, err := r.exec(ctx, `
		DELETE FROM kg_graph_scope_leases
		WHERE owner_tenant_id = $1
		  AND owner_app_id IS NOT DISTINCT FROM $2
		  AND graph_scope = $3
		  AND version_id = $4
	`, ownerTenantID, nullString(ownerAppID), graphScope, versionID)
	return err
}

func (r *Repository) GetScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope string) (write.ScopeLeaseRecord, bool) {
	row := r.queryRow(ctx, `
		SELECT owner_tenant_id, owner_app_id, graph_scope, version_id, expires_at, created_at, updated_at
		FROM kg_graph_scope_leases
		WHERE owner_tenant_id = $1
		  AND owner_app_id IS NOT DISTINCT FROM $2
		  AND graph_scope = $3
	`, ownerTenantID, nullString(ownerAppID), graphScope)
	var record write.ScopeLeaseRecord
	var ownerApp sql.NullString
	if err := row.Scan(&record.OwnerTenantID, &ownerApp, &record.GraphScope, &record.VersionID, &record.ExpiresAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return write.ScopeLeaseRecord{}, false
	}
	record.OwnerAppID = ownerApp.String
	return record, true
}

func (r *Repository) GetGraphVersionEntities(versionID string) []write.GraphVersionEntityRecord {
	rows, err := r.query(context.Background(), `
		SELECT version_id, entity_kind, entity_id, change_kind
		FROM kg_graph_version_entities
		WHERE version_id = $1
		ORDER BY entity_kind, entity_id
	`, versionID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]write.GraphVersionEntityRecord, 0)
	for rows.Next() {
		var record write.GraphVersionEntityRecord
		if err := rows.Scan(&record.VersionID, &record.EntityKind, &record.EntityID, &record.ChangeKind); err != nil {
			continue
		}
		result = append(result, record)
	}
	return result
}

func (r *Repository) UpsertGraphProjectionHead(ctx context.Context, record write.GraphProjectionHeadRecord) error {
	_, err := r.exec(ctx, `
		INSERT INTO kg_graph_projection_heads (
			identifier_id, backend_kind, backend_name, applied_version_id, applied_version_number, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (identifier_id, backend_kind, backend_name) DO UPDATE SET
			applied_version_id = EXCLUDED.applied_version_id,
			applied_version_number = EXCLUDED.applied_version_number,
			updated_at = EXCLUDED.updated_at
		WHERE EXCLUDED.applied_version_number >= kg_graph_projection_heads.applied_version_number
	`, record.IdentifierID, record.BackendKind, record.BackendName, nullString(record.AppliedVersionID), record.AppliedVersionNumber, func() time.Time {
		if record.UpdatedAt.IsZero() {
			return time.Now().UTC()
		}
		return record.UpdatedAt
	}())
	return err
}

func (r *Repository) GetGraphProjectionHead(identifierID, backendKind, backendName string) (write.GraphProjectionHeadRecord, bool) {
	row := r.queryRow(context.Background(), `
		SELECT identifier_id, backend_kind, backend_name, COALESCE(applied_version_id::text, ''), applied_version_number, updated_at
		FROM kg_graph_projection_heads
		WHERE identifier_id = $1 AND backend_kind = $2 AND backend_name = $3
	`, identifierID, backendKind, backendName)
	var record write.GraphProjectionHeadRecord
	if err := row.Scan(&record.IdentifierID, &record.BackendKind, &record.BackendName, &record.AppliedVersionID, &record.AppliedVersionNumber, &record.UpdatedAt); err != nil {
		return write.GraphProjectionHeadRecord{}, false
	}
	return record, true
}

func buildGraphVersionEntitiesInsertQuery(versionID string, entities []write.GraphVersionEntityRecord) (string, []any, error) {
	var sb strings.Builder
	args := make([]any, 0, len(entities)*4)
	sb.WriteString(`
		INSERT INTO kg_graph_version_entities (
			version_id, entity_kind, entity_id, change_kind
		) VALUES
	`)
	for i, entity := range entities {
		if i > 0 {
			sb.WriteString(",")
		}
		base := len(args) + 1
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d)", base, base+1, base+2, base+3))
		args = append(args, versionID, entity.EntityKind, entity.EntityID, entity.ChangeKind)
	}
	return sb.String(), args, nil
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

func (r *Repository) ListProjectionVersionsBatch(afterEntityKind, afterEntityID string, limit int) []write.ProjectionVersionRecord {
	query := `
		SELECT entity_id, entity_kind, source_version, source_event_id, source_updated_at,
		       COALESCE(graph_backend, ''), COALESCE(graph_version, 0), graph_synced_at,
		       COALESCE(vector_backend, ''), COALESCE(vector_version, 0), vector_synced_at
		FROM kg_projection_versions
	`
	args := []any{}
	if afterEntityKind != "" {
		query += " WHERE (entity_kind, entity_id) > ($1, $2)"
		args = append(args, afterEntityKind, afterEntityID)
	}
	query += " ORDER BY entity_kind, entity_id"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := r.query(context.Background(), query, args...)
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

func buildNodeBulkUpsertQuery(nodes []write.NodeRecord) (string, []any, error) {
	var sb strings.Builder
	args := make([]any, 0, len(nodes)*13)
	sb.WriteString(`
		INSERT INTO kg_nodes (
			id, node_type, domain_id, owner_tenant_id, owner_app_id, visibility, properties, domain_version, external_ref, status_value, is_deleted, created_at, updated_at
		) VALUES
	`)
	for i, node := range nodes {
		if i > 0 {
			sb.WriteString(",")
		}
		payload, err := json.Marshal(node.Properties)
		if err != nil {
			return "", nil, err
		}
		base := len(args) + 1
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, NULLIF($%d, ''), NULLIF($%d, ''), $%d, $%d, $%d)", base, base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10, base+11, base+12))
		args = append(args, node.ID, node.NodeType, node.DomainID, node.OwnerTenantID, nullString(node.OwnerAppID), node.Visibility, payload, node.DomainVersion, node.ExternalRef, node.StatusValue, node.IsDeleted, node.CreatedAt, node.UpdatedAt)
	}
	sb.WriteString(`
		ON CONFLICT (id) DO UPDATE SET
			node_type = EXCLUDED.node_type,
			domain_id = EXCLUDED.domain_id,
			owner_tenant_id = EXCLUDED.owner_tenant_id,
			owner_app_id = EXCLUDED.owner_app_id,
			visibility = EXCLUDED.visibility,
			properties = EXCLUDED.properties,
			domain_version = EXCLUDED.domain_version,
			external_ref = EXCLUDED.external_ref,
			status_value = EXCLUDED.status_value,
			is_deleted = EXCLUDED.is_deleted,
			updated_at = EXCLUDED.updated_at
	`)
	return sb.String(), args, nil
}

func buildRelationshipBulkInsertQuery(rels []write.RelationshipRecord) (string, []any, error) {
	var sb strings.Builder
	args := make([]any, 0, len(rels)*11)
	sb.WriteString(`
		INSERT INTO kg_relationships (
			id, rel_type, from_node_id, to_node_id, domain_id, owner_tenant_id, owner_app_id, domain_version, properties, is_deleted, created_at
		) VALUES
	`)
	for i, rel := range rels {
		if i > 0 {
			sb.WriteString(",")
		}
		payload, err := json.Marshal(rel.Properties)
		if err != nil {
			return "", nil, err
		}
		base := len(args) + 1
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", base, base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10))
		args = append(args, rel.ID, rel.RelType, rel.FromNodeID, rel.ToNodeID, rel.DomainID, rel.OwnerTenantID, nullString(rel.OwnerAppID), rel.DomainVersion, payload, rel.IsDeleted, rel.CreatedAt)
	}
	sb.WriteString(`
		ON CONFLICT (id) DO UPDATE SET
			rel_type = EXCLUDED.rel_type,
			from_node_id = EXCLUDED.from_node_id,
			to_node_id = EXCLUDED.to_node_id,
			domain_id = EXCLUDED.domain_id,
			owner_tenant_id = EXCLUDED.owner_tenant_id,
			owner_app_id = EXCLUDED.owner_app_id,
			domain_version = EXCLUDED.domain_version,
			properties = EXCLUDED.properties,
			is_deleted = EXCLUDED.is_deleted
	`)
	return sb.String(), args, nil
}

func buildOutboxBulkInsertQuery(events []write.OutboxEvent) (string, []any, error) {
	var sb strings.Builder
	args := make([]any, 0, len(events)*9)
	sb.WriteString(`
		INSERT INTO kg_outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at, processed_at
		) VALUES
	`)
	for i, event := range events {
		if i > 0 {
			sb.WriteString(",")
		}
		payload, err := json.Marshal(event.Payload)
		if err != nil {
			return "", nil, err
		}
		base := len(args) + 1
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", base, base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8))
		args = append(args, event.ID, event.AggregateType, event.AggregateID, event.EventType, payload, event.Status, event.RetryCount, event.CreatedAt, event.ProcessedAt)
	}
	return sb.String(), args, nil
}

func arrayLiteral(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	escaped := make([]string, 0, len(values))
	for _, value := range values {
		escaped = append(escaped, strings.ReplaceAll(value, `"`, `\"`))
	}
	return `{"` + strings.Join(escaped, `","`) + `"}`
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

func normalizeRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	if normalized := normalizeWriteError(err); normalized != err {
		return normalized
	}
	msg := err.Error()
	if strings.Contains(msg, "owner_tenant_id_fkey") || strings.Contains(msg, "owner_app_id_fkey") {
		return errors.Join(write.ErrControlPlaneNotReady, err)
	}
	return err
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "23505")
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func chunkNodeRecords(nodes []write.NodeRecord) [][]write.NodeRecord {
	return chunkByBindLimit(nodes, 13)
}

func chunkRelationshipRecords(rels []write.RelationshipRecord) [][]write.RelationshipRecord {
	return chunkByBindLimit(rels, 11)
}

func chunkOutboxEvents(events []write.OutboxEvent) [][]write.OutboxEvent {
	return chunkByBindLimit(events, 9)
}

func chunkByBindLimit[T any](items []T, bindsPerItem int) [][]T {
	if len(items) == 0 {
		return nil
	}
	chunkSize := postgresMaxBindParameters / bindsPerItem
	if chunkSize <= 0 {
		chunkSize = 1
	}
	if len(items) <= chunkSize {
		return [][]T{items}
	}
	chunks := make([][]T, 0, (len(items)+chunkSize-1)/chunkSize)
	for start := 0; start < len(items); start += chunkSize {
		end := start + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
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
