-- Rollback: 0040_observe_service.down.sql
DROP TABLE IF EXISTS dedup_cache CASCADE;
DROP TABLE IF EXISTS compressed_observations CASCADE;
DROP TABLE IF EXISTS raw_observations CASCADE;
DROP TABLE IF EXISTS agent_sessions CASCADE;
