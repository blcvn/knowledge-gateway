-- Rollback: 0044_governance.down.sql
DROP TABLE IF EXISTS snapshot_records CASCADE;
DROP TABLE IF EXISTS audit_entries CASCADE;
