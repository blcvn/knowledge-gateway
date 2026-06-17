CREATE OR REPLACE FUNCTION create_access_audit_log_partition(partition_start DATE)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    partition_end DATE := (partition_start + INTERVAL '1 month')::DATE;
    partition_name TEXT := format('access_audit_log_%s', to_char(partition_start, 'YYYYMM'));
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF access_audit_log FOR VALUES FROM (%L) TO (%L)',
        partition_name,
        partition_start,
        partition_end
    );
END;
$$;

SELECT create_access_audit_log_partition(date_trunc('month', CURRENT_DATE)::DATE);
SELECT create_access_audit_log_partition((date_trunc('month', CURRENT_DATE) + INTERVAL '1 month')::DATE);
