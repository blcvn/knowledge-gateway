---
trigger: always_on
---

# Read-Only Services

The following service directories belong to **external projects** and MUST NOT be modified:

## Protected Paths

- `services/kgs-platform/` — KGS Platform (Knowledge Graph Service)

## Rules

1. **No source code changes**: Do NOT create, edit, or delete any files under the protected paths listed above.
2. **Deploy configs are OK**: Files under `deployment/dev/configs/` (e.g., `kgs-platform.yaml`, `ui-knowledge-service.yaml`) are deployment configurations and CAN be modified — they are NOT part of the external project source code.
3. **Docker compose is OK**: Service definitions in `deployment/dev/docker-compose.server.yaml` CAN be modified.
4. **Binary builds are OK**: The `deployment/dev/deploy.sh` script cross-compiles binaries from these services — this is a read-only operation and is allowed.

## When Debugging

- Read source code in protected paths for understanding, but do NOT propose modifications.
- If a bug is found in a protected service, document the issue and suggest the fix as a recommendation to the upstream project owner — do NOT apply the fix directly.
