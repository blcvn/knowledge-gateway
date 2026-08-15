# Migration duplicate cleanup (version 000014 collision)

## What was wrong

`migrations/` carried two different files at version `000014`:

- `000014_identity_ontology_alignment.{up,down}.sql` — added 2026-07-09 (`2111ae9b`)
- `000014_kg_uuid_identity_backfill.{up,down}.sql` — added 2026-07-07 (`8611eba4`)

golang-migrate refuses to load a source directory with a duplicate version prefix
(`duplicate migration version`). The `migrate/migrate` sidecar in
`deploy/compose/docker-compose.yml` therefore failed before applying anything, which meant the
service could not be brought up from a clean database.

The 2026-07-07 commit had duplicated two earlier migrations verbatim:

| Duplicate (removed) | Original (kept) | Relationship |
|---|---|---|
| `000014_kg_uuid_identity_backfill` | `000010_kg_uuid_identity_backfill` | byte-identical (`md5 839a30b3…`, up and down) |
| `000015_optimize_kg_hot_fks` | `000011_optimize_kg_hot_fks` | byte-identical (`md5 9356690a…`, up and down) |

They carried no content of their own, so removing them loses nothing.

## What was done

1. Deleted `000014_kg_uuid_identity_backfill.{up,down}.sql` and
   `000015_optimize_kg_hot_fks.{up,down}.sql`.
2. Left the highest existing version at `000014` (`identity_ontology_alignment`).
3. Numbered the next migrations **`000016`** and **`000017`**, deliberately skipping `000015`.

## Why `000015` is skipped on purpose

Between 2026-07-07 and 2026-07-09 the migration set was internally consistent
(`…, 000013, 000014_kg_uuid_identity_backfill, 000015_optimize_kg_hot_fks`), so an environment
provisioned in that window can hold `schema_migrations.version = 15`.

If a new migration reused version `000015`, golang-migrate on such an environment would consider
15 already applied and skip straight to 16 — silently omitting the new migration. Skipping the
number removes that failure mode entirely. golang-migrate does not require contiguous versions.

## Effect per environment

| Current `schema_migrations.version` | Result of `migrate up` |
|---|---|
| empty (fresh database) | applies `000001` … `000014`, then `000016`, `000017` |
| `14` (identity alignment applied) | applies `000016`, `000017` |
| `15` (provisioned in the 07-07 → 07-09 window) | applies `000016`, `000017` |

No manual `UPDATE schema_migrations` is required in any case, and no migration is skipped or
re-applied.

`migrate down` from a database sitting at version `15` cannot find file `000015` and will error.
That is expected and harmless: roll forward instead, or, if a rollback below 15 is genuinely
needed, set `schema_migrations.version = 14` first — versions 14-backfill and 15 were duplicates
of 10 and 11, whose effects are already recorded at those versions.

## Guard

`tests/migrations/migrations_test.go` fails the build if a duplicate version prefix reappears, if
an `.up.sql` has no matching `.down.sql`, or if version `000015` is reintroduced.
