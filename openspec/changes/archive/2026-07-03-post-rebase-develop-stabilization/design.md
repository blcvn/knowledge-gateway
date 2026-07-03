# Design: Post-Rebase Develop Stabilization

## Overview

This change is a maintenance-only stabilization pass after rebasing `main` into `develop`.
The goal is to restore a trustworthy baseline before any follow-up feature work continues.

## Goals

- recover a clean compile path across the repo
- reconcile config helpers and call sites that diverged during rebase resolution
- verify no existing functionality regresses under the current automated tests

## Approach

### Config loading reconciliation

Treat `internal/config` as the first stabilization boundary because compile errors there fan out into
bootstrap, workers, observability, and integration packages.

The cleanup should:

- restore any env parsing helper still required by merged config fields;
- update config loading call sites to handle helper return values consistently;
- preserve the current default values and validation behavior.

### Redundant fragment cleanup

When the rebase leaves partial field wiring or dead call patterns behind, prefer the smallest
possible reconciliation:

- keep the merged behavior that matches the current config structs and tests;
- remove stale fragments only when they are no longer part of a reachable or valid code path;
- avoid opportunistic refactors while the goal is stabilization.

## Verification Plan

1. run `go test ./...` from the repo root;
2. confirm config package tests still cover env parsing and validation paths;
3. rely on the existing integration suite to catch unintended behavior changes in write, read, and
   worker flows.
