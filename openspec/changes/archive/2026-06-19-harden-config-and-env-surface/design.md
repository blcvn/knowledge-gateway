# Design

## Current Behavior

The service loads runtime configuration from environment variables inside `internal/config`. That loader has useful local defaults, but the broader deployment experience is split across three places:

- Go runtime config in `internal/config`
- deployment-time profile defaults in `scripts/runtime-profile.sh`
- operator guidance in `docs/deployment/*`

That split creates drift:

- runtime parsing panics for some malformed values
- public health output includes a full DSN
- Kubernetes does not inject all values implied by runtime profiles
- docs describe defaults or examples that do not match script behavior

## Problem Statement

Operators need one trustworthy configuration contract for startup and deployment. Today, the contract is only partially encoded in code and partially encoded in scripts and prose. That makes configuration less friendly than it should be, especially for Kubernetes and profile-based deployments.

## Goals

- Fail startup with clear configuration errors instead of panics for malformed environment variables.
- Ensure public health output never exposes sensitive configuration secrets.
- Keep deploy scripts, manifests, and docs aligned on required and optional environment variables.
- Document the supported environment surface in one operator-facing place.

## Non-Goals

- Replacing environment variables with a new config file format.
- Reworking runtime profile selection semantics.
- Redesigning backend adapter behavior beyond the config surface needed to start them safely.
- Introducing secret managers or cloud-specific deployment tooling.

## Key Decisions

### 1. Treat startup config parsing as validation, not process control

Environment parsing should feed into `config.Load()` errors rather than crash the process. This preserves fail-fast startup while making mistakes actionable for operators and automation.

### 2. Keep `/healthz` public but restrict it to safe metadata

The public health endpoint can continue to support readiness and liveness probes, but it must not expose credential-bearing DSNs or equivalent sensitive configuration values.

### 3. Make deployment assets honor the same env contract

The runtime profile shell helpers, Compose manifests, Kubernetes manifests, and VM docs should all agree on the same named variables. If a profile defines a graph database name, every supported deployment target must be able to pass it through.

### 4. Add a source-of-truth env inventory

The repository should explicitly list:

- variable name
- purpose
- default value, if any
- whether it is always required or conditionally required
- which deployment paths commonly set it

This inventory should be operator-facing and easy to keep updated alongside runtime changes.

## Risks And Mitigations

- Docs may drift again after future runtime changes.
  - Mitigation: make the env inventory explicit and link it from deployment docs.
- Sanitizing health output too aggressively may reduce troubleshooting value.
  - Mitigation: keep non-sensitive connection metadata such as host, port, adapter kind, or db index where useful.
- Tightening config validation may surface previously hidden deployment misconfigurations.
  - Mitigation: return clear startup errors that name the invalid variable and expected format.

## Validation Strategy

- Verify malformed integer and duration env values return non-panicking errors from config loading.
- Verify the public health payload no longer includes raw DSNs or secrets.
- Verify Kubernetes deployment assets pass through graph database selection along with existing profile variables.
- Verify deployment docs and env inventory match the actual deployment scripts and manifests.
