-- KGS Platform Upgrade Migrations
-- Run these manually (not via GORM auto-migrate) to avoid locking production tables.

-- ============================================================
-- P1: GIN index for JSONB property filtering (QueryNodes)
-- ============================================================
-- This index enables fast @> containment queries on kg_entities.properties
-- Required by: POST /v1/kg/{ns}/entities/query
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_entity_props_gin
  ON kg_entities USING gin(properties);

-- GIN index for edges properties (future-proofing)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_edge_props_gin
  ON kg_edges USING gin(properties);

-- ============================================================
-- P3: App Registry external_id for lookup by project UUID
-- ============================================================
-- This enables consumers to call GetAppByExternalID(projectID) → app_id
-- Required by: GET /v1/apps/by-external-id/{external_id}
ALTER TABLE apps ADD COLUMN IF NOT EXISTS external_id VARCHAR(128);

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_app_external_id
  ON apps(external_id) WHERE external_id IS NOT NULL;

-- ============================================================
-- Verify indexes created
-- ============================================================
-- SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'kg_entities' AND indexname LIKE 'idx_%';
-- SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'apps' AND indexname LIKE 'idx_%';
