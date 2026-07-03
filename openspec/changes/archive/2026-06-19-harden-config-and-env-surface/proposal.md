# Harden Config And Environment Surface

## Why

The current configuration and deployment surface is functional for local bootstrap, but it is not friendly enough for repeatable operator use. During review we found four concrete gaps:

- the public health endpoint exposes the full Postgres DSN, including credentials
- the Kubernetes deployment path does not pass `KG_GRAPH_DATABASE`, even though runtime profiles define and some graph backends use it
- deployment docs do not fully match the actual behavior of the deploy scripts
- invalid numeric or duration environment variables can crash the process with a panic instead of returning a configuration error

These gaps make the environment contract harder to trust and harder to operate safely across Compose, Kubernetes, and VM deployments.

## What Changes

- Harden the runtime configuration surface so invalid environment values return actionable startup errors instead of panicking.
- Remove sensitive connection details from the public health payload.
- Align deployment manifests, scripts, and docs on the same environment variable contract, including graph database selection.
- Publish a single operator-facing inventory of supported environment variables, defaults, and when each value is required.

## Capabilities

- Safer public health output
- Predictable startup failures for invalid environment values
- Consistent environment handling across Compose, Kubernetes, and VM targets
- A documented source of truth for config and environment variables

## Impact

- Reduces accidental secret exposure.
- Makes deployment failures easier to diagnose.
- Lowers operator guesswork when switching runtime profiles.
- Gives the team a tracked change for config and environment quality improvements.
