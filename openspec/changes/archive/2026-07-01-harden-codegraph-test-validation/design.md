# Design: Harden CodeGraph Test Validation Scripts

## Overview

This change is a narrow reliability pass over the repo's test entrypoints. The goal is not to add
new validation logic, but to remove behavior mismatches between documentation, flags, and actual
execution for the CodeGraph flow, and to make the expected sync coverage explicit.

## Design Decisions

### Environment guards

The CodeGraph Compose and validation scripts should treat `EMBEDDING_PROVIDER` the same way they
already treat other required environment variables: validate it explicitly and exit with a helpful
message.

Using `${EMBEDDING_PROVIDER:-}` preserves `set -u` safety while avoiding an unbound-variable abort.

### Verification flag semantics

`validate-codegraph-runtime.sh --skip-verify` should skip both:

- ontology verification immediately after bootstrap or reuse;
- the post-sync verification block later in the script.

This keeps runtime behavior aligned with the script help text and the rerun workflow documented in
`docs/guides/testing.md`.

### Create and update sync coverage

The CodeGraph validation flow should validate two distinct runtime states:

1. a clean or freshly cleared namespace where sync creates CodeGraph nodes for the first time;
2. a rerun against an existing synced symbol where a changed source artifact is pushed as an update.

The second path needs a stronger assertion than “the sync command succeeds”. It should prove the
service observed a new logical graph version for the updated symbol or sync scope.

### Version validation for update

The update path should capture a stable version signal before and after the changed symbol is
synced. The exact probe can be implementation-specific, but the validation must ensure:

- the pre-update symbol is readable after the initial sync;
- a deliberate content change is made to the source fixture or probe entity;
- the subsequent sync updates the same logical entity rather than creating a duplicate;
- the version metadata for that entity or enclosing sync session changes after the update.

This keeps the validation aligned with the repo's graph-versioned write model instead of checking
only surface-level query success.

## Verification Plan

Run the checks that are deterministic in the current workspace:

1. `bash -n` on the touched shell scripts;
2. `go test ./tests/integration/...`;
3. `go test ./examples/codegraph/...` for bridge-level coverage around stale/update behavior;
4. attempt the lightweight CodeGraph validation entrypoint where environment limitations allow.

If Docker is unavailable, record that as an environment limitation rather than broadening the change
scope.
