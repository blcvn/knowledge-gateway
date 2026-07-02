# SQL And Migration Policy

## Taxonomy

- `migration/`: schema changes that must be idempotent and safe to apply once in order.
- `verify/`: read-only checks that confirm a migration landed correctly.
- `backfill/`: data population or reshaping jobs that may run for a long time and may be rerun.
- `repair/`: corrective operations for orphan cleanup, rebuild, or rollback-adjacent recovery.

## Naming

- Use numeric prefixes for ordered migrations, for example `000011_optimize_kg_hot_fks.up.sql`.
- Use verb-first filenames for scripts, for example `verify-kg-hot-fks.sh`, `repair-orphan-vector-docs.sql`.

## Lock And Transaction Rules

- Keep hot-table migrations short and DDL-only.
- Do not mix backfill scans, repair loops, or long-running maintenance inside migration files.
- Use explicit transaction boundaries when a script mutates hot tables.
- Prefer chunked updates and idempotent predicates for repair scripts.

## Rollback Boundary

- `down.sql` is for reversible schema changes only.
- Repairing data drift should happen in repair scripts, not in migration rollback logic.
