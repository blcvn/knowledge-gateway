# Tasks

## Milestone: Example Packaging

- [x] **T1** - Finalize the canonical example layout under `examples/codegraph/`, including command
  entrypoint, internal packages, test files, wrapper scripts, README, and example env/config files.
- [x] **T2** - Move the current `codegraph-sync` implementation and its tests into the example layout
  without changing the bridge's HTTP behavior or MCP tool surface.
- [x] **T3** - Update repository-owned command entrypoints such as `Makefile` targets and any
  compatibility wrappers so contributors can still run build, dry-run sync, sync, and MCP flows
  repeatably after the move.

## Milestone: Validation And Runtime Automation

- [x] **T4** - Update `scripts/validate-codegraph-runtime.sh` and any related runtime-validation
  references to use the new example path, including the default state-file location if it moves under
  `examples/codegraph/`.
- [x] **T5** - Run focused bridge tests from the new example package location and confirm the
  repository-owned validation flow still has a single documented CodeGraph path.

## Milestone: Documentation Refresh

- [x] **T6** - Update active documentation that references the bridge location, including `README.md`,
  `docs/codegraph/*`, `docs/guides/*`, `docs/deployment/*`, and any other maintained contributor or
  operator docs that still point at `codegraph-sync/`.
- [x] **T7** - Refresh documentation and guidance wording so the bridge is consistently described as a
  `kg-service` example under `examples/codegraph/`, not as a separate root-level product surface.
- [x] **T8** - Audit `docs/api/README.md`, `docs/api/openapi.yaml`, and `docs/api/maintenance.md`
  against the latest live runtime and update any API-adjacent references or examples that are stale.
- [x] **T9** - Run `bash scripts/check-api-route-inventory.sh` and resolve any route-documentation
  drift discovered during the refresh.
