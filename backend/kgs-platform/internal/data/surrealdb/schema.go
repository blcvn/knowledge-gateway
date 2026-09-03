package surrealdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

// InitSchema creates all SurrealDB tables, fields, and indexes
// equivalent to the GORM auto-migrate in data.go.
// This is idempotent — safe to run on every startup.
func InitSchema(ctx context.Context, client *Client, vectorSize int, logger log.Logger) error {
	l := log.NewHelper(logger)
	l.Info("[KGS][SurrealDB] Initializing schema...")

	if vectorSize <= 0 {
		vectorSize = 1536
	}

	statements := buildSchemaStatements(vectorSize)

	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if _, err := client.Query(ctx, trimmed, nil); err != nil {
			return fmt.Errorf("schema init failed on statement %q: %w", truncate(trimmed, 120), err)
		}
	}

	l.Infof("[KGS][SurrealDB] Schema initialized: %d statements executed", len(statements))
	return nil
}

func buildSchemaStatements(vectorSize int) []string {
	return []string{
		// ── App Registry (L4 Governance) ──────────────────────────
		`DEFINE TABLE kgs_apps SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_apps TYPE string`,
		`DEFINE FIELD app_name ON kgs_apps TYPE string`,
		`DEFINE FIELD description ON kgs_apps TYPE option<string>`,
		`DEFINE FIELD owner ON kgs_apps TYPE string`,
		`DEFINE FIELD status ON kgs_apps TYPE string DEFAULT "ACTIVE"`,
		`DEFINE FIELD created_at ON kgs_apps TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD updated_at ON kgs_apps TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD deleted_at ON kgs_apps TYPE option<datetime>`,
		`DEFINE INDEX idx_apps_app_id ON kgs_apps FIELDS app_id UNIQUE`,

		// ── API Keys ──────────────────────────────────────────────
		`DEFINE TABLE kgs_api_keys SCHEMALESS`,
		`DEFINE FIELD key_hash ON kgs_api_keys TYPE string`,
		`DEFINE FIELD app_id ON kgs_api_keys TYPE string`,
		`DEFINE FIELD key_prefix ON kgs_api_keys TYPE string`,
		`DEFINE FIELD name ON kgs_api_keys TYPE option<string>`,
		`DEFINE FIELD scopes ON kgs_api_keys TYPE option<string>`,
		`DEFINE FIELD is_revoked ON kgs_api_keys TYPE bool DEFAULT false`,
		`DEFINE FIELD expires_at ON kgs_api_keys TYPE option<datetime>`,
		`DEFINE FIELD created_at ON kgs_api_keys TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_api_keys_hash ON kgs_api_keys FIELDS key_hash UNIQUE`,
		`DEFINE INDEX idx_api_keys_app ON kgs_api_keys FIELDS app_id`,

		// ── Quotas ────────────────────────────────────────────────
		`DEFINE TABLE kgs_quotas SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_quotas TYPE string`,
		`DEFINE FIELD quota_type ON kgs_quotas TYPE string`,
		`DEFINE FIELD limit_value ON kgs_quotas TYPE int`,
		`DEFINE INDEX idx_quotas_app_type ON kgs_quotas FIELDS app_id, quota_type UNIQUE`,

		// ── Audit Logs ────────────────────────────────────────────
		`DEFINE TABLE kgs_audit_logs SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_audit_logs TYPE string`,
		`DEFINE FIELD action ON kgs_audit_logs TYPE string`,
		`DEFINE FIELD actor ON kgs_audit_logs TYPE string`,
		`DEFINE FIELD details ON kgs_audit_logs TYPE option<string>`,
		`DEFINE FIELD created_at ON kgs_audit_logs TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_audit_app ON kgs_audit_logs FIELDS app_id`,

		// ── Ontology: Entity Types ────────────────────────────────
		`DEFINE TABLE kgs_entity_types SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_entity_types TYPE string`,
		`DEFINE FIELD tenant_id ON kgs_entity_types TYPE string DEFAULT "default"`,
		`DEFINE FIELD name ON kgs_entity_types TYPE string`,
		`DEFINE FIELD description ON kgs_entity_types TYPE option<string>`,
		`DEFINE FIELD schema ON kgs_entity_types TYPE object`,
		`DEFINE FIELD created_at ON kgs_entity_types TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD updated_at ON kgs_entity_types TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_entity_types_unique ON kgs_entity_types FIELDS app_id, tenant_id, name UNIQUE`,

		// ── Ontology: Relation Types ──────────────────────────────
		`DEFINE TABLE kgs_relation_types SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_relation_types TYPE string`,
		`DEFINE FIELD tenant_id ON kgs_relation_types TYPE string DEFAULT "default"`,
		`DEFINE FIELD name ON kgs_relation_types TYPE string`,
		`DEFINE FIELD description ON kgs_relation_types TYPE option<string>`,
		`DEFINE FIELD properties ON kgs_relation_types TYPE option<object>`,
		`DEFINE FIELD source_types ON kgs_relation_types TYPE option<array>`,
		`DEFINE FIELD target_types ON kgs_relation_types TYPE option<array>`,
		`DEFINE FIELD created_at ON kgs_relation_types TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD updated_at ON kgs_relation_types TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_relation_types_unique ON kgs_relation_types FIELDS app_id, tenant_id, name UNIQUE`,

		// ── Rules ─────────────────────────────────────────────────
		`DEFINE TABLE kgs_rules SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_rules TYPE string`,
		`DEFINE FIELD tenant_id ON kgs_rules TYPE string DEFAULT "default"`,
		`DEFINE FIELD name ON kgs_rules TYPE string`,
		`DEFINE FIELD trigger_type ON kgs_rules TYPE string`,
		`DEFINE FIELD cron ON kgs_rules TYPE option<string>`,
		`DEFINE FIELD cypher_query ON kgs_rules TYPE string`,
		`DEFINE FIELD action ON kgs_rules TYPE option<string>`,
		`DEFINE FIELD payload ON kgs_rules TYPE option<object>`,
		`DEFINE FIELD is_active ON kgs_rules TYPE bool DEFAULT true`,
		`DEFINE FIELD created_at ON kgs_rules TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_rules_app ON kgs_rules FIELDS app_id, tenant_id`,

		// ── Rule Executions ───────────────────────────────────────
		`DEFINE TABLE kgs_rule_executions SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_rule_executions TYPE string`,
		`DEFINE FIELD tenant_id ON kgs_rule_executions TYPE string DEFAULT "default"`,
		`DEFINE FIELD rule_id ON kgs_rule_executions TYPE string`,
		`DEFINE FIELD status ON kgs_rule_executions TYPE string`,
		`DEFINE FIELD message ON kgs_rule_executions TYPE option<string>`,
		`DEFINE FIELD started_at ON kgs_rule_executions TYPE datetime`,
		`DEFINE FIELD ended_at ON kgs_rule_executions TYPE option<datetime>`,
		`DEFINE INDEX idx_rule_exec_app ON kgs_rule_executions FIELDS app_id, tenant_id`,

		// ── Policies ──────────────────────────────────────────────
		`DEFINE TABLE kgs_policies SCHEMALESS`,
		`DEFINE FIELD app_id ON kgs_policies TYPE string`,
		`DEFINE FIELD tenant_id ON kgs_policies TYPE string DEFAULT "default"`,
		`DEFINE FIELD name ON kgs_policies TYPE string`,
		`DEFINE FIELD description ON kgs_policies TYPE option<string>`,
		`DEFINE FIELD rego_content ON kgs_policies TYPE string`,
		`DEFINE FIELD is_active ON kgs_policies TYPE bool DEFAULT true`,
		`DEFINE FIELD created_at ON kgs_policies TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_policies_app ON kgs_policies FIELDS app_id, tenant_id`,

		// ── Knowledge Graph: Entities (source-of-truth) ───────────
		`DEFINE TABLE kg_entities SCHEMALESS`,
		`DEFINE FIELD entity_id ON kg_entities TYPE string`,
		`DEFINE FIELD app_id ON kg_entities TYPE string`,
		`DEFINE FIELD tenant_id ON kg_entities TYPE string`,
		`DEFINE FIELD entity_type ON kg_entities TYPE string`,
		`DEFINE FIELD name ON kg_entities TYPE string`,
		`DEFINE FIELD properties ON kg_entities TYPE option<object>`,
		`DEFINE FIELD confidence ON kg_entities TYPE float DEFAULT 1.0`,
		`DEFINE FIELD source_file ON kg_entities TYPE option<string>`,
		`DEFINE FIELD chunk_id ON kg_entities TYPE option<string>`,
		`DEFINE FIELD skill_id ON kg_entities TYPE option<string>`,
		`DEFINE FIELD version_id ON kg_entities TYPE option<string>`,
		`DEFINE FIELD provenance_type ON kg_entities TYPE option<string>`,
		`DEFINE FIELD domains ON kg_entities TYPE option<array>`,
		`DEFINE FIELD aliases ON kg_entities TYPE option<array>`,
		`DEFINE FIELD version ON kg_entities TYPE int DEFAULT 1`,
		`DEFINE FIELD is_deleted ON kg_entities TYPE bool DEFAULT false`,
		`DEFINE FIELD created_at ON kg_entities TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD updated_at ON kg_entities TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD deleted_at ON kg_entities TYPE option<datetime>`,
		`DEFINE FIELD embedding ON kg_entities TYPE option<array>`,
		`DEFINE INDEX idx_entities_id ON kg_entities FIELDS entity_id UNIQUE`,
		`DEFINE INDEX idx_entities_app_tenant ON kg_entities FIELDS app_id, tenant_id`,
		`DEFINE INDEX idx_entities_type ON kg_entities FIELDS app_id, tenant_id, entity_type`,
		// Vector index for semantic search
		fmt.Sprintf(`DEFINE INDEX idx_entity_vector ON kg_entities FIELDS embedding MTREE DIMENSION %d DIST COSINE`, vectorSize),

		// Full-text search index
		`DEFINE ANALYZER kgs_text TOKENIZERS blank, class FILTERS lowercase, ascii`,
		`DEFINE INDEX idx_entity_fulltext ON kg_entities FIELDS name SEARCH ANALYZER kgs_text BM25`,

		// ── Knowledge Graph: Edges ────────────────────────────────
		// Using SurrealDB RELATE for graph edges
		`DEFINE TABLE kg_edges SCHEMALESS`,
		`DEFINE FIELD edge_id ON kg_edges TYPE string`,
		`DEFINE FIELD app_id ON kg_edges TYPE string`,
		`DEFINE FIELD tenant_id ON kg_edges TYPE string`,
		`DEFINE FIELD from_entity_id ON kg_edges TYPE string`,
		`DEFINE FIELD to_entity_id ON kg_edges TYPE string`,
		`DEFINE FIELD relation_type ON kg_edges TYPE string`,
		`DEFINE FIELD properties ON kg_edges TYPE option<object>`,
		`DEFINE FIELD confidence ON kg_edges TYPE float DEFAULT 1.0`,
		`DEFINE FIELD version_id ON kg_edges TYPE option<string>`,
		`DEFINE FIELD is_deleted ON kg_edges TYPE bool DEFAULT false`,
		`DEFINE FIELD created_at ON kg_edges TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD updated_at ON kg_edges TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_edges_id ON kg_edges FIELDS edge_id UNIQUE`,
		`DEFINE INDEX idx_edges_app_tenant ON kg_edges FIELDS app_id, tenant_id`,
		`DEFINE INDEX idx_edges_from ON kg_edges FIELDS from_entity_id`,
		`DEFINE INDEX idx_edges_to ON kg_edges FIELDS to_entity_id`,

		// ── Graph Versions ────────────────────────────────────────
		`DEFINE TABLE graph_versions SCHEMALESS`,
		`DEFINE FIELD app_id ON graph_versions TYPE string`,
		`DEFINE FIELD tenant_id ON graph_versions TYPE string`,
		`DEFINE FIELD version ON graph_versions TYPE int`,
		`DEFINE FIELD description ON graph_versions TYPE option<string>`,
		`DEFINE FIELD created_at ON graph_versions TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_gv_app_tenant ON graph_versions FIELDS app_id, tenant_id`,

		// ── View Definitions (Projection) ─────────────────────────
		`DEFINE TABLE view_definitions SCHEMALESS`,
		`DEFINE FIELD app_id ON view_definitions TYPE string`,
		`DEFINE FIELD tenant_id ON view_definitions TYPE string DEFAULT "default"`,
		`DEFINE FIELD name ON view_definitions TYPE string`,
		`DEFINE FIELD role ON view_definitions TYPE string`,
		`DEFINE FIELD allowed_entity_types ON view_definitions TYPE option<array>`,
		`DEFINE FIELD allowed_fields ON view_definitions TYPE option<array>`,
		`DEFINE FIELD pii_fields ON view_definitions TYPE option<array>`,
		`DEFINE FIELD created_at ON view_definitions TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_view_defs_unique ON view_definitions FIELDS app_id, tenant_id, name, role UNIQUE`,

		// ── Overlay Graphs (session-based staging) ────────────────
		`DEFINE TABLE kg_overlays SCHEMALESS`,
		`DEFINE FIELD overlay_id ON kg_overlays TYPE string`,
		`DEFINE FIELD namespace ON kg_overlays TYPE string`,
		`DEFINE FIELD entity_deltas ON kg_overlays TYPE option<array>`,
		`DEFINE FIELD edge_deltas ON kg_overlays TYPE option<array>`,
		`DEFINE FIELD delete_entity_ids ON kg_overlays TYPE option<array>`,
		`DEFINE FIELD delete_edge_ids ON kg_overlays TYPE option<array>`,
		`DEFINE FIELD created_at ON kg_overlays TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD expires_at ON kg_overlays TYPE datetime`,
		`DEFINE INDEX idx_overlays_id ON kg_overlays FIELDS overlay_id UNIQUE`,

		// ── Overlay Sessions ──────────────────────────────────────
		`DEFINE TABLE kg_overlay_sessions SCHEMALESS`,
		`DEFINE FIELD session_id ON kg_overlay_sessions TYPE string`,
		`DEFINE FIELD overlay_id ON kg_overlay_sessions TYPE string`,
		`DEFINE FIELD created_at ON kg_overlay_sessions TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD expires_at ON kg_overlay_sessions TYPE datetime`,
		`DEFINE INDEX idx_overlay_sessions_id ON kg_overlay_sessions FIELDS session_id UNIQUE`,

		// ── Distributed Locks ─────────────────────────────────────
		`DEFINE TABLE kg_locks SCHEMALESS`,
		`DEFINE FIELD lock_key ON kg_locks TYPE string`,
		`DEFINE FIELD token ON kg_locks TYPE string`,
		`DEFINE FIELD owner ON kg_locks TYPE option<string>`,
		`DEFINE FIELD expires_at ON kg_locks TYPE datetime`,
		`DEFINE FIELD created_at ON kg_locks TYPE datetime DEFAULT time::now()`,
		`DEFINE INDEX idx_locks_key ON kg_locks FIELDS lock_key UNIQUE`,
	}
}
