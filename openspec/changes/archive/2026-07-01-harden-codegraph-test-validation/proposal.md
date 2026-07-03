# Proposal: Harden CodeGraph Test Validation Scripts

## Problem

The repository's integration and CodeGraph validation entrypoints mostly line up with the testing
guide, but the CodeGraph-specific scripts still have a couple of sharp edges:

1. `set -u` causes `deploy-compose-codegraph-runtime.sh` and `validate-codegraph-runtime.sh` to
   abort with an unhelpful unbound-variable error when `EMBEDDING_PROVIDER` is missing.
2. `validate-codegraph-runtime.sh --skip-verify` still runs ontology verification even though the
   flag claims it skips post-bootstrap verification.
3. the CodeGraph validation flow does not yet state or enforce both sync paths that matter in
   practice:
   - an initial create path that writes brand-new `code-graph` nodes;
   - a repeat update path that modifies an already-synced symbol and proves the logical graph
     version advances.

These mismatches make local reruns and CI-like script execution less predictable, especially when
debugging CodeGraph validation flows.

## Proposed Solution

Create a focused change that audits the integration and CodeGraph validation scripts, then hardens
the failing paths without expanding the runtime surface:

1. make CodeGraph script env validation fail with the documented message instead of a shell error;
2. align `--skip-verify` behavior with the script usage text;
3. extend the CodeGraph validation contract so it explicitly covers:
   - first-time sync that creates CodeGraph entities;
   - rerun sync that updates an existing entity;
   - verification that the update path advances the stored graph version metadata;
4. rerun the lightweight integration checks that are available in the current environment.

## Scope

### In scope

- CodeGraph validation shell-script behavior
- integration test audit for repo-local Go coverage
- CodeGraph validation coverage for create and update sync flows
- verification that script syntax and available tests still pass

### Out of scope

- changes to CodeGraph sync semantics
- new integration scenarios
- Compose or Docker infrastructure redesign

## Success Criteria

- missing `EMBEDDING_PROVIDER` now fails with a clear validation message
- `--skip-verify` skips ontology verification as documented
- CodeGraph validation covers both create and update sync paths
- the update validation proves the graph version changes after a symbol mutation is synced
- `go test ./tests/integration/...` passes after the script changes
