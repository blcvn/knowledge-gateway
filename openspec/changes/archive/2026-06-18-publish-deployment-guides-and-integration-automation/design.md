# Design

## Current Behavior

The repository already has strong consumer-facing documentation:

- `docs/guides/` covers quickstart, integration workflows, MCP usage, and troubleshooting.
- `docs/api/` covers the published API contract and maintenance notes.
- `docs/operations/` covers incident recovery and operational response.

That leaves a gap between "how to use the service" and "how to deploy and verify it" for supported runtime targets. There is no first-class deployment guide for:

- Docker Compose as the easy local or single-host stack
- Kubernetes as the cluster deployment path
- VM as the bare-metal or systemd-style deployment path

The repo also contains integration tests under `tests/integration`, but there is no standardized script that operators can run as a post-deploy validation step.

## Problem Statement

Operators need a predictable, repository-owned path to:

- deploy the service into supported environments
- supply the right configuration for each environment
- verify the deployed service with repeatable checks
- understand what is local bootstrap behavior versus what is a deployment requirement

Without that path, deployment steps become tribal knowledge and integration checks drift from one environment to another.

## Goals

- Publish deployment guidance for Docker Compose, Kubernetes, and VM targets.
- Provide executable scripts or entrypoints that make each supported deployment path repeatable.
- Provide a single integration test script that can validate a running deployment.
- Provide a repository-owned command entrypoint and `Makefile` so the same build/run commands can be reused across local and deployed workflows.
- Keep the deployment and validation guidance aligned with the current runtime behavior and existing test surface.

## Non-Goals

- Designing a full CI/CD pipeline.
- Introducing cloud-provider-specific infrastructure templates.
- Reworking service runtime architecture, persistence, or scaling behavior.
- Changing API semantics or integration test assertions as part of this change.

## Key Decisions

### 1. Separate operator deployment docs from consumer guides

The deployment material should live alongside other operator-facing documentation rather than inside the consumer guides. That keeps the usage docs focused on how to call the service and keeps the deploy material focused on how to start and verify it.

### 2. Treat Compose, Kubernetes, and VM as supported deployment surfaces

Each target should have its own documented path, but the guidance should share the same core model:

- required environment variables and configuration inputs
- startup command or manifest entrypoint
- health/readiness verification
- shutdown or redeploy guidance

This avoids having three unrelated sets of instructions.

### 3. Make the scripts the source of truth for repeatable actions

The change should provide scripts or script entrypoints that encode the supported deploy and integration-test workflows. The docs should explain how to use those scripts and what they validate, rather than duplicating shell logic in prose.

### 4. Keep integration validation target-agnostic

The integration test script should work against any reachable deployment target through a base URL or equivalent configuration. That lets the same validation flow run against Compose, Kubernetes, or VM deployments without rewriting the test procedure for each environment.

### 5. Surface bootstrap assumptions explicitly

Some current behaviors are bootstrap conveniences, not production guarantees. The new deployment docs should label those assumptions clearly so that operators do not mistake local shortcuts for mandatory production architecture.

## Risks And Mitigations

- Environment-specific scripts may diverge over time.
  - Mitigation: keep shared options and verification steps consistent across all three targets.
- Kubernetes and VM guidance can become too opinionated if infrastructure details are underspecified.
  - Mitigation: scope the change to repo-owned scripts, manifests, and documented prerequisites rather than trying to model every platform choice.
- Integration scripts can become brittle if they assert too much environment detail.
  - Mitigation: keep the validation focused on service health, auth, and the current core integration path.

### 6. Keep the command surface small and repository-owned

The service should expose a minimal command package with a `serve` subcommand, similar to the pattern used in other Go services in the workspace, while keeping the executable entrypoint at repository root in `main.go`. The `Makefile` should wrap that command along with deploy and validation helpers so operators and contributors have one consistent entrypoint for common actions.

## Validation Strategy

- Verify the Compose path can start the service and pass a health check.
- Verify the Kubernetes path has a documented and repeatable deploy-and-verify flow.
- Verify the VM path has a documented and repeatable deploy-and-verify flow.
- Verify the integration test script can run against a deployed instance and returns a clear non-zero failure on regressions.
- Verify the new docs are linked from the main repository navigation so operators can find them easily.
