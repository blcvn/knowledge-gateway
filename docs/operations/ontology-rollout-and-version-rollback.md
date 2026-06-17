# Ontology Rollout And Version Rollback Runbook

## When To Use

Use this runbook when a new ontology version, node type, relationship type, or template rollout needs to be introduced or rolled back.

## Rollout Steps

1. Create or update the ontology object in the ontology service.
2. Validate schema compatibility and access control for the owning tenant.
3. Activate query templates only after the pattern DSL and parameter schema are validated.
4. Verify the generic read route resolves the active template.

## Rollback Steps

1. Revert the active status of the newly introduced template or version.
2. Restore the previous ontology version or config state.
3. Re-run the read and search smoke tests for the affected domain.

## Validation

- Effective ontology listing reflects the expected versions and active templates.
- Read routes reject inactive templates and accept active ones only.

