# Tasks

## Milestone: `docs/deployment`

- [x] Add operator-facing deployment guidance for Docker Compose, Kubernetes, and VM targets.
- [x] Document required configuration, ports, dependencies, and health verification for each deployment target.
- [x] Add clear notes that separate bootstrap conveniences from deployment-time requirements.

## Milestone: `scripts/deploy`

- [x] Add a Docker Compose deployment entrypoint or script.
- [x] Add a Kubernetes deployment entrypoint or script.
- [x] Add a VM deployment entrypoint or script.
- [x] Keep the deploy entrypoints aligned on shared inputs such as image selection, environment variables, and verification steps.

## Milestone: `scripts/integration-test`

- [x] Add a repeatable integration test script that can run against a deployed instance.
- [x] Make the script validate the current service health and a core integration path.
- [x] Ensure failures are surfaced with clear exit codes and actionable output.

## Milestone: `cmd-and-makefile`

- [x] Add a repository-owned `cmd` package with a `serve` subcommand.
- [x] Update the binary entrypoint to dispatch through the new command package.
- [x] Add a `Makefile` that wraps build, run, deploy, migration, and integration validation targets.
- [x] Update Dockerfile and deployment launchers so they invoke the `serve` command explicitly where needed.

## Milestone: `docs/alignment`

- [x] Link the new deployment and integration validation docs from the main README or docs index.
- [x] Document how operators should choose between Compose, Kubernetes, and VM deployment paths.
- [x] Add a maintenance note so deploy scripts and validation steps stay aligned with runtime changes.
